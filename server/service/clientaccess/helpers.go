package clientaccess

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	clientmodel "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	"gorm.io/gorm"
)

func identityFromContext(ctx context.Context) (SessionIdentity, error) {
	identity, ok := SessionIdentityFromContext(ctx)
	if !ok || identity.SessionID == "" || identity.CareClientID == 0 || identity.DeptID == 0 {
		return SessionIdentity{}, invalidSession()
	}
	return identity, nil
}

func allowedTask(identity SessionIdentity, taskID uint) bool {
	for _, allowed := range identity.AllowedTaskIDs {
		if allowed == taskID {
			return true
		}
	}
	return false
}

func requestHash(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func runIdempotent[T any](s *ClientAccessService, ctx context.Context, operation, key string, request any, fn func(*gorm.DB) (T, error)) (T, error) {
	var result T
	identity, err := identityFromContext(ctx)
	if err != nil {
		return result, err
	}
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 128 {
		return result, clientmodel.NewDomainError(clientmodel.CodeInvalidArgument, "Idempotency-Key 必填且不超过 128 字符")
	}
	hash, err := requestHash(request)
	if err != nil {
		return result, err
	}
	keyDigest := DigestToken(key)
	err = s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var receipt clientmodel.ClientTaskCommandReceipt
		receiptErr := tx.Where("care_client_id = ? AND operation = ? AND key_digest = ?", identity.CareClientID, operation, keyDigest).First(&receipt).Error
		if receiptErr == nil {
			if receipt.RequestHash != hash {
				return clientmodel.NewDomainError(clientmodel.CodeIdempotencyConflict, "幂等键已用于不同请求")
			}
			return json.Unmarshal([]byte(receipt.ResultJSON), &result)
		}
		if !errors.Is(receiptErr, gorm.ErrRecordNotFound) {
			return receiptErr
		}
		result, err = fn(tx)
		if err != nil {
			return err
		}
		encoded, err := json.Marshal(result)
		if err != nil {
			return err
		}
		receipt = clientmodel.ClientTaskCommandReceipt{
			CareClientID: identity.CareClientID, Operation: operation, KeyDigest: keyDigest,
			RequestHash: hash, ResultJSON: string(encoded), DeptId: identity.DeptID, CreatedBy: identity.CareClientID,
		}
		if err = tx.Create(&receipt).Error; err != nil {
			if duplicateError(err) {
				return clientmodel.NewDomainError(clientmodel.CodeIdempotencyConflict, "幂等请求发生并发冲突，请重试")
			}
			return err
		}
		return nil
	})
	return result, err
}

func duplicateError(err error) bool {
	value := strings.ToLower(err.Error())
	return strings.Contains(value, "duplicate") || strings.Contains(value, "unique")
}

func decodeRuleIDs(raw []byte) ([]uint, error) {
	ids := []uint{}
	if len(raw) == 0 || json.Unmarshal(raw, &ids) != nil {
		return nil, clientmodel.NewDomainError(clientmodel.CodeContentDisabled, "任务绑定内容无效")
	}
	return ids, nil
}

func normalizeQuestionnaireError(err error) error {
	var domainErr *qmodel.DomainError
	if !errors.As(err, &domainErr) {
		return err
	}
	message := domainErr.Message
	switch domainErr.Code {
	case qmodel.CodeContentNotPublished:
		message = "任务内容尚不可用"
	case qmodel.CodeContentDisabled:
		message = "任务内容不可用"
	case qmodel.CodeRuleExecutionDisabled:
		message = "任务规则暂不可用"
	}
	return &clientmodel.DomainError{Code: domainErr.Code, Message: message, HTTPStatus: domainErr.HTTPStatus}
}

func normalizeCaseWorkError(err error) error {
	var domainErr *caseworkmodel.DomainError
	if !errors.As(err, &domainErr) {
		return err
	}
	return &clientmodel.DomainError{Code: domainErr.Code, Message: domainErr.Message, HTTPStatus: domainErr.HTTPStatus}
}

func normalizeSupervisionError(err error) error {
	var domainErr *supervisionmodel.DomainError
	if !errors.As(err, &domainErr) {
		return err
	}
	return &clientmodel.DomainError{Code: domainErr.Code, Message: domainErr.Message, HTTPStatus: domainErr.HTTPStatus}
}

func scopeDenied() error {
	return clientmodel.NewHTTPError(clientmodel.CodeAccessScopeDenied, http.StatusForbidden, "任务不存在或不在当前访问范围")
}

func operation(kind string, taskID uint) string {
	return fmt.Sprintf("%s:%d", kind, taskID)
}

func taskActionError(task pathmodel.TaskInstance) error {
	switch task.ExecutionStatus {
	case pathmodel.ExecutionCancelled:
		return clientmodel.NewDomainError(clientmodel.CodeTaskCancelled, "任务已取消")
	case pathmodel.ExecutionSubmitted:
		return clientmodel.NewDomainError(clientmodel.CodeOperationNotAllowed, "任务已提交")
	}
	return nil
}
