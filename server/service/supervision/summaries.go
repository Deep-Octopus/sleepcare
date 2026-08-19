package supervision

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	supervisionreq "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/request"
	supervisionres "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/response"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type computedSummary struct {
	Detail         supervisionres.DailySummaryDetail
	FocusCasesJSON datatypes.JSON
	SourceDigest   string
	SourceCutoffAt time.Time
}

func (s *SupervisionService) ListDailySummaries(ctx context.Context, req supervisionreq.DailySummarySearch) ([]supervisionres.DailySummary, int64, error) {
	_, organizationID, err := s.supervisorScope(ctx)
	if err != nil {
		return nil, 0, err
	}
	now := s.now()
	businessDate, err := summaryDate(req.BusinessDate, now)
	if err != nil {
		return nil, 0, err
	}
	currentDate, _ := summaryDate("", now)
	includePreview := stringsEqualDate(businessDate, currentDate)

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	limit, offset := req.LimitOffset()

	query := s.db().WithContext(ctx).Model(&supervisionmodel.DailySummaryVersion{}).
		Where("organization_id = ? AND synthetic = ?", organizationID, true)
	if req.BusinessDate != "" {
		query = query.Where("business_date = ?", businessDate)
	}
	var snapshotTotal int64
	if err = query.Count(&snapshotTotal).Error; err != nil {
		return nil, 0, err
	}
	total := snapshotTotal
	if includePreview {
		total++
	}

	items := make([]supervisionres.DailySummary, 0, limit)
	snapshotOffset := offset
	snapshotLimit := limit
	if includePreview {
		if offset == 0 && limit > 0 {
			preview, computeErr := s.computeSummary(ctx, organizationID, businessDate, now)
			if computeErr != nil {
				return nil, 0, computeErr
			}
			items = append(items, preview.Detail.DailySummary)
			snapshotLimit--
		} else if offset > 0 {
			snapshotOffset--
		}
	}
	if snapshotLimit <= 0 {
		return items, total, nil
	}

	var snapshots []supervisionmodel.DailySummaryVersion
	if err = query.Order("business_date DESC, version DESC, id DESC").
		Limit(snapshotLimit).Offset(snapshotOffset).Find(&snapshots).Error; err != nil {
		return nil, 0, err
	}
	latestVersions := make(map[string]uint, len(snapshots))
	if len(snapshots) > 0 {
		businessDates := make([]time.Time, 0, len(snapshots))
		for i := range snapshots {
			businessDates = append(businessDates, snapshots[i].BusinessDate)
		}
		var rows []struct {
			BusinessDate time.Time
			Version      uint
		}
		if err = s.db().WithContext(ctx).Model(&supervisionmodel.DailySummaryVersion{}).
			Select("business_date, MAX(version) AS version").
			Where(
				"organization_id = ? AND synthetic = ? AND business_date IN ?",
				organizationID,
				true,
				businessDates,
			).
			Group("business_date").
			Scan(&rows).Error; err != nil {
			return nil, 0, err
		}
		for _, row := range rows {
			latestVersions[row.BusinessDate.In(summaryLocation).Format("2006-01-02")] = row.Version
		}
	}
	for i := range snapshots {
		item := summaryFromSnapshot(snapshots[i])
		item.IsLatest = snapshots[i].Version == latestVersions[item.BusinessDate]
		items = append(items, item)
	}
	return items, total, nil
}

