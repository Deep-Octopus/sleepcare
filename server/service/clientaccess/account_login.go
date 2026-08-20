package clientaccess

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	clientmodel "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess"
	clientreq "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess/request"
	clientres "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	maxCredentialFailures = 5
	credentialLockWindow  = 15 * time.Minute
)

var dummyCredentialHash = utils.BcryptHash("client-login-timing-equalizer")

func (s *ClientAccessService) Login(ctx context.Context, req clientreq.Login) (clientres.LoginResult, string, error) {
	username := normalizeUsername(req.Username)
	if len(username) < 3 || len(username) > 64 || len(req.Password) == 0 || len(req.Password) > 128 {
		utils.BcryptCheck(req.Password, dummyCredentialHash)
		return clientres.LoginResult{}, "", invalidCredentials()
	}

	now := s.now()
	var result clientres.LoginResult
	var rawSession string
	var loginErr error
	err := s.db().WithContext(ctx).Set("data_scope:skip", true).Transaction(func(tx *gorm.DB) error {
		var credential clientmodel.CareClientCredential
		query := locking(tx.Set("data_scope:skip", true)).Where("username = ?", username)
		if err := query.First(&credential).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				utils.BcryptCheck(req.Password, dummyCredentialHash)
				loginErr = invalidCredentials()
				return nil
			}
			return err
		}
		if credential.Status != clientmodel.CredentialStatusActive || !credential.Synthetic || !s.syntheticFixturesEnabled() {
			utils.BcryptCheck(req.Password, credential.PasswordHash)
			loginErr = invalidCredentials()
			return nil
		}
		if credential.LockedUntil != nil && now.Before(*credential.LockedUntil) {
			loginErr = credentialLocked()
			return nil
		}
		if !utils.BcryptCheck(req.Password, credential.PasswordHash) {
			failed := credential.FailedLoginCount + 1
			updates := map[string]any{
				"failed_login_count": failed,
				"version":            gorm.Expr("version + 1"),
			}
			if failed >= maxCredentialFailures {
				lockedUntil := now.Add(credentialLockWindow)
				updates["locked_until"] = lockedUntil
				loginErr = credentialLocked()
			} else {
				loginErr = invalidCredentials()
			}
			return tx.Model(&credential).Updates(updates).Error
		}

		var account clientmodel.CareClientAccount
		if err := tx.Set("data_scope:skip", true).Where("id = ?", credential.AccountID).First(&account).Error; err != nil {
			loginErr = invalidCredentials()
			return nil
		}
		var client caremodel.CareClient
		if err := tx.Set("data_scope:skip", true).Where("id = ?", account.CareClientID).First(&client).Error; err != nil {
			loginErr = invalidCredentials()
			return nil
		}
		if account.Status != clientmodel.AccountStatusActive || !account.Synthetic || client.Status != caremodel.ClientStatusActive ||
			!client.Synthetic || account.DeptId != credential.DeptId || client.DeptId != credential.DeptId {
			loginErr = invalidCredentials()
			return nil
		}

		var tokenErr error
		rawSession, tokenErr = NewOpaqueToken()
		if tokenErr != nil {
			return tokenErr
		}
		expiresAt := now.Add(s.sessionTTL())
		session := clientmodel.ClientSession{
			SessionID: uuid.NewString(), AccountID: account.ID, CareClientID: client.ID,
			TokenDigest: DigestToken(rawSession), AllowedTaskIDsJSON: datatypes.JSON([]byte("[]")),
			AuthType: clientmodel.SessionAuthAccount, Status: clientmodel.SessionStatusActive,
			ExpiresAt: expiresAt, Synthetic: true, DeptId: client.DeptId, CreatedBy: client.ID,
		}
		if err := tx.Create(&session).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"failed_login_count": 0,
			"locked_until":       nil,
			"last_login_at":      now,
			"version":            gorm.Expr("version + 1"),
			"updated_by":         client.ID,
		}
		if err := tx.Model(&credential).Updates(updates).Error; err != nil {
			return err
		}
		result = clientres.LoginResult{
			ExpiresAt: expiresAt,
			Profile:   clientres.ClientProfile{DisplayName: client.DisplayName, DisplayCode: client.DisplayCode},
		}
		return nil
	})
	if err != nil {
		return clientres.LoginResult{}, "", err
	}
	if loginErr != nil {
		return clientres.LoginResult{}, "", loginErr
	}
	return result, rawSession, nil
}

func (s *ClientAccessService) GetProfile(ctx context.Context) (clientres.SessionProfile, error) {
	identity, err := identityFromContext(ctx)
	if err != nil {
		return clientres.SessionProfile{}, err
	}
	var client caremodel.CareClient
	if err = s.db().WithContext(ctx).Where("id = ?", identity.CareClientID).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return clientres.SessionProfile{}, invalidSession()
		}
		return clientres.SessionProfile{}, err
	}
	return clientres.SessionProfile{
		ClientProfile: clientres.ClientProfile{DisplayName: client.DisplayName, DisplayCode: client.DisplayCode},
		ExpiresAt:     identity.ExpiresAt,
	}, nil
}

func (s *ClientAccessService) Logout(ctx context.Context) (clientres.LogoutResult, error) {
	identity, err := identityFromContext(ctx)
	if err != nil {
		return clientres.LogoutResult{}, err
	}
	now := s.now()
	result := s.db().WithContext(ctx).Set("data_scope:skip", true).Model(&clientmodel.ClientSession{}).
		Where("session_id = ? AND account_id = ? AND status = ?", identity.SessionID, identity.AccountID, clientmodel.SessionStatusActive).
		Updates(map[string]any{"status": clientmodel.SessionStatusRevoked, "revoked_at": now, "updated_by": identity.CareClientID})
	if result.Error != nil {
		return clientres.LogoutResult{}, result.Error
	}
	if result.RowsAffected != 1 {
		return clientres.LogoutResult{}, invalidSession()
	}
	return clientres.LogoutResult{SignedOut: true}, nil
}

func normalizeUsername(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func invalidCredentials() error {
	return clientmodel.NewHTTPError(clientmodel.CodeCredentialsInvalid, http.StatusUnauthorized, "账号或密码不正确")
}

func credentialLocked() error {
	return clientmodel.NewHTTPError(clientmodel.CodeCredentialLocked, http.StatusTooManyRequests, "登录尝试过多，请稍后再试")
}
