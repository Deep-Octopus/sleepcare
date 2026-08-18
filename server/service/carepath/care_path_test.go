package carepath

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	pathreq "github.com/flipped-aurora/gin-vue-admin/server/model/carepath/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type mutableClock struct{ value time.Time }

func (c *mutableClock) Now() time.Time { return c.value }

type acceptingBindingValidator struct{}

func (acceptingBindingValidator) ValidateFrozenBinding(context.Context, uint, []uint, bool) error {
	return nil
}

func TestPreviewAndStartAreIdempotentAndCreateExactlyD1ToD5(t *testing.T) {
	service, db, clock, ctx := newCarePathTestService(t)
	previewRequest := pathreq.PreviewPlan{PlanTemplateVersionID: 201, AnchorAt: clock.value}
	firstPreview, err := service.PreviewPlan(ctx, 301, "preview-key", previewRequest)
	if err != nil {
		t.Fatal(err)
	}
	secondPreview, err := service.PreviewPlan(ctx, 301, "preview-key", previewRequest)
	if err != nil {
		t.Fatal(err)
	}
	if firstPreview.PreviewID != secondPreview.PreviewID || len(firstPreview.Tasks) != 5 {
		t.Fatalf("idempotent preview differs: first=%+v second=%+v", firstPreview, secondPreview)
	}
	assertModelCount(t, db, &pathmodel.PlanPreview{}, 1)

	startRequest := pathreq.StartPlan{ExpectedClientVersion: 1, PreviewID: firstPreview.PreviewID}
	firstStart, err := service.StartPlan(ctx, 301, "start-key", startRequest)
	if err != nil {
		t.Fatal(err)
	}
	secondStart, err := service.StartPlan(ctx, 301, "start-key", startRequest)
	if err != nil {
		t.Fatal(err)
	}
	if firstStart.EnrollmentID != secondStart.EnrollmentID || firstStart.PlanInstanceID != secondStart.PlanInstanceID ||
		firstStart.CareClientID != secondStart.CareClientID || !firstStart.AnchorAt.Equal(secondStart.AnchorAt) ||
		firstStart.Status != secondStart.Status || fmt.Sprint(firstStart.TaskIDs) != fmt.Sprint(secondStart.TaskIDs) ||
		firstStart.Version != secondStart.Version {
		t.Fatalf("idempotent start differs: first=%+v second=%+v", firstStart, secondStart)
	}
	consumedReplay, err := service.StartPlan(ctx, 301, "start-key-2", startRequest)
	if err != nil {
		t.Fatal(err)
	}
	if consumedReplay.PlanInstanceID != firstStart.PlanInstanceID {
		t.Fatalf("consumed preview created another plan: %+v", consumedReplay)
	}
	assertModelCount(t, db, &pathmodel.Enrollment{}, 1)
	assertModelCount(t, db, &pathmodel.PlanInstance{}, 1)
	assertModelCount(t, db, &pathmodel.TaskInstance{}, 5)
	assertModelCount(t, db, &outbox.Event{}, 2)

	var events []pathmodel.CarePathEvent
	if err = db.WithContext(ctx).Order("id ASC").Find(&events).Error; err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].EventType != pathmodel.EventPlanStarted || events[1].EventType != pathmodel.EventTaskOpened {
		t.Fatalf("unexpected event order/types: %+v", events)
	}
	var outboxEvents []outbox.Event
	if err = db.WithContext(ctx).Order("id ASC").Find(&outboxEvents).Error; err != nil {
		t.Fatal(err)
	}
	if len(outboxEvents) != 2 || outboxEvents[0].EventType != pathmodel.EventPlanStarted ||
		outboxEvents[1].EventType != pathmodel.EventTaskOpened || outboxEvents[1].AggregateType != "CareTask" {
		t.Fatalf("unexpected outbox events: %+v", outboxEvents)
	}

	var tasks []pathmodel.TaskInstance
	if err = db.WithContext(ctx).Order("sort ASC").Find(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	for i, task := range tasks {
		wantDay := fmt.Sprintf("D%d", i+1)
		if task.DayCode != wantDay || task.ExecutionRole != pathmodel.ExecutionRoleCareClient {
			t.Fatalf("task %d = %+v", i, task)
		}
		wantOpen := clock.value.Add(time.Duration(i) * 24 * time.Hour)
		wantDue := wantOpen.Add(11 * time.Hour)
		if !task.OpenAt.Equal(wantOpen) || !task.DueAt.Equal(wantDue) {
			t.Fatalf("%s window=%s..%s, want=%s..%s", task.DayCode, task.OpenAt, task.DueAt, wantOpen, wantDue)
		}
		if i == 0 {
			if task.ExecutionStatus != pathmodel.ExecutionOpen || task.ReviewStatus != pathmodel.ReviewNotReady ||
				task.QuestionnaireVersionID == nil || *task.QuestionnaireVersionID != p1D1QuestionnaireVersionID {
				t.Fatalf("D1 binding/status=%+v", task)
			}
			if string(task.BoundRuleVersionIDsJSON) != "[9501]" {
				t.Fatalf("D1 rule binding=%s", task.BoundRuleVersionIDsJSON)
			}
			continue
		}
		if task.ExecutionStatus != pathmodel.ExecutionScheduled || task.ReviewStatus != pathmodel.ReviewNotRequired ||
			task.QuestionnaireVersionID != nil || string(task.BoundRuleVersionIDsJSON) != "[]" {
			t.Fatalf("%s binding/status=%+v", task.DayCode, task)
		}
	}

	changedPreview := previewRequest
	changedPreview.AnchorAt = changedPreview.AnchorAt.Add(time.Minute)
	if _, err = service.PreviewPlan(ctx, 301, "preview-key", changedPreview); carePathCode(err) != pathmodel.CodeIdempotencyConflict {
		t.Fatalf("changed preview request should conflict, got %v", err)
	}
	thirdPreview, err := service.PreviewPlan(ctx, 301, "preview-key-3", changedPreview)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.StartPlan(ctx, 301, "start-key-3", pathreq.StartPlan{ExpectedClientVersion: 2, PreviewID: thirdPreview.PreviewID}); carePathCode(err) != pathmodel.CodeActiveEnrollmentConflict {
		t.Fatalf("second active enrollment should conflict, got %v", err)
	}
}

