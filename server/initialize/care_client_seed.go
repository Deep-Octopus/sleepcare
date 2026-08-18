package initialize

import (
	"context"
	"errors"
	"fmt"
	"time"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	syntheticOrgAID         = 9001
	syntheticOrgBID         = 9002
	syntheticTeamAID        = 9101
	syntheticTeamBID        = 9102
	syntheticStewardAID     = 9201
	syntheticClinicianAID   = 9202
	syntheticSupervisorAID  = 9203
	syntheticStewardA2ID    = 9204
	syntheticStewardBID     = 9299
	syntheticStewardRole    = 9801
	syntheticClinicianRole  = 9802
	syntheticSupervisorRole = 9803
)

var careClientAPIs = []system.SysApi{
	{ApiGroup: "康养用户", Method: "GET", Path: "/care/clients", Description: "获取康养用户列表"},
	{ApiGroup: "康养用户", Method: "GET", Path: "/care/clients/:id", Description: "获取康养用户详情"},
	{ApiGroup: "康养用户", Method: "GET", Path: "/care/client-options", Description: "获取康养用户维护选项"},
	{ApiGroup: "康养用户", Method: "POST", Path: "/care/clients", Description: "新建合成康养用户"},
	{ApiGroup: "康养用户", Method: "PUT", Path: "/care/clients/:id", Description: "更新合成康养用户"},
	{ApiGroup: "康养用户", Method: "POST", Path: "/care/clients/:id/assignments", Description: "记录康养用户责任关系"},
	{ApiGroup: "康养用户", Method: "POST", Path: "/care/clients/:id/consent-records", Description: "记录合成测试授权事实"},
}

// EnsureCareClientData keeps metadata available for already initialized GVA
// databases. Synthetic people and clients are opt-in and use a local secret.
func EnsureCareClientData() error {
	if global.GVA_DB == nil {
		return nil
	}
	ctx := datascope.WithSystem(context.Background())
	if err := ensureCareClientMetadata(global.GVA_DB.WithContext(ctx)); err != nil {
		return err
	}
	if !global.GVA_CONFIG.Care.SyntheticFixturesEnabled {
		return nil
	}
	if global.GVA_CONFIG.Care.FixturePassword == "" {
		return errors.New("care synthetic fixtures enabled but fixture password is empty")
	}
	return ensureCareClientSyntheticFixtures(global.GVA_DB.WithContext(ctx), global.GVA_CONFIG.Care.FixturePassword)
}

