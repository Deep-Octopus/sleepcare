package careclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	carereq "github.com/flipped-aurora/gin-vue-admin/server/model/careclient/request"
	careres "github.com/flipped-aurora/gin-vue-admin/server/model/careclient/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CareClientService struct {
	DB  *gorm.DB
	Now func() time.Time
}

func (s *CareClientService) db() *gorm.DB {
	if s.DB != nil {
		return s.DB
	}
	return global.GVA_DB
}

func (s *CareClientService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *CareClientService) List(ctx context.Context, req carereq.CareClientSearch) ([]careres.CareClientSummary, int64, error) {
	db := s.db()
	decision, err := accesspolicy.ResolveCareClient(ctx, db)
	if err != nil {
		return nil, 0, err
	}
	query := decision.Scope(db.WithContext(ctx).Model(&caremodel.CareClient{}), s.now())
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("display_code LIKE ? OR display_name LIKE ?", like, like)
	}
	if req.OrganizationID != 0 {
		query = query.Where("organization_id = ?", req.OrganizationID)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
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
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	var clients []caremodel.CareClient
	err = query.Order("care_clients.id ASC").Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize).Find(&clients).Error
	if err != nil {
		return nil, 0, err
	}
	items, err := s.buildSummaries(ctx, decision, clients, true)
	return items, total, err
}

func (s *CareClientService) Get(ctx context.Context, id uint) (careres.CareClientDetail, error) {
	db := s.db()
	decision, err := accesspolicy.ResolveCareClient(ctx, db)
	if err != nil {
		return careres.CareClientDetail{}, err
	}
	var client caremodel.CareClient
	err = decision.Scope(db.WithContext(ctx).Model(&caremodel.CareClient{}), s.now()).Where("care_clients.id = ?", id).First(&client).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return careres.CareClientDetail{}, caremodel.NewDomainError(caremodel.CodeResourceNotFound, "康养用户不存在或不在可见范围")
	}
	if err != nil {
		return careres.CareClientDetail{}, err
	}
	summaries, err := s.buildSummaries(ctx, decision, []caremodel.CareClient{client}, false)
	if err != nil {
		return careres.CareClientDetail{}, err
	}
	detail := careres.CareClientDetail{CareClientSummary: summaries[0], ConsentStatus: caremodel.ConsentStatusPending}

	var assignments []caremodel.CareAssignment
	if err = db.WithContext(ctx).Where("care_client_id = ?", id).Order("valid_from DESC, id DESC").Find(&assignments).Error; err != nil {
		return detail, err
	}
	detail.Assignments, err = s.assignmentSummaries(ctx, assignments)
	if err != nil {
		return detail, err
	}
	var consents []caremodel.ConsentRecord
	if err = db.WithContext(ctx).Where("care_client_id = ?", id).Order("occurred_at DESC, id DESC").Find(&consents).Error; err != nil {
		return detail, err
	}
	for _, record := range consents {
		detail.ConsentRecords = append(detail.ConsentRecords, careres.ConsentRecordSummary{
			ID: record.ID, ConsentType: record.ConsentType, Action: record.Action, TextVersion: record.TextVersion,
			OccurredAt: record.OccurredAt, Source: record.Source, Reason: record.Reason, RecordedBy: record.RecordedBy,
		})
	}
	if len(consents) > 0 {
		if consents[0].Action == caremodel.ConsentActionGrant {
			detail.ConsentStatus = caremodel.ConsentStatusGranted
		} else {
			detail.ConsentStatus = caremodel.ConsentStatusWithdrawn
		}
	}
	return detail, nil
}

