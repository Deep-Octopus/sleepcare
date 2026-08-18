package careclient

import "github.com/flipped-aurora/gin-vue-admin/server/global"

// CareClient is the single public profile shared by all care paths. It is not
// a system login account and must not contain path-specific medical facts.
type CareClient struct {
	global.GVA_MODEL
	DisplayCode        string `json:"displayCode" gorm:"type:varchar(64);uniqueIndex;not null;comment:合成康养用户稳定编码"`
	DisplayName        string `json:"displayName" gorm:"type:varchar(100);not null;comment:显示名称"`
	ContactMobile      string `json:"contactMobile" gorm:"type:varchar(32);comment:联系电话"`
	ServiceReason      string `json:"serviceReason" gorm:"type:varchar(500);comment:非医疗服务原因"`
	ServicePackageCode string `json:"servicePackageCode" gorm:"type:varchar(64);comment:服务包编码"`
	OrganizationID     uint   `json:"organizationId" gorm:"index;not null;comment:机构部门ID"`
	TeamID             *uint  `json:"teamId" gorm:"index;comment:团队部门ID"`
	Status             string `json:"status" gorm:"type:varchar(24);index;not null;default:ACTIVE;comment:状态"`
	SensitivityLevel   string `json:"sensitivityLevel" gorm:"type:varchar(32);not null;default:BASIC;comment:敏感等级"`
	Synthetic          bool   `json:"synthetic" gorm:"index;not null;default:false;comment:是否合成数据"`
	Version            uint   `json:"version" gorm:"not null;default:1;comment:乐观锁版本"`
	DeptId             uint   `json:"deptId" gorm:"column:dept_id;index;not null;comment:数据权限归属部门"`
	CreatedBy          uint   `json:"createdBy" gorm:"column:created_by;index;comment:创建人"`
	UpdatedBy          uint   `json:"updatedBy" gorm:"column:updated_by;comment:更新人"`
	DeletedBy          uint   `json:"-" gorm:"column:deleted_by;comment:删除人"`
}

func (CareClient) TableName() string { return "care_clients" }
