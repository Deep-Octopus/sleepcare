package initialize

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	platformoutbox "github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	notificationmodel "github.com/flipped-aurora/gin-vue-admin/server/model/notification"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	notificationservice "github.com/flipped-aurora/gin-vue-admin/server/service/notification"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	scenarioBEnrollmentID  = 22002
	scenarioBPreviewID     = 22902
	scenarioBPlanID        = 23002
	scenarioBTaskD1ID      = 24101
	scenarioARequestID     = 27901
	scenarioBRequestID     = 27902
	scenarioAAttemptID     = 28001
	scenarioBAttempt1ID    = 28101
	scenarioBAttempt2ID    = 28102
	scenarioBPlanEventID   = 24201
	scenarioBTaskEventID   = 24202
	scenarioBPreviewOpaque = "p1-09-scenario-b-preview"
)

var notificationAPIs = []system.SysApi{
	{ApiGroup: "通知记录", Method: "GET", Path: "/care/deliveries", Description: "查询授权范围通知尝试"},
	{ApiGroup: "通知记录", Method: "POST", Path: "/care/deliveries/:id/resend", Description: "为终态通知创建补发尝试"},
	{ApiGroup: "康养任务", Method: "POST", Path: "/care/tasks/:id/contact-records", Description: "追加人工联系记录"},
}

func EnsureNotificationData() error {
	if global.GVA_DB == nil {
		return nil
	}
	ctx := datascope.WithSystem(context.Background())
	db := global.GVA_DB.WithContext(ctx)
	if err := ensureNotificationMetadata(db, global.GVA_CONFIG.Care.SyntheticFixturesEnabled); err != nil {
		return err
	}
	if !global.GVA_CONFIG.Care.SyntheticFixturesEnabled {
		return nil
	}
	if err := ensureScenarioBPlan(db); err != nil {
		return err
	}
	return ensureNotificationFixtures(ctx, db)
}

