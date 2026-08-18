package accesspolicy

import (
	"context"
	"errors"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
)

func TestResolvePlanContentDoesNotGrantRuntimeAccess(t *testing.T) {
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
	for _, authorityID := range []uint{1, 2, 3, 4} {
		ctx := datascope.WithIdentity(context.Background(), &datascope.Identity{UserID: 10, AuthorityID: authorityID})
		if _, err := ResolvePlanContent(ctx, db); err != nil {
			t.Fatalf("authority %d should read plan content: %v", authorityID, err)
		}
	}

	contentCtx := datascope.WithIdentity(context.Background(), &datascope.Identity{UserID: 10, AuthorityID: 4})
	_, err := ResolveCareClient(contentCtx, db)
	var domainErr *careclient.DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != careclient.CodeAccessScopeDenied {
		t.Fatalf("content administrator must not receive runtime access, got %v", err)
	}
	for _, ctx := range []context.Context{
		context.Background(),
		datascope.WithIdentity(context.Background(), &datascope.Identity{UserID: 10, AuthorityID: 888}),
		datascope.WithSystem(context.Background()),
	} {
		if _, err = ResolvePlanContent(ctx, db); !errors.As(err, &domainErr) || domainErr.Code != careclient.CodeAccessScopeDenied {
			t.Fatalf("invalid content identity should fail closed, got %v", err)
		}
	}
}
