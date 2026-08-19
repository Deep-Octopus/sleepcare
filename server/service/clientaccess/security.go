package clientaccess

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	clientmodel "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess"
	clientres "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess/response"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func NewOpaqueToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func DigestToken(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (s *ClientAccessService) Redeem(ctx context.Context, rawGrant string) (clientres.RedeemResult, string, error) {
	rawGrant = strings.TrimSpace(rawGrant)
	if len(rawGrant) < 32 || len(rawGrant) > 512 {
		return clientres.RedeemResult{}, "", invalidGrant()
	}
	var discovered clientmodel.ClientAccessGrant
	err := s.db().WithContext(ctx).Set("data_scope:skip", true).
		Where("token_digest = ?", DigestToken(rawGrant)).First(&discovered).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return clientres.RedeemResult{}, "", invalidGrant()
		}
		return clientres.RedeemResult{}, "", err
	}
	identity := SessionIdentity{
		AccountID: discovered.AccountID, CareClientID: discovered.CareClientID, DeptID: discovered.DeptId,
		Synthetic: discovered.Synthetic,
	}
	secureCtx := ContextWithSessionIdentity(ctx, identity)
	now := s.now()
	rawSession, err := NewOpaqueToken()
	if err != nil {
		return clientres.RedeemResult{}, "", err
	}
	expiresAt := now.Add(s.sessionTTL())
	allowedCount := 0
	err = s.db().WithContext(secureCtx).Transaction(func(tx *gorm.DB) error {
		var grant clientmodel.ClientAccessGrant
		if err := locking(tx).Where("id = ? AND token_digest = ?", discovered.ID, DigestToken(rawGrant)).First(&grant).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return invalidGrant()
			}
			return err
		}
		if grant.Status != clientmodel.GrantStatusIssued || grant.RedeemedAt != nil || grant.RevokedAt != nil || !now.Before(grant.ExpiresAt) || !grant.Synthetic || !s.syntheticFixturesEnabled() {
			return invalidGrant()
		}
		var account clientmodel.CareClientAccount
		if err := tx.Where("id = ? AND care_client_id = ?", grant.AccountID, grant.CareClientID).First(&account).Error; err != nil {
			return invalidGrant()
		}
		var client caremodel.CareClient
		if err := tx.Where("id = ?", grant.CareClientID).First(&client).Error; err != nil {
			return invalidGrant()
		}
		if account.Status != clientmodel.AccountStatusActive || !account.Synthetic || client.Status != caremodel.ClientStatusActive || !client.Synthetic || account.DeptId != grant.DeptId || client.DeptId != grant.DeptId {
			return invalidGrant()
		}
		allowed, err := decodeTaskIDs(grant.AllowedTaskIDsJSON)
		if err != nil || len(allowed) == 0 {
			return invalidGrant()
		}
		var taskCount int64
		if err = tx.Model(&pathmodel.TaskInstance{}).
			Where("id IN ? AND care_client_id = ? AND execution_role = ? AND synthetic = ?", allowed, grant.CareClientID, pathmodel.ExecutionRoleCareClient, true).
			Count(&taskCount).Error; err != nil {
			return err
		}
		if taskCount != int64(len(allowed)) {
			return invalidGrant()
		}
		result := tx.Model(&clientmodel.ClientAccessGrant{}).
			Where("id = ? AND status = ? AND redeemed_at IS NULL AND revoked_at IS NULL", grant.ID, clientmodel.GrantStatusIssued).
			Updates(map[string]any{"status": clientmodel.GrantStatusRedeemed, "redeemed_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return invalidGrant()
		}
		session := clientmodel.ClientSession{
			SessionID: uuid.NewString(), AccountID: account.ID, GrantID: grant.ID, CareClientID: grant.CareClientID,
			TokenDigest: DigestToken(rawSession), AllowedTaskIDsJSON: grant.AllowedTaskIDsJSON,
			Status: clientmodel.SessionStatusActive, ExpiresAt: expiresAt, Synthetic: true, DeptId: grant.DeptId,
			CreatedBy: grant.CareClientID,
		}
		if err = tx.Create(&session).Error; err != nil {
			return err
		}
		allowedCount = len(allowed)
		return nil
	})
	if err != nil {
		return clientres.RedeemResult{}, "", err
	}
	return clientres.RedeemResult{ExpiresAt: expiresAt, AllowedTaskCount: allowedCount}, rawSession, nil
}

func (s *ClientAccessService) Authenticate(ctx context.Context, rawSession string) (SessionIdentity, error) {
	rawSession = strings.TrimSpace(rawSession)
	if len(rawSession) < 32 || len(rawSession) > 512 {
		return SessionIdentity{}, invalidSession()
	}
	var session clientmodel.ClientSession
	err := s.db().WithContext(ctx).Set("data_scope:skip", true).
		Where("token_digest = ?", DigestToken(rawSession)).First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SessionIdentity{}, invalidSession()
		}
		return SessionIdentity{}, err
	}
	if session.Status != clientmodel.SessionStatusActive || session.RevokedAt != nil || !s.now().Before(session.ExpiresAt) || !session.Synthetic || !s.syntheticFixturesEnabled() {
		return SessionIdentity{}, invalidSession()
	}
	identity := SessionIdentity{
		SessionID: session.SessionID, AccountID: session.AccountID, CareClientID: session.CareClientID, DeptID: session.DeptId,
		Synthetic: session.Synthetic,
	}
	secureCtx := ContextWithSessionIdentity(ctx, identity)
	var account clientmodel.CareClientAccount
	if err = s.db().WithContext(secureCtx).Where("id = ? AND care_client_id = ?", session.AccountID, session.CareClientID).First(&account).Error; err != nil {
		return SessionIdentity{}, invalidSession()
	}
	var client caremodel.CareClient
	if err = s.db().WithContext(secureCtx).Where("id = ?", session.CareClientID).First(&client).Error; err != nil {
		return SessionIdentity{}, invalidSession()
	}
	if account.Status != clientmodel.AccountStatusActive || !account.Synthetic || client.Status != caremodel.ClientStatusActive || !client.Synthetic || account.DeptId != session.DeptId || client.DeptId != session.DeptId {
		return SessionIdentity{}, invalidSession()
	}
	allowed, err := decodeTaskIDs(session.AllowedTaskIDsJSON)
	if err != nil || len(allowed) == 0 {
		return SessionIdentity{}, invalidSession()
	}
	identity.AllowedTaskIDs = allowed
	return identity, nil
}

func invalidGrant() error {
	return clientmodel.NewDomainError(clientmodel.CodeGrantInvalid, "访问链接无效或已失效")
}

func invalidSession() error {
	return clientmodel.NewHTTPError(clientmodel.CodeSessionInvalid, http.StatusUnauthorized, "访问会话无效或已失效")
}

func decodeTaskIDs(raw []byte) ([]uint, error) {
	var ids []uint
	if len(raw) == 0 || json.Unmarshal(raw, &ids) != nil {
		return nil, clientmodel.NewDomainError(clientmodel.CodeAccessScopeDenied, "任务访问范围无效")
	}
	seen := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			return nil, clientmodel.NewDomainError(clientmodel.CodeAccessScopeDenied, "任务访问范围无效")
		}
		if _, ok := seen[id]; ok {
			return nil, clientmodel.NewDomainError(clientmodel.CodeAccessScopeDenied, "任务访问范围无效")
		}
		seen[id] = struct{}{}
	}
	return ids, nil
}

func locking(db *gorm.DB) *gorm.DB {
	if db.Dialector.Name() == "mysql" || db.Dialector.Name() == "postgres" {
		return db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return db
}