func (s *SupervisionService) GetDailySummary(ctx context.Context, id uint) (supervisionres.DailySummaryDetail, error) {
	if id == 0 {
		return supervisionres.DailySummaryDetail{}, supervisionmodel.NewDomainError(supervisionmodel.CodeInvalidArgument, "日报版本标识必填")
	}
	_, organizationID, err := s.supervisorScope(ctx)
	if err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	var snapshot supervisionmodel.DailySummaryVersion
	err = s.db().WithContext(ctx).
		Where("id = ? AND organization_id = ? AND synthetic = ?", id, organizationID, true).
		First(&snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return supervisionres.DailySummaryDetail{}, supervisionmodel.NewForbiddenError(supervisionmodel.CodeReviewScopeDenied, "日报版本不存在或不在当前管理范围")
	}
	if err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	var latestVersion uint
	if err = s.db().WithContext(ctx).Model(&supervisionmodel.DailySummaryVersion{}).
		Where("organization_id = ? AND business_date = ? AND synthetic = ?", organizationID, snapshot.BusinessDate, true).
		Select("COALESCE(MAX(version), 0)").Scan(&latestVersion).Error; err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	return summaryDetailFromSnapshot(snapshot, snapshot.Version == latestVersion)
}

func (s *SupervisionService) computeSummary(
	ctx context.Context,
	organizationID uint,
	businessDate time.Time,
	now time.Time,
) (computedSummary, error) {
	start, queryCutoff, err := summaryBounds(businessDate, now)
	if err != nil {
		return computedSummary{}, err
	}
	sourceCutoff := start.AddDate(0, 0, 1)
	if stringsEqualDate(start, now) {
		sourceCutoff = now
	}
	generatedAt := now
	detail := supervisionres.DailySummaryDetail{
		DailySummary: supervisionres.DailySummary{
			BusinessDate:            start.Format("2006-01-02"),
			SummaryType:             supervisionmodel.SummaryTypeRealtimePreview,
			MetricDefinitionVersion: supervisionmodel.MetricDefinitionVersionV2,
			GeneratedAt:             &generatedAt,
			SourceCutoffAt:          &sourceCutoff,
		},
		FocusCases:      make([]caseworkres.AttentionCaseSummary, 0),
		RevisionChanges: make([]supervisionres.MetricChange, 0),
	}

	var clientIDs []uint
	if err = s.db().WithContext(ctx).Model(&careclient.CareClient{}).
		Where("organization_id = ? AND status = ? AND synthetic = ?", organizationID, careclient.ClientStatusActive, true).
		Order("id ASC").Pluck("id", &clientIDs).Error; err != nil {
		return computedSummary{}, err
	}
	if len(clientIDs) > 0 {
		if err = s.populateSummaryMetrics(ctx, clientIDs, start, queryCutoff, &detail); err != nil {
			return computedSummary{}, err
		}
	}
	focusJSON, err := json.Marshal(detail.FocusCases)
	if err != nil {
		return computedSummary{}, err
	}
	sourceDigest, err := summarySourceDigest(detail, sourceCutoff)
	if err != nil {
		return computedSummary{}, err
	}
	return computedSummary{
		Detail: detail, FocusCasesJSON: datatypes.JSON(focusJSON),
		SourceDigest: sourceDigest, SourceCutoffAt: sourceCutoff,
	}, nil
}