func TestRecordTaskContactAppendsAuditEventWithoutChangingTaskState(t *testing.T) {
	service, db, clock, ctx := newCarePathTestService(t)
	preview, err := service.PreviewPlan(ctx, 301, "contact-preview", pathreq.PreviewPlan{
		PlanTemplateVersionID: 201,
		AnchorAt:              clock.value,
	})
	if err != nil {
		t.Fatal(err)
	}
	started, err := service.StartPlan(ctx, 301, "contact-start", pathreq.StartPlan{
		ExpectedClientVersion: 1,
		PreviewID:             preview.PreviewID,
	})
	if err != nil {
		t.Fatal(err)
	}
	taskID := started.TaskIDs[0]
	request := pathreq.TaskContactRecord{
		ExpectedVersion: 1,
		Channel:         pathmodel.ContactChannelPhone,
		Result:          "已完成固定流程联系，等待后续人工确认。",
		OccurredAt:      clock.value.Add(20 * time.Minute),
	}
	first, err := service.RecordTaskContact(ctx, taskID, "contact-key", request)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := service.RecordTaskContact(ctx, taskID, "contact-key", request)
	if err != nil {
		t.Fatal(err)
	}
	if first.ResourceID != replayed.ResourceID || first.ActionID != replayed.ActionID ||
		first.Status != replayed.Status || first.Version != replayed.Version ||
		!first.OccurredAt.Equal(replayed.OccurredAt) ||
		first.Status != pathmodel.ExecutionOpen || first.Version != 2 {
		t.Fatalf("unexpected contact result: first=%+v replay=%+v", first, replayed)
	}
	var task pathmodel.TaskInstance
	if err = db.WithContext(ctx).First(&task, taskID).Error; err != nil {
		t.Fatal(err)
	}
	if task.ExecutionStatus != pathmodel.ExecutionOpen || task.Version != 2 {
		t.Fatalf("contact changed task lifecycle: %+v", task)
	}
	var event pathmodel.CarePathEvent
	if err = db.WithContext(ctx).Where("task_instance_id = ? AND event_type = ?", taskID, pathmodel.EventTaskContactRecorded).
		First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Channel != pathmodel.ContactChannelPhone || event.Reason != request.Result {
		t.Fatalf("structured contact event differs: %+v", event)
	}
	detail, err := service.GetTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	last := detail.Timeline[len(detail.Timeline)-1]
	if last.EventType != pathmodel.EventTaskContactRecorded || !strings.Contains(last.Summary, pathmodel.ContactChannelPhone) {
		t.Fatalf("contact timeline missing: %+v", detail.Timeline)
	}
	_, err = service.RecordTaskContact(ctx, taskID, "contact-stale", request)
	if carePathCode(err) != pathmodel.CodeVersionConflict {
		t.Fatalf("stale contact version should conflict, got %v", err)
	}
}

