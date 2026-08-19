package initialize

import (
	"context"
	"fmt"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

var aiAssistAPIs = []system.SysApi{
	{
		ApiGroup:    "AI 辅助",
		Method:      "GET",
		Path:        "/care/ai-shadow-readiness",
		Description: "查询工作人员 AI 影子能力关闭态",
	},
}

func EnsureAIAssistData() error {
	if global.GVA_DB == nil {
		return nil
	}
	db := global.GVA_DB.WithContext(datascope.WithSystem(context.Background()))
	return ensureAIAssistMetadata(db)
}

func ensureAIAssistMetadata(db *gorm.DB) error {
	for _, api := range aiAssistAPIs {
		if err := db.Where("path = ? AND method = ?", api.Path, api.Method).
			Attrs(api).FirstOrCreate(&system.SysApi{}).Error; err != nil {
			return fmt.Errorf("ensure AI assist API %s: %w", api.Path, err)
		}
	}
	return nil
}
