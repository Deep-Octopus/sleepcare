package casework

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func runIdempotent[T any](s *CaseWorkService, ctx context.Context, operation string, actorID uint, key string, request any, fn func(*gorm.DB, string) (T, error)) (T, error) {
	var zero T
	key = strings.TrimSpace(key)
	if key == "" || actorID == 0 {
		return zero, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "Idempotency-Key 和操作者必填")
	}
	keyDigest := digest(key)
	requestHash, err := hashRequest(request)
	if err != nil {
		return zero, err
	}
	err = s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var receipt caseworkmodel.CommandReceipt
		receiptErr := tx.Where("actor_id = ? AND operation = ? AND key_digest = ?", actorID, operation, keyDigest).First(&receipt).Error
		if receiptErr == nil {
			if receipt.RequestHash != requestHash {
				return caseworkmodel.NewDomainError(caseworkmodel.CodeIdempotencyConflict, "幂等键已用于不同请求")
			}
			return json.Unmarshal([]byte(receipt.ResultJSON), &zero)
		}
		if !errors.Is(receiptErr, gorm.ErrRecordNotFound) {
			return receiptErr
		}
		result, err := fn(tx, keyDigest)
		if err != nil {
			return err
		}
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return err
		}
		receipt = caseworkmodel.CommandReceipt{
			ActorID: actorID, Operation: operation, KeyDigest: keyDigest,
			RequestHash: requestHash, ResultJSON: string(resultJSON),
		}
		if err = tx.Create(&receipt).Error; err != nil {
			if duplicateError(err) {
				return caseworkmodel.NewDomainError(caseworkmodel.CodeIdempotencyConflict, "幂等请求发生并发冲突，请重试")
			}
			return err
		}
		zero = result
		return nil
	})
	return zero, err
}

func locking(db *gorm.DB) *gorm.DB {
	if db.Dialector.Name() == "mysql" || db.Dialector.Name() == "postgres" {
		return db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return db
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func hashRequest(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return digest(string(payload)), nil
}
