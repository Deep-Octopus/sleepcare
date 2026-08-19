package supervision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	supervisionreq "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/request"
	supervisionres "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type snapshotMetadata struct {
	GenerationType     string
	PreviousVersionID  *uint
	CorrectionReason   string
	RevisionChanges    []supervisionres.MetricChange
	FocusCasesChanged  bool
	CommandActorID     uint
	CommandOperation   string
	CommandKeyDigest   *string
	CommandRequestHash string
	SystemOwned        bool
}

func (s *SupervisionService) GenerateSnapshot(ctx context.Context, organizationID uint, businessDate time.Time) (supervisionres.DailySummaryDetail, error) {
	identity, ok := datascope.FromContext(ctx)
	if !ok || identity == nil || !identity.IsSystem || organizationID == 0 {
		return supervisionres.DailySummaryDetail{}, supervisionmodel.NewForbiddenError(
			supervisionmodel.CodeSummaryGenerationDenied,
			"日报快照只能由受控系统流程生成",
		)
	}
	now := s.now()
	businessDate, err := summaryDate(businessDate.In(summaryLocation).Format("2006-01-02"), now)
	if err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	return s.generateSnapshot(ctx, organizationID, businessDate, now, snapshotMetadata{
		GenerationType: supervisionmodel.SummaryGenerationSystemRecompute,
		SystemOwned:    true,
	})
}

func (s *SupervisionService) EnsureScheduledSnapshot(
	ctx context.Context,
	organizationID uint,
	businessDate time.Time,
) (supervisionres.DailySummaryDetail, bool, error) {
	identity, ok := datascope.FromContext(ctx)
	if !ok || identity == nil || !identity.IsSystem || organizationID == 0 {
		return supervisionres.DailySummaryDetail{}, false, supervisionmodel.NewForbiddenError(
			supervisionmodel.CodeSummaryGenerationDenied,
			"日报快照只能由受控系统流程生成",
		)
	}
	now := s.now()
	businessDate, err := summaryDate(businessDate.In(summaryLocation).Format("2006-01-02"), now)
	if err != nil {
		return supervisionres.DailySummaryDetail{}, false, err
	}
	currentDate, _ := summaryDate("", now)
	if !businessDate.Before(currentDate) {
		return supervisionres.DailySummaryDetail{}, false, supervisionmodel.NewDomainError(
			supervisionmodel.CodeInvalidArgument,
			"自动日报只能生成已结束的业务日期",
		)
	}

	var existing supervisionmodel.DailySummaryVersion
	err = s.db().WithContext(ctx).
		Where(
			"organization_id = ? AND business_date = ? AND metric_definition_version = ? AND synthetic = ?",
			organizationID,
			businessDate,
			supervisionmodel.MetricDefinitionVersionV2,
			true,
		).
		Order("version DESC").
		First(&existing).Error
	if err == nil {
		detail, detailErr := summaryDetailFromSnapshot(existing, true)
		return detail, false, detailErr
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return supervisionres.DailySummaryDetail{}, false, err
	}

	result, err := s.generateSnapshot(ctx, organizationID, businessDate, now, snapshotMetadata{
		GenerationType: supervisionmodel.SummaryGenerationScheduled,
		SystemOwned:    true,
	})
	if err != nil {
		if duplicateError(err) {
			var replay supervisionmodel.DailySummaryVersion
			findErr := s.db().WithContext(ctx).
				Where(
					"organization_id = ? AND business_date = ? AND metric_definition_version = ? AND synthetic = ?",
					organizationID,
					businessDate,
					supervisionmodel.MetricDefinitionVersionV2,
					true,
				).
				Order("version DESC").
				First(&replay).Error
			if findErr == nil {
				detail, detailErr := summaryDetailFromSnapshot(replay, true)
				return detail, false, detailErr
			}
		}
		return supervisionres.DailySummaryDetail{}, false, err
	}
	return result, true, nil
}

func (s *SupervisionService) GenerateScheduledSnapshots(ctx context.Context) error {
	if !s.syntheticFixturesEnabled() {
		return nil
	}
	identity, ok := datascope.FromContext(ctx)
	if !ok || identity == nil || !identity.IsSystem {
		return supervisionmodel.NewForbiddenError(
			supervisionmodel.CodeSummaryGenerationDenied,
			"日报定时任务缺少系统身份",
		)
	}
	now := s.now()
	currentDate, _ := summaryDate("", now)
	businessDate := currentDate.AddDate(0, 0, -1)
	var organizationIDs []uint
	if err := s.db().WithContext(ctx).Model(&careclient.CareOrgUnitProfile{}).
		Where("unit_type = ? AND active = ? AND synthetic = ?", careclient.OrgUnitTypeOrganization, true, true).
		Order("organization_id ASC").
		Distinct("organization_id").
		Pluck("organization_id", &organizationIDs).Error; err != nil {
		return err
	}
	var runErrors []error
	for _, organizationID := range organizationIDs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, _, err := s.EnsureScheduledSnapshot(ctx, organizationID, businessDate); err != nil {
			runErrors = append(runErrors, fmt.Errorf("organization %d: %w", organizationID, err))
		}
	}
	return errors.Join(runErrors...)
}