func (s *SupervisionService) populateSummaryMetrics(
	ctx context.Context,
	clientIDs []uint,
	start time.Time,
	cutoff time.Time,
	detail *supervisionres.DailySummaryDetail,
) error {
	dayEnd := start.AddDate(0, 0, 1)
	taskBase := func() *gorm.DB {
		return s.db().WithContext(ctx).Model(&pathmodel.TaskInstance{}).
			Where("care_client_id IN ? AND synthetic = ?", clientIDs, true)
	}
	if err := taskBase().Where("due_at >= ? AND due_at < ?", start, dayEnd).
		Distinct("care_client_id").Count(&detail.ServedClients).Error; err != nil {
		return err
	}
	if err := taskBase().Where("due_at >= ? AND due_at < ?", start, dayEnd).
		Count(&detail.DueTasks).Error; err != nil {
		return err
	}
	if err := taskBase().Where("submitted_at >= ? AND submitted_at < ?", start, cutoff).
		Count(&detail.SubmittedTasks).Error; err != nil {
		return err
	}
	if err := taskBase().Where("due_at >= ? AND due_at < ?", start, cutoff).
		Where("submitted_at IS NULL OR submitted_at > due_at").
		Count(&detail.OverdueTasks).Error; err != nil {
		return err
	}

	if err := s.db().WithContext(ctx).Model(&caseworkmodel.TodoItem{}).
		Where("care_client_id IN ? AND synthetic = ? AND category = ?", clientIDs, true, caseworkmodel.TodoCategoryDeliveryIssue).
		Where("opened_at < ? AND (completed_at IS NULL OR completed_at >= ?)", cutoff, cutoff).
		Count(&detail.DeliveryIssues).Error; err != nil {
		return err
	}
	if err := s.db().WithContext(ctx).Model(&caseworkmodel.TodoItem{}).
		Where("care_client_id IN ? AND synthetic = ?", clientIDs, true).
		Where("opened_at < ? AND (completed_at IS NULL OR completed_at >= ?)", cutoff, cutoff).
		Count(&detail.OpenTodos).Error; err != nil {
		return err
	}

	openCases := func() *gorm.DB {
		return s.db().WithContext(ctx).Model(&caseworkmodel.AttentionCase{}).
			Where("care_client_id IN ? AND synthetic = ?", clientIDs, true).
			Where("opened_at < ? AND (closed_at IS NULL OR closed_at >= ?)", cutoff, cutoff)
	}
	if err := openCases().Count(&detail.OpenAttentionCases).Error; err != nil {
		return err
	}
	if err := s.db().WithContext(ctx).Model(&caseworkmodel.AttentionCase{}).
		Where("care_client_id IN ? AND synthetic = ?", clientIDs, true).
		Where("resolved_at >= ? AND resolved_at < ?", start, cutoff).
		Count(&detail.ResolvedAttentionCases).Error; err != nil {
		return err
	}
	if err := openCases().Where("status = ?", caseworkmodel.CaseStatusWaitingSupervisor).
		Count(&detail.ReviewRequired).Error; err != nil {
		return err
	}

	consultationBase := func() *gorm.DB {
		return s.db().WithContext(ctx).Model(&caseworkmodel.Consultation{}).
			Where("care_client_id IN ? AND synthetic = ?", clientIDs, true)
	}
	if err := consultationBase().Where("opened_at >= ? AND opened_at < ?", start, cutoff).
		Count(&detail.ConsultationsOpened).Error; err != nil {
		return err
	}
	if err := consultationBase().Where("closed_at >= ? AND closed_at < ?", start, cutoff).
		Count(&detail.ConsultationsClosed).Error; err != nil {
		return err
	}
	if err := consultationBase().Where("opened_at < ? AND (closed_at IS NULL OR closed_at >= ?)", cutoff, cutoff).
		Count(&detail.OpenConsultations).Error; err != nil {
		return err
	}

	var cases []caseworkmodel.AttentionCase
	if err := openCases().Order("opened_at DESC, id DESC").Limit(100).Find(&cases).Error; err != nil {
		return err
	}
	for i := range cases {
		detail.FocusCases = append(detail.FocusCases, focusCaseSummary(cases[i]))
	}
	return nil
}

