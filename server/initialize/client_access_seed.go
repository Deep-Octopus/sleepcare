package initialize

import (
	"context"
	"errors"
	"fmt"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	clientmodel "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

const clientAccessFixtureAccountID = 9701

// EnsureClientAccessData creates only the client security principal needed by
// the fixed local task. It deliberately does not persist or print a usable
// grant; tests create short-lived bearer values in memory.
func EnsureClientAccessData() error {
	if global.GVA_DB == nil || !global.GVA_CONFIG.Care.SyntheticFixturesEnabled {
		return nil
	}
	db := global.GVA_DB.WithContext(datascope.WithSystem(context.Background()))
	return ensureClientAccessFixture(db)
}

func ensureClientAccessFixture(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var client caremodel.CareClient
		if err := tx.Where("id = ? AND status = ? AND synthetic = ?", 20001, caremodel.ClientStatusActive, true).First(&client).Error; err != nil {
			return fmt.Errorf("client access fixture client unavailable: %w", err)
		}
		var task pathmodel.TaskInstance
		if err := tx.Where("id = ? AND care_client_id = ? AND synthetic = ?", syntheticTaskInstanceD1ID, client.ID, true).First(&task).Error; err != nil {
			return fmt.Errorf("client access fixture task unavailable: %w", err)
		}
		account := clientmodel.CareClientAccount{
			GVA_MODEL:    global.GVA_MODEL{ID: clientAccessFixtureAccountID},
			CareClientID: client.ID, Status: clientmodel.AccountStatusActive,
			Version: 1, Synthetic: true, DeptId: client.DeptId, CreatedBy: syntheticSupervisorAID,
		}
		var existing clientmodel.CareClientAccount
		err := tx.Where("care_client_id = ?", client.ID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&account).Error
		}
		if err != nil {
			return err
		}
		if existing.ID != account.ID || existing.Status != account.Status || existing.DeptId != account.DeptId || !existing.Synthetic {
			return errors.New("client access fixture account conflicts with fixed definition")
		}
		return nil
	})
}
