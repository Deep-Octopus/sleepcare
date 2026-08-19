package aiassist

import (
	"context"
	"errors"

	careconfig "github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	aiassistmodel "github.com/flipped-aurora/gin-vue-admin/server/model/aiassist"
	aiassistres "github.com/flipped-aurora/gin-vue-admin/server/model/aiassist/response"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
)

func (s *AIShadowService) GetShadowReadiness(ctx context.Context) (aiassistres.ShadowReadiness, error) {
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return aiassistres.ShadowReadiness{}, normalizeAccessError(err)
	}
	switch decision.RoleType {
	case caremodel.AuthorityRoleCareSteward, caremodel.AuthorityRoleClinician, caremodel.AuthorityRoleSupervisor:
	default:
		return aiassistres.ShadowReadiness{}, aiassistmodel.NewForbiddenError("当前角色无权查看 AI 影子能力门禁")
	}
	return shadowReadiness(s.config()), nil
}

func shadowReadiness(config careconfig.AIShadow) aiassistres.ShadowReadiness {
	return aiassistres.ShadowReadiness{
		Mode:            config.NormalizedMode(),
		SelectionStatus: aiassistmodel.ShadowSelectionNotSelected,
		UsageScope:      aiassistmodel.ShadowUsageScopeNotEnabled,
		Blockers: []string{
			aiassistmodel.BlockerPhaseTwoNotSelected,
			aiassistmodel.BlockerKnowledgeNotReviewed,
			aiassistmodel.BlockerReviewFlowMissing,
			aiassistmodel.BlockerLineageMissing,
			aiassistmodel.BlockerScenarioPolicy,
			aiassistmodel.BlockerDataPolicy,
			aiassistmodel.BlockerEvaluation,
			aiassistmodel.BlockerProvider,
		},
	}
}

func normalizeAccessError(err error) error {
	var domainErr *caremodel.DomainError
	if errors.As(err, &domainErr) {
		return &aiassistmodel.DomainError{
			Code:       domainErr.Code,
			Message:    domainErr.Message,
			HTTPStatus: domainErr.HTTPStatus,
		}
	}
	return err
}
