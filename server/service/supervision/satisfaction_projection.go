package supervision

import (
	"context"
	"errors"
	"time"

	platformoutbox "github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

type ClientSatisfactionIdentity struct {
	CareClientID uint
	DeptID       uint
	Synthetic    bool
}

func (s *SupervisionService) ProjectConsultationClosed(
	ctx context.Context,
	tx *gorm.DB,
	consultation caseworkmodel.Consultation,
	interaction caseworkmodel.ConsultationInteraction,
) error {
	if !s.syntheticFixturesEnabled() || !consultation.Synthetic ||
		interaction.ActionType != caseworkmodel.ConsultationActionClose || interaction.ID == 0 {
		return nil
	}
	if tx == nil {
		tx = s.db().WithContext(ctx)
	}
	systemCtx := datascope.WithSystem(ctx)
	systemDB := func() *gorm.DB {
		return tx.Session(&gorm.Session{NewDB: true}).
			WithContext(systemCtx).
			Set("data_scope:skip", true)
	}
	var existing supervisionmodel.SatisfactionRequest
	err := systemDB().Where(
		"source_type = ? AND source_event_id = ?",
		supervisionmodel.SatisfactionSourceConsultation,
		interaction.ID,
	).First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	policy, found, err := s.satisfactionPolicyAt(systemDB(), interaction.OccurredAt)
	if err != nil || !found {
		return err
	}
	var client caremodel.CareClient
	if err = systemDB().Where("id = ? AND synthetic = ?", consultation.CareClientID, true).
		First(&client).Error; err != nil {
		return err
	}
	expiresAt := interaction.OccurredAt.Add(time.Duration(policy.ValidForHours) * time.Hour)
	status := supervisionmodel.SatisfactionRequestPending
	if !s.now().Before(expiresAt) {
		status = supervisionmodel.SatisfactionRequestExpired
	}
	request := supervisionmodel.SatisfactionRequest{
		SourceType:          supervisionmodel.SatisfactionSourceConsultation,
		SourceID:            consultation.ID,
		SourceEventID:       interaction.ID,
		CareClientID:        consultation.CareClientID,
		OrganizationID:      client.OrganizationID,
		ServiceAssigneeID:   consultation.AssigneeID,
		ServiceAssigneeRole: consultation.AssigneeRole,
		PolicyID:            policy.ID,
		PolicyCode:          policy.Code,
		PolicyVersion:       policy.Version,
		AnonymityMode:       policy.AnonymityMode,
		LowScoreThreshold:   policy.LowScoreThreshold,
		Status:              status,
		InvitedAt:           interaction.OccurredAt,
		ExpiresAt:           expiresAt,
		Version:             1,
		Synthetic:           true,
		DeptId:              consultation.DeptId,
		CreatedBy:           interaction.ActorID,
	}
	if err = systemDB().Create(&request).Error; err != nil {
		if duplicateError(err) {
			return nil
		}
		return err
	}
	return platformoutbox.Append(systemDB(), platformoutbox.AppendInput{
		EventType:     supervisionmodel.EventSatisfactionRequested,
		AggregateType: "SatisfactionRequest",
		AggregateID:   request.ID,
		Payload: map[string]any{
			"requestId":      request.ID,
			"sourceType":     request.SourceType,
			"sourceId":       request.SourceID,
			"sourceEventId":  request.SourceEventID,
			"careClientId":   request.CareClientID,
			"organizationId": request.OrganizationID,
			"status":         request.Status,
			"policyCode":     request.PolicyCode,
			"policyVersion":  request.PolicyVersion,
			"synthetic":      request.Synthetic,
		},
		OccurredAt:  request.InvitedAt,
		CausationID: interaction.CommandKeyDigest,
		Synthetic:   request.Synthetic,
		DeptID:      request.DeptId,
		CreatedBy:   request.CreatedBy,
	})
}

func (s *SupervisionService) ReconcileClientSatisfactionRequests(
	ctx context.Context,
	identity ClientSatisfactionIdentity,
) error {
	if !s.syntheticFixturesEnabled() || !identity.Synthetic || identity.CareClientID == 0 || identity.DeptID == 0 {
		return supervisionmodel.NewForbiddenError(
			supervisionmodel.CodeSatisfactionScopeDenied,
			"当前会话不能查看服务评价",
		)
	}
	var client caremodel.CareClient
	if err := s.db().WithContext(ctx).Where(
		"id = ? AND dept_id = ? AND status = ? AND synthetic = ?",
		identity.CareClientID,
		identity.DeptID,
		caremodel.ClientStatusActive,
		true,
	).First(&client).Error; err != nil {
		return supervisionmodel.NewForbiddenError(
			supervisionmodel.CodeSatisfactionScopeDenied,
			"当前会话不能查看服务评价",
		)
	}
	return s.reconcileSatisfactionRequests(ctx, identity.CareClientID, 0)
}

func (s *SupervisionService) reconcileOrganizationSatisfactionRequests(
	ctx context.Context,
	organizationID uint,
) error {
	if !s.syntheticFixturesEnabled() || organizationID == 0 {
		return nil
	}
	return s.reconcileSatisfactionRequests(ctx, 0, organizationID)
}

func (s *SupervisionService) reconcileSatisfactionRequests(
	ctx context.Context,
	careClientID uint,
	organizationID uint,
) error {
	systemCtx := datascope.WithSystem(ctx)
	query := s.db().WithContext(systemCtx).Set("data_scope:skip", true).
		Model(&caseworkmodel.ConsultationInteraction{}).
		Select("consultation_interactions.*").
		Joins("JOIN consultations ON consultations.id = consultation_interactions.consultation_id AND consultations.deleted_at IS NULL").
		Joins("JOIN care_clients ON care_clients.id = consultations.care_client_id AND care_clients.deleted_at IS NULL").
		Where(
			"consultation_interactions.action_type = ? AND consultation_interactions.synthetic = ? AND consultations.synthetic = ?",
			caseworkmodel.ConsultationActionClose,
			true,
			true,
		)
	if careClientID != 0 {
		query = query.Where("consultations.care_client_id = ?", careClientID)
	}
	if organizationID != 0 {
		query = query.Where("care_clients.organization_id = ?", organizationID)
	}
	var interactions []caseworkmodel.ConsultationInteraction
	if err := query.Order("consultation_interactions.id ASC").Find(&interactions).Error; err != nil {
		return err
	}
	if len(interactions) == 0 {
		return nil
	}
	return s.db().WithContext(systemCtx).Transaction(func(tx *gorm.DB) error {
		for _, interaction := range interactions {
			var consultation caseworkmodel.Consultation
			if err := tx.Set("data_scope:skip", true).
				Where("id = ?", interaction.ConsultationID).
				First(&consultation).Error; err != nil {
				return err
			}
			if err := s.ProjectConsultationClosed(systemCtx, tx, consultation, interaction); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *SupervisionService) satisfactionPolicyAt(
	db *gorm.DB,
	occurredAt time.Time,
) (supervisionmodel.SatisfactionPolicy, bool, error) {
	var policy supervisionmodel.SatisfactionPolicy
	err := db.Where(
		"trigger_type = ? AND status = ? AND synthetic = ? AND effective_from <= ?",
		supervisionmodel.SatisfactionTriggerConsultation,
		supervisionmodel.SatisfactionPolicyStatusPublished,
		true,
		occurredAt,
	).Order("effective_from DESC, version DESC, id DESC").First(&policy).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return policy, false, nil
	}
	return policy, err == nil, err
}