func TestStartPlanRollsBackWhenOutboxWriteFails(t *testing.T) {
	service, db, clock, ctx := newCarePathTestService(t)
	preview, err := service.PreviewPlan(ctx, 301, "preview-rollback", pathreq.PreviewPlan{
		PlanTemplateVersionID: 201,
		AnchorAt:              clock.value,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.WithContext(datascope.WithSystem(context.Background())).Migrator().DropTable(&outbox.Event{}); err != nil {
		t.Fatal(err)
	}
	_, err = service.StartPlan(ctx, 301, "start-rollback", pathreq.StartPlan{
		ExpectedClientVersion: 1,
		PreviewID:             preview.PreviewID,
	})
	if err == nil {
		t.Fatal("expected missing outbox table to fail plan start")
	}
	assertModelCount(t, db, &pathmodel.Enrollment{}, 0)
	assertModelCount(t, db, &pathmodel.PlanInstance{}, 0)
	assertModelCount(t, db, &pathmodel.TaskInstance{}, 0)
	assertModelCount(t, db, &pathmodel.CarePathEvent{}, 0)
	var startReceipts int64
	if err = db.WithContext(datascope.WithSystem(context.Background())).Model(&pathmodel.CommandReceipt{}).
		Where("operation = ?", operation("START_PLAN", 301)).Count(&startReceipts).Error; err != nil {
		t.Fatal(err)
	}
	if startReceipts != 0 {
		t.Fatalf("failed start persisted %d command receipts", startReceipts)
	}
	var client caremodel.CareClient
	if err = db.WithContext(datascope.WithSystem(context.Background())).Where("id = ?", 301).First(&client).Error; err != nil {
		t.Fatal(err)
	}
	if client.Version != 1 {
		t.Fatalf("failed start changed client version to %d", client.Version)
	}
	var persistedPreview pathmodel.PlanPreview
	if err = db.WithContext(datascope.WithSystem(context.Background())).Where("preview_id = ?", preview.PreviewID).First(&persistedPreview).Error; err != nil {
		t.Fatal(err)
	}
	if persistedPreview.ConsumedAt != nil || persistedPreview.PlanInstanceID != nil {
		t.Fatalf("failed start consumed preview: %+v", persistedPreview)
	}
}

func TestPauseResumeKeepsWindowsAndOnlyOpensDueTasksAfterResume(t *testing.T) {
	service, db, clock, ctx := newCarePathTestService(t)
	preview, err := service.PreviewPlan(ctx, 301, "preview", pathreq.PreviewPlan{PlanTemplateVersionID: 201, AnchorAt: clock.value})
	if err != nil {
		t.Fatal(err)
	}
	started, err := service.StartPlan(ctx, 301, "start", pathreq.StartPlan{ExpectedClientVersion: 1, PreviewID: preview.PreviewID})
	if err != nil {
		t.Fatal(err)
	}
	var before []pathmodel.TaskInstance
	if err = db.WithContext(ctx).Where("plan_instance_id = ?", started.PlanInstanceID).Order("sort ASC").Find(&before).Error; err != nil {
		t.Fatal(err)
	}
	paused, err := service.PausePlan(ctx, started.PlanInstanceID, "pause", pathreq.PlanStateAction{ExpectedVersion: 1, Reason: "合成暂停验证"})
	if err != nil || paused.Status != pathmodel.EnrollmentPaused {
		t.Fatalf("pause=%+v err=%v", paused, err)
	}
	pausedReplay, err := service.PausePlan(ctx, started.PlanInstanceID, "pause", pathreq.PlanStateAction{ExpectedVersion: 1, Reason: "合成暂停验证"})
	if err != nil || pausedReplay != paused {
		t.Fatalf("idempotent pause replay=%+v err=%v", pausedReplay, err)
	}
	clock.value = clock.value.Add(25 * time.Hour)
	if err = service.ReconcilePlanTasks(ctx, started.PlanInstanceID); err != nil {
		t.Fatal(err)
	}
	var whilePaused pathmodel.TaskInstance
	if err = db.WithContext(ctx).Where("plan_instance_id = ? AND day_code = ?", started.PlanInstanceID, "D2").First(&whilePaused).Error; err != nil {
		t.Fatal(err)
	}
	if whilePaused.ExecutionStatus != pathmodel.ExecutionScheduled {
		t.Fatalf("paused plan opened D2: %+v", whilePaused)
	}
	resumed, err := service.ResumePlan(ctx, started.PlanInstanceID, "resume", pathreq.PlanStateAction{ExpectedVersion: 2, Reason: "合成恢复验证"})
	if err != nil || resumed.Status != pathmodel.EnrollmentActive || resumed.Version != 3 {
		t.Fatalf("resume=%+v err=%v", resumed, err)
	}
	var after []pathmodel.TaskInstance
	if err = db.WithContext(ctx).Where("plan_instance_id = ?", started.PlanInstanceID).Order("sort ASC").Find(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after[1].ExecutionStatus != pathmodel.ExecutionOpen || after[2].ExecutionStatus != pathmodel.ExecutionScheduled {
		t.Fatalf("unexpected resume reconciliation: D2=%s D3=%s", after[1].ExecutionStatus, after[2].ExecutionStatus)
	}
	for i := range before {
		if !before[i].OpenAt.Equal(after[i].OpenAt) || !before[i].DueAt.Equal(after[i].DueAt) {
			t.Fatalf("KEEP_WINDOWS shifted %s", before[i].DayCode)
		}
	}
	var taskOpenedOutboxCount int64
	if err = db.WithContext(ctx).Model(&outbox.Event{}).Where("event_type = ?", pathmodel.EventTaskOpened).Count(&taskOpenedOutboxCount).Error; err != nil {
		t.Fatal(err)
	}
	if taskOpenedOutboxCount != 2 {
		t.Fatalf("TaskOpened outbox count=%d, want=2", taskOpenedOutboxCount)
	}
	var events []pathmodel.CarePathEvent
	if err = db.WithContext(ctx).Where("plan_instance_id = ?", started.PlanInstanceID).Order("id ASC").Find(&events).Error; err != nil {
		t.Fatal(err)
	}
	if len(events) != 5 || events[3].EventType != pathmodel.EventPlanResumed || events[4].EventType != pathmodel.EventTaskOpened {
		t.Fatalf("resume/task-open event order=%+v", events)
	}
}

func TestSystemReconciliationKeepsEventAndOutboxDepartmentOwnership(t *testing.T) {
	service, db, clock, ctx := newCarePathTestService(t)
	preview, err := service.PreviewPlan(ctx, 301, "preview-system", pathreq.PreviewPlan{PlanTemplateVersionID: 201, AnchorAt: clock.value})
	if err != nil {
		t.Fatal(err)
	}
	started, err := service.StartPlan(ctx, 301, "start-system", pathreq.StartPlan{ExpectedClientVersion: 1, PreviewID: preview.PreviewID})
	if err != nil {
		t.Fatal(err)
	}
	clock.value = clock.value.Add(25 * time.Hour)
	if err = service.ReconcilePlanTasks(datascope.WithSystem(context.Background()), started.PlanInstanceID); err != nil {
		t.Fatal(err)
	}

	var event pathmodel.CarePathEvent
	if err = db.WithContext(datascope.WithSystem(context.Background())).Where(
		"plan_instance_id = ? AND task_instance_id IS NOT NULL AND event_type = ?", started.PlanInstanceID, pathmodel.EventTaskOpened,
	).Order("id DESC").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.DeptId != 10 {
		t.Fatalf("system TaskOpened event dept_id=%d, want=10", event.DeptId)
	}
	var outboxEvent outbox.Event
	if err = db.WithContext(datascope.WithSystem(context.Background())).Where(
		"aggregate_type = ? AND aggregate_id = ? AND event_type = ?", "CareTask", fmt.Sprint(started.TaskIDs[1]), pathmodel.EventTaskOpened,
	).First(&outboxEvent).Error; err != nil {
		t.Fatal(err)
	}
	if outboxEvent.DeptId != 10 {
		t.Fatalf("system TaskOpened outbox dept_id=%d, want=10", outboxEvent.DeptId)
	}
}

func TestCarePathCommandValidationRejectsOversizedKeysAndReasons(t *testing.T) {
	service, _, clock, ctx := newCarePathTestService(t)
	longKey := strings.Repeat("k", 129)
	if _, err := service.PreviewPlan(ctx, 301, longKey, pathreq.PreviewPlan{PlanTemplateVersionID: 201, AnchorAt: clock.value}); carePathCode(err) != pathmodel.CodeInvalidArgument {
		t.Fatalf("oversized idempotency key should fail, got %v", err)
	}
	preview, err := service.PreviewPlan(ctx, 301, "preview", pathreq.PreviewPlan{PlanTemplateVersionID: 201, AnchorAt: clock.value})
	if err != nil {
		t.Fatal(err)
	}
	started, err := service.StartPlan(ctx, 301, "start", pathreq.StartPlan{ExpectedClientVersion: 1, PreviewID: preview.PreviewID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.PausePlan(ctx, started.PlanInstanceID, "pause", pathreq.PlanStateAction{ExpectedVersion: 1, Reason: strings.Repeat("r", 1001)}); carePathCode(err) != pathmodel.CodeInvalidArgument {
		t.Fatalf("oversized reason should fail, got %v", err)
	}
	if _, err = service.PausePlan(ctx, started.PlanInstanceID, "pause-unicode", pathreq.PlanStateAction{ExpectedVersion: 1, Reason: strings.Repeat("合", 1001)}); carePathCode(err) != pathmodel.CodeInvalidArgument {
		t.Fatalf("oversized unicode reason should fail, got %v", err)
	}
}

func TestPreviewRejectsTamperedPathAndTaskDefinitions(t *testing.T) {
	t.Run("path hash", func(t *testing.T) {
		service, db, clock, ctx := newCarePathTestService(t)
		if err := db.WithContext(datascope.WithSystem(context.Background())).Model(&pathmodel.PathDefinitionVersion{}).
			Where("id = ?", 200).Update("title", "被篡改的合成路径").Error; err != nil {
			t.Fatal(err)
		}
		_, err := service.PreviewPlan(ctx, 301, "tampered-path", pathreq.PreviewPlan{PlanTemplateVersionID: 201, AnchorAt: clock.value})
		if carePathCode(err) != pathmodel.CodeContentDisabled {
			t.Fatalf("tampered path should be rejected, got %v", err)
		}
	})

	t.Run("task hash", func(t *testing.T) {
		service, db, clock, ctx := newCarePathTestService(t)
		if err := db.WithContext(datascope.WithSystem(context.Background())).Model(&pathmodel.PlanTaskDefinition{}).
			Where("id = ?", 210).Update("title", "被篡改的合成任务").Error; err != nil {
			t.Fatal(err)
		}
		_, err := service.PreviewPlan(ctx, 301, "tampered-task", pathreq.PreviewPlan{PlanTemplateVersionID: 201, AnchorAt: clock.value})
		if carePathCode(err) != pathmodel.CodeContentDisabled {
			t.Fatalf("tampered task should be rejected, got %v", err)
		}
	})

	t.Run("out-of-scope D1 binding", func(t *testing.T) {
		service, db, clock, ctx := newCarePathTestService(t)
		wrongQuestionnaireID := uint(9402)
		systemDB := db.WithContext(datascope.WithSystem(context.Background()))
		if err := systemDB.Model(&pathmodel.PlanTaskDefinition{}).Where("id = ?", 210).
			Update("questionnaire_version_id", wrongQuestionnaireID).Error; err != nil {
			t.Fatal(err)
		}
		value, err := service.loadTemplate(ctx, db, 201)
		if err != nil {
			t.Fatal(err)
		}
		document, err := definitionDocument(value)
		if err != nil {
			t.Fatal(err)
		}
		hash, err := pathmodel.HashDefinition(document)
		if err != nil {
			t.Fatal(err)
		}
		if err = systemDB.Model(&pathmodel.PlanTemplateVersion{}).Where("id = ?", 201).
			Update("definition_hash", hash).Error; err != nil {
			t.Fatal(err)
		}
		_, err = service.PreviewPlan(ctx, 301, "wrong-d1-binding", pathreq.PreviewPlan{PlanTemplateVersionID: 201, AnchorAt: clock.value})
		if carePathCode(err) != pathmodel.CodeContentDisabled {
			t.Fatalf("out-of-scope D1 binding should be rejected, got %v", err)
		}
	})
}

func TestPlanWritesRequireCurrentStewardOrClinicianResponsibility(t *testing.T) {
	service, db, clock, stewardCtx := newCarePathTestService(t)
	supervisorCtx := identityContext(3, 103, caremodel.AuthorityRoleSupervisor)
	if _, _, err := service.ListPlanVersions(supervisorCtx, pathreq.PlanVersionSearch{}); err != nil {
		t.Fatalf("supervisor should read plan versions: %v", err)
	}
	if _, err := service.PreviewPlan(supervisorCtx, 301, "supervisor-preview", pathreq.PreviewPlan{PlanTemplateVersionID: 201, AnchorAt: clock.value}); carePathCode(err) != pathmodel.CodeAccessScopeDenied {
		t.Fatalf("supervisor write should be denied, got %v", err)
	}
	unassignedCtx := identityContext(2, 101, caremodel.AuthorityRoleCareSteward)
	if _, err := service.PreviewPlan(unassignedCtx, 301, "unassigned-preview", pathreq.PreviewPlan{PlanTemplateVersionID: 201, AnchorAt: clock.value}); carePathCode(err) != pathmodel.CodeAccessScopeDenied {
		t.Fatalf("same-team unassigned write should be denied, got %v", err)
	}
	if _, err := service.PreviewPlan(stewardCtx, 301, "steward-preview", pathreq.PreviewPlan{PlanTemplateVersionID: 201, AnchorAt: clock.value}); err != nil {
		t.Fatalf("responsible steward should preview: %v", err)
	}
	clinicianAssignment := caremodel.CareAssignment{
		CareClientID: 301, OrganizationID: 10, TeamID: 10, AssigneeID: 2,
		RoleType: caremodel.AssignmentRoleClinician, ValidFrom: clock.value.Add(-time.Hour),
		Reason: "合成医护责任关系", Synthetic: true, DeptId: 10, CreatedBy: 3,
	}
	if err := db.WithContext(datascope.WithSystem(context.Background())).Create(&clinicianAssignment).Error; err != nil {
		t.Fatal(err)
	}
	clinicianCtx := identityContext(2, 102, caremodel.AuthorityRoleClinician)
	if _, err := service.PreviewPlan(clinicianCtx, 301, "clinician-preview", pathreq.PreviewPlan{PlanTemplateVersionID: 201, AnchorAt: clock.value}); err != nil {
		t.Fatalf("responsible clinician should preview: %v", err)
	}
}

func TestRuntimeReadsAllowSupervisorAndDenyUnassignedOrUnmappedRoles(t *testing.T) {
	service, _, clock, stewardCtx := newCarePathTestService(t)
	preview, err := service.PreviewPlan(stewardCtx, 301, "preview-read-scope", pathreq.PreviewPlan{PlanTemplateVersionID: 201, AnchorAt: clock.value})
	if err != nil {
		t.Fatal(err)
	}
	started, err := service.StartPlan(stewardCtx, 301, "start-read-scope", pathreq.StartPlan{ExpectedClientVersion: 1, PreviewID: preview.PreviewID})
	if err != nil {
		t.Fatal(err)
	}

	supervisorCtx := identityContext(3, 103, caremodel.AuthorityRoleSupervisor)
	plans, err := service.ListClientPlans(supervisorCtx, 301)
	if err != nil || len(plans) != 1 {
		t.Fatalf("supervisor plan read failed: plans=%+v err=%v", plans, err)
	}
	tasks, total, err := service.ListTasks(supervisorCtx, pathreq.TaskSearch{})
	if err != nil || total != 5 || len(tasks) != 5 {
		t.Fatalf("supervisor task read failed: total=%d tasks=%d err=%v", total, len(tasks), err)
	}
	if _, err = service.GetTask(supervisorCtx, started.TaskIDs[0]); err != nil {
		t.Fatalf("supervisor task detail failed: %v", err)
	}

	unassignedCtx := identityContext(2, 101, caremodel.AuthorityRoleCareSteward)
	if _, err = service.ListClientPlans(unassignedCtx, 301); carePathCode(err) != pathmodel.CodeAccessScopeDenied {
		t.Fatalf("unassigned steward plan read should fail closed, got %v", err)
	}
	tasks, total, err = service.ListTasks(unassignedCtx, pathreq.TaskSearch{})
	if err != nil || total != 0 || len(tasks) != 0 {
		t.Fatalf("unassigned steward task list should be empty: total=%d tasks=%d err=%v", total, len(tasks), err)
	}
	if _, err = service.GetTask(unassignedCtx, started.TaskIDs[0]); carePathCode(err) != pathmodel.CodeAccessScopeDenied {
		t.Fatalf("unassigned steward task detail should fail closed, got %v", err)
	}

	adminCtx := identityContext(8, 888, "")
	if _, _, err = service.ListTasks(adminCtx, pathreq.TaskSearch{}); carePathCode(err) != pathmodel.CodeAccessScopeDenied {
		t.Fatalf("unmapped administrator should be denied, got %v", err)
	}
	if _, _, err = service.ListPlanVersions(context.Background(), pathreq.PlanVersionSearch{}); carePathCode(err) != pathmodel.CodeAccessScopeDenied {
		t.Fatalf("missing identity should be denied, got %v", err)
	}
}

func TestTaskTimingStatusUsesInjectedInstantAndDenyPolicy(t *testing.T) {
	openAt := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	task := pathmodel.TaskInstance{OpenAt: openAt, DueAt: openAt.Add(11 * time.Hour), LateSubmissionPolicy: pathmodel.LateSubmissionDeny}
	if got := task.TimingStatus(openAt.Add(-time.Nanosecond)); got != pathmodel.TimingNotOpen {
		t.Fatalf("before open=%s", got)
	}
	if got := task.TimingStatus(openAt); got != pathmodel.TimingWithinWindow {
		t.Fatalf("at open=%s", got)
	}
	if got := task.TimingStatus(task.DueAt); got != pathmodel.TimingExpired {
		t.Fatalf("at due with DENY=%s", got)
	}
}

func TestRuntimeQueriesAndCommandsExcludeNonSyntheticRows(t *testing.T) {
	service, db, clock, ctx := newCarePathTestService(t)
	systemCtx := datascope.WithSystem(context.Background())
	plan := pathmodel.PlanInstance{
		GVA_MODEL: global.GVA_MODEL{ID: 801}, EnrollmentID: 802, CareClientID: 301,
		PlanTemplateVersionID: 201, PreviewID: 803, AnchorAt: clock.value,
		Status: pathmodel.EnrollmentActive, PauseStrategy: pathmodel.PauseStrategyKeepWindows,
		Version: 1, Synthetic: false, DeptId: 10, CreatedBy: 1,
	}
	if err := db.WithContext(systemCtx).Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	task := pathmodel.TaskInstance{
		GVA_MODEL: global.GVA_MODEL{ID: 804}, PlanInstanceID: plan.ID, CareClientID: 301,
		TaskDefinitionID: 210, DayCode: "D1", Title: "非合成任务", Sort: 1,
		ExecutionRole: pathmodel.ExecutionRoleCareClient, ExecutionStatus: pathmodel.ExecutionScheduled,
		ReviewStatus: pathmodel.ReviewNotRequired, OpenAt: clock.value, DueAt: clock.value.Add(11 * time.Hour),
		BoundRuleVersionIDsJSON: datatypes.JSON([]byte("[]")), LateSubmissionPolicy: pathmodel.LateSubmissionDeny,
		NotificationPolicy: pathmodel.NotificationPolicyDisabled, Version: 1, Synthetic: false,
		DeptId: 10, CreatedBy: 1,
	}
	if err := db.WithContext(systemCtx).Create(&task).Error; err != nil {
		t.Fatal(err)
	}

	plans, err := service.ListClientPlans(ctx, 301)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 0 {
		t.Fatalf("non-synthetic plan leaked through list: %+v", plans)
	}
	tasks, total, err := service.ListTasks(ctx, pathreq.TaskSearch{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(tasks) != 0 {
		t.Fatalf("non-synthetic task leaked through list: total=%d tasks=%+v", total, tasks)
	}
	if _, err = service.GetTask(ctx, task.ID); carePathCode(err) != pathmodel.CodeAccessScopeDenied {
		t.Fatalf("non-synthetic task detail should fail closed, got %v", err)
	}
	if _, err = service.PausePlan(ctx, plan.ID, "pause-non-synthetic", pathreq.PlanStateAction{ExpectedVersion: 1, Reason: "合成边界验证"}); carePathCode(err) != pathmodel.CodeAccessScopeDenied {
		t.Fatalf("non-synthetic plan action should fail closed, got %v", err)
	}
	if err = service.ReconcilePlanTasks(systemCtx, plan.ID); carePathCode(err) != pathmodel.CodeResourceNotFound {
		t.Fatalf("non-synthetic plan reconciliation should fail closed, got %v", err)
	}
}

func newCarePathTestService(t *testing.T) (*CarePathService, *gorm.DB, *mutableClock, context.Context) {
	t.Helper()
	models := []any{
		&caremodel.CareClient{}, &caremodel.CareAssignment{}, &caremodel.CareAuthorityProfile{},
		&pathmodel.PathDefinitionVersion{}, &pathmodel.PlanTemplateVersion{}, &pathmodel.PlanTaskDefinition{},
		&pathmodel.PlanTaskDependency{}, &pathmodel.Enrollment{}, &pathmodel.PlanPreview{},
		&pathmodel.PlanInstance{}, &pathmodel.TaskInstance{}, &pathmodel.CarePathEvent{},
		&pathmodel.CommandReceipt{}, &outbox.Event{}, testutil.WithDataScopeCallbacks(),
	}
	db := testutil.NewMemoryDB(t, models...)
	fixed := time.Date(2026, 8, 18, 9, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	clock := &mutableClock{value: fixed}
	enabled := true
	service := &CarePathService{DB: db, Clock: clock, BindingValidator: acceptingBindingValidator{}, SyntheticFixturesEnabled: &enabled}
	systemCtx := datascope.WithSystem(context.Background())
	profiles := []caremodel.CareAuthorityProfile{
		{AuthorityID: 101, RoleType: caremodel.AuthorityRoleCareSteward, Active: true, Synthetic: true},
		{AuthorityID: 102, RoleType: caremodel.AuthorityRoleClinician, Active: true, Synthetic: true},
		{AuthorityID: 103, RoleType: caremodel.AuthorityRoleSupervisor, Active: true, Synthetic: true},
	}
	if err := db.WithContext(systemCtx).Create(&profiles).Error; err != nil {
		t.Fatal(err)
	}
	client := caremodel.CareClient{
		GVA_MODEL: global.GVA_MODEL{ID: 301}, DisplayCode: "SYN-CLIENT-301", DisplayName: "[合成] 测试用户",
		OrganizationID: 10, Status: caremodel.ClientStatusActive, SensitivityLevel: caremodel.SensitivitySensitive,
		Synthetic: true, Version: 1, DeptId: 10, CreatedBy: 1,
	}
	if err := db.WithContext(systemCtx).Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	assignment := caremodel.CareAssignment{
		CareClientID: client.ID, OrganizationID: 10, TeamID: 10, AssigneeID: 1,
		RoleType: caremodel.AssignmentRoleCareSteward, ValidFrom: fixed.Add(-time.Hour),
		Reason: "合成测试责任关系", Synthetic: true, DeptId: 10, CreatedBy: 3,
	}
	if err := db.WithContext(systemCtx).Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	seedTestDefinition(t, db.WithContext(systemCtx), fixed)
	return service, db, clock, identityContext(1, 101, caremodel.AuthorityRoleCareSteward)
}

func seedTestDefinition(t *testing.T, db *gorm.DB, fixed time.Time) {
	t.Helper()
	path := pathmodel.PathDefinitionVersion{
		GVA_MODEL: global.GVA_MODEL{ID: 200}, Code: p1PathCode, Version: p1PathVersion,
		Title: "合成路径", Purpose: "合成软件测试", Status: pathmodel.LifecyclePublished,
		UsageScope: pathmodel.UsageScopeTestOnly, Synthetic: true, ReviewType: pathmodel.ReviewTypeEngineering,
		ReviewedBy: 3, ReviewedAt: &fixed, PublishedAt: &fixed, DefinitionHash: "test", RowVersion: 1,
	}
	pathHash, err := pathmodel.HashDefinition(pathmodel.PathDefinitionDocument{
		Code: path.Code, Version: path.Version, Title: path.Title, Purpose: path.Purpose,
		UsageScope: path.UsageScope, Synthetic: path.Synthetic, ProductionEnabled: path.ProductionEnabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	path.DefinitionHash = pathHash
	template := pathmodel.PlanTemplateVersion{
		GVA_MODEL: global.GVA_MODEL{ID: 201}, PathDefinitionVersionID: path.ID,
		Code: p1PlanTemplateCode, Version: p1PlanTemplateVersion, Title: "合成计划", Purpose: "合成软件测试",
		Status: pathmodel.LifecyclePublished, UsageScope: pathmodel.UsageScopeTestOnly,
		Synthetic: true, ReviewType: pathmodel.ReviewTypeEngineering, ReviewedAt: &fixed, PublishedAt: &fixed,
		ReviewedBy:           3,
		AnchorDefinition:     pathmodel.AnchorFirstValidSyntheticDeviceUse,
		LateSubmissionPolicy: pathmodel.LateSubmissionDeny, PauseStrategy: pathmodel.PauseStrategyKeepWindows,
		DefinitionSchemaVersion: "v1", RowVersion: 1,
	}
	tasks := make([]pathmodel.PlanTaskDefinition, 0, 5)
	for i := 0; i < 5; i++ {
		definition := pathmodel.PlanTaskDefinition{
			GVA_MODEL: global.GVA_MODEL{ID: uint(210 + i)}, PlanTemplateVersionID: template.ID,
			DayCode: fmt.Sprintf("D%d", i+1), Title: fmt.Sprintf("D%d 合成任务", i+1), Sort: i + 1,
			ExecutionRole: pathmodel.ExecutionRoleCareClient, OpenOffsetSeconds: int64(i) * 24 * 60 * 60,
			DueOffsetSeconds:        int64(i)*24*60*60 + 11*60*60,
			BoundRuleVersionIDsJSON: datatypes.JSON([]byte("[]")), NotificationPolicy: pathmodel.NotificationPolicyDisabled,
		}
		if i == 0 {
			questionnaireID := p1D1QuestionnaireVersionID
			definition.QuestionnaireVersionID = &questionnaireID
			definition.BoundRuleVersionIDsJSON = datatypes.JSON([]byte("[9501]"))
			definition.ReviewRequired = true
			definition.ReviewRole = pathmodel.ExecutionRoleClinician
		}
		tasks = append(tasks, definition)
	}
	value := loadedTemplate{Path: path, Template: template, Tasks: tasks}
	document, err := definitionDocument(value)
	if err != nil {
		t.Fatal(err)
	}
	template.DefinitionHash, err = pathmodel.HashDefinition(document)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&path).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&template).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
}

func identityContext(userID, authorityID uint, _ string) context.Context {
	return datascope.WithIdentity(context.Background(), &datascope.Identity{
		UserID: userID, AuthorityID: authorityID, DeptID: 10, DeptIDs: []uint{10}, VisibleDeptIDs: []uint{10}, Scope: datascope.ScopeDept,
	})
}

func carePathCode(err error) int {
	var domainErr *pathmodel.DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code
	}
	return 0
}

func assertModelCount(t *testing.T, db *gorm.DB, model any, want int64) {
	t.Helper()
	var got int64
	if err := db.WithContext(datascope.WithSystem(context.Background())).Model(model).Count(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%T count=%d, want=%d", model, got, want)
	}
}
