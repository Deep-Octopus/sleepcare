package casework

import (
	"context"
	"errors"
	"sort"
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkreq "github.com/flipped-aurora/gin-vue-admin/server/model/casework/request"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"gorm.io/gorm"
)

func (s *CaseWorkService) ListConsultations(
	ctx context.Context,
	req caseworkreq.ConsultationSearch,
) ([]caseworkres.ConsultationSummary, int64, error) {
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return nil, 0, normalizeAccessError(err)
	}
	if req.Status != "" && !validConsultationStatus(req.Status) {
		return nil, 0, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "咨询状态无效")
	}
	if req.Urgency != "" && !validConsultationUrgency(req.Urgency) {
		return nil, 0, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "联系优先级无效")
	}
	query := decision.ScopeConsultations(
		s.db().WithContext(ctx).Model(&caseworkmodel.Consultation{}),
		s.now(),
	).Where("consultations.synthetic = ?", true)
	if req.Status != "" {
		query = query.Where("consultations.status = ?", req.Status)
	}
	if req.Urgency != "" {
		query = query.Where("consultations.urgency = ?", req.Urgency)
	}
	if req.AssigneeID != 0 {
		query = query.Where("consultations.assignee_id = ?", req.AssigneeID)
	}
	var total int64
	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	limit, offset := req.LimitOffset()
	var rows []caseworkmodel.Consultation
	if err = query.Order("consultations.opened_at DESC, consultations.id DESC").
		Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items, err := s.consultationSummaries(ctx, rows)
	return items, total, err
}

func (s *CaseWorkService) GetConsultation(
	ctx context.Context,
	id uint,
) (caseworkres.ConsultationDetail, error) {
	if id == 0 {
		return caseworkres.ConsultationDetail{}, caseworkmodel.NewDomainError(
			caseworkmodel.CodeInvalidArgument,
			"咨询标识必填",
		)
	}
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.ConsultationDetail{}, normalizeAccessError(err)
	}
	var consultation caseworkmodel.Consultation
	query := decision.ScopeConsultations(
		s.db().WithContext(ctx).Model(&caseworkmodel.Consultation{}),
		s.now(),
	).Where("consultations.synthetic = ?", true)
	err = query.Where("consultations.id = ?", id).First(&consultation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return caseworkres.ConsultationDetail{}, caseworkmodel.NewForbiddenError(
			caseworkmodel.CodeAccessScopeDenied,
			"咨询不存在或不在当前访问范围",
		)
	}
	if err != nil {
		return caseworkres.ConsultationDetail{}, err
	}

	var interactions []caseworkmodel.ConsultationInteraction
	if err = s.db().WithContext(ctx).Set("data_scope:skip", true).
		Where("consultation_id = ?", consultation.ID).
		Order("occurred_at ASC, id ASC").Find(&interactions).Error; err != nil {
		return caseworkres.ConsultationDetail{}, err
	}
	summaries, err := s.consultationSummaries(ctx, []caseworkmodel.Consultation{consultation})
	if err != nil {
		return caseworkres.ConsultationDetail{}, err
	}
	detail := caseworkres.ConsultationDetail{
		ConsultationSummary: summaries[0],
		InitialQuestion:     consultation.InitialQuestion,
		Resolution:          optionalString(consultation.Resolution),
		FollowUpPlan:        optionalString(consultation.FollowUpPlan),
		CloseReason:         optionalString(consultation.CloseReason),
		Interactions:        make([]caseworkres.ConsultationInteraction, 0, len(interactions)),
	}
	actorNames, err := s.consultationActorNames(ctx, interactions)
	if err != nil {
		return caseworkres.ConsultationDetail{}, err
	}
	for _, interaction := range interactions {
		detail.Interactions = append(detail.Interactions, caseworkres.ConsultationInteraction{
			ID:         interaction.ID,
			ActionType: interaction.ActionType,
			ActorType:  interaction.ActorType,
			ActorRole:  interaction.ActorRole,
			ActorName:  actorNames[interactionActorKey(interaction.ActorType, interaction.ActorID)],
			Content:    interaction.Content,
			Reason:     optionalString(interaction.Reason),
			FromStatus: interaction.FromStatus,
			ToStatus:   interaction.ToStatus,
			TargetRole: interaction.TargetRole,
			OccurredAt: interaction.OccurredAt,
		})
	}
	return detail, nil
}

