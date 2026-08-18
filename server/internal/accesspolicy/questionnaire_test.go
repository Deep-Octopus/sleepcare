package accesspolicy

import (
	"context"
	"errors"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	"github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
)

func TestResolveQuestionnaireAllowsContentReviewRoles(t *testing.T) {
	db := testutil.NewMemoryDB(t, &careclient.CareAuthorityProfile{})
	profiles := []careclient.CareAuthorityProfile{
		{AuthorityID: 1, RoleType: careclient.AuthorityRoleCareSteward, Active: true, Synthetic: true},
		{AuthorityID: 2, RoleType: careclient.AuthorityRoleClinician, Active: true, Synthetic: true},
		{AuthorityID: 3, RoleType: careclient.AuthorityRoleSupervisor, Active: true, Synthetic: true},
		{AuthorityID: 4, RoleType: careclient.AuthorityRoleContentAdmin, Active: true, Synthetic: true},
	}
	if err := db.Create(&profiles).Error; err != nil {
		t.Fatal(err)
	}
	for _, authorityID := range []uint{2, 3, 4} {
		ctx := datascope.WithIdentity(context.Background(), &datascope.Identity{UserID: 10, AuthorityID: authorityID})
		if _, err := ResolveQuestionnaire(ctx, db); err != nil {
			t.Fatalf("authority %d should be allowed: %v", authorityID, err)
		}
	}
	for _, authorityID := range []uint{0, 1, 888} {
		ctx := datascope.WithIdentity(context.Background(), &datascope.Identity{UserID: 10, AuthorityID: authorityID})
		_, err := ResolveQuestionnaire(ctx, db)
		var domainErr *questionnaire.DomainError
		if !errors.As(err, &domainErr) || domainErr.Code != questionnaire.CodeAccessScopeDenied {
			t.Fatalf("authority %d should fail closed, got %v", authorityID, err)
		}
	}
}
