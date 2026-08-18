package supervision

import (
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
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *SupervisionService) ListDailySummaries(ctx context.Context, req supervisionreq.DailySummarySearch) ([]supervisionres.DailySummary, int64, error) {
	_, organizationID, err := s.supervisorScope(ctx)
	if err != nil {
		return nil, 0, err
	}
	businessDate, err := summaryDate(req.BusinessDate, s.now())
	if err != nil {
		return nil, 0, err
	}
	currentDate, _ := summaryDate("", s.now())
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
			preview, computeErr := s.computeSummary(ctx, organizationID, businessDate)
			if computeErr != nil {
				return nil, 0, computeErr
			}
			items = append(items, preview.DailySummary)
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
	for i := range snapshots {
		items = append(items, summaryFromSnapshot(snapshots[i]))
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
	var focusCases []caseworkres.AttentionCaseSummary
	if err = json.Unmarshal(snapshot.FocusCasesJSON, &focusCases); err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	return supervisionres.DailySummaryDetail{
		DailySummary: summaryFromSnapshot(snapshot),
		FocusCases:   focusCases,
	}, nil
}

func (s *SupervisionService) GenerateSnapshot(ctx context.Context, organizationID uint, businessDate time.Time) (supervisionres.DailySummaryDetail, error) {
	identity, ok := datascope.FromContext(ctx)
	if !ok || identity == nil || !identity.IsSystem || organizationID == 0 {
		return supervisionres.DailySummaryDetail{}, supervisionmodel.NewForbiddenError(supervisionmodel.CodeReviewScopeDenied, "日报快照只能由受控系统流程生成")
	}
	businessDate, err := summaryDate(businessDate.In(summaryLocation).Format("2006-01-02"), s.now())
	if err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	if _, _, err = summaryBounds(businessDate, s.now()); err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}

	var result supervisionres.DailySummaryDetail
	err = s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		worker := &SupervisionService{DB: tx, Now: s.Now}
		computed, computeErr := worker.computeSummary(ctx, organizationID, businessDate)
		if computeErr != nil {
			return computeErr
		}
		var latest supervisionmodel.DailySummaryVersion
		findErr := summaryLocking(tx).
			Where("organization_id = ? AND business_date = ?", organizationID, businessDate).
			Order("version DESC").First(&latest).Error
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		version := uint(1)
		if findErr == nil {
			version = latest.Version + 1
		}
		focusJSON, marshalErr := json.Marshal(computed.FocusCases)
		if marshalErr != nil {
			return marshalErr
		}
		snapshot := supervisionmodel.DailySummaryVersion{
			OrganizationID:          organizationID,
			BusinessDate:            businessDate,
			Version:                 version,
			MetricDefinitionVersion: supervisionmodel.MetricDefinitionVersionV1,
			GeneratedAt:             s.now(),
			ServedClients:           computed.ServedClients,
			DueTasks:                computed.DueTasks,
			SubmittedTasks:          computed.SubmittedTasks,
			DeliveryIssues:          computed.DeliveryIssues,
			OpenAttentionCases:      computed.OpenAttentionCases,
			ResolvedAttentionCases:  computed.ResolvedAttentionCases,
			ReviewRequired:          computed.ReviewRequired,
			FocusCasesJSON:          datatypes.JSON(focusJSON),
			Synthetic:               true,
			DeptId:                  organizationID,
		}
		if createErr := tx.Create(&snapshot).Error; createErr != nil {
			return createErr
		}
		computed.ID = snapshot.ID
		computed.SummaryType = supervisionmodel.SummaryTypeVersionedSnapshot
		computed.Version = &snapshot.Version
		result = computed
		return nil
	})
	return result, err
}