func (s *CareClientService) Create(ctx context.Context, key string, req carereq.CreateCareClient) (careres.ActionResult, error) {
	db := s.db()
	decision, err := accesspolicy.ResolveCareClient(ctx, db)
	if err != nil {
		return careres.ActionResult{}, err
	}
	if !decision.CanManage() {
		return careres.ActionResult{}, caremodel.NewForbiddenError(caremodel.CodeAccessScopeDenied, "仅督导角色可新建康养用户")
	}
	if err = validateCreate(req); err != nil {
		return careres.ActionResult{}, err
	}
	if !req.Synthetic {
		return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeOperationNotAllowed, "P1-02 仅允许合成数据")
	}
	deptID, err := s.validateOrgTeam(ctx, decision, req.OrganizationID, req.TeamID)
	if err != nil {
		return careres.ActionResult{}, err
	}
	commandCtx := withDepartment(ctx, deptID)
	return s.runIdempotent(commandCtx, "CREATE_CLIENT", key, req, func(tx *gorm.DB) (careres.ActionResult, error) {
		client := caremodel.CareClient{
			DisplayCode: strings.TrimSpace(req.DisplayCode), DisplayName: strings.TrimSpace(req.DisplayName),
			ContactMobile: strings.TrimSpace(req.ContactMobile), ServiceReason: strings.TrimSpace(req.ServiceReason),
			ServicePackageCode: strings.TrimSpace(req.ServicePackageCode), OrganizationID: req.OrganizationID,
			TeamID: req.TeamID, Status: caremodel.ClientStatusActive, SensitivityLevel: caremodel.SensitivitySensitive,
			Synthetic: true, Version: 1,
		}
		if err := tx.Create(&client).Error; err != nil {
			return careres.ActionResult{}, mapDuplicate(err, "康养用户编码已存在")
		}
		return careres.ActionResult{CareClientID: client.ID, ResourceID: client.ID, Version: client.Version}, nil
	})
}

func (s *CareClientService) Update(ctx context.Context, id uint, key string, req carereq.UpdateCareClient) (careres.ActionResult, error) {
	decision, client, err := s.manageableClient(ctx, id)
	if err != nil {
		return careres.ActionResult{}, err
	}
	if req.ExpectedVersion == 0 || strings.TrimSpace(req.DisplayName) == "" || !validClientStatus(req.Status) {
		return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeInvalidArgument, "expectedVersion、显示名称和有效状态必填")
	}
	if !strings.Contains(req.DisplayName, "合成") {
		return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeInvalidArgument, "显示名称必须醒目标注为合成数据")
	}
	if !sameOptionalID(client.TeamID, req.TeamID) {
		var openAssignmentCount int64
		now := s.now()
		if err := s.db().WithContext(ctx).Model(&caremodel.CareAssignment{}).
			Where("care_client_id = ? AND cancelled_at IS NULL AND (valid_until IS NULL OR valid_until > ?)", id, now).
			Count(&openAssignmentCount).Error; err != nil {
			return careres.ActionResult{}, err
		}
		if openAssignmentCount > 0 {
			return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeOperationNotAllowed, "存在生效或已排期责任关系时不能直接变更团队")
		}
	}
	deptID, err := s.validateOrgTeam(ctx, decision, client.OrganizationID, req.TeamID)
	if err != nil {
		return careres.ActionResult{}, err
	}
	commandCtx := withDepartment(ctx, deptID)
	operation := fmt.Sprintf("UPDATE_CLIENT:%d", id)
	return s.runIdempotent(commandCtx, operation, key, req, func(tx *gorm.DB) (careres.ActionResult, error) {
		updates := map[string]any{
			"display_name": strings.TrimSpace(req.DisplayName), "contact_mobile": strings.TrimSpace(req.ContactMobile),
			"service_reason": strings.TrimSpace(req.ServiceReason), "service_package_code": strings.TrimSpace(req.ServicePackageCode),
			"team_id": req.TeamID, "dept_id": deptID, "status": req.Status, "version": gorm.Expr("version + 1"),
		}
		result := tx.Model(&caremodel.CareClient{}).Where("id = ? AND version = ?", id, req.ExpectedVersion).Updates(updates)
		if result.Error != nil {
			return careres.ActionResult{}, result.Error
		}
		if result.RowsAffected != 1 {
			return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeVersionConflict, "康养用户已被其他操作更新")
		}
		return careres.ActionResult{CareClientID: id, ResourceID: id, Version: req.ExpectedVersion + 1}, nil
	})
}