func (s *CaseWorkService) ListConsultationAssigneeOptions(
	ctx context.Context,
	id uint,
) ([]caseworkres.ConsultationAssigneeOption, error) {
	if id == 0 {
		return nil, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "咨询标识必填")
	}
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return nil, normalizeAccessError(err)
	}
	var consultation caseworkmodel.Consultation
	query := decision.ScopeConsultations(
		s.db().WithContext(ctx).Model(&caseworkmodel.Consultation{}),
		s.now(),
	).Where("consultations.synthetic = ?", true)
	err = query.Where("consultations.id = ?", id).First(&consultation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, caseworkmodel.NewForbiddenError(
			caseworkmodel.CodeAccessScopeDenied,
			"咨询不存在或不在当前访问范围",
		)
	}
	if err != nil {
		return nil, err
	}

	var client caremodel.CareClient
	if err = s.db().WithContext(ctx).Set("data_scope:skip", true).
		Where("id = ?", consultation.CareClientID).First(&client).Error; err != nil {
		return nil, err
	}
	now := s.now()
	var assignments []caremodel.CareAssignment
	if err = s.db().WithContext(ctx).Set("data_scope:skip", true).
		Where("care_client_id = ? AND synthetic = ? AND cancelled_at IS NULL AND valid_from <= ?", client.ID, true, now).
		Where("valid_until IS NULL OR valid_until > ?", now).
		Where("role_type IN ?", []string{caremodel.AssignmentRoleCareSteward, caremodel.AssignmentRoleClinician}).
		Order("role_type ASC, id DESC").Find(&assignments).Error; err != nil {
		return nil, err
	}

	userIDs := make([]uint, 0, len(assignments))
	for _, assignment := range assignments {
		userIDs = append(userIDs, assignment.AssigneeID)
	}
	var supervisors []system.SysUser
	if err = s.db().WithContext(ctx).Set("data_scope:skip", true).
		Where("dept_id = ? AND enable = ?", client.OrganizationID, 1).
		Order("id ASC").Find(&supervisors).Error; err != nil {
		return nil, err
	}
	for _, supervisor := range supervisors {
		userIDs = append(userIDs, supervisor.ID)
	}
	users := make(map[uint]system.SysUser)
	if len(userIDs) > 0 {
		var values []system.SysUser
		if err = s.db().WithContext(ctx).Set("data_scope:skip", true).
			Where("id IN ? AND enable = ?", userIDs, 1).Find(&values).Error; err != nil {
			return nil, err
		}
		for _, value := range values {
			users[value.ID] = value
		}
	}

	options := make([]caseworkres.ConsultationAssigneeOption, 0, len(assignments)+len(supervisors))
	seen := make(map[string]struct{})
	appendOption := func(userID uint, role string) {
		user, ok := users[userID]
		if !ok {
			return
		}
		key := interactionActorKey(role, userID)
		if _, ok = seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		options = append(options, caseworkres.ConsultationAssigneeOption{
			ID:          userID,
			DisplayName: user.NickName,
			RoleType:    role,
		})
	}
	for _, assignment := range assignments {
		appendOption(assignment.AssigneeID, assignment.RoleType)
	}
	for _, supervisor := range supervisors {
		var profile caremodel.CareAuthorityProfile
		profileErr := s.db().WithContext(ctx).Set("data_scope:skip", true).
			Where(
				"authority_id = ? AND role_type = ? AND active = ?",
				supervisor.AuthorityId,
				caremodel.AuthorityRoleSupervisor,
				true,
			).First(&profile).Error
		if errors.Is(profileErr, gorm.ErrRecordNotFound) {
			continue
		}
		if profileErr != nil {
			return nil, profileErr
		}
		appendOption(supervisor.ID, caremodel.AuthorityRoleSupervisor)
	}
	sort.Slice(options, func(i, j int) bool {
		if options[i].RoleType == options[j].RoleType {
			return options[i].ID < options[j].ID
		}
		return options[i].RoleType < options[j].RoleType
	})
	return options, nil
}

