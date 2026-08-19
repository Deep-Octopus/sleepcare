package supervision

import (
	"context"

	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	supervisionreq "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/request"
	supervisionres "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/response"
)

const (
	defaultDashboardDays = 7
	maxDashboardDays     = 31
)

func (s *SupervisionService) GetOperationsDashboard(
	ctx context.Context,
	req supervisionreq.OperationsDashboardQuery,
) (supervisionres.OperationsDashboard, error) {
	_, organizationID, err := s.supervisorScope(ctx)
	if err != nil {
		return supervisionres.OperationsDashboard{}, err
	}
	if req.Days == 0 {
		req.Days = defaultDashboardDays
	}
	if req.Days < 1 || req.Days > maxDashboardDays {
		return supervisionres.OperationsDashboard{}, supervisionmodel.NewDomainError(
			supervisionmodel.CodeInvalidArgument,
			"days 必须在 1 到 31 之间",
		)
	}
	now := s.now()
	currentDate, _ := summaryDate("", now)
	current, err := s.computeSummary(ctx, organizationID, currentDate, now)
	if err != nil {
		return supervisionres.OperationsDashboard{}, err
	}
	pastDays := req.Days - 1
	coverage := supervisionres.DashboardCoverage{
		RequestedPastDays: pastDays,
		MissingDates:      make([]string, 0),
	}
	recent := make([]supervisionres.DailySummary, 0, pastDays)
	if pastDays > 0 {
		startDate := currentDate.AddDate(0, 0, -pastDays)
		var snapshots []supervisionmodel.DailySummaryVersion
		if err = s.db().WithContext(ctx).
			Where(
				"organization_id = ? AND synthetic = ? AND business_date >= ? AND business_date < ?",
				organizationID,
				true,
				startDate,
				currentDate,
			).
			Order("business_date DESC, version DESC, id DESC").
			Find(&snapshots).Error; err != nil {
			return supervisionres.OperationsDashboard{}, err
		}
		latestByDate := make(map[string]supervisionmodel.DailySummaryVersion, pastDays)
		revisedDates := make(map[string]struct{}, pastDays)
		for _, snapshot := range snapshots {
			dateKey := snapshot.BusinessDate.In(summaryLocation).Format("2006-01-02")
			if snapshot.GenerationType == supervisionmodel.SummaryGenerationCorrection {
				revisedDates[dateKey] = struct{}{}
			}
			if _, exists := latestByDate[dateKey]; !exists {
				latestByDate[dateKey] = snapshot
			}
		}
		for offset := 1; offset <= pastDays; offset++ {
			dateKey := currentDate.AddDate(0, 0, -offset).Format("2006-01-02")
			snapshot, exists := latestByDate[dateKey]
			if !exists {
				coverage.MissingDates = append(coverage.MissingDates, dateKey)
				continue
			}
			row := summaryFromSnapshot(snapshot)
			row.IsLatest = true
			recent = append(recent, row)
			coverage.SnapshotDays++
			if _, revised := revisedDates[dateKey]; revised {
				coverage.RevisedDates++
			}
		}
	}
	return supervisionres.OperationsDashboard{
		AsOf:                    now,
		TimeZone:                "Asia/Shanghai",
		UsageScope:              supervisionmodel.SummaryUsageTestOnly,
		FormalReportingEnabled:  false,
		AttributionPolicyStatus: supervisionmodel.SummaryAttributionPending,
		MetricDefinitionVersion: supervisionmodel.MetricDefinitionVersionV2,
		Current:                 current.Detail.DailySummary,
		RecentSnapshots:         recent,
		Coverage:                coverage,
	}, nil
}