func (s *CareClientService) CreateAssignment(ctx context.Context, id uint, key string, req carereq.CreateAssignment) (careres.ActionResult, error) {
	decision, client, err := s.manageableClient(ctx, id)
	if err != nil {
		return careres.ActionResult{}, err
	}
	if req.ExpectedVersion == 0 || req.AssigneeID == 0 || req.TeamID == 0 || req.ValidFrom.IsZero() || strings.TrimSpace(req.Reason) == "" || !validAssignmentRole(req.RoleType) {
		return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeInvalidArgument, "责任角色、责任人、团队、生效时间、原因和 expectedVersion 必填")
	}
	if req.ValidUntil != nil && !req.ValidUntil.After(req.ValidFrom) {
		return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeInvalidArgument, "责任结束时间必须晚于生效时间")
	}
	if client.TeamID == nil || *client.TeamID == 0 || req.TeamID != *client.TeamID {
		return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeOperationNotAllowed, "P1-02 责任人必须属于康养用户当前团队")
	}
	teamID := req.TeamID
	if _, err = s.validateOrgTeam(ctx, decision, client.OrganizationID, &teamID); err != nil {
		return careres.ActionResult{}, err
	}
	if err = s.validateAssignee(ctx, req.AssigneeID, req.TeamID, req.RoleType); err != nil {
		return careres.ActionResult{}, err
	}
	operation := fmt.Sprintf("CREATE_ASSIGNMENT:%d", id)
	commandCtx := withDepartment(ctx, client.DeptId)
	return s.runIdempotent(commandCtx, operation, key, req, func(tx *gorm.DB) (careres.ActionResult, error) {
		if err := lockClient(tx, id, req.ExpectedVersion); err != nil {
			return careres.ActionResult{}, err
		}
		now := s.now()
		var current caremodel.CareAssignment
		overlap := lockQuery(tx).Where("care_client_id = ? AND role_type = ? AND cancelled_at IS NULL", id, req.RoleType).
			Where("valid_until IS NULL OR valid_until > ?", req.ValidFrom)
		if req.ValidUntil != nil {
			overlap = overlap.Where("valid_from < ?", *req.ValidUntil)
		}
		currentErr := overlap.Order("valid_from ASC, id ASC").First(&current).Error
		if currentErr == nil {
			if req.ReplacesAssignmentID == nil || *req.ReplacesAssignmentID != current.ID {
				return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeOperationNotAllowed, "已有生效责任关系，必须明确 replacesAssignmentId")
			}
			if current.ValidFrom.After(req.ValidFrom) {
				return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeOperationNotAllowed, "新责任关系不能早于被替代的已排期关系")
			}
			if err := tx.Model(&current).Updates(map[string]any{"valid_until": req.ValidFrom, "ended_at": now, "end_reason": "由新责任关系替代"}).Error; err != nil {
				return careres.ActionResult{}, err
			}
		} else if !errors.Is(currentErr, gorm.ErrRecordNotFound) {
			return careres.ActionResult{}, currentErr
		} else if req.ReplacesAssignmentID != nil {
			return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeOperationNotAllowed, "被替代责任关系不存在或已失效")
		}
		assignment := caremodel.CareAssignment{
			CareClientID: id, OrganizationID: client.OrganizationID, TeamID: req.TeamID, AssigneeID: req.AssigneeID,
			RoleType: req.RoleType, ValidFrom: req.ValidFrom, ValidUntil: req.ValidUntil,
			ReplacesAssignmentID: req.ReplacesAssignmentID, Reason: strings.TrimSpace(req.Reason), Synthetic: true,
		}
		if err := tx.Create(&assignment).Error; err != nil {
			return careres.ActionResult{}, err
		}
		if err := bumpVersion(tx, id, req.ExpectedVersion); err != nil {
			return careres.ActionResult{}, err
		}
		return careres.ActionResult{CareClientID: id, ResourceID: assignment.ID, Version: req.ExpectedVersion + 1}, nil
	})
}