func summaryFromSnapshot(snapshot supervisionmodel.DailySummaryVersion) supervisionres.DailySummary {
	version := snapshot.Version
	generatedAt := snapshot.GeneratedAt
	generationType := snapshot.GenerationType
	if generationType == "" {
		generationType = supervisionmodel.SummaryGenerationLegacy
	}
	return supervisionres.DailySummary{
		ID: snapshot.ID, BusinessDate: snapshot.BusinessDate.Format("2006-01-02"),
		SummaryType: supervisionmodel.SummaryTypeVersionedSnapshot, Version: &version,
		MetricDefinitionVersion: snapshot.MetricDefinitionVersion,
		GenerationType:          generationType, GeneratedAt: &generatedAt,
		SourceCutoffAt: snapshot.SourceCutoffAt, PreviousVersionID: snapshot.PreviousVersionID,
		CorrectionReason: snapshot.CorrectionReason,
		ServedClients:    snapshot.ServedClients, DueTasks: snapshot.DueTasks,
		SubmittedTasks: snapshot.SubmittedTasks, OverdueTasks: snapshot.OverdueTasks,
		DeliveryIssues: snapshot.DeliveryIssues, OpenAttentionCases: snapshot.OpenAttentionCases,
		ResolvedAttentionCases: snapshot.ResolvedAttentionCases,
		ConsultationsOpened:    snapshot.ConsultationsOpened, ConsultationsClosed: snapshot.ConsultationsClosed,
		OpenConsultations: snapshot.OpenConsultations, OpenTodos: snapshot.OpenTodos,
		ReviewRequired: snapshot.ReviewRequired,
	}
}

func summaryDetailFromSnapshot(snapshot supervisionmodel.DailySummaryVersion, latest bool) (supervisionres.DailySummaryDetail, error) {
	focusCases := make([]caseworkres.AttentionCaseSummary, 0)
	if len(snapshot.FocusCasesJSON) > 0 {
		if err := json.Unmarshal(snapshot.FocusCasesJSON, &focusCases); err != nil {
			return supervisionres.DailySummaryDetail{}, err
		}
	}
	changes := make([]supervisionres.MetricChange, 0)
	if len(snapshot.RevisionChangesJSON) > 0 {
		if err := json.Unmarshal(snapshot.RevisionChangesJSON, &changes); err != nil {
			return supervisionres.DailySummaryDetail{}, err
		}
	}
	summary := summaryFromSnapshot(snapshot)
	summary.IsLatest = latest
	return supervisionres.DailySummaryDetail{
		DailySummary: summary, FocusCases: focusCases,
		RevisionChanges: changes, FocusCasesChanged: snapshot.FocusCasesChanged,
	}, nil
}

type summaryMetricValues struct {
	ServedClients          int64 `json:"servedClients"`
	DueTasks               int64 `json:"dueTasks"`
	SubmittedTasks         int64 `json:"submittedTasks"`
	OverdueTasks           int64 `json:"overdueTasks"`
	DeliveryIssues         int64 `json:"deliveryIssues"`
	OpenAttentionCases     int64 `json:"openAttentionCases"`
	ResolvedAttentionCases int64 `json:"resolvedAttentionCases"`
	ConsultationsOpened    int64 `json:"consultationsOpened"`
	ConsultationsClosed    int64 `json:"consultationsClosed"`
	OpenConsultations      int64 `json:"openConsultations"`
	OpenTodos              int64 `json:"openTodos"`
	ReviewRequired         int64 `json:"reviewRequired"`
}

func metricValuesFromSummary(summary supervisionres.DailySummary) summaryMetricValues {
	return summaryMetricValues{
		ServedClients: summary.ServedClients, DueTasks: summary.DueTasks,
		SubmittedTasks: summary.SubmittedTasks, OverdueTasks: summary.OverdueTasks,
		DeliveryIssues: summary.DeliveryIssues, OpenAttentionCases: summary.OpenAttentionCases,
		ResolvedAttentionCases: summary.ResolvedAttentionCases,
		ConsultationsOpened:    summary.ConsultationsOpened, ConsultationsClosed: summary.ConsultationsClosed,
		OpenConsultations: summary.OpenConsultations, OpenTodos: summary.OpenTodos,
		ReviewRequired: summary.ReviewRequired,
	}
}

