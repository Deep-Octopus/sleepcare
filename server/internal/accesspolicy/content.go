package accesspolicy

import (
	"context"

	"github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

// ContentDecision authorizes read-only access to versioned care content. It is
// intentionally separate from CareClientDecision so content administrators do
// not inherit access to clients or runtime records.
type ContentDecision struct {
	Identity *datascope.Identity
	RoleType string
}

func ResolvePlanContent(ctx context.Context, db *gorm.DB) (*ContentDecision, error) {
	identity, roleType, err := resolveContentIdentity(ctx, db)
	if err != nil {
		return nil, err
	}
	switch roleType {
	case careclient.AuthorityRoleCareSteward,
		careclient.AuthorityRoleClinician,
		careclient.AuthorityRoleSupervisor,
		careclient.AuthorityRoleContentAdmin:
		return &ContentDecision{Identity: identity, RoleType: roleType}, nil
	default:
		return nil, careclient.NewForbiddenError(careclient.CodeAccessScopeDenied, "当前角色未获方案内容预览授权")
	}
}

func resolveContentIdentity(ctx context.Context, db *gorm.DB) (*datascope.Identity, string, error) {
	identity, ok := datascope.FromContext(ctx)
	if !ok || identity == nil || identity.IsSystem || identity.UserID == 0 || identity.AuthorityID == 0 {
		return nil, "", careclient.NewForbiddenError(careclient.CodeAccessScopeDenied, "缺少有效的内容业务身份")
	}
	var profile careclient.CareAuthorityProfile
	err := db.WithContext(ctx).Set("data_scope:skip", true).
		Where("authority_id = ? AND active = ?", identity.AuthorityID, true).
		First(&profile).Error
	if err != nil {
		return nil, "", careclient.NewForbiddenError(careclient.CodeAccessScopeDenied, "当前角色未获内容预览授权")
	}
	return identity, profile.RoleType, nil
}