func (s *CareClientService) CreateConsent(ctx context.Context, id uint, key string, req carereq.CreateConsentRecord) (careres.ActionResult, error) {
	_, client, err := s.manageableClient(ctx, id)
	if err != nil {
		return careres.ActionResult{}, err
	}
	if !client.Synthetic || req.ExpectedVersion == 0 || req.ConsentType != caremodel.ConsentTypeSyntheticTestParticipation ||
		(req.Action != caremodel.ConsentActionGrant && req.Action != caremodel.ConsentActionWithdraw) || strings.TrimSpace(req.TextVersion) == "" || req.OccurredAt.IsZero() ||
		req.Source != caremodel.ConsentSourceStaffRecorded {
		return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeInvalidArgument, "仅允许为合成测试记录有效的授权或撤回事实")
	}
	operation := fmt.Sprintf("CREATE_CONSENT:%d", id)
	commandCtx := withDepartment(ctx, client.DeptId)
	return s.runIdempotent(commandCtx, operation, key, req, func(tx *gorm.DB) (careres.ActionResult, error) {
		if err := lockClient(tx, id, req.ExpectedVersion); err != nil {
			return careres.ActionResult{}, err
		}
		var latest caremodel.ConsentRecord
		err := lockQuery(tx).Where("care_client_id = ? AND consent_type = ?", id, req.ConsentType).Order("occurred_at DESC, id DESC").First(&latest).Error
		if req.Action == caremodel.ConsentActionWithdraw && (errors.Is(err, gorm.ErrRecordNotFound) || latest.Action != caremodel.ConsentActionGrant) {
			return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeOperationNotAllowed, "当前没有可撤回的有效授权")
		}
		if req.Action == caremodel.ConsentActionGrant && err == nil && latest.Action == caremodel.ConsentActionGrant {
			return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeOperationNotAllowed, "授权已生效，无需重复记录")
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return careres.ActionResult{}, err
		}
		actorID := actorID(commandCtx)
		record := caremodel.ConsentRecord{
			CareClientID: id, ConsentType: req.ConsentType, Action: req.Action, TextVersion: strings.TrimSpace(req.TextVersion),
			OccurredAt: req.OccurredAt, Source: req.Source, Reason: strings.TrimSpace(req.Reason), RecordedBy: actorID, Synthetic: true,
		}
		if err := tx.Create(&record).Error; err != nil {
			return careres.ActionResult{}, err
		}
		if err := bumpVersion(tx, id, req.ExpectedVersion); err != nil {
			return careres.ActionResult{}, err
		}
		return careres.ActionResult{CareClientID: id, ResourceID: record.ID, Version: req.ExpectedVersion + 1}, nil
	})
}

func (s *CareClientService) Options(ctx context.Context) (careres.ClientOptions, error) {
	db := s.db()
	decision, err := accesspolicy.ResolveCareClient(ctx, db)
	if err != nil {
		return careres.ClientOptions{}, err
	}
	if !decision.CanManage() {
		return careres.ClientOptions{}, caremodel.NewForbiddenError(caremodel.CodeAccessScopeDenied, "仅督导角色可读取维护选项")
	}
	var profiles []caremodel.CareOrgUnitProfile
	if err = db.WithContext(ctx).Where("active = ?", true).Order("unit_type, code").Find(&profiles).Error; err != nil {
		return careres.ClientOptions{}, err
	}
	deptIDs := make([]uint, 0, len(profiles))
	for _, profile := range profiles {
		deptIDs = append(deptIDs, profile.DepartmentID)
	}
	var departments []system.SysDepartment
	db.WithContext(ctx).Set("data_scope:skip", true).Where("id IN ?", nonEmptyIDs(deptIDs)).Find(&departments)
	deptNames := make(map[uint]string, len(departments))
	for _, dept := range departments {
		deptNames[dept.ID] = dept.Name
	}
	options := careres.ClientOptions{}
	for _, profile := range profiles {
		options.OrgUnits = append(options.OrgUnits, careres.OrgUnitOption{
			DepartmentID: profile.DepartmentID, OrganizationID: profile.OrganizationID, Code: profile.Code,
			Name: deptNames[profile.DepartmentID], UnitType: profile.UnitType,
		})
	}
	var users []system.SysUser
	if err = db.WithContext(ctx).Set("data_scope:skip", true).
		Where("dept_id IN ? AND enable = ?", nonEmptyIDs(deptIDs), 1).Order("id").Find(&users).Error; err != nil {
		return options, err
	}
	for _, user := range users {
		var roleProfile caremodel.CareAuthorityProfile
		if err := db.WithContext(ctx).Set("data_scope:skip", true).Where("authority_id = ? AND active = ?", user.AuthorityId, true).First(&roleProfile).Error; err != nil {
			continue
		}
		if roleProfile.RoleType != caremodel.AuthorityRoleCareSteward && roleProfile.RoleType != caremodel.AuthorityRoleClinician {
			continue
		}
		options.Assignees = append(options.Assignees, careres.AssigneeOption{ID: user.ID, DisplayName: user.NickName, RoleType: roleProfile.RoleType, TeamID: user.DeptId})
	}
	return options, nil
}

