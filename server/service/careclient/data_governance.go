package careclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	carereq "github.com/flipped-aurora/gin-vue-admin/server/model/careclient/request"
	careres "github.com/flipped-aurora/gin-vue-admin/server/model/careclient/response"
	"gorm.io/gorm"
)

func (s *CareClientService) GetDataGovernanceReadiness(ctx context.Context) (careres.DataGovernanceReadiness, error) {
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return careres.DataGovernanceReadiness{}, err
	}
	if !decision.CanManage() {
		return careres.DataGovernanceReadiness{}, caremodel.NewForbiddenError(caremodel.CodeAccessScopeDenied, "仅督导角色可查看数据治理准备状态")
	}
	readiness, _ := buildDataGovernanceReadiness(s.dataGovernanceConfig(), s.syntheticFixturesEnabled())
	return readiness, nil
}

func (s *CareClientService) ListDataLifecycleRequests(
	ctx context.Context,
	id uint,
	req carereq.DataLifecycleRequestSearch,
) ([]careres.DataLifecycleRequestSummary, int64, error) {
	if _, _, err := s.manageableClient(ctx, id); err != nil {
		return nil, 0, err
	}
	if req.RequestType != "" && !validLifecycleRequestType(req.RequestType) {
		return nil, 0, caremodel.NewDomainError(caremodel.CodeLifecycleRequestInvalid, "数据生命周期请求类型无效")
	}
	query := s.db().WithContext(ctx).Model(&caremodel.DataLifecycleRequest{}).
		Where("care_client_id = ? AND synthetic = ?", id, true)
	if req.RequestType != "" {
		query = query.Where("request_type = ?", req.RequestType)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	limit, offset := req.LimitOffset()
	var records []caremodel.DataLifecycleRequest
	if err := query.Order("requested_at DESC, id DESC").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	items := make([]careres.DataLifecycleRequestSummary, 0, len(records))
	for _, record := range records {
		items = append(items, dataLifecycleRequestSummary(record))
	}
	return items, total, nil
}

func (s *CareClientService) CreateDataLifecycleRequest(
	ctx context.Context,
	id uint,
	key string,
	req carereq.CreateDataLifecycleRequest,
) (careres.ActionResult, error) {
	_, client, err := s.manageableClient(ctx, id)
	if err != nil {
		return careres.ActionResult{}, err
	}
	reason := strings.TrimSpace(req.Reason)
	if req.ExpectedVersion == 0 || !validLifecycleRequestType(req.RequestType) || req.RequestedAt.IsZero() ||
		req.Source != caremodel.LifecycleRequestSourceStaffRecorded || reason == "" || len([]rune(reason)) > 1000 {
		return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeLifecycleRequestInvalid, "expectedVersion、有效请求类型、请求时间、记录来源和原因必填")
	}
	commandCtx := withDepartment(ctx, client.DeptId)
	operation := fmt.Sprintf("CREATE_DATA_LIFECYCLE_REQUEST:%d", id)
	return s.runIdempotent(commandCtx, operation, key, req, func(tx *gorm.DB) (careres.ActionResult, error) {
		cfg := s.dataGovernanceConfig()
		readiness, digest := buildDataGovernanceReadiness(cfg, s.syntheticFixturesEnabled())
		if !readiness.RequestRecordingEnabled {
			return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeDataGovernanceDisabled, "数据生命周期请求记录门禁未开启")
		}
		if err := lockClient(tx, id, req.ExpectedVersion); err != nil {
			return careres.ActionResult{}, err
		}
		record := caremodel.DataLifecycleRequest{
			OrganizationID:             client.OrganizationID,
			CareClientID:               id,
			RequestType:                req.RequestType,
			Status:                     caremodel.LifecycleRequestStatusPendingPolicy,
			RequestedAt:                req.RequestedAt,
			Source:                     req.Source,
			Reason:                     reason,
			IdentityVerificationStatus: caremodel.IdentityVerificationStatusNotConfigured,
			GovernanceMode:             cfg.NormalizedMode(),
			PolicySnapshotDigest:       digest,
			ExecutionAllowed:           false,
			Synthetic:                  true,
		}
		if err := tx.Create(&record).Error; err != nil {
			return careres.ActionResult{}, err
		}
		if err := bumpVersion(tx, id, req.ExpectedVersion); err != nil {
			return careres.ActionResult{}, err
		}
		return careres.ActionResult{
			CareClientID: id,
			ResourceID:   record.ID,
			Version:      req.ExpectedVersion + 1,
		}, nil
	})
}

