package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
)

func bizModel() error {
	db := global.GVA_DB
	err := db.AutoMigrate(
		&careclient.CareClient{},
		&careclient.CareAssignment{},
		&careclient.ConsentRecord{},
		&careclient.CareOrgUnitProfile{},
		&careclient.CareAuthorityProfile{},
		&careclient.CareClientCommandReceipt{},
	)
	if err != nil {
		return err
	}
	return nil
}