func ensureCareClientMetadata(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, api := range careClientAPIs {
			if err := tx.Where("path = ? AND method = ?", api.Path, api.Method).Attrs(api).FirstOrCreate(&system.SysApi{}).Error; err != nil {
				return fmt.Errorf("ensure care API %s: %w", api.Path, err)
			}
		}
		root := system.SysBaseMenu{Path: "sleep-care", Name: "SleepCare", Component: "view/routerHolder.vue", Sort: 20, Meta: system.Meta{Title: "睡眠康养随访", Icon: "user"}}
		if err := tx.Where("name = ?", root.Name).Attrs(root).FirstOrCreate(&root).Error; err != nil {
			return err
		}
		leaf := system.SysBaseMenu{ParentId: root.ID, Path: "care-clients", Name: "CareClients", Component: "view/sleep-care/clients/index.vue", Sort: 1, Meta: system.Meta{Title: "康养用户", Icon: "user"}}
		if err := tx.Where("name = ?", leaf.Name).Attrs(leaf).FirstOrCreate(&leaf).Error; err != nil {
			return err
		}
		buttons := []system.SysBaseMenuBtn{
			{Name: "viewDetail", Desc: "查看详情", SysBaseMenuID: leaf.ID},
			{Name: "createClient", Desc: "新建合成康养用户", SysBaseMenuID: leaf.ID},
			{Name: "maintainClient", Desc: "维护公开资料", SysBaseMenuID: leaf.ID},
			{Name: "assignCare", Desc: "记录责任关系", SysBaseMenuID: leaf.ID},
			{Name: "recordConsent", Desc: "记录合成测试授权", SysBaseMenuID: leaf.ID},
		}
		for i := range buttons {
			if err := tx.Where("name = ? AND sys_base_menu_id = ?", buttons[i].Name, leaf.ID).Attrs(buttons[i]).FirstOrCreate(&buttons[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ensureCareClientSyntheticFixtures(db *gorm.DB, password string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := seedCareDepartments(tx); err != nil {
			return err
		}
		if err := seedCareAuthorities(tx); err != nil {
			return err
		}
		if err := seedCareUsers(tx, password); err != nil {
			return err
		}
		root, leaf, buttons, err := loadCareMenu(tx)
		if err != nil {
			return err
		}
		if err = grantCareAccess(tx, root, leaf, buttons); err != nil {
			return err
		}
		return seedCareClients(tx)
	})
}

func seedCareDepartments(tx *gorm.DB) error {
	active := true
	departments := []system.SysDepartment{
		{GVA_MODEL: global.GVA_MODEL{ID: syntheticOrgAID}, Name: "[合成] 睡眠康养机构A", ParentId: 1, Ancestors: "0,1", Sort: 90, Status: &active},
		{GVA_MODEL: global.GVA_MODEL{ID: syntheticOrgBID}, Name: "[合成] 睡眠康养机构B", ParentId: 1, Ancestors: "0,1", Sort: 91, Status: &active},
		{GVA_MODEL: global.GVA_MODEL{ID: syntheticTeamAID}, Name: "[合成] 随访团队A", ParentId: syntheticOrgAID, Ancestors: "0,1,9001", Sort: 1, Status: &active},
		{GVA_MODEL: global.GVA_MODEL{ID: syntheticTeamBID}, Name: "[合成] 随访团队B", ParentId: syntheticOrgBID, Ancestors: "0,1,9002", Sort: 1, Status: &active},
	}
	for i := range departments {
		var existing system.SysDepartment
		err := tx.Unscoped().Where("id = ?", departments[i].ID).First(&existing).Error
		if err == nil {
			if existing.Name != departments[i].Name || existing.DeletedAt.Valid {
				return fmt.Errorf("synthetic department id %d is occupied", departments[i].ID)
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err = tx.Create(&departments[i]).Error; err != nil {
			return err
		}
	}
	profiles := []caremodel.CareOrgUnitProfile{
		{DepartmentID: syntheticOrgAID, OrganizationID: syntheticOrgAID, Code: "SYN-ORG-A", UnitType: caremodel.OrgUnitTypeOrganization, Synthetic: true, Active: true, DeptId: syntheticOrgAID},
		{DepartmentID: syntheticOrgBID, OrganizationID: syntheticOrgBID, Code: "SYN-ORG-B", UnitType: caremodel.OrgUnitTypeOrganization, Synthetic: true, Active: true, DeptId: syntheticOrgBID},
		{DepartmentID: syntheticTeamAID, OrganizationID: syntheticOrgAID, Code: "SYN-TEAM-A", UnitType: caremodel.OrgUnitTypeTeam, Synthetic: true, Active: true, DeptId: syntheticTeamAID},
		{DepartmentID: syntheticTeamBID, OrganizationID: syntheticOrgBID, Code: "SYN-TEAM-B", UnitType: caremodel.OrgUnitTypeTeam, Synthetic: true, Active: true, DeptId: syntheticTeamBID},
	}
	for i := range profiles {
		if err := tx.Where("department_id = ?", profiles[i].DepartmentID).Attrs(profiles[i]).FirstOrCreate(&profiles[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedCareAuthorities(tx *gorm.DB) error {
	parentID := uint(888)
	authorities := []system.SysAuthority{
		{AuthorityId: syntheticStewardRole, AuthorityName: "[合成] 健康管家", ParentId: &parentID, DataScope: datascope.ScopeDept, DefaultRouter: "CareClients"},
		{AuthorityId: syntheticClinicianRole, AuthorityName: "[合成] 一线医护", ParentId: &parentID, DataScope: datascope.ScopeDept, DefaultRouter: "CareClients"},
		{AuthorityId: syntheticSupervisorRole, AuthorityName: "[合成] 上级医师", ParentId: &parentID, DataScope: datascope.ScopeDeptAndChild, DefaultRouter: "CareClients"},
	}
	profiles := []caremodel.CareAuthorityProfile{
		{AuthorityID: syntheticStewardRole, RoleType: caremodel.AuthorityRoleCareSteward, Synthetic: true, Active: true},
		{AuthorityID: syntheticClinicianRole, RoleType: caremodel.AuthorityRoleClinician, Synthetic: true, Active: true},
		{AuthorityID: syntheticSupervisorRole, RoleType: caremodel.AuthorityRoleSupervisor, Synthetic: true, Active: true},
	}
	for i := range authorities {
		var existing system.SysAuthority
		err := tx.Unscoped().Where("authority_id = ?", authorities[i].AuthorityId).First(&existing).Error
		if err == nil {
			if existing.AuthorityName != authorities[i].AuthorityName || existing.DeletedAt != nil {
				return fmt.Errorf("synthetic authority id %d is occupied", authorities[i].AuthorityId)
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			if err = tx.Create(&authorities[i]).Error; err != nil {
				return err
			}
		} else {
			return err
		}
		if err := tx.Where("authority_id = ?", profiles[i].AuthorityID).Attrs(profiles[i]).FirstOrCreate(&profiles[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedCareUsers(tx *gorm.DB, password string) error {
	users := []system.SysUser{
		{GVA_MODEL: global.GVA_MODEL{ID: syntheticStewardAID}, UUID: uuid.New(), Username: "syn_steward_a", NickName: "[合成] 管家甲", AuthorityId: syntheticStewardRole, DeptId: syntheticTeamAID, Enable: 1},
		{GVA_MODEL: global.GVA_MODEL{ID: syntheticClinicianAID}, UUID: uuid.New(), Username: "syn_clinician_a", NickName: "[合成] 医护甲", AuthorityId: syntheticClinicianRole, DeptId: syntheticTeamAID, Enable: 1},
		{GVA_MODEL: global.GVA_MODEL{ID: syntheticSupervisorAID}, UUID: uuid.New(), Username: "syn_supervisor_a", NickName: "[合成] 上级医师甲", AuthorityId: syntheticSupervisorRole, DeptId: syntheticOrgAID, Enable: 1},
		{GVA_MODEL: global.GVA_MODEL{ID: syntheticStewardA2ID}, UUID: uuid.New(), Username: "syn_steward_a2", NickName: "[合成] 管家乙", AuthorityId: syntheticStewardRole, DeptId: syntheticTeamAID, Enable: 1},
		{GVA_MODEL: global.GVA_MODEL{ID: syntheticStewardBID}, UUID: uuid.New(), Username: "syn_steward_b", NickName: "[合成] 跨机构管家", AuthorityId: syntheticStewardRole, DeptId: syntheticTeamBID, Enable: 1},
	}
	passwordHash := utils.BcryptHash(password)
	for i := range users {
		var existing system.SysUser
		err := tx.Unscoped().Where("id = ?", users[i].ID).First(&existing).Error
		if err == nil {
			if existing.Username != users[i].Username || existing.DeletedAt.Valid {
				return fmt.Errorf("synthetic user id %d is occupied", users[i].ID)
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			users[i].Password = passwordHash
			if err = tx.Create(&users[i]).Error; err != nil {
				return err
			}
		} else {
			return err
		}
		joins := []any{
			&system.SysUserAuthority{SysUserId: users[i].ID, SysAuthorityAuthorityId: users[i].AuthorityId},
			&system.SysUserDepartment{SysUserId: users[i].ID, SysDepartmentId: users[i].DeptId},
		}
		for _, join := range joins {
			if err := tx.Where(join).FirstOrCreate(join).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func loadCareMenu(tx *gorm.DB) (system.SysBaseMenu, system.SysBaseMenu, []system.SysBaseMenuBtn, error) {
	var root, leaf system.SysBaseMenu
	if err := tx.Where("name = ?", "SleepCare").First(&root).Error; err != nil {
		return root, leaf, nil, err
	}
	if err := tx.Where("name = ?", "CareClients").First(&leaf).Error; err != nil {
		return root, leaf, nil, err
	}
	var buttons []system.SysBaseMenuBtn
	if err := tx.Where("sys_base_menu_id = ?", leaf.ID).Find(&buttons).Error; err != nil {
		return root, leaf, nil, err
	}
	return root, leaf, buttons, nil
}

func grantCareAccess(tx *gorm.DB, root, leaf system.SysBaseMenu, buttons []system.SysBaseMenuBtn) error {
	roles := []uint{syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole}
	for _, role := range roles {
		for _, menuID := range []uint{root.ID, leaf.ID} {
			link := system.SysAuthorityMenu{MenuId: fmt.Sprint(menuID), AuthorityId: fmt.Sprint(role)}
			if err := tx.Where(link).FirstOrCreate(&link).Error; err != nil {
				return err
			}
		}
	}
	for _, button := range buttons {
		var allowedRoles []uint
		switch button.Name {
		case "viewDetail":
			allowedRoles = roles
		case "startPlan", "pausePlan", "resumePlan":
			allowedRoles = []uint{syntheticStewardRole, syntheticClinicianRole}
		case "createClient", "maintainClient", "assignCare", "recordConsent":
			allowedRoles = []uint{syntheticSupervisorRole}
		default:
			continue
		}
		for _, role := range allowedRoles {
			link := system.SysAuthorityBtn{AuthorityId: role, SysMenuID: leaf.ID, SysBaseMenuBtnID: button.ID}
			if err := tx.Where("authority_id = ? AND sys_menu_id = ? AND sys_base_menu_btn_id = ?", role, leaf.ID, button.ID).FirstOrCreate(&link).Error; err != nil {
				return err
			}
		}
	}
	readPaths := map[string]bool{"GET /care/clients": true, "GET /care/clients/:id": true}
	for _, role := range roles {
		for _, api := range careClientAPIs {
			if role != syntheticSupervisorRole && !readPaths[api.Method+" "+api.Path] {
				continue
			}
			policy := adapter.CasbinRule{Ptype: "p", V0: fmt.Sprint(role), V1: api.Path, V2: api.Method}
			if err := tx.Where(policy).FirstOrCreate(&policy).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedCareClients(tx *gorm.DB) error {
	fixed := time.Date(2026, time.August, 18, 9, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	teamA, teamB := uint(syntheticTeamAID), uint(syntheticTeamBID)
	clients := []caremodel.CareClient{
		{GVA_MODEL: global.GVA_MODEL{ID: 20001}, DisplayCode: "SYN-CLIENT-A001", DisplayName: "[合成] 康养用户甲", ContactMobile: "18800001001", ServiceReason: "合成测试：验证机构内责任关系与授权留痕", ServicePackageCode: "SYN-PACKAGE-A", OrganizationID: syntheticOrgAID, TeamID: &teamA, Status: caremodel.ClientStatusActive, SensitivityLevel: caremodel.SensitivitySensitive, Synthetic: true, Version: 1, DeptId: syntheticTeamAID, CreatedBy: syntheticSupervisorAID},
		{GVA_MODEL: global.GVA_MODEL{ID: 20002}, DisplayCode: "SYN-CLIENT-A002", DisplayName: "[合成] 康养用户乙", ContactMobile: "18800001002", ServiceReason: "合成测试：验证同机构责任隔离", ServicePackageCode: "SYN-PACKAGE-A", OrganizationID: syntheticOrgAID, TeamID: &teamA, Status: caremodel.ClientStatusActive, SensitivityLevel: caremodel.SensitivitySensitive, Synthetic: true, Version: 1, DeptId: syntheticTeamAID, CreatedBy: syntheticSupervisorAID},
		{GVA_MODEL: global.GVA_MODEL{ID: 20003}, DisplayCode: "SYN-CLIENT-B001", DisplayName: "[合成] 康养用户丙", ContactMobile: "18800002001", ServiceReason: "合成测试：验证跨机构不可见", ServicePackageCode: "SYN-PACKAGE-B", OrganizationID: syntheticOrgBID, TeamID: &teamB, Status: caremodel.ClientStatusActive, SensitivityLevel: caremodel.SensitivitySensitive, Synthetic: true, Version: 1, DeptId: syntheticTeamBID, CreatedBy: syntheticStewardBID},
	}
	for i := range clients {
		var existing caremodel.CareClient
		err := tx.Unscoped().Where("id = ?", clients[i].ID).First(&existing).Error
		if err == nil {
			if existing.DisplayCode != clients[i].DisplayCode || !existing.Synthetic || existing.DeletedAt.Valid {
				return fmt.Errorf("synthetic care client id %d is occupied", clients[i].ID)
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err = tx.Create(&clients[i]).Error; err != nil {
			return err
		}
	}
	assignments := []caremodel.CareAssignment{
		{CareClientID: 20001, OrganizationID: syntheticOrgAID, TeamID: syntheticTeamAID, AssigneeID: syntheticStewardAID, RoleType: caremodel.AssignmentRoleCareSteward, ValidFrom: fixed, Reason: "合成测试初始责任关系", Synthetic: true, DeptId: syntheticTeamAID, CreatedBy: syntheticSupervisorAID},
		{CareClientID: 20001, OrganizationID: syntheticOrgAID, TeamID: syntheticTeamAID, AssigneeID: syntheticClinicianAID, RoleType: caremodel.AssignmentRoleClinician, ValidFrom: fixed, Reason: "合成测试初始责任关系", Synthetic: true, DeptId: syntheticTeamAID, CreatedBy: syntheticSupervisorAID},
		{CareClientID: 20002, OrganizationID: syntheticOrgAID, TeamID: syntheticTeamAID, AssigneeID: syntheticStewardA2ID, RoleType: caremodel.AssignmentRoleCareSteward, ValidFrom: fixed, Reason: "合成测试同机构责任隔离", Synthetic: true, DeptId: syntheticTeamAID, CreatedBy: syntheticSupervisorAID},
		{CareClientID: 20003, OrganizationID: syntheticOrgBID, TeamID: syntheticTeamBID, AssigneeID: syntheticStewardBID, RoleType: caremodel.AssignmentRoleCareSteward, ValidFrom: fixed, Reason: "合成测试跨机构责任隔离", Synthetic: true, DeptId: syntheticTeamBID, CreatedBy: syntheticStewardBID},
	}
	for i := range assignments {
		where := caremodel.CareAssignment{CareClientID: assignments[i].CareClientID, AssigneeID: assignments[i].AssigneeID, RoleType: assignments[i].RoleType, Synthetic: true}
		if err := tx.Where(where).Attrs(assignments[i]).FirstOrCreate(&assignments[i]).Error; err != nil {
			return err
		}
	}
	consent := caremodel.ConsentRecord{CareClientID: 20001, ConsentType: caremodel.ConsentTypeSyntheticTestParticipation, Action: caremodel.ConsentActionGrant, TextVersion: "SYNTHETIC-V1", OccurredAt: fixed, Source: caremodel.ConsentSourceStaffRecorded, Reason: "合成测试授权夹具，不代表真实授权", RecordedBy: syntheticSupervisorAID, Synthetic: true, DeptId: syntheticTeamAID, CreatedBy: syntheticSupervisorAID}
	return tx.Where("care_client_id = ? AND consent_type = ? AND occurred_at = ?", consent.CareClientID, consent.ConsentType, consent.OccurredAt).Attrs(consent).FirstOrCreate(&consent).Error
}
