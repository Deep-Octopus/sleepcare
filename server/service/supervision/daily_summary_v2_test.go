package supervision

import (
	"context"
	"testing"
	"time"

	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	commonreq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	supervisionreq "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/request"
	supervisionres "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
)

func TestDailySummaryV2MetricsAndDashboardCoverage(t *testing.T) {
	fixture := newSupervisionFixture(t)
	reportNow := fixture.now.Add(24 * time.Hour)
	fixture.service.Now = func() time.Time { return reportNow }
	enabled := true
	fixture.service.SyntheticFixturesEnabled = &enabled

	lateSubmittedAt := fixture.now.Add(-5 * time.Hour)
	lateTask := supervisionTask(
		fixture.attentionCase.CareClientID,
		101,
		81,
		fixture.now.Add(-9*time.Hour),
		fixture.now.Add(-6*time.Hour),
		pathmodel.ExecutionSubmitted,
		&lateSubmittedAt,
	)
	if err := fixture.db.WithContext(fixture.systemCtx).Create(&lateTask).Error; err != nil {
		t.Fatal(err)
	}
	closedAt := fixture.now.Add(-time.Hour)
	consultations := []caseworkmodel.Consultation{
		{
			CareClientID:    fixture.attentionCase.CareClientID,
			Source:          caseworkmodel.ConsultationSourceOnline,
			Subject:         "流程咨询记录",
			InitialQuestion: "请协助确认服务流程",
			Urgency:         caseworkmodel.ConsultationUrgencyRoutine,
			Status:          caseworkmodel.ConsultationStatusClosed,
			OpenedAt:        fixture.now.Add(-7 * time.Hour),
			ClosedAt:        &closedAt,
			CloseReason:     "流程记录已完成",
			Version:         1,
			Synthetic:       true,
			DeptId:          101,
		},
		{
			CareClientID:    fixture.attentionCase.CareClientID,
			Source:          caseworkmodel.ConsultationSourceOnline,
			Subject:         "跨日流程咨询",
			InitialQuestion: "请协助确认后续安排",
			Urgency:         caseworkmodel.ConsultationUrgencyRoutine,
			Status:          caseworkmodel.ConsultationStatusHandling,
			OpenedAt:        fixture.now.Add(-26 * time.Hour),
			Version:         1,
			Synthetic:       true,
			DeptId:          101,
		},
	}
	if err := fixture.db.WithContext(fixture.systemCtx).Create(&consultations).Error; err != nil {
		t.Fatal(err)
	}

	businessDate := time.Date(2026, time.August, 18, 0, 0, 0, 0, summaryLocation)
	snapshot, created, err := fixture.service.EnsureScheduledSnapshot(fixture.systemCtx, 100, businessDate)
	if err != nil {
		t.Fatal(err)
	}
	if !created || snapshot.Version == nil || *snapshot.Version != 1 || snapshot.GenerationType != supervisionmodel.SummaryGenerationScheduled {
		t.Fatalf("unexpected scheduled snapshot: created=%v snapshot=%+v", created, snapshot)
	}
	assertSummaryV2Metrics(t, snapshot.DailySummary, summaryMetricValues{
		ServedClients:          1,
		DueTasks:               3,
		SubmittedTasks:         2,
		OverdueTasks:           2,
		DeliveryIssues:         1,
		OpenAttentionCases:     1,
		ResolvedAttentionCases: 1,
		ConsultationsOpened:    1,
		ConsultationsClosed:    1,
		OpenConsultations:      1,
		OpenTodos:              2,
		ReviewRequired:         1,
	})
	if _, created, err = fixture.service.EnsureScheduledSnapshot(fixture.systemCtx, 100, businessDate); err != nil || created {
		t.Fatalf("scheduled snapshot must be idempotent: created=%v err=%v", created, err)
	}

	dashboard, err := fixture.service.GetOperationsDashboard(
		fixture.supervisorCtx,
		supervisionreq.OperationsDashboardQuery{Days: 2},
	)
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.FormalReportingEnabled || dashboard.UsageScope != supervisionmodel.SummaryUsageTestOnly ||
		dashboard.AttributionPolicyStatus != supervisionmodel.SummaryAttributionPending {
		t.Fatalf("dashboard opened an unapproved reporting boundary: %+v", dashboard)
	}
	if len(dashboard.RecentSnapshots) != 1 || dashboard.RecentSnapshots[0].ID != snapshot.ID ||
		dashboard.Coverage.SnapshotDays != 1 || len(dashboard.Coverage.MissingDates) != 0 {
		t.Fatalf("unexpected dashboard coverage: %+v", dashboard)
	}
	cross, err := fixture.service.GetOperationsDashboard(
		fixture.crossCtx,
		supervisionreq.OperationsDashboardQuery{Days: 2},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(cross.RecentSnapshots) != 0 {
		t.Fatalf("cross-organization dashboard leaked snapshots: %+v", cross.RecentSnapshots)
	}
	if _, err = fixture.service.GetOperationsDashboard(fixture.supervisorCtx, supervisionreq.OperationsDashboardQuery{Days: 32}); domainCode(err) != supervisionmodel.CodeInvalidArgument {
		t.Fatalf("invalid dashboard window should fail, got %v", err)
	}
}

func TestDailySummaryCorrectionIsAppendOnlyChangedAndIdempotent(t *testing.T) {
	fixture := newSupervisionFixture(t)
	reportNow := fixture.now.Add(24 * time.Hour)
	fixture.service.Now = func() time.Time { return reportNow }
	businessDate := time.Date(2026, time.August, 18, 0, 0, 0, 0, summaryLocation)
	base, _, err := fixture.service.EnsureScheduledSnapshot(fixture.systemCtx, 100, businessDate)
	if err != nil {
		t.Fatal(err)
	}

	correctedSubmission := fixture.now.Add(time.Hour + 30*time.Minute)
	if err = fixture.db.WithContext(fixture.systemCtx).Model(&pathmodel.TaskInstance{}).
		Where("care_client_id = ? AND task_definition_id = ?", fixture.attentionCase.CareClientID, 2).
		Updates(map[string]any{
			"execution_status": pathmodel.ExecutionSubmitted,
			"submitted_at":     correctedSubmission,
		}).Error; err != nil {
		t.Fatal(err)
	}

	revisionRequest := supervisionreq.ReviseDailySummary{
		ExpectedVersion: *base.Version,
		Reason:          "补录记录已核对，重新复算历史汇总",
	}
	revised, err := fixture.service.ReviseDailySummary(
		fixture.supervisorCtx,
		base.ID,
		"summary-revision-001",
		revisionRequest,
	)
	if err != nil {
		t.Fatal(err)
	}
	if revised.Version == nil || *revised.Version != 2 || revised.PreviousVersionID == nil ||
		*revised.PreviousVersionID != base.ID || revised.GenerationType != supervisionmodel.SummaryGenerationCorrection ||
		!revised.IsLatest {
		t.Fatalf("unexpected revision metadata: %+v", revised)
	}
	assertMetricChange(t, revised.RevisionChanges, "submittedTasks", 1, 2)
	assertMetricChange(t, revised.RevisionChanges, "overdueTasks", 1, 0)

	replayed, err := fixture.service.ReviseDailySummary(
		fixture.supervisorCtx,
		base.ID,
		"summary-revision-001",
		revisionRequest,
	)
	if err != nil || replayed.ID != revised.ID {
		t.Fatalf("revision replay changed result: replay=%+v err=%v", replayed, err)
	}
	changedRequest := revisionRequest
	changedRequest.Reason = "另一条修正原因"
	if _, err = fixture.service.ReviseDailySummary(
		fixture.supervisorCtx,
		base.ID,
		"summary-revision-001",
		changedRequest,
	); domainCode(err) != supervisionmodel.CodeIdempotencyConflict {
		t.Fatalf("changed revision request should conflict, got %v", err)
	}
	if _, err = fixture.service.ReviseDailySummary(
		fixture.supervisorCtx,
		base.ID,
		"summary-revision-stale",
		revisionRequest,
	); domainCode(err) != supervisionmodel.CodeSummaryNotLatest {
		t.Fatalf("stale summary revision should fail, got %v", err)
	}
	if _, err = fixture.service.ReviseDailySummary(
		fixture.supervisorCtx,
		revised.ID,
		"summary-revision-no-change",
		supervisionreq.ReviseDailySummary{ExpectedVersion: 2, Reason: "再次核对"},
	); domainCode(err) != supervisionmodel.CodeSummaryUnchanged {
		t.Fatalf("unchanged recomputation should not append a version, got %v", err)
	}
	if err = fixture.db.WithContext(fixture.systemCtx).Model(&pathmodel.TaskInstance{}).
		Where("care_client_id = ? AND task_definition_id = ?", fixture.attentionCase.CareClientID, 2).
		Updates(map[string]any{
			"execution_status": pathmodel.ExecutionOpen,
			"submitted_at":     nil,
		}).Error; err != nil {
		t.Fatal(err)
	}
	latest, err := fixture.service.ReviseDailySummary(
		fixture.supervisorCtx,
		revised.ID,
		"summary-revision-002",
		supervisionreq.ReviseDailySummary{ExpectedVersion: 2, Reason: "撤销错误补录后重新核对"},
	)
	if err != nil || latest.Version == nil || *latest.Version != 3 || !latest.IsLatest {
		t.Fatalf("second revision was not appended: latest=%+v err=%v", latest, err)
	}
	replayed, err = fixture.service.ReviseDailySummary(
		fixture.supervisorCtx,
		base.ID,
		"summary-revision-001",
		revisionRequest,
	)
	if err != nil || replayed.ID != revised.ID || replayed.IsLatest {
		t.Fatalf("older replay should keep its original result without latest marker: replay=%+v err=%v", replayed, err)
	}
	items, _, err := fixture.service.ListDailySummaries(
		fixture.supervisorCtx,
		supervisionreq.DailySummarySearch{
			PageInfo:     commonreq.PageInfo{Page: 1, PageSize: 10},
			BusinessDate: businessDate.Format("2006-01-02"),
		},
	)
	if err != nil || len(items) != 3 || !items[0].IsLatest || items[1].IsLatest || items[2].IsLatest {
		t.Fatalf("summary list latest markers are wrong: items=%+v err=%v", items, err)
	}
	dashboard, err := fixture.service.GetOperationsDashboard(
		fixture.supervisorCtx,
		supervisionreq.OperationsDashboardQuery{Days: 2},
	)
	if err != nil || dashboard.Coverage.RevisedDates != 1 {
		t.Fatalf("dashboard revision coverage is wrong: coverage=%+v err=%v", dashboard.Coverage, err)
	}

	storedBase, err := fixture.service.GetDailySummary(fixture.supervisorCtx, base.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedBase.SubmittedTasks != 1 || storedBase.OverdueTasks != 1 || storedBase.IsLatest {
		t.Fatalf("base snapshot was changed: %+v", storedBase)
	}
	var count int64
	if err = fixture.db.WithContext(fixture.systemCtx).Model(&supervisionmodel.DailySummaryVersion{}).
		Where("organization_id = ? AND business_date = ?", 100, businessDate).
		Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("revision count = %d, want 3", count)
	}
}

func TestGenerateScheduledSnapshotsUsesSystemScopeAndAllFixedOrganizations(t *testing.T) {
	fixture := newSupervisionFixture(t)
	reportNow := fixture.now.Add(24 * time.Hour)
	enabled := true
	fixture.service.Now = func() time.Time { return reportNow }
	fixture.service.SyntheticFixturesEnabled = &enabled

	if err := fixture.service.GenerateScheduledSnapshots(context.Background()); domainCode(err) != supervisionmodel.CodeSummaryGenerationDenied {
		t.Fatalf("scheduled generation without system scope should fail, got %v", err)
	}
	if err := fixture.service.GenerateScheduledSnapshots(datascope.WithSystem(context.Background())); err != nil {
		t.Fatal(err)
	}
	if err := fixture.service.GenerateScheduledSnapshots(datascope.WithSystem(context.Background())); err != nil {
		t.Fatal(err)
	}
	var snapshots []supervisionmodel.DailySummaryVersion
	if err := fixture.db.WithContext(fixture.systemCtx).Order("organization_id ASC").Find(&snapshots).Error; err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 2 || snapshots[0].OrganizationID != 100 || snapshots[1].OrganizationID != 200 {
		t.Fatalf("scheduled generation did not cover both fixed organizations exactly once: %+v", snapshots)
	}
}

func assertSummaryV2Metrics(t *testing.T, summary supervisionres.DailySummary, want summaryMetricValues) {
	t.Helper()
	got := metricValuesFromSummary(summary)
	if got != want {
		t.Fatalf("unexpected v2 summary metrics: got=%+v want=%+v", got, want)
	}
}

func assertMetricChange(t *testing.T, changes []supervisionres.MetricChange, key string, before, after int64) {
	t.Helper()
	for _, change := range changes {
		if change.Key == key {
			if change.Before != before || change.After != after {
				t.Fatalf("metric change %s = %+v, want %d -> %d", key, change, before, after)
			}
			return
		}
	}
	t.Fatalf("metric change %s missing from %+v", key, changes)
}
