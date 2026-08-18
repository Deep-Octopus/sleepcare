package careclient

import "github.com/flipped-aurora/gin-vue-admin/server/global"

// CareOrgUnitProfile classifies a SysDepartment without duplicating its tree.
type CareOrgUnitProfile struct {
	global.GVA_MODEL
	DepartmentID   uint   `json:"departmentId" gorm:"uniqueIndex;not null"`
	OrganizationID uint   `json:"organizationId" gorm:"index;not null"`
	Code           string `json:"code" gorm:"type:varchar(64);uniqueIndex;not null"`
	UnitType       string `json:"unitType" gorm:"type:varchar(24);index;not null"`
	Synthetic      bool   `json:"synthetic" gorm:"index;not null;default:false"`
	Active         bool   `json:"active" gorm:"index;not null;default:true"`
	DeptId         uint   `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy      uint   `json:"createdBy" gorm:"column:created_by"`
	UpdatedBy      uint   `json:"updatedBy" gorm:"column:updated_by"`
	DeletedBy      uint   `json:"-" gorm:"column:deleted_by"`
}

func (CareOrgUnitProfile) TableName() string { return "care_org_unit_profiles" }

// CareAuthorityProfile maps a mutable GVA authority to a stable domain role.
type CareAuthorityProfile struct {
	global.GVA_MODEL
	AuthorityID uint   `json:"authorityId" gorm:"uniqueIndex;not null"`
	RoleType    string `json:"roleType" gorm:"type:varchar(32);index;not null"`
	Synthetic   bool   `json:"synthetic" gorm:"index;not null;default:false"`
	Active      bool   `json:"active" gorm:"index;not null;default:true"`
}

func (CareAuthorityProfile) TableName() string { return "care_authority_profiles" }