func (s *CareClientService) manageableClient(ctx context.Context, id uint) (*accesspolicy.CareClientDecision, caremodel.CareClient, error) {
	db := s.db()
	decision, err := accesspolicy.ResolveCareClient(ctx, db)
	if err != nil {
		return nil, caremodel.CareClient{}, err
	}
	if !decision.CanManage() {
		return nil, caremodel.CareClient{}, caremodel.NewForbiddenError(caremodel.CodeAccessScopeDenied, "仅督导角色可维护康养用户")
	}
	var client caremodel.CareClient
	err = decision.Scope(db.WithContext(ctx).Model(&caremodel.CareClient{}), s.now()).Where("care_clients.id = ?", id).First(&client).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, client, caremodel.NewDomainError(caremodel.CodeResourceNotFound, "康养用户不存在或不在可见范围")
	}
	if err == nil && !client.Synthetic {
		return nil, client, caremodel.NewDomainError(caremodel.CodeOperationNotAllowed, "P1-02 仅允许维护合成数据")
	}
	return decision, client, err
}

func (s *CareClientService) validateOrgTeam(ctx context.Context, decision *accesspolicy.CareClientDecision, organizationID uint, teamID *uint) (uint, error) {
	if organizationID == 0 || !decision.CanAccessDepartment(organizationID) {
		return 0, caremodel.NewForbiddenError(caremodel.CodeAccessScopeDenied, "机构不在可管理范围")
	}
	var organization caremodel.CareOrgUnitProfile
	if err := s.db().WithContext(ctx).Set("data_scope:skip", true).Where("department_id = ? AND organization_id = ? AND unit_type = ? AND active = ?", organizationID, organizationID, caremodel.OrgUnitTypeOrganization, true).First(&organization).Error; err != nil {
		return 0, caremodel.NewDomainError(caremodel.CodeInvalidArgument, "机构无效")
	}
	if teamID == nil || *teamID == 0 {
		return organizationID, nil
	}
	if !decision.CanAccessDepartment(*teamID) {
		return 0, caremodel.NewForbiddenError(caremodel.CodeAccessScopeDenied, "团队不在可管理范围")
	}
	var team caremodel.CareOrgUnitProfile
	if err := s.db().WithContext(ctx).Set("data_scope:skip", true).Where("department_id = ? AND organization_id = ? AND unit_type = ? AND active = ?", *teamID, organizationID, caremodel.OrgUnitTypeTeam, true).First(&team).Error; err != nil {
		return 0, caremodel.NewDomainError(caremodel.CodeInvalidArgument, "团队无效或不属于所选机构")
	}
	return *teamID, nil
}

func (s *CareClientService) validateAssignee(ctx context.Context, userID, teamID uint, assignmentRole string) error {
	var user system.SysUser
	if err := s.db().WithContext(ctx).Set("data_scope:skip", true).Where("id = ? AND dept_id = ? AND enable = ?", userID, teamID, 1).First(&user).Error; err != nil {
		return caremodel.NewDomainError(caremodel.CodeInvalidArgument, "责任人无效或不属于所选团队")
	}
	var profile caremodel.CareAuthorityProfile
	if err := s.db().WithContext(ctx).Set("data_scope:skip", true).Where("authority_id = ? AND active = ?", user.AuthorityId, true).First(&profile).Error; err != nil {
		return caremodel.NewDomainError(caremodel.CodeInvalidArgument, "责任人没有有效的康养业务角色")
	}
	expectedRole := caremodel.AuthorityRoleCareSteward
	if assignmentRole == caremodel.AssignmentRoleClinician {
		expectedRole = caremodel.AuthorityRoleClinician
	}
	if profile.RoleType != expectedRole {
		return caremodel.NewDomainError(caremodel.CodeInvalidArgument, "责任人角色与责任类型不匹配")
	}
	return nil
}