func (s *SupervisionService) ReviseDailySummary(
	ctx context.Context,
	id uint,
	key string,
	req supervisionreq.ReviseDailySummary,
) (supervisionres.DailySummaryDetail, error) {
	reason := strings.TrimSpace(req.Reason)
	key = strings.TrimSpace(key)
	if id == 0 || req.ExpectedVersion == 0 || reason == "" || key == "" {
		return supervisionres.DailySummaryDetail{}, supervisionmodel.NewDomainError(
			supervisionmodel.CodeSummaryRevisionRequired,
			"日报版本、expectedVersion、修正原因和 Idempotency-Key 必填",
		)
	}
	decision, organizationID, err := s.supervisorScope(ctx)
	if err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	operation := fmt.Sprintf("REVISE_DAILY_SUMMARY:%d", id)
	commandDigest := digest(fmt.Sprintf("%d:%s:%s", decision.Identity.UserID, operation, key))
	requestHash, err := hashRequest(req)
	if err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	now := s.now()
	var result supervisionres.DailySummaryDetail
	err = s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing supervisionmodel.DailySummaryVersion
		existingErr := tx.Where("command_key_digest = ?", commandDigest).First(&existing).Error
		if existingErr == nil {
			if existing.OrganizationID != organizationID || existing.CommandRequestHash != requestHash {
				return supervisionmodel.NewDomainError(supervisionmodel.CodeIdempotencyConflict, "幂等键已用于不同日报修正")
			}
			var latestID uint
			if loadErr := tx.Model(&supervisionmodel.DailySummaryVersion{}).
				Where(
					"organization_id = ? AND business_date = ? AND synthetic = ?",
					existing.OrganizationID,
					existing.BusinessDate,
					true,
				).
				Order("version DESC").
				Limit(1).
				Pluck("id", &latestID).Error; loadErr != nil {
				return loadErr
			}
			var replayErr error
			result, replayErr = summaryDetailFromSnapshot(existing, existing.ID == latestID)
			return replayErr
		}
		if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
			return existingErr
		}

		var base supervisionmodel.DailySummaryVersion
		if loadErr := summaryLocking(tx).
			Where("id = ? AND organization_id = ? AND synthetic = ?", id, organizationID, true).
			First(&base).Error; loadErr != nil {
			if errors.Is(loadErr, gorm.ErrRecordNotFound) {
				return supervisionmodel.NewForbiddenError(supervisionmodel.CodeReviewScopeDenied, "日报版本不存在或不在当前管理范围")
			}
			return loadErr
		}
		var latest supervisionmodel.DailySummaryVersion
		if loadErr := summaryLocking(tx).
			Where("organization_id = ? AND business_date = ? AND synthetic = ?", organizationID, base.BusinessDate, true).
			Order("version DESC").
			First(&latest).Error; loadErr != nil {
			return loadErr
		}
		if latest.ID != base.ID || base.Version != req.ExpectedVersion {
			return supervisionmodel.NewDomainError(supervisionmodel.CodeSummaryNotLatest, "只能基于该业务日期的最新日报版本修正")
		}
		currentDate, _ := summaryDate("", now)
		if !base.BusinessDate.Before(currentDate) {
			return supervisionmodel.NewDomainError(supervisionmodel.CodeSummaryRevisionRequired, "今日实时数据不能创建历史修正版")
		}

		worker := &SupervisionService{DB: tx, Now: s.Now, SyntheticFixturesEnabled: s.SyntheticFixturesEnabled}
		computed, computeErr := worker.computeSummary(ctx, organizationID, base.BusinessDate, now)
		if computeErr != nil {
			return computeErr
		}
		if latest.MetricDefinitionVersion == supervisionmodel.MetricDefinitionVersionV2 &&
			latest.SourceDigest == computed.SourceDigest {
			return supervisionmodel.NewDomainError(supervisionmodel.CodeSummaryUnchanged, "原始记录复算结果没有变化，无需生成新版本")
		}
		changes := buildMetricChanges(latest, computed.Detail.DailySummary)
		focusChanged := focusCasesChanged(latest.FocusCasesJSON, computed.FocusCasesJSON)
		previousID := latest.ID
		keyDigest := commandDigest
		created, createErr := worker.createSnapshot(tx, organizationID, latest.Version+1, computed, snapshotMetadata{
			GenerationType:     supervisionmodel.SummaryGenerationCorrection,
			PreviousVersionID:  &previousID,
			CorrectionReason:   reason,
			RevisionChanges:    changes,
			FocusCasesChanged:  focusChanged,
			CommandActorID:     decision.Identity.UserID,
			CommandOperation:   operation,
			CommandKeyDigest:   &keyDigest,
			CommandRequestHash: requestHash,
		})
		if createErr != nil {
			if duplicateError(createErr) {
				return supervisionmodel.NewDomainError(supervisionmodel.CodeIdempotencyConflict, "日报修正发生并发冲突，请重试")
			}
			return createErr
		}
		result = created
		return nil
	})
	return result, err
}

