package accesspolicy

import (
	"context"

	"github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	"github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

type QuestionnaireDecision struct {
	Identity *datascope.Identity
	RoleType string
}

// ResolveQuestionnaire fails closed. Content preview is intentionally limited
// to clinician and supervisor roles; care stewards and unmapped admins are not
// questionnaire content reviewers in P1-03.
func ResolveQuestionnaire(ctx context.Context, db *gorm.DB) (*QuestionnaireDecision, error) {
	id, ok := datascope.FromContext(ctx)
	if !ok || id == nil || id.IsSystem || id.UserID == 0 || id.AuthorityID == 0 {
		return nil, questionnaire.NewForbiddenError("缺少有效的问卷业务身份")
	}
	var profile careclient.CareAuthorityProfile
	err := db.WithContext(ctx).Set("data_scope:skip", true).
		Where("authority_id = ? AND active = ?", id.AuthorityID, true).
		First(&profile).Error
	if err != nil {
		return nil, questionnaire.NewForbiddenError("当前角色未获问卷预览授权")
	}
	if profile.RoleType != careclient.AuthorityRoleClinician && profile.RoleType != careclient.AuthorityRoleSupervisor {
		return nil, questionnaire.NewForbiddenError("当前业务角色未获问卷预览授权")
	}
	return &QuestionnaireDecision{Identity: id, RoleType: profile.RoleType}, nil
}
