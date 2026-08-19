package notification

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	notificationmodel "github.com/flipped-aurora/gin-vue-admin/server/model/notification"
	notificationreq "github.com/flipped-aurora/gin-vue-admin/server/model/notification/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

const maxProviderCallbackBodyBytes = 64 * 1024

func (s *NotificationService) ApplyProviderCallback(
	ctx context.Context,
	providerCode string,
	body []byte,
	signature notificationreq.ProviderCallbackSignature,
) (caseworkres.ActionResult, error) {
	config := s.providerConfig()
	readiness := providerReadiness(config, s.fixturesEnabled())
	providerCode = strings.TrimSpace(providerCode)
	if !readiness.CallbackEndpointEnabled || !strings.EqualFold(providerCode, config.ProviderCode) {
		return caseworkres.ActionResult{}, callbackDomainError(notificationmodel.CodeProviderGateClosed, "通知回执未被接受", http.StatusNotFound)
	}
	if len(body) == 0 || len(body) > maxProviderCallbackBodyBytes {
		return caseworkres.ActionResult{}, callbackDomainError(notificationmodel.CodeDeliveryEventInvalid, "通知回执格式无效", http.StatusBadRequest)
	}
	signedAt, err := verifyProviderCallbackSignature(config.CallbackVerificationSecret, config.CallbackMaxSkewSeconds, s.now(), body, signature)
	if err != nil {
		return caseworkres.ActionResult{}, err
	}
	payload, err := decodeProviderCallback(body)
	if err != nil {
		return caseworkres.ActionResult{}, err
	}
	if payload.OccurredAt.After(s.now().Add(time.Duration(config.CallbackMaxSkewSeconds) * time.Second)) {
		return caseworkres.ActionResult{}, callbackDomainError(notificationmodel.CodeDeliveryEventInvalid, "通知回执时间无效", http.StatusBadRequest)
	}

	payloadDigest := digest(string(body))
	eventDigest := digest(payload.EventID)
	nonceDigest := digest(signature.Nonce)
	messageDigest := digest(payload.ProviderMessageID)
	systemCtx := datascope.WithSystem(ctx)
	var result caseworkres.ActionResult
	err = s.db().WithContext(systemCtx).Transaction(func(tx *gorm.DB) error {
		var existing notificationmodel.NotificationProviderCallback
		existingErr := tx.Where("provider_code = ? AND event_key_digest = ?", config.ProviderCode, eventDigest).
			First(&existing).Error
		if existingErr == nil {
			if existing.PayloadDigest != payloadDigest || existing.NonceDigest != nonceDigest || existing.Status != payload.Status {
				return callbackDomainError(notificationmodel.CodeCallbackReplay, "通知回执事件发生冲突", http.StatusConflict)
			}
			var replayErr error
			result, replayErr = s.currentActionTx(tx, existing.NotificationAttemptID)
			return replayErr
		}
		if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
			return existingErr
		}
		var nonceCount int64
		if err := tx.Model(&notificationmodel.NotificationProviderCallback{}).
			Where("provider_code = ? AND nonce_digest = ?", config.ProviderCode, nonceDigest).
			Count(&nonceCount).Error; err != nil {
			return err
		}
		if nonceCount != 0 {
			return callbackDomainError(notificationmodel.CodeCallbackReplay, "通知回执 nonce 已使用", http.StatusConflict)
		}
		var attempt notificationmodel.NotificationAttempt
		attemptErr := locking(tx).
			Where("provider_code = ? AND provider_message_id_hash = ?", config.ProviderCode, messageDigest).
			First(&attempt).Error
		if attemptErr != nil {
			if errors.Is(attemptErr, gorm.ErrRecordNotFound) {
				return callbackDomainError(notificationmodel.CodeCallbackUnauthorized, "通知回执未被接受", http.StatusUnauthorized)
			}
			return attemptErr
		}
		if attempt.Channel != notificationmodel.ChannelProviderContract || attempt.DispatchPolicyCode != config.PolicyCode ||
			attempt.DispatchPolicyVersion != config.PolicyVersion {
			return callbackDomainError(notificationmodel.CodeCallbackUnauthorized, "通知回执未被接受", http.StatusUnauthorized)
		}
		callback := notificationmodel.NotificationProviderCallback{
			ProviderCode: config.ProviderCode, EventKeyDigest: eventDigest, NonceDigest: nonceDigest,
			PayloadDigest: payloadDigest, NotificationAttemptID: attempt.ID,
			Status: payload.Status, OccurredAt: payload.OccurredAt, ReceivedAt: s.now(),
			SignatureTimestamp: signedAt, Synthetic: attempt.Synthetic,
			DeptId: attempt.DeptId, CreatedBy: attempt.CreatedBy,
		}
		if err := tx.Create(&callback).Error; err != nil {
			if duplicateError(err) {
				return callbackDomainError(notificationmodel.CodeCallbackReplay, "通知回执已处理", http.StatusConflict)
			}
			return err
		}
		result, err = s.applyDeliveryReceipt(tx, attempt.ID, DeliveryReceipt{
			EventKey: "callback:" + eventDigest,
			Status:   payload.Status, OccurredAt: payload.OccurredAt,
			FailureCode: payload.FailureCode, AdapterReference: payload.ProviderMessageID,
		})
		if err != nil {
			return err
		}
		if err := tx.Model(&notificationmodel.NotificationProviderCallback{}).
			Where("id = ?", callback.ID).Update("delivery_event_id", result.ActionID).Error; err != nil {
			return err
		}
		return nil
	})
	return result, err
}