func (s *SupervisionService) generateSnapshot(
	ctx context.Context,
	organizationID uint,
	businessDate time.Time,
	now time.Time,
	metadata snapshotMetadata,
) (supervisionres.DailySummaryDetail, error) {
	var result supervisionres.DailySummaryDetail
	err := s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		worker := &SupervisionService{DB: tx, Now: s.Now, SyntheticFixturesEnabled: s.SyntheticFixturesEnabled}
		computed, computeErr := worker.computeSummary(ctx, organizationID, businessDate, now)
		if computeErr != nil {
			return computeErr
		}
		var latest supervisionmodel.DailySummaryVersion
		findErr := summaryLocking(tx).
			Where("organization_id = ? AND business_date = ? AND synthetic = ?", organizationID, businessDate, true).
			Order("version DESC").
			First(&latest).Error
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		version := uint(1)
		if findErr == nil {
			version = latest.Version + 1
			previousID := latest.ID
			metadata.PreviousVersionID = &previousID
			metadata.RevisionChanges = buildMetricChanges(latest, computed.Detail.DailySummary)
			metadata.FocusCasesChanged = focusCasesChanged(latest.FocusCasesJSON, computed.FocusCasesJSON)
		}
		created, createErr := worker.createSnapshot(tx, organizationID, version, computed, metadata)
		if createErr != nil {
			return createErr
		}
		result = created
		return nil
	})
	return result, err
}

func (s *SupervisionService) createSnapshot(
	tx *gorm.DB,
	organizationID uint,
	version uint,
	computed computedSummary,
	metadata snapshotMetadata,
) (supervisionres.DailySummaryDetail, error) {
	changesJSON, err := json.Marshal(metadata.RevisionChanges)
	if err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	businessDate, err := summaryDate(computed.Detail.BusinessDate, computed.SourceCutoffAt)
	if err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	sourceCutoff := computed.SourceCutoffAt
	snapshot := supervisionmodel.DailySummaryVersion{
		OrganizationID:          organizationID,
		BusinessDate:            businessDate,
		Version:                 version,
		MetricDefinitionVersion: supervisionmodel.MetricDefinitionVersionV2,
		GenerationType:          metadata.GenerationType,
		GeneratedAt:             s.now(),
		SourceCutoffAt:          &sourceCutoff,
		PreviousVersionID:       metadata.PreviousVersionID,
		CorrectionReason:        metadata.CorrectionReason,
		SourceDigest:            computed.SourceDigest,
		RevisionChangesJSON:     datatypes.JSON(changesJSON),
		FocusCasesChanged:       metadata.FocusCasesChanged,
		CommandActorID:          metadata.CommandActorID,
		CommandOperation:        metadata.CommandOperation,
		CommandKeyDigest:        metadata.CommandKeyDigest,
		CommandRequestHash:      metadata.CommandRequestHash,
		ServedClients:           computed.Detail.ServedClients,
		DueTasks:                computed.Detail.DueTasks,
		SubmittedTasks:          computed.Detail.SubmittedTasks,
		OverdueTasks:            computed.Detail.OverdueTasks,
		DeliveryIssues:          computed.Detail.DeliveryIssues,
		OpenAttentionCases:      computed.Detail.OpenAttentionCases,
		ResolvedAttentionCases:  computed.Detail.ResolvedAttentionCases,
		ConsultationsOpened:     computed.Detail.ConsultationsOpened,
		ConsultationsClosed:     computed.Detail.ConsultationsClosed,
		OpenConsultations:       computed.Detail.OpenConsultations,
		OpenTodos:               computed.Detail.OpenTodos,
		ReviewRequired:          computed.Detail.ReviewRequired,
		FocusCasesJSON:          computed.FocusCasesJSON,
		Synthetic:               true,
	}
	if metadata.SystemOwned {
		snapshot.DeptId = organizationID
	}
	if err = tx.Create(&snapshot).Error; err != nil {
		return supervisionres.DailySummaryDetail{}, err
	}
	return summaryDetailFromSnapshot(snapshot, true)
}