func ensureNotificationMetadata(db *gorm.DB, grantPolicies bool) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, api := range notificationAPIs {
			if err := tx.Where("path = ? AND method = ?", api.Path, api.Method).
				Attrs(api).FirstOrCreate(&system.SysApi{}).Error; err != nil {
				return fmt.Errorf("ensure notification API %s: %w", api.Path, err)
			}
		}
		var root, executionMenu, taskMenu system.SysBaseMenu
		if err := tx.Where("name = ?", "SleepCare").First(&root).Error; err != nil {
			return err
		}
		if err := tx.Where("name = ?", "CareExecution").First(&executionMenu).Error; err != nil {
			return err
		}
		if err := tx.Where("name = ?", "CareTasks").First(&taskMenu).Error; err != nil {
			return err
		}
		deliveryMenu := system.SysBaseMenu{
			ParentId: executionMenu.ID, Path: "care-deliveries", Name: "CareDeliveries",
			Component: "view/sleep-care/deliveries/index.vue", Sort: 3,
			Meta: system.Meta{Title: "通知记录", Icon: "notification"},
		}
		if err := ensureCaseWorkMenu(tx, &deliveryMenu); err != nil {
			return err
		}
		resendButton := system.SysBaseMenuBtn{Name: "resendNotice", Desc: "创建补发尝试", SysBaseMenuID: deliveryMenu.ID}
		if err := tx.Where("name = ? AND sys_base_menu_id = ?", resendButton.Name, deliveryMenu.ID).
			Attrs(resendButton).FirstOrCreate(&resendButton).Error; err != nil {
			return err
		}
		contactButton := system.SysBaseMenuBtn{Name: "recordContact", Desc: "追加人工联系记录", SysBaseMenuID: taskMenu.ID}
		if err := tx.Where("name = ? AND sys_base_menu_id = ?", contactButton.Name, taskMenu.ID).
			Attrs(contactButton).FirstOrCreate(&contactButton).Error; err != nil {
			return err
		}
		if !grantPolicies {
			return nil
		}
		roles := []uint{syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole}
		for _, role := range roles {
			for _, menuID := range []uint{root.ID, executionMenu.ID, deliveryMenu.ID} {
				link := system.SysAuthorityMenu{MenuId: fmt.Sprint(menuID), AuthorityId: fmt.Sprint(role)}
				if err := tx.Where(link).FirstOrCreate(&link).Error; err != nil {
					return err
				}
			}
		}
		if err := ensureAuthorityButton(tx, syntheticStewardRole, deliveryMenu.ID, resendButton.ID); err != nil {
			return err
		}
		for _, role := range []uint{syntheticStewardRole, syntheticClinicianRole} {
			if err := ensureAuthorityButton(tx, role, taskMenu.ID, contactButton.ID); err != nil {
				return err
			}
		}
		rolesByRoute := map[string][]uint{
			"GET /care/deliveries":                 roles,
			"POST /care/deliveries/:id/resend":     {syntheticStewardRole},
			"POST /care/tasks/:id/contact-records": {syntheticStewardRole, syntheticClinicianRole},
		}
		for _, api := range notificationAPIs {
			for _, role := range rolesByRoute[api.Method+" "+api.Path] {
				policy := adapter.CasbinRule{Ptype: "p", V0: fmt.Sprint(role), V1: api.Path, V2: api.Method}
				if err := tx.Where(policy).FirstOrCreate(&policy).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func ensureScenarioBPlan(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var existing pathmodel.PlanInstance
		err := tx.Unscoped().Where("id = ?", scenarioBPlanID).First(&existing).Error
		if err == nil {
			return verifyScenarioBPlan(tx, existing)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var client caremodel.CareClient
		if err := tx.Where("id = ? AND synthetic = ?", 20002, true).First(&client).Error; err != nil {
			return fmt.Errorf("scenario B client missing: %w", err)
		}
		var template pathmodel.PlanTemplateVersion
		if err := tx.Where("id = ? AND synthetic = ?", syntheticPlanTemplateID, true).First(&template).Error; err != nil {
			return fmt.Errorf("scenario B plan template missing: %w", err)
		}
		var definitions []pathmodel.PlanTaskDefinition
		if err := tx.Where("plan_template_version_id = ?", syntheticPlanTemplateID).
			Order("sort ASC").Find(&definitions).Error; err != nil {
			return err
		}
		if len(definitions) != 5 {
			return fmt.Errorf("scenario B requires exactly five task definitions")
		}
		var conflicting int64
		if err := tx.Model(&pathmodel.Enrollment{}).
			Where("care_client_id = ? AND path_code = ? AND active_slot IS NOT NULL", client.ID, "OSA").
			Count(&conflicting).Error; err != nil {
			return err
		}
		if conflicting != 0 {
			return fmt.Errorf("scenario B client already has a different active path")
		}
		fixed := carePathFixedTime()
		activeSlot := "OSA"
		enrollment := pathmodel.Enrollment{
			GVA_MODEL:    global.GVA_MODEL{ID: scenarioBEnrollmentID},
			CareClientID: client.ID, PathDefinitionVersionID: syntheticPathVersionID,
			PathCode: "OSA", ActiveSlot: &activeSlot, Status: pathmodel.EnrollmentActive,
			StartedAt: &fixed, Version: 1, Synthetic: true,
			DeptId: syntheticTeamAID, CreatedBy: syntheticStewardA2ID,
		}
		consumedAt := fixed
		planID := uint(scenarioBPlanID)
		preview := pathmodel.PlanPreview{
			GVA_MODEL: global.GVA_MODEL{ID: scenarioBPreviewID}, PreviewID: scenarioBPreviewOpaque,
			CareClientID: client.ID, PlanTemplateVersionID: syntheticPlanTemplateID,
			AnchorAt: fixed, ExpiresAt: fixed.Add(30 * time.Minute),
			TemplateDefinitionHash: template.DefinitionHash, ConsumedAt: &consumedAt, PlanInstanceID: &planID,
			Synthetic: true, DeptId: syntheticTeamAID, CreatedBy: syntheticStewardA2ID,
		}
		plan := pathmodel.PlanInstance{
			GVA_MODEL:    global.GVA_MODEL{ID: scenarioBPlanID},
			EnrollmentID: enrollment.ID, CareClientID: client.ID,
			PlanTemplateVersionID: syntheticPlanTemplateID, PreviewID: preview.ID,
			AnchorAt: fixed, Status: pathmodel.EnrollmentActive,
			PauseStrategy: pathmodel.PauseStrategyKeepWindows, Version: 1,
			Synthetic: true, DeptId: syntheticTeamAID, CreatedBy: syntheticStewardA2ID,
		}
		if err := tx.Create(&enrollment).Error; err != nil {
			return err
		}
		if err := tx.Create(&preview).Error; err != nil {
			return err
		}
		if err := tx.Create(&plan).Error; err != nil {
			return err
		}
		tasks := make([]pathmodel.TaskInstance, 0, len(definitions))
		for i, definition := range definitions {
			status := pathmodel.ExecutionScheduled
			var openedAt *time.Time
			if i == 0 {
				status = pathmodel.ExecutionOpen
				openedAt = &fixed
			}
			reviewStatus := pathmodel.ReviewNotRequired
			if definition.ReviewRequired {
				reviewStatus = pathmodel.ReviewNotReady
			}
			tasks = append(tasks, pathmodel.TaskInstance{
				GVA_MODEL:      global.GVA_MODEL{ID: uint(scenarioBTaskD1ID + i)},
				PlanInstanceID: plan.ID, CareClientID: client.ID, TaskDefinitionID: definition.ID,
				DayCode: definition.DayCode, Title: definition.Title, Sort: definition.Sort,
				ExecutionRole: definition.ExecutionRole, ExecutionStatus: status,
				ReviewStatus: reviewStatus, ReviewRole: definition.ReviewRole,
				OpenAt:                  fixed.Add(time.Duration(definition.OpenOffsetSeconds) * time.Second),
				DueAt:                   fixed.Add(time.Duration(definition.DueOffsetSeconds) * time.Second),
				QuestionnaireVersionID:  definition.QuestionnaireVersionID,
				BoundRuleVersionIDsJSON: append(datatypes.JSON(nil), definition.BoundRuleVersionIDsJSON...),
				LateSubmissionPolicy:    pathmodel.LateSubmissionDeny,
				NotificationPolicy:      pathmodel.NotificationPolicyDisabled,
				OpenedAt:                openedAt, Version: 1, Synthetic: true,
				DeptId: syntheticTeamAID, CreatedBy: syntheticStewardA2ID,
			})
		}
		if err := tx.Create(&tasks).Error; err != nil {
			return err
		}
		d1ID := tasks[0].ID
		events := []pathmodel.CarePathEvent{
			{
				GVA_MODEL: global.GVA_MODEL{ID: scenarioBPlanEventID},
				EventID:   "00000000-0000-4000-8000-000000024201",
				EventType: pathmodel.EventPlanStarted, CareClientID: client.ID,
				EnrollmentID: enrollment.ID, PlanInstanceID: plan.ID,
				ActorID: syntheticStewardA2ID, Source: pathmodel.EventSourceCareSteward,
				FromStatus: pathmodel.EnrollmentPendingStart, ToStatus: pathmodel.EnrollmentActive,
				OccurredAt: fixed, Synthetic: true, DeptId: syntheticTeamAID, CreatedBy: syntheticStewardA2ID,
			},
			{
				GVA_MODEL: global.GVA_MODEL{ID: scenarioBTaskEventID},
				EventID:   "00000000-0000-4000-8000-000000024202",
				EventType: pathmodel.EventTaskOpened, CareClientID: client.ID,
				EnrollmentID: enrollment.ID, PlanInstanceID: plan.ID, TaskInstanceID: &d1ID,
				ActorID: syntheticStewardA2ID, Source: pathmodel.EventSourceSystem,
				FromStatus: pathmodel.ExecutionScheduled, ToStatus: pathmodel.ExecutionOpen,
				OccurredAt: fixed, Synthetic: true, DeptId: syntheticTeamAID, CreatedBy: syntheticStewardA2ID,
			},
		}
		for i := range events {
			if err := tx.Create(&events[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func verifyScenarioBPlan(tx *gorm.DB, plan pathmodel.PlanInstance) error {
	if plan.CareClientID != 20002 || plan.EnrollmentID != scenarioBEnrollmentID ||
		plan.Status != pathmodel.EnrollmentActive || !plan.Synthetic || plan.DeletedAt.Valid {
		return fmt.Errorf("scenario B fixed plan differs")
	}
	var tasks []pathmodel.TaskInstance
	if err := tx.Unscoped().Where("plan_instance_id = ?", plan.ID).Order("sort ASC").Find(&tasks).Error; err != nil {
		return err
	}
	if len(tasks) != 5 {
		return fmt.Errorf("scenario B fixed plan must retain five tasks")
	}
	for i, task := range tasks {
		expectedStatus := pathmodel.ExecutionScheduled
		if i == 0 {
			expectedStatus = pathmodel.ExecutionOpen
		}
		if task.ID != uint(scenarioBTaskD1ID+i) || task.ExecutionStatus != expectedStatus || task.DeletedAt.Valid {
			return fmt.Errorf("scenario B task D%d differs", i+1)
		}
	}
	return nil
}

func ensureNotificationFixtures(ctx context.Context, db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		type fixture struct {
			requestID uint
			attemptID uint
			taskID    uint
			outcome   string
			requested time.Time
			createdBy uint
		}
		zone := time.FixedZone("CST", 8*60*60)
		fixtures := []fixture{
			{
				requestID: scenarioARequestID, attemptID: scenarioAAttemptID,
				taskID: syntheticTaskInstanceD1ID, outcome: notificationmodel.AttemptStatusDelivered,
				requested: time.Date(2026, time.August, 18, 8, 54, 0, 0, zone), createdBy: syntheticStewardAID,
			},
			{
				requestID: scenarioBRequestID, attemptID: scenarioBAttempt1ID,
				taskID: scenarioBTaskD1ID, outcome: notificationmodel.AttemptStatusFailed,
				requested: time.Date(2026, time.August, 18, 8, 54, 0, 0, zone), createdBy: syntheticStewardA2ID,
			},
		}
		for _, item := range fixtures {
			if err := ensureInitialNotificationFixture(ctx, tx, item.requestID, item.attemptID, item.taskID, item.outcome, item.requested, item.createdBy); err != nil {
				return err
			}
		}
		return ensureRetryNotificationFixture(ctx, tx)
	})
}

func ensureInitialNotificationFixture(ctx context.Context, db *gorm.DB, requestID, attemptID, taskID uint, outcome string, requestedAt time.Time, createdBy uint) error {
	var existing notificationmodel.NotificationAttempt
	err := db.Unscoped().Where("id = ?", attemptID).First(&existing).Error
	if err == nil {
		if existing.TaskID != taskID || existing.AttemptNo != 1 || existing.Status != outcome || existing.DeletedAt.Valid {
			return fmt.Errorf("notification attempt %d differs", attemptID)
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		var task pathmodel.TaskInstance
		if err := tx.Where("id = ? AND synthetic = ?", taskID, true).First(&task).Error; err != nil {
			return err
		}
		request := notificationmodel.NotificationRequest{
			GVA_MODEL: global.GVA_MODEL{ID: requestID}, TaskID: task.ID, CareClientID: task.CareClientID,
			Channel: notificationmodel.ChannelDemo, RequestedAt: requestedAt,
			Synthetic: true, DeptId: task.DeptId, CreatedBy: createdBy,
		}
		if err := tx.Create(&request).Error; err != nil {
			return err
		}
		attempt := notificationmodel.NotificationAttempt{
			GVA_MODEL:             global.GVA_MODEL{ID: attemptID},
			NotificationRequestID: request.ID, AttemptNo: 1, TaskID: task.ID, CareClientID: task.CareClientID,
			Channel: notificationmodel.ChannelDemo, Status: notificationmodel.AttemptStatusPending,
			RequestedAt: requestedAt, Version: 1, ActorID: 0,
			Operation:        fmt.Sprintf("FIXED_INITIAL:%d", task.ID),
			CommandKeyDigest: seedDigest(fmt.Sprintf("initial:%d", task.ID)),
			RequestHash:      seedDigest(fmt.Sprint(task.ID)),
			Synthetic:        true, DeptId: task.DeptId, CreatedBy: createdBy,
		}
		if err := tx.Create(&attempt).Error; err != nil {
			return err
		}
		event := notificationmodel.DeliveryEvent{
			EventID:  fmt.Sprintf("00000000-0000-4000-8000-%012d", attemptID),
			EventKey: "requested", NotificationRequestID: request.ID, NotificationAttemptID: attempt.ID,
			EventType: notificationmodel.EventNotificationRequested, ToStatus: notificationmodel.AttemptStatusPending,
			OccurredAt: requestedAt, Synthetic: true, DeptId: task.DeptId, CreatedBy: createdBy,
		}
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
		return platformoutbox.Append(tx, platformoutbox.AppendInput{
			EventType: event.EventType, AggregateType: "NotificationRequest", AggregateID: request.ID,
			Payload: map[string]any{
				"notificationRequestId": request.ID, "notificationAttemptId": attempt.ID,
				"taskId": task.ID, "careClientId": task.CareClientID,
				"attemptNo": 1, "toStatus": notificationmodel.AttemptStatusPending, "synthetic": true,
			},
			OccurredAt: requestedAt, CausationID: event.EventID,
			Synthetic: true, DeptID: task.DeptId, CreatedBy: createdBy,
		})
	})
	if err != nil {
		return err
	}
	service := notificationservice.NotificationService{DB: db, Clock: notificationservice.FixedClock{Time: requestedAt}}
	receipts := fixedReceipts(attemptID, requestedAt, outcome)
	for _, receipt := range receipts {
		if _, err := service.ApplyDeliveryReceipt(ctx, attemptID, receipt); err != nil {
			return err
		}
	}
	return nil
}

func ensureRetryNotificationFixture(ctx context.Context, db *gorm.DB) error {
	var existing notificationmodel.NotificationAttempt
	err := db.Unscoped().Where("id = ?", scenarioBAttempt2ID).First(&existing).Error
	if err == nil {
		if existing.NotificationRequestID != scenarioBRequestID || existing.AttemptNo != 2 ||
			existing.Status != notificationmodel.AttemptStatusUnknown || existing.RetryOfAttemptID == nil ||
			*existing.RetryOfAttemptID != scenarioBAttempt1ID || existing.DeletedAt.Valid {
			return fmt.Errorf("notification retry attempt differs")
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	requestedAt := time.Date(2026, time.August, 18, 9, 10, 0, 0, time.FixedZone("CST", 8*60*60))
	err = db.Transaction(func(tx *gorm.DB) error {
		var request notificationmodel.NotificationRequest
		if err := tx.Where("id = ?", scenarioBRequestID).First(&request).Error; err != nil {
			return err
		}
		retryOf := uint(scenarioBAttempt1ID)
		attempt := notificationmodel.NotificationAttempt{
			GVA_MODEL:             global.GVA_MODEL{ID: scenarioBAttempt2ID},
			NotificationRequestID: request.ID, AttemptNo: 2, TaskID: request.TaskID, CareClientID: request.CareClientID,
			RetryOfAttemptID: &retryOf, Channel: notificationmodel.ChannelDemo,
			Status: notificationmodel.AttemptStatusPending, RequestedAt: requestedAt,
			ResendReason: "验证补发保留旧尝试", Version: 1,
			ActorID: syntheticStewardA2ID, Operation: fmt.Sprintf("RESEND_NOTIFICATION:%d", scenarioBAttempt1ID),
			CommandKeyDigest: seedDigest("scenario-b-resend"), RequestHash: seedDigest("scenario-b-resend-request"),
			Synthetic: true, DeptId: syntheticTeamAID, CreatedBy: syntheticStewardA2ID,
		}
		if err := tx.Create(&attempt).Error; err != nil {
			return err
		}
		event := notificationmodel.DeliveryEvent{
			EventID:  "00000000-0000-4000-8000-000000028102",
			EventKey: "requested", NotificationRequestID: request.ID, NotificationAttemptID: attempt.ID,
			EventType: notificationmodel.EventNotificationRequested, ToStatus: notificationmodel.AttemptStatusPending,
			OccurredAt: requestedAt, Synthetic: true, DeptId: syntheticTeamAID, CreatedBy: syntheticStewardA2ID,
		}
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
		return platformoutbox.Append(tx, platformoutbox.AppendInput{
			EventType: event.EventType, AggregateType: "NotificationRequest", AggregateID: request.ID,
			Payload: map[string]any{
				"notificationRequestId": request.ID, "notificationAttemptId": attempt.ID,
				"taskId": request.TaskID, "careClientId": request.CareClientID,
				"attemptNo": 2, "toStatus": notificationmodel.AttemptStatusPending, "synthetic": true,
			},
			OccurredAt: requestedAt, CausationID: event.EventID,
			Synthetic: true, DeptID: syntheticTeamAID, CreatedBy: syntheticStewardA2ID,
		})
	})
	if err != nil {
		return err
	}
	service := notificationservice.NotificationService{DB: db, Clock: notificationservice.FixedClock{Time: requestedAt}}
	for _, receipt := range fixedReceipts(scenarioBAttempt2ID, requestedAt, notificationmodel.AttemptStatusUnknown) {
		if _, err := service.ApplyDeliveryReceipt(ctx, scenarioBAttempt2ID, receipt); err != nil {
			return err
		}
	}
	return nil
}

func fixedReceipts(attemptID uint, requestedAt time.Time, outcome string) []notificationservice.DeliveryReceipt {
	failureCode := ""
	if outcome == notificationmodel.AttemptStatusFailed {
		failureCode = notificationmodel.DemoFailureCode
	}
	if outcome == notificationmodel.AttemptStatusUnknown {
		failureCode = notificationmodel.DemoUnknownCode
	}
	return []notificationservice.DeliveryReceipt{
		{
			EventKey:         fmt.Sprintf("fixed:%d:submitted", attemptID),
			Status:           notificationmodel.AttemptStatusSubmittedToProvider,
			OccurredAt:       requestedAt.Add(time.Minute),
			AdapterReference: fmt.Sprintf("fixed:%d:provider", attemptID),
		},
		{
			EventKey:         fmt.Sprintf("fixed:%d:accepted", attemptID),
			Status:           notificationmodel.AttemptStatusAccepted,
			OccurredAt:       requestedAt.Add(time.Minute + time.Second),
			AdapterReference: fmt.Sprintf("fixed:%d:accepted", attemptID),
		},
		{
			EventKey: fmt.Sprintf("fixed:%d:final", attemptID),
			Status:   outcome, OccurredAt: requestedAt.Add(2 * time.Minute),
			FailureCode: failureCode, AdapterReference: fmt.Sprintf("fixed:%d:final", attemptID),
		},
	}
}

func seedDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