func (s *CaseWorkService) consultationSummaries(
	ctx context.Context,
	rows []caseworkmodel.Consultation,
) ([]caseworkres.ConsultationSummary, error) {
	clientIDs := make([]uint, 0, len(rows))
	assigneeIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		clientIDs = append(clientIDs, row.CareClientID)
		if row.AssigneeID != nil {
			assigneeIDs = append(assigneeIDs, *row.AssigneeID)
		}
	}
	clients := make(map[uint]caremodel.CareClient)
	if len(clientIDs) > 0 {
		var values []caremodel.CareClient
		if err := s.db().WithContext(ctx).Set("data_scope:skip", true).
			Where("id IN ?", clientIDs).Find(&values).Error; err != nil {
			return nil, err
		}
		for _, value := range values {
			clients[value.ID] = value
		}
	}
	users := make(map[uint]system.SysUser)
	if len(assigneeIDs) > 0 {
		var values []system.SysUser
		if err := s.db().WithContext(ctx).Set("data_scope:skip", true).
			Where("id IN ?", assigneeIDs).Find(&values).Error; err != nil {
			return nil, err
		}
		for _, value := range values {
			users[value.ID] = value
		}
	}
	items := make([]caseworkres.ConsultationSummary, 0, len(rows))
	for _, row := range rows {
		client := clients[row.CareClientID]
		assigneeName := ""
		if row.AssigneeID != nil {
			assigneeName = users[*row.AssigneeID].NickName
		}
		items = append(items, caseworkres.ConsultationSummary{
			ID:                row.ID,
			CareClientID:      row.CareClientID,
			ClientDisplayCode: client.DisplayCode,
			ClientDisplayName: client.DisplayName,
			Subject:           row.Subject,
			Source:            row.Source,
			Urgency:           row.Urgency,
			Status:            row.Status,
			AssigneeID:        row.AssigneeID,
			AssigneeRole:      row.AssigneeRole,
			AssigneeName:      assigneeName,
			OpenedAt:          row.OpenedAt,
			FirstRespondedAt:  row.FirstRespondedAt,
			ResolvedAt:        row.ResolvedAt,
			ClosedAt:          row.ClosedAt,
			Version:           row.Version,
		})
	}
	return items, nil
}

func (s *CaseWorkService) consultationActorNames(
	ctx context.Context,
	interactions []caseworkmodel.ConsultationInteraction,
) (map[string]string, error) {
	clientIDs := make([]uint, 0)
	staffIDs := make([]uint, 0)
	for _, interaction := range interactions {
		switch interaction.ActorType {
		case caseworkmodel.ConsultationActorClient:
			clientIDs = append(clientIDs, interaction.ActorID)
		case caseworkmodel.ConsultationActorStaff:
			staffIDs = append(staffIDs, interaction.ActorID)
		}
	}
	names := make(map[string]string)
	if len(clientIDs) > 0 {
		var clients []caremodel.CareClient
		if err := s.db().WithContext(ctx).Set("data_scope:skip", true).
			Where("id IN ?", clientIDs).Find(&clients).Error; err != nil {
			return nil, err
		}
		for _, client := range clients {
			names[interactionActorKey(caseworkmodel.ConsultationActorClient, client.ID)] = client.DisplayName
		}
	}
	if len(staffIDs) > 0 {
		var users []system.SysUser
		if err := s.db().WithContext(ctx).Set("data_scope:skip", true).
			Where("id IN ?", staffIDs).Find(&users).Error; err != nil {
			return nil, err
		}
		for _, user := range users {
			names[interactionActorKey(caseworkmodel.ConsultationActorStaff, user.ID)] = user.NickName
		}
	}
	for _, interaction := range interactions {
		if interaction.ActorType == caseworkmodel.ConsultationActorSystem {
			names[interactionActorKey(interaction.ActorType, interaction.ActorID)] = "系统"
		}
	}
	return names, nil
}

func interactionActorKey(actorType string, actorID uint) string {
	return actorType + ":" + strconv.FormatUint(uint64(actorID), 10)
}
