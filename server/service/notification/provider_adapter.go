package notification

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	notificationmodel "github.com/flipped-aurora/gin-vue-admin/server/model/notification"
)

type AdapterDescriptor struct {
	Channel                       string
	ProviderCode                  string
	UsageScope                    string
	PolicyCode                    string
	PolicyVersion                 int
	TemplateCode                  string
	QualificationEvidenceReviewed bool
	TemplateEvidenceReviewed      bool
	ReceiptSemanticsReviewed      bool
	RetryPolicyReviewed           bool
	FallbackReviewed              bool
	CostBoundaryReviewed          bool
	MaxAttemptsPerRequest         int
	RateLimitWindowSeconds        int
	RateLimitCount                int64
	CostCurrency                  string
	EstimatedCostMinor            int64
	DailyCostLimitMinor           int64
}

func (d AdapterDescriptor) requiresReservation() bool {
	return d.Channel == notificationmodel.ChannelProviderContract
}

func (d AdapterDescriptor) validate() error {
	if d.Channel == notificationmodel.ChannelDemo {
		return nil
	}
	if d.Channel != notificationmodel.ChannelProviderContract || d.UsageScope != notificationmodel.ProviderUsageTestOnly {
		return notificationmodel.NewDomainError(notificationmodel.CodeProviderGateClosed, "外部通知契约未通过测试范围门禁")
	}
	if strings.TrimSpace(d.ProviderCode) == "" || strings.TrimSpace(d.PolicyCode) == "" || d.PolicyVersion <= 0 ||
		strings.TrimSpace(d.TemplateCode) == "" {
		return notificationmodel.NewDomainError(notificationmodel.CodeProviderConfigInvalid, "通知供应商契约标识不完整")
	}
	if !d.QualificationEvidenceReviewed || !d.TemplateEvidenceReviewed || !d.ReceiptSemanticsReviewed ||
		!d.RetryPolicyReviewed || !d.FallbackReviewed || !d.CostBoundaryReviewed {
		return notificationmodel.NewDomainError(notificationmodel.CodeProviderGateClosed, "通知供应商契约仍有未关闭门禁")
	}
	if d.MaxAttemptsPerRequest <= 0 || d.RateLimitWindowSeconds <= 0 || d.RateLimitCount <= 0 ||
		d.EstimatedCostMinor <= 0 || d.DailyCostLimitMinor <= 0 || d.EstimatedCostMinor > d.DailyCostLimitMinor ||
		len(strings.TrimSpace(d.CostCurrency)) != 3 {
		return notificationmodel.NewDomainError(notificationmodel.CodeProviderConfigInvalid, "通知重试、限流或费用边界不完整")
	}
	return nil
}

type SignedProviderSubmission struct {
	ProviderCode string
	Timestamp    string
	Nonce        string
	Signature    string
	Body         []byte
}

type ProviderSubmissionReceipt struct {
	ProviderMessageID string
	AcceptedAt        time.Time
}

// ProviderGateway is intentionally only a port. This task does not ship a
// network implementation or any destination/content fields.
type ProviderGateway interface {
	Submit(context.Context, SignedProviderSubmission) (ProviderSubmissionReceipt, error)
}

type ProviderContractAdapter struct {
	Policy               AdapterDescriptor
	RequestSigningSecret []byte
	Gateway              ProviderGateway
	Clock                Clock
	Nonce                func() (string, error)
}

func (a ProviderContractAdapter) Descriptor() AdapterDescriptor { return a.Policy }

type providerSubmissionPayload struct {
	RequestReference string    `json:"requestReference"`
	AttemptReference string    `json:"attemptReference"`
	TemplateCode     string    `json:"templateCode"`
	RequestedAt      time.Time `json:"requestedAt"`
}

func (a ProviderContractAdapter) Submit(ctx context.Context, command SendCommand) ([]DeliveryReceipt, error) {
	if err := a.Policy.validate(); err != nil {
		return nil, err
	}
	if len(a.RequestSigningSecret) < 32 || a.Gateway == nil {
		return nil, notificationmodel.NewDomainError(notificationmodel.CodeProviderGateClosed, "通知请求签名或 provider gateway 未配置")
	}
	nonceSource := a.Nonce
	if nonceSource == nil {
		nonceSource = randomProviderNonce
	}
	nonce, err := nonceSource()
	if err != nil || strings.TrimSpace(nonce) == "" {
		return nil, notificationmodel.NewDomainError(notificationmodel.CodeProviderConfigInvalid, "通知请求 nonce 生成失败")
	}
	payload := providerSubmissionPayload{
		RequestReference: digest(fmt.Sprintf("%s:request:%d", a.Policy.ProviderCode, command.NotificationRequestID)),
		AttemptReference: digest(fmt.Sprintf("%s:attempt:%d", a.Policy.ProviderCode, command.NotificationAttemptID)),
		TemplateCode:     a.Policy.TemplateCode,
		RequestedAt:      command.RequestedAt,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	now := command.RequestedAt
	if a.Clock != nil {
		now = a.Clock.Now()
	}
	if now.Before(command.RequestedAt) {
		now = command.RequestedAt
	}
	timestamp := strconv.FormatInt(now.Unix(), 10)
	result, err := a.Gateway.Submit(ctx, SignedProviderSubmission{
		ProviderCode: a.Policy.ProviderCode,
		Timestamp:    timestamp,
		Nonce:        nonce,
		Signature:    ProviderSignature(a.RequestSigningSecret, timestamp, nonce, body),
		Body:         body,
	})
	if err != nil {
		return nil, err
	}
	messageID := strings.TrimSpace(result.ProviderMessageID)
	if messageID == "" || len(messageID) > 256 || result.AcceptedAt.IsZero() || result.AcceptedAt.Before(now) {
		return nil, notificationmodel.NewDomainError(notificationmodel.CodeDeliveryEventInvalid, "provider 受理回执不完整")
	}
	prefix := "provider:" + digest(a.Policy.ProviderCode+":"+messageID)
	return []DeliveryReceipt{
		{
			EventKey: prefix + ":submitted", Status: notificationmodel.AttemptStatusSubmittedToProvider,
			OccurredAt: now, AdapterReference: messageID,
		},
		{
			EventKey: prefix + ":accepted", Status: notificationmodel.AttemptStatusAccepted,
			OccurredAt: result.AcceptedAt, AdapterReference: messageID,
		},
	}, nil
}

func ProviderSignature(secret []byte, timestamp, nonce string, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write([]byte(nonce))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func randomProviderNonce() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}
