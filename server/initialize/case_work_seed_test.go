package initialize

import (
	"fmt"
	"testing"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestEnsureCaseWorkMetadataIsIdempotentAndRoleScoped(t *testing.T) {
	db := testutil.NewMemoryDBWithoutGlobal(t, &system.SysApi{}, &adapter.CasbinRule{})
	require.NoError(t, ensureCaseWorkMetadata(db, true))
	require.NoError(t, ensureCaseWorkMetadata(db, true))

	var apiCount int64
	require.NoError(t, db.Model(&system.SysApi{}).Where("path LIKE ?", "/care/attention-cases%").Count(&apiCount).Error)
	require.Equal(t, int64(7), apiCount)

	assertCaseWorkPolicy(t, db, syntheticStewardRole, "/care/attention-cases", "GET", true)
	assertCaseWorkPolicy(t, db, syntheticStewardRole, "/care/attention-cases/:id/acknowledge", "POST", true)
	assertCaseWorkPolicy(t, db, syntheticStewardRole, "/care/attention-cases/:id/close", "POST", false)
	assertCaseWorkPolicy(t, db, syntheticClinicianRole, "/care/attention-cases/:id/close", "POST", true)
	assertCaseWorkPolicy(t, db, syntheticClinicianRole, "/care/attention-cases/:id/reopen", "POST", false)
	assertCaseWorkPolicy(t, db, syntheticSupervisorRole, "/care/attention-cases/:id/acknowledge", "POST", false)
	assertCaseWorkPolicy(t, db, syntheticSupervisorRole, "/care/attention-cases/:id/reopen", "POST", true)
}

func assertCaseWorkPolicy(t *testing.T, db *gorm.DB, role uint, path, method string, want bool) {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&adapter.CasbinRule{}).
		Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", fmt.Sprint(role), path, method).
		Count(&count).Error)
	if want {
		require.Equal(t, int64(1), count)
		return
	}
	require.Zero(t, count)
}