func (s *CareClientService) runIdempotent(ctx context.Context, operation, key string, request any, fn func(*gorm.DB) (careres.ActionResult, error)) (careres.ActionResult, error) {
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 128 {
		return careres.ActionResult{}, caremodel.NewDomainError(caremodel.CodeInvalidArgument, "Idempotency-Key 必填且不超过 128 字符")
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return careres.ActionResult{}, err
	}
	sum := sha256.Sum256(payload)
	hash := hex.EncodeToString(sum[:])
	actor := actorID(ctx)
	if actor == 0 {
		return careres.ActionResult{}, caremodel.NewForbiddenError(caremodel.CodeAccessScopeDenied, "缺少有效操作人")
	}
	var result careres.ActionResult
	err = s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var receipt caremodel.CareClientCommandReceipt
		err := tx.Where("actor_id = ? AND operation = ? AND idempotency_key = ?", actor, operation, key).First(&receipt).Error
		if err == nil {
			if receipt.RequestHash != hash {
				return caremodel.NewDomainError(caremodel.CodeIdempotencyConflict, "相同 Idempotency-Key 对应了不同请求")
			}
			return json.Unmarshal([]byte(receipt.ResultJSON), &result)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		result, err = fn(tx)
		if err != nil {
			return err
		}
		encoded, err := json.Marshal(result)
		if err != nil {
			return err
		}
		receipt = caremodel.CareClientCommandReceipt{Operation: operation, ActorID: actor, IdempotencyKey: key, RequestHash: hash, ResultJSON: string(encoded)}
		return tx.Create(&receipt).Error
	})
	return result, err
}

func (s *CareClientService) buildSummaries(ctx context.Context, decision *accesspolicy.CareClientDecision, clients []caremodel.CareClient, currentOnly bool) ([]careres.CareClientSummary, error) {
	if len(clients) == 0 {
		return []careres.CareClientSummary{}, nil
	}
	ids := make([]uint, 0, len(clients))
	deptIDs := make([]uint, 0, len(clients)*2)
	for _, client := range clients {
		ids = append(ids, client.ID)
		deptIDs = append(deptIDs, client.OrganizationID)
		if client.TeamID != nil {
			deptIDs = append(deptIDs, *client.TeamID)
		}
	}
	var departments []system.SysDepartment
	s.db().WithContext(ctx).Set("data_scope:skip", true).Where("id IN ?", deptIDs).Find(&departments)
	deptNames := map[uint]string{}
	for _, dept := range departments {
		deptNames[dept.ID] = dept.Name
	}
	query := s.db().WithContext(ctx).Where("care_client_id IN ?", ids)
	if currentOnly {
		now := s.now()
		query = query.Where("cancelled_at IS NULL AND valid_from <= ? AND (valid_until IS NULL OR valid_until > ?)", now, now)
	}
	var assignments []caremodel.CareAssignment
	if err := query.Order("care_client_id, role_type, valid_from DESC").Find(&assignments).Error; err != nil {
		return nil, err
	}
	assignmentDTOs, err := s.assignmentSummaries(ctx, assignments)
	if err != nil {
		return nil, err
	}
	byClient := map[uint][]careres.CareAssignmentSummary{}
	for i, assignment := range assignments {
		byClient[assignment.CareClientID] = append(byClient[assignment.CareClientID], assignmentDTOs[i])
	}
	items := make([]careres.CareClientSummary, 0, len(clients))
	for _, client := range clients {
		mobile := client.ContactMobile
		if decision.DataLevel == caremodel.DataLevelBasic {
			mobile = maskMobile(mobile)
		}
		item := careres.CareClientSummary{
			ID: client.ID, DisplayCode: client.DisplayCode, DisplayName: client.DisplayName, ContactMobile: mobile,
			ServiceReason: client.ServiceReason, ServicePackageCode: client.ServicePackageCode,
			OrganizationID: client.OrganizationID, OrganizationName: deptNames[client.OrganizationID], TeamID: client.TeamID,
			Status: client.Status, SensitivityLevel: client.SensitivityLevel, Synthetic: client.Synthetic, Version: client.Version,
			CurrentAssignments: byClient[client.ID],
		}
		if client.TeamID != nil {
			item.TeamName = deptNames[*client.TeamID]
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *CareClientService) assignmentSummaries(ctx context.Context, assignments []caremodel.CareAssignment) ([]careres.CareAssignmentSummary, error) {
	if len(assignments) == 0 {
		return []careres.CareAssignmentSummary{}, nil
	}
	userIDs, teamIDs := make([]uint, 0, len(assignments)), make([]uint, 0, len(assignments))
	for _, assignment := range assignments {
		userIDs = append(userIDs, assignment.AssigneeID)
		teamIDs = append(teamIDs, assignment.TeamID)
	}
	var users []system.SysUser
	if err := s.db().WithContext(ctx).Set("data_scope:skip", true).Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	var teams []system.SysDepartment
	if err := s.db().WithContext(ctx).Set("data_scope:skip", true).Where("id IN ?", teamIDs).Find(&teams).Error; err != nil {
		return nil, err
	}
	userNames, teamNames := map[uint]string{}, map[uint]string{}
	for _, user := range users {
		userNames[user.ID] = user.NickName
	}
	for _, team := range teams {
		teamNames[team.ID] = team.Name
	}
	result := make([]careres.CareAssignmentSummary, 0, len(assignments))
	for _, assignment := range assignments {
		result = append(result, careres.CareAssignmentSummary{
			ID: assignment.ID, RoleType: assignment.RoleType, AssigneeID: assignment.AssigneeID,
			AssigneeDisplayName: userNames[assignment.AssigneeID], TeamID: assignment.TeamID, TeamName: teamNames[assignment.TeamID],
			Status: assignment.EffectiveStatus(s.now()), ValidFrom: assignment.ValidFrom, ValidUntil: assignment.ValidUntil,
			ReplacesAssignmentID: assignment.ReplacesAssignmentID, Reason: assignment.Reason, EndReason: assignment.EndReason,
		})
	}
	return result, nil
}

func validateCreate(req carereq.CreateCareClient) error {
	if strings.TrimSpace(req.DisplayCode) == "" || strings.TrimSpace(req.DisplayName) == "" || req.OrganizationID == 0 {
		return caremodel.NewDomainError(caremodel.CodeInvalidArgument, "显示编码、显示名称和机构必填")
	}
	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(req.DisplayCode)), "SYN-") || !strings.Contains(req.DisplayName, "合成") {
		return caremodel.NewDomainError(caremodel.CodeInvalidArgument, "显示编码和名称必须醒目标注为合成数据")
	}
	return nil
}

