package initialize

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	clientmodel "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

const (
	clientAccessFixtureAccountID    = 9701
	clientAccessFixtureCredentialID = 9702
	clientAccessFixtureUsername     = "linanran"
)

// EnsureClientAccessData creates only the client security principal needed by
// the fixed local task. It deliberately does not persist or print a usable
// grant; tests create short-lived bearer values in memory.
func EnsureClientAccessData() error {
	if global.GVA_DB == nil || !global.GVA_CONFIG.Care.SyntheticFixturesEnabled {
		return nil
	}
	if global.GVA_CONFIG.Care.FixturePassword == "" {
		return errors.New("care client login enabled but fixture password is empty")
	}
	db := global.GVA_DB.WithContext(datascope.WithSystem(context.Background()))
	return ensureClientAccessFixture(db, global.GVA_CONFIG.Care.FixturePassword)
}

func ensureClientAccessFixture(db *gorm.DB, password string) error {
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
			if err = tx.Create(&account).Error; err != nil {
				return err
			}
			existing = account
		} else if err != nil {
			return err
		} else if existing.ID != account.ID || existing.Status != account.Status || existing.DeptId != account.DeptId || !existing.Synthetic {
			return errors.New("client access fixture account conflicts with fixed definition")
		}

		passwordHash := utils.BcryptHash(password)
		passwordUpdatedAt := time.Date(2026, time.August, 18, 9, 0, 0, 0, time.FixedZone("CST", 8*60*60))
		credential := clientmodel.CareClientCredential{
			GVA_MODEL: global.GVA_MODEL{ID: clientAccessFixtureCredentialID}, AccountID: existing.ID,
			Username: clientAccessFixtureUsername, PasswordHash: passwordHash, Status: clientmodel.CredentialStatusActive,
			PasswordUpdatedAt: passwordUpdatedAt, Version: 1, Synthetic: true, DeptId: client.DeptId,
			CreatedBy: syntheticSupervisorAID,
		}
		var stored clientmodel.CareClientCredential
		err = tx.Where("account_id = ?", existing.ID).First(&stored).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&credential).Error
		}
		if err != nil {
			return err
		}
		if stored.ID != credential.ID || stored.Username != credential.Username || stored.DeptId != credential.DeptId || !stored.Synthetic {
			return errors.New("client access fixture credential conflicts with fixed definition")
		}
		if !utils.BcryptCheck(password, stored.PasswordHash) {
			return tx.Model(&stored).Updates(map[string]any{
				"password_hash":       passwordHash,
				"password_updated_at": global.GVA_CONFIG.Care.Now(),
				"failed_login_count":  0,
				"locked_until":        nil,
				"version":             gorm.Expr("version + 1"),
			}).Error
		}
		return nil
	})
}
