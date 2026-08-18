package initialize

import (
	"context"
	"fmt"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	caseworkservice "github.com/flipped-aurora/gin-vue-admin/server/service/casework"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

var caseWorkAPIs = []system.SysApi{
	{ApiGroup: "关注事项", Method: "GET", Path: "/care/attention-cases", Description: "获取授权范围关注事项列表"},
	{ApiGroup: "关注事项", Method: "GET", Path: "/care/attention-cases/:id", Description: "获取授权范围关注事项详情"},
	{ApiGroup: "关注事项", Method: "POST", Path: "/care/attention-cases/:id/acknowledge", Description: "确认关注事项"},
	{ApiGroup: "关注事项", Method: "POST", Path: "/care/attention-cases/:id/handling-records", Description: "追加关注事项处理记录"},
	{ApiGroup: "关注事项", Method: "POST", Path: "/care/attention-cases/:id/escalate", Description: "升级关注事项"},
	{ApiGroup: "关注事项", Method: "POST", Path: "/care/attention-cases/:id/close", Description: "关闭关注事项"},
	{ApiGroup: "关注事项", Method: "POST", Path: "/care/attention-cases/:id/reopen", Description: "重开关注事项"},
}

func EnsureCaseWorkData() error {
	if global.GVA_DB == nil {
		return nil
	}
	ctx := datascope.WithSystem(context.Background())
	db := global.GVA_DB.WithContext(ctx)
	if err := ensureCaseWorkMetadata(db, global.GVA_CONFIG.Care.SyntheticFixturesEnabled); err != nil {
		return err
	}
	if !global.GVA_CONFIG.Care.SyntheticFixturesEnabled {
		return nil
	}
	enabled := true
	_, err := (&caseworkservice.CaseWorkService{DB: db, SyntheticFixturesEnabled: &enabled}).ReconcileRuleHits(ctx)
	return err
}

func ensureCaseWorkMetadata(db *gorm.DB, grantPolicies bool) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, api := range caseWorkAPIs {
			if err := tx.Where("path = ? AND method = ?", api.Path, api.Method).Attrs(api).FirstOrCreate(&system.SysApi{}).Error; err != nil {
				return fmt.Errorf("ensure attention case API %s: %w", api.Path, err)
			}
		}
		if !grantPolicies {
			return nil
		}
		rolesByRoute := map[string][]uint{
			"GET /care/attention-cases":                       {syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole},
			"GET /care/attention-cases/:id":                   {syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole},
			"POST /care/attention-cases/:id/acknowledge":      {syntheticStewardRole, syntheticClinicianRole},
			"POST /care/attention-cases/:id/handling-records": {syntheticStewardRole, syntheticClinicianRole},
			"POST /care/attention-cases/:id/escalate":         {syntheticStewardRole, syntheticClinicianRole},
			"POST /care/attention-cases/:id/close":            {syntheticClinicianRole, syntheticSupervisorRole},
			"POST /care/attention-cases/:id/reopen":           {syntheticSupervisorRole},
		}
		for _, api := range caseWorkAPIs {
			for _, role := range rolesByRoute[api.Method+" "+api.Path] {
				policy := adapter.CasbinRule{Ptype: "p", V0: fmt.Sprint(role), V1: api.Path, V2: api.Method}
				if err := tx.Where(policy).FirstOrCreate(&policy).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