func validClientStatus(status string) bool {
	return status == caremodel.ClientStatusActive || status == caremodel.ClientStatusInactive
}

func validAssignmentRole(role string) bool {
	return role == caremodel.AssignmentRoleCareSteward || role == caremodel.AssignmentRoleClinician
}

func withDepartment(ctx context.Context, departmentID uint) context.Context {
	id, ok := datascope.FromContext(ctx)
	if !ok || id == nil {
		return ctx
	}
	copyID := *id
	copyID.DeptID = departmentID
	return datascope.WithIdentity(ctx, &copyID)
}

func actorID(ctx context.Context) uint {
	if id, ok := datascope.FromContext(ctx); ok && id != nil {
		return id.UserID
	}
	return 0
}

func lockQuery(db *gorm.DB) *gorm.DB {
	name := db.Dialector.Name()
	if name == "mysql" || name == "postgres" {
		return db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return db
}

func lockClient(db *gorm.DB, id, version uint) error {
	var client caremodel.CareClient
	err := lockQuery(db).Where("id = ?", id).First(&client).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return caremodel.NewDomainError(caremodel.CodeResourceNotFound, "康养用户不存在")
	}
	if err != nil {
		return err
	}
	if client.Version != version {
		return caremodel.NewDomainError(caremodel.CodeVersionConflict, "康养用户已被其他操作更新")
	}
	return nil
}

func bumpVersion(db *gorm.DB, id, expected uint) error {
	result := db.Model(&caremodel.CareClient{}).Where("id = ? AND version = ?", id, expected).Update("version", gorm.Expr("version + 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return caremodel.NewDomainError(caremodel.CodeVersionConflict, "康养用户已被其他操作更新")
	}
	return nil
}

func mapDuplicate(err error, message string) error {
	text := strings.ToLower(err.Error())
	if strings.Contains(text, "duplicate") || strings.Contains(text, "unique") {
		return caremodel.NewDomainError(caremodel.CodeOperationNotAllowed, message)
	}
	return err
}

func maskMobile(value string) string {
	runes := []rune(value)
	if len(runes) <= 4 {
		return "****"
	}
	return "****" + string(runes[len(runes)-4:])
}

func nonEmptyIDs(ids []uint) []uint {
	if len(ids) == 0 {
		return []uint{0}
	}
	return ids
}

func sameOptionalID(left, right *uint) bool {
	if left == nil || *left == 0 {
		return right == nil || *right == 0
	}
	return right != nil && *left == *right
}