func buildDataGovernanceReadiness(cfg config.DataGovernance, fixturesEnabled bool) (careres.DataGovernanceReadiness, string) {
	requestRecordingEnabled := cfg.ContractTestEnabled(fixturesEnabled)
	requirements := []careres.ConsentPolicyRequirement{
		{
			ConsentType:      caremodel.ConsentRequirementServiceNotice,
			PolicyVersion:    strings.TrimSpace(cfg.ServiceNoticeVersion),
			ContentReviewed:  cfg.ServiceNoticeReviewed,
			RecordingEnabled: false,
		},
		{
			ConsentType:      caremodel.ConsentRequirementPrivacyNotice,
			PolicyVersion:    strings.TrimSpace(cfg.PrivacyNoticeVersion),
			ContentReviewed:  cfg.PrivacyNoticeReviewed,
			RecordingEnabled: false,
		},
		{
			ConsentType:      caremodel.ConsentRequirementNotification,
			PolicyVersion:    strings.TrimSpace(cfg.NotificationConsentVersion),
			ContentReviewed:  cfg.NotificationConsentReviewed,
			RecordingEnabled: false,
		},
		{
			ConsentType:      caremodel.ConsentRequirementAIProcessing,
			PolicyVersion:    strings.TrimSpace(cfg.AIProcessingConsentVersion),
			ContentReviewed:  cfg.AIProcessingConsentReviewed,
			RecordingEnabled: false,
		},
	}
	gates := []careres.GovernanceReviewGate{
		{Key: "IDENTITY_VERIFICATION", Reviewed: cfg.IdentityVerificationReviewed},
		{Key: "CONSENT_EVIDENCE", Reviewed: cfg.ConsentEvidenceReviewed},
		{Key: "WITHDRAWAL_POLICY", Reviewed: cfg.WithdrawalPolicyReviewed},
		{Key: "MINIMUM_NECESSARY_FIELDS", Reviewed: cfg.MinimumNecessaryFieldsReviewed},
		{Key: "RETENTION_POLICY", Reviewed: cfg.RetentionPolicyReviewed},
		{Key: "CORRECTION_POLICY", Reviewed: cfg.CorrectionPolicyReviewed},
		{Key: "ERASURE_POLICY", Reviewed: cfg.ErasurePolicyReviewed},
		{Key: "EXPORT_POLICY", Reviewed: cfg.ExportPolicyReviewed},
		{Key: "SENSITIVE_ACCESS_AUDIT", Reviewed: cfg.SensitiveAccessAuditReviewed},
		{Key: "BACKUP_RESTORE", Reviewed: cfg.BackupRestoreReviewed},
	}
	blockingItems := []string{
		"REAL_DATA_MODE_UNAVAILABLE",
		"FORMAL_CONSENT_RECORDING_UNAVAILABLE",
		"LIFECYCLE_EXECUTION_UNAVAILABLE",
	}
	if !requestRecordingEnabled {
		blockingItems = append(blockingItems, "CONTRACT_TEST_RECORDING_DISABLED")
	}
	for _, requirement := range requirements {
		if !requirement.ContentReviewed || requirement.PolicyVersion == "" {
			blockingItems = append(blockingItems, requirement.ConsentType+"_NOT_APPROVED")
		}
	}
	for _, gate := range gates {
		if !gate.Reviewed {
			blockingItems = append(blockingItems, gate.Key+"_NOT_APPROVED")
		}
	}
	readiness := careres.DataGovernanceReadiness{
		Mode:                      cfg.NormalizedMode(),
		UsageScope:                caremodel.DataGovernanceUsageTestOnly,
		RealDataEnabled:           false,
		FormalConsentEnabled:      false,
		LifecycleExecutionEnabled: false,
		RequestRecordingEnabled:   requestRecordingEnabled,
		ConsentRequirements:       requirements,
		ReviewGates:               gates,
		BlockingItems:             blockingItems,
	}
	payload, _ := json.Marshal(cfg)
	sum := sha256.Sum256(payload)
	return readiness, hex.EncodeToString(sum[:])
}

func validLifecycleRequestType(value string) bool {
	switch value {
	case caremodel.LifecycleRequestAccessCopy,
		caremodel.LifecycleRequestCorrection,
		caremodel.LifecycleRequestRestriction,
		caremodel.LifecycleRequestErasure:
		return true
	default:
		return false
	}
}

func dataLifecycleRequestSummary(record caremodel.DataLifecycleRequest) careres.DataLifecycleRequestSummary {
	return careres.DataLifecycleRequestSummary{
		ID:                         record.ID,
		CareClientID:               record.CareClientID,
		RequestType:                record.RequestType,
		Status:                     record.Status,
		RequestedAt:                record.RequestedAt,
		Source:                     record.Source,
		Reason:                     record.Reason,
		IdentityVerificationStatus: record.IdentityVerificationStatus,
		GovernanceMode:             record.GovernanceMode,
		ExecutionAllowed:           record.ExecutionAllowed,
		RecordedAt:                 record.CreatedAt,
	}
}