func (s *SupervisionService) computeSummary(ctx context.Context, organizationID uint, businessDate time.Time) (supervisionres.DailySummaryDetail, error) {
	start, cutoff, err := summaryBounds(businessDate, s.now())
	if err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	detail := supervisionres.DailySummaryDetail{
		DailySummary: supervisionres.DailySummary{
			BusinessDate: start.Format("2006-01-02"),
			SummaryType:  supervisionmodel.SummaryTypeRealtimePreview,
		},
		FocusCases: make([]caseworkres.AttentionCaseSummary, 0),
	}

	var clientIDs []uint
	if err = s.db().WithContext(ctx).Model(&careclient.CareClient{}).
		Where("organization_id = ? AND status = ? AND synthetic = ?", organizationID, careclient.ClientStatusActive, true).
		Order("id ASC").Pluck("id", &clientIDs).Error; err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	if len(clientIDs) == 0 {
		return detail, nil
	}

	taskBase := func() *gorm.DB {
		return s.db().WithContext(ctx).Model(&pathmodel.TaskInstance{}).
			Where("care_client_id IN ? AND synthetic = ?", clientIDs, true)
	}
	dayEnd := start.AddDate(0, 0, 1)
	if err = taskBase().Where("due_at >= ? AND due_at < ?", start, dayEnd).
		Distinct("care_client_id").Count(&detail.ServedClients).Error; err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	if err = taskBase().Where("due_at >= ? AND due_at < ?", start, dayEnd).
		Count(&detail.DueTasks).Error; err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	if err = taskBase().Where("submitted_at >= ? AND submitted_at < ?", start, cutoff).
		Count(&detail.SubmittedTasks).Error; err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}

	if err = s.db().WithContext(ctx).Model(&caseworkmodel.TodoItem{}).
		Where("care_client_id IN ? AND synthetic = ? AND category = ?", clientIDs, true, caseworkmodel.TodoCategoryDeliveryIssue).
		Where("opened_at < ? AND (completed_at IS NULL OR completed_at >= ?)", cutoff, cutoff).
		Count(&detail.DeliveryIssues).Error; err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}

	openCases := func() *gorm.DB {
		return s.db().WithContext(ctx).Model(&caseworkmodel.AttentionCase{}).
			Where("care_client_id IN ? AND synthetic = ?", clientIDs, true).
			Where("opened_at < ? AND (closed_at IS NULL OR closed_at >= ?)", cutoff, cutoff)
	}
	if err = openCases().Count(&detail.OpenAttentionCases).Error; err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	if err = s.db().WithContext(ctx).Model(&caseworkmodel.AttentionCase{}).
		Where("care_client_id IN ? AND synthetic = ?", clientIDs, true).
		Where("resolved_at >= ? AND resolved_at < ?", start, cutoff).
		Count(&detail.ResolvedAttentionCases).Error; err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	if err = openCases().Where("status = ?", caseworkmodel.CaseStatusWaitingSupervisor).
		Count(&detail.ReviewRequired).Error; err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}

	var cases []caseworkmodel.AttentionCase
	if err = openCases().Order("opened_at DESC, id DESC").Limit(100).Find(&cases).Error; err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	for i := range cases {
		detail.FocusCases = append(detail.FocusCases, focusCaseSummary(cases[i]))
	}
	return detail, nil
}

func summaryFromSnapshot(snapshot supervisionmodel.DailySummaryVersion) supervisionres.DailySummary {
	version := snapshot.Version
	return supervisionres.DailySummary{
		ID:                     snapshot.ID,
		BusinessDate:           snapshot.BusinessDate.Format("2006-01-02"),
		SummaryType:            supervisionmodel.SummaryTypeVersionedSnapshot,
		Version:                &version,
		ServedClients:          snapshot.ServedClients,
		DueTasks:               snapshot.DueTasks,
		SubmittedTasks:         snapshot.SubmittedTasks,
		DeliveryIssues:         snapshot.DeliveryIssues,
		OpenAttentionCases:     snapshot.OpenAttentionCases,
		ResolvedAttentionCases: snapshot.ResolvedAttentionCases,
		ReviewRequired:         snapshot.ReviewRequired,
	}
}

func focusCaseSummary(attentionCase caseworkmodel.AttentionCase) caseworkres.AttentionCaseSummary {
	return caseworkres.AttentionCaseSummary{
		ID:             attentionCase.ID,
		CareClientID:   attentionCase.CareClientID,
		TaskID:         attentionCase.TaskID,
		Status:         attentionCase.Status,
		AttentionLevel: attentionCase.AttentionLevel,
		ReasonSummary:  attentionCase.ReasonSummary,
		AssigneeID:     attentionCase.AssigneeID,
		OpenedAt:       attentionCase.OpenedAt,
		DueAt:          attentionCase.DueAt,
		Version:        attentionCase.Version,
	}
}

func stringsEqualDate(left, right time.Time) bool {
	return left.In(summaryLocation).Format("2006-01-02") == right.In(summaryLocation).Format("2006-01-02")
}
