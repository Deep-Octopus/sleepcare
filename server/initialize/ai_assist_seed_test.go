package initialize

import (
	"context"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
)

func TestEnsureAIAssistMetadataIsIdempotentAndReadOnly(t *testing.T) {
	db := testutil.NewMemoryDBWithoutGlobal(t, &system.SysApi{}).
		WithContext(datascope.WithSystem(context.Background()))

	for i := 0; i < 2; i++ {
		if err := ensureAIAssistMetadata(db); err != nil {
			t.Fatal(err)
		}
	}

	var count int64
	if err := db.Model(&system.SysApi{}).
		Where("path = ? AND method = ?", "/care/ai-shadow-readiness", "GET").
		Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("AI shadow readiness API metadata count = %d, want 1", count)
	}
	if err := db.Model(&system.SysApi{}).
		Where("api_group = ? AND method <> ?", "AI 辅助", "GET").
		Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("AI assist metadata registered %d write APIs", count)
	}
}
