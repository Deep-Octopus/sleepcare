package aiassist

import (
	"context"
	"errors"
	"reflect"
	"testing"

	careconfig "github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	aiassistmodel "github.com/flipped-aurora/gin-vue-admin/server/model/aiassist"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
)

func TestShadowReadinessIsClosedForEveryConfiguredValue(t *testing.T) {
	wantBlockers := []string{
		aiassistmodel.BlockerPhaseTwoNotSelected,
		aiassistmodel.BlockerKnowledgeNotReviewed,
		aiassistmodel.BlockerReviewFlowMissing,
		aiassistmodel.BlockerLineageMissing,
		aiassistmodel.BlockerScenarioPolicy,
		aiassistmodel.BlockerDataPolicy,
		aiassistmodel.BlockerEvaluation,
		aiassistmodel.BlockerProvider,
	}
	for _, mode := range []string{"", aiassistmodel.ShadowModeDisabled, "SHADOW"} {
		readiness := shadowReadiness(careconfig.AIShadow{Mode: mode})
		if readiness.SelectionStatus != aiassistmodel.ShadowSelectionNotSelected ||
			readiness.UsageScope != aiassistmodel.ShadowUsageScopeNotEnabled {
			t.Fatalf("unexpected closed-state identity for mode %q: %+v", mode, readiness)
		}
		if readiness.StaffShadowEnabled || readiness.SuggestionGenerationEnabled ||
			readiness.KnowledgeRetrievalEnabled || readiness.ExternalModelEnabled ||
			readiness.UserFacingAIEnabled || readiness.DirectSendEnabled ||
			readiness.ReviewedKnowledgeReady || readiness.HumanReviewWorkflowReady ||
			readiness.LineagePersistenceReady || readiness.ProhibitedScenarioPolicyReady ||
			readiness.DataProcessingReviewReady || readiness.EvaluationProtocolReady ||
			readiness.ModelProviderReviewReady {
			t.Fatalf("AI capability unexpectedly enabled for mode %q: %+v", mode, readiness)
		}
		if !reflect.DeepEqual(readiness.Blockers, wantBlockers) {
			t.Fatalf("blockers for mode %q = %v, want %v", mode, readiness.Blockers, wantBlockers)
		}
	}
}

func TestGetShadowReadinessAllowsOnlyCareRuntimeRoles(t *testing.T) {
	db := testutil.NewMemoryDB(t, &caremodel.CareAuthorityProfile{})
	profiles := []caremodel.CareAuthorityProfile{
		{AuthorityID: 8101, RoleType: caremodel.AuthorityRoleCareSteward, Synthetic: true, Active: true},
		{AuthorityID: 8102, RoleType: caremodel.AuthorityRoleClinician, Synthetic: true, Active: true},
		{AuthorityID: 8103, RoleType: caremodel.AuthorityRoleSupervisor, Synthetic: true, Active: true},
		{AuthorityID: 8104, RoleType: caremodel.AuthorityRoleContentAdmin, Synthetic: true, Active: true},
	}
	if err := db.Create(&profiles).Error; err != nil {
		t.Fatal(err)
	}
	config := careconfig.AIShadow{Mode: aiassistmodel.ShadowModeDisabled}
	service := AIShadowService{DB: db, Config: &config}

	for _, authorityID := range []uint{8101, 8102, 8103} {
		ctx := datascope.WithIdentity(context.Background(), &datascope.Identity{
			UserID: authorityID + 100, AuthorityID: authorityID,
		})
		readiness, err := service.GetShadowReadiness(ctx)
		if err != nil {
			t.Fatalf("authority %d could not read closed state: %v", authorityID, err)
		}
		if readiness.StaffShadowEnabled || readiness.DirectSendEnabled {
			t.Fatalf("authority %d received an enabled capability: %+v", authorityID, readiness)
		}
	}

	for _, ctx := range []context.Context{
		context.Background(),
		datascope.WithIdentity(context.Background(), &datascope.Identity{UserID: 8204, AuthorityID: 8104}),
		datascope.WithIdentity(context.Background(), &datascope.Identity{UserID: 888, AuthorityID: 888}),
	} {
		_, err := service.GetShadowReadiness(ctx)
		var domainErr *aiassistmodel.DomainError
		if !errors.As(err, &domainErr) || domainErr.Code != aiassistmodel.CodeAccessScopeDenied {
			t.Fatalf("invalid runtime identity did not fail closed: %v", err)
		}
	}
}