func metricValuesFromSnapshot(snapshot supervisionmodel.DailySummaryVersion) summaryMetricValues {
	return summaryMetricValues{
		ServedClients: snapshot.ServedClients, DueTasks: snapshot.DueTasks,
		SubmittedTasks: snapshot.SubmittedTasks, OverdueTasks: snapshot.OverdueTasks,
		DeliveryIssues: snapshot.DeliveryIssues, OpenAttentionCases: snapshot.OpenAttentionCases,
		ResolvedAttentionCases: snapshot.ResolvedAttentionCases,
		ConsultationsOpened:    snapshot.ConsultationsOpened, ConsultationsClosed: snapshot.ConsultationsClosed,
		OpenConsultations: snapshot.OpenConsultations, OpenTodos: snapshot.OpenTodos,
		ReviewRequired: snapshot.ReviewRequired,
	}
}

func summarySourceDigest(detail supervisionres.DailySummaryDetail, cutoff time.Time) (string, error) {
	payload, err := json.Marshal(struct {
		BusinessDate string                             `json:"businessDate"`
		CutoffAt     time.Time                          `json:"cutoffAt"`
		Metrics      summaryMetricValues                `json:"metrics"`
		FocusCases   []caseworkres.AttentionCaseSummary `json:"focusCases"`
	}{
		BusinessDate: detail.BusinessDate,
		CutoffAt:     cutoff,
		Metrics:      metricValuesFromSummary(detail.DailySummary),
		FocusCases:   detail.FocusCases,
	})
	if err != nil {
		return "", err
	}
	return digest(string(payload)), nil
}

func buildMetricChanges(before supervisionmodel.DailySummaryVersion, after supervisionres.DailySummary) []supervisionres.MetricChange {
	previous := metricValuesFromSnapshot(before)
	next := metricValuesFromSummary(after)
	values := []struct {
		key           string
		before, after int64
	}{
		{"servedClients", previous.ServedClients, next.ServedClients},
		{"dueTasks", previous.DueTasks, next.DueTasks},
		{"submittedTasks", previous.SubmittedTasks, next.SubmittedTasks},
		{"overdueTasks", previous.OverdueTasks, next.OverdueTasks},
		{"deliveryIssues", previous.DeliveryIssues, next.DeliveryIssues},
		{"openAttentionCases", previous.OpenAttentionCases, next.OpenAttentionCases},
		{"resolvedAttentionCases", previous.ResolvedAttentionCases, next.ResolvedAttentionCases},
		{"consultationsOpened", previous.ConsultationsOpened, next.ConsultationsOpened},
		{"consultationsClosed", previous.ConsultationsClosed, next.ConsultationsClosed},
		{"openConsultations", previous.OpenConsultations, next.OpenConsultations},
		{"openTodos", previous.OpenTodos, next.OpenTodos},
		{"reviewRequired", previous.ReviewRequired, next.ReviewRequired},
	}
	changes := make([]supervisionres.MetricChange, 0, len(values))
	for _, value := range values {
		if value.before != value.after {
			changes = append(changes, supervisionres.MetricChange{
				Key: value.key, Before: value.before, After: value.after,
			})
		}
	}
	return changes
}

func focusCasesChanged(before, after datatypes.JSON) bool {
	return !bytes.Equal(bytes.TrimSpace(before), bytes.TrimSpace(after))
}

func focusCaseSummary(attentionCase caseworkmodel.AttentionCase) caseworkres.AttentionCaseSummary {
	return caseworkres.AttentionCaseSummary{
		ID: attentionCase.ID, CareClientID: attentionCase.CareClientID,
		TaskID: attentionCase.TaskID, Status: attentionCase.Status,
		AttentionLevel: attentionCase.AttentionLevel, ReasonSummary: attentionCase.ReasonSummary,
		AssigneeID: attentionCase.AssigneeID, OpenedAt: attentionCase.OpenedAt,
		DueAt: attentionCase.DueAt, Version: attentionCase.Version,
	}
}

func stringsEqualDate(left, right time.Time) bool {
	return left.In(summaryLocation).Format("2006-01-02") == right.In(summaryLocation).Format("2006-01-02")
}
