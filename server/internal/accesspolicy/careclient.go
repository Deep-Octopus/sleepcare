package accesspolicy

import (
	"context"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

// CareClientDecision is the fail-closed domain authorization decision for one
// request. GVA's DataScope still applies independently to every owned table.
type CareClientDecision struct {
	Identity  *datascope.Identity
	RoleType  string
	DataLevel string
}

func ResolveCareClient(ctx context.Context, db *gorm.DB) (*CareClientDecision, error) {
	id, ok := datascope.FromContext(ctx)
	if !ok || id == nil || id.IsSystem || id.UserID == 0 || id.AuthorityID == 0 {
		return nil, careclient.NewForbiddenError(careclient.CodeAccessScopeDenied, "缺少有效的康养业务身份")
	}
	var profile careclient.CareAuthorityProfile
	err := db.WithContext(ctx).Set("data_scope:skip", true).
		Where("authority_id = ? AND active = ?", id.AuthorityID, true).
		First(&profile).Error
	if err != nil {
		return nil, careclient.NewForbiddenError(careclient.CodeAccessScopeDenied, "当前角色未获康养用户访问授权")
	}

	decision := &CareClientDecision{Identity: id, RoleType: profile.RoleType, DataLevel: careclient.DataLevelBasic}
	switch profile.RoleType {
	case careclient.AuthorityRoleSupervisor, careclient.AuthorityRoleClinician:
		decision.DataLevel = careclient.DataLevelSensitive
	case careclient.AuthorityRoleCareSteward:
	default:
		return nil, careclient.NewForbiddenError(careclient.CodeAccessScopeDenied, "康养业务角色无效")
	}
	return decision, nil
}

func (d *CareClientDecision) CanManage() bool {
	return d != nil && d.RoleType == careclient.AuthorityRoleSupervisor
}

// Scope applies the responsibility relation in addition to the global
// department DataScope callback. Supervisors rely on DataScope; stewards and
// clinicians must also have an active assignment for the same aggregate.
func (d *CareClientDecision) Scope(db *gorm.DB, now time.Time) *gorm.DB {
	if d == nil || d.Identity == nil {
		return db.Where("1 = 0")
	}
	if d.RoleType == careclient.AuthorityRoleSupervisor {
		return db
	}
	assignmentRole := careclient.AssignmentRoleCareSteward
	if d.RoleType == careclient.AuthorityRoleClinician {
		assignmentRole = careclient.AssignmentRoleClinician
	}
	return db.Where(`EXISTS (
		SELECT 1 FROM care_assignments ca
		WHERE ca.care_client_id = care_clients.id
		  AND ca.assignee_id = ? AND ca.role_type = ?
		  AND ca.deleted_at IS NULL AND ca.cancelled_at IS NULL
		  AND ca.valid_from <= ?
		  AND (ca.valid_until IS NULL OR ca.valid_until > ?)
	)`, d.Identity.UserID, assignmentRole, now, now)
}

// ScopeAttentionCases applies the responsibility relation to the caseWork
// aggregate. Department ownership remains enforced independently by DataScope.
func (d *CareClientDecision) ScopeAttentionCases(db *gorm.DB, now time.Time) *gorm.DB {
	if d == nil || d.Identity == nil {
		return db.Where("1 = 0")
	}
	if d.RoleType == careclient.AuthorityRoleSupervisor {
		return db
	}
	assignmentRole := careclient.AssignmentRoleCareSteward
	if d.RoleType == careclient.AuthorityRoleClinician {
		assignmentRole = careclient.AssignmentRoleClinician
	}
	return db.Where(`EXISTS (
		SELECT 1 FROM care_assignments ca
		WHERE ca.care_client_id = attention_cases.care_client_id
		  AND ca.assignee_id = ? AND ca.role_type = ?
		  AND ca.deleted_at IS NULL AND ca.cancelled_at IS NULL
		  AND ca.valid_from <= ?
		  AND (ca.valid_until IS NULL OR ca.valid_until > ?)
	)`, d.Identity.UserID, assignmentRole, now, now)
}

func (d *CareClientDecision) CanAccessDepartment(departmentID uint) bool {
	if d == nil || d.Identity == nil || departmentID == 0 {
		return false
	}
	id := d.Identity
	switch id.Scope {
	case datascope.ScopeAll:
		return true
	case datascope.ScopeDeptAndChild:
		return containsDepartment(id.VisibleDeptIDs, departmentID)
	case datascope.ScopeDept:
		return containsDepartment(id.DeptIDs, departmentID)
	case datascope.ScopeCustom:
		return containsDepartment(id.CustomDeptIDs, departmentID)
	case datascope.ScopeSelf:
		return departmentID == id.DeptID
	default:
		return false
	}
}

func containsDepartment(ids []uint, target uint) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}