func verifyProviderCallbackSignature(
	secret string,
	maxSkewSeconds int,
	now time.Time,
	body []byte,
	signature notificationreq.ProviderCallbackSignature,
) (time.Time, error) {
	if len(secret) < 32 || maxSkewSeconds < 30 {
		return time.Time{}, callbackDomainError(notificationmodel.CodeProviderGateClosed, "通知回执未被接受", http.StatusNotFound)
	}
	timestampValue := strings.TrimSpace(signature.Timestamp)
	nonce := strings.TrimSpace(signature.Nonce)
	provided := strings.ToLower(strings.TrimSpace(signature.Signature))
	unixSeconds, err := strconv.ParseInt(timestampValue, 10, 64)
	if err != nil || !validCallbackNonce(nonce) || len(provided) != 64 {
		return time.Time{}, callbackDomainError(notificationmodel.CodeCallbackUnauthorized, "通知回执认证失败", http.StatusUnauthorized)
	}
	signedAt := time.Unix(unixSeconds, 0)
	skew := now.Sub(signedAt)
	if skew < 0 {
		skew = -skew
	}
	if skew > time.Duration(maxSkewSeconds)*time.Second {
		return time.Time{}, callbackDomainError(notificationmodel.CodeCallbackUnauthorized, "通知回执认证已过期", http.StatusUnauthorized)
	}
	expected := ProviderSignature([]byte(secret), timestampValue, nonce, body)
	if !hmac.Equal([]byte(expected), []byte(provided)) {
		return time.Time{}, callbackDomainError(notificationmodel.CodeCallbackUnauthorized, "通知回执认证失败", http.StatusUnauthorized)
	}
	return signedAt, nil
}

func validCallbackNonce(value string) bool {
	if len(value) < 16 || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}

func decodeProviderCallback(body []byte) (notificationreq.ProviderCallback, error) {
	var payload notificationreq.ProviderCallback
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return payload, callbackDomainError(notificationmodel.CodeDeliveryEventInvalid, "通知回执格式无效", http.StatusBadRequest)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return payload, callbackDomainError(notificationmodel.CodeDeliveryEventInvalid, "通知回执只能包含一个 JSON 对象", http.StatusBadRequest)
	}
	payload.EventID = strings.TrimSpace(payload.EventID)
	payload.ProviderMessageID = strings.TrimSpace(payload.ProviderMessageID)
	payload.Status = strings.ToUpper(strings.TrimSpace(payload.Status))
	payload.FailureCode = strings.ToUpper(strings.TrimSpace(payload.FailureCode))
	if payload.EventID == "" || len(payload.EventID) > 128 || payload.ProviderMessageID == "" || len(payload.ProviderMessageID) > 256 ||
		payload.OccurredAt.IsZero() || !notificationmodel.IsProviderCallbackStatus(payload.Status) {
		return payload, callbackDomainError(notificationmodel.CodeDeliveryEventInvalid, "通知回执字段无效", http.StatusBadRequest)
	}
	switch payload.Status {
	case notificationmodel.AttemptStatusAccepted, notificationmodel.AttemptStatusDelivered:
		if payload.FailureCode != "" {
			return payload, callbackDomainError(notificationmodel.CodeDeliveryEventInvalid, "成功回执不能携带失败码", http.StatusBadRequest)
		}
	case notificationmodel.AttemptStatusFailed:
		if payload.FailureCode != notificationmodel.ProviderFailureRejected && payload.FailureCode != notificationmodel.ProviderFailureUnavailable {
			return payload, callbackDomainError(notificationmodel.CodeDeliveryEventInvalid, "失败回执必须使用标准失败码", http.StatusBadRequest)
		}
	case notificationmodel.AttemptStatusUnknown:
		if payload.FailureCode != notificationmodel.DemoUnknownCode {
			return payload, callbackDomainError(notificationmodel.CodeDeliveryEventInvalid, "未知回执必须使用标准未知码", http.StatusBadRequest)
		}
	}
	return payload, nil
}

func callbackDomainError(code int, message string, status int) error {
	return &notificationmodel.DomainError{Code: code, Message: message, HTTPStatus: status}
}
