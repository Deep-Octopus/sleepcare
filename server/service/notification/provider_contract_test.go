package notification

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	careconfig "github.com/flipped-aurora/gin-vue-admin/server/config"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	notificationmodel "github.com/flipped-aurora/gin-vue-admin/server/model/notification"
	notificationreq "github.com/flipped-aurora/gin-vue-admin/server/model/notification/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
)

func TestProviderContractAdapterSignsMinimalPayloadAndAppliesVerifiedCallback(t *testing.T) {
	fixture := newNotificationFixture(t)
	config := providerTestConfig()
	gateway := &recordingProviderGateway{
		receipt: ProviderSubmissionReceipt{
			ProviderMessageID: "provider-message-001",
			AcceptedAt:        fixture.now.Add(time.Second),
		},
	}
	fixture.useProviderAdapter(config, gateway)
	systemCtx := datascope.WithSystem(context.Background())

	accepted, err := fixture.service.CreateInitial(systemCtx, fixture.task.ID, "provider-initial")
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != notificationmodel.AttemptStatusAccepted || accepted.Version != 3 || gateway.calls != 1 {
		t.Fatalf("unexpected provider acceptance: result=%+v calls=%d", accepted, gateway.calls)
	}
	if got := ProviderSignature([]byte(config.RequestSigningSecret), gateway.last.Timestamp, gateway.last.Nonce, gateway.last.Body); got != gateway.last.Signature {
		t.Fatalf("provider request signature mismatch: got %s want %s", got, gateway.last.Signature)
	}
	var outbound map[string]any
	if err = json.Unmarshal(gateway.last.Body, &outbound); err != nil {
		t.Fatal(err)
	}
	if len(outbound) != 4 || outbound["requestReference"] == "" || outbound["attemptReference"] == "" || outbound["templateCode"] != config.TemplateCode {
		t.Fatalf("unexpected provider request payload: %+v", outbound)
	}
	for _, forbidden := range []string{"phone", "mobile", "destination", "content", "body"} {
		if strings.Contains(strings.ToLower(string(gateway.last.Body)), forbidden) {
			t.Fatalf("provider request contains forbidden field %q: %s", forbidden, gateway.last.Body)
		}
	}

	delivered := notificationreq.ProviderCallback{
		EventID: "provider-event-delivered-001", ProviderMessageID: gateway.receipt.ProviderMessageID,
		Status: notificationmodel.AttemptStatusDelivered, OccurredAt: fixture.now.Add(2 * time.Second),
	}
	body := mustCallbackBody(t, delivered)
	signature := fixture.callbackSignature(config, body, "callback-nonce-0001")
	result, err := fixture.service.ApplyProviderCallback(context.Background(), config.ProviderCode, body, signature)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != notificationmodel.AttemptStatusDelivered || result.Version != 4 {
		t.Fatalf("unexpected delivered result: %+v", result)
	}
	replayed, err := fixture.service.ApplyProviderCallback(context.Background(), config.ProviderCode, body, signature)
	if err != nil {
		t.Fatalf("same callback should replay: %v", err)
	}
	if replayed.ActionID != result.ActionID || replayed.ResourceID != result.ResourceID {
		t.Fatalf("callback replay differs: first=%+v replay=%+v", result, replayed)
	}
	assertNotificationCount(t, fixture.db, &notificationmodel.NotificationProviderCallback{}, "notification_attempt_id = ?", result.ResourceID, 1)
	assertNotificationCount(t, fixture.db, &notificationmodel.DeliveryEvent{}, "notification_attempt_id = ?", result.ResourceID, 4)

	var attempt notificationmodel.NotificationAttempt
	if err = fixture.db.WithContext(systemCtx).First(&attempt, result.ResourceID).Error; err != nil {
		t.Fatal(err)
	}
	if attempt.ProviderMessageIDHash == nil || *attempt.ProviderMessageIDHash == gateway.receipt.ProviderMessageID {
		t.Fatalf("provider message identifier was not digested: %+v", attempt.ProviderMessageIDHash)
	}
	var reservation notificationmodel.NotificationDispatchReservation
	if err = fixture.db.WithContext(systemCtx).Where("notification_attempt_id = ?", attempt.ID).First(&reservation).Error; err != nil {
		t.Fatal(err)
	}
	if reservation.Status != notificationmodel.DispatchReservationFinalized || reservation.EstimatedCostMinor != config.EstimatedCostMinor {
		t.Fatalf("dispatch reservation did not finalize: %+v", reservation)
	}
}

func TestProviderCallbackRejectsTamperingExpiryAndNonceReplay(t *testing.T) {
	fixture := newNotificationFixture(t)
	config := providerTestConfig()
	gateway := &recordingProviderGateway{receipt: ProviderSubmissionReceipt{
		ProviderMessageID: "provider-message-002", AcceptedAt: fixture.now.Add(time.Second),
	}}
	fixture.useProviderAdapter(config, gateway)
	accepted, err := fixture.service.CreateInitial(datascope.WithSystem(context.Background()), fixture.task.ID, "provider-security")
	if err != nil {
		t.Fatal(err)
	}
	payload := notificationreq.ProviderCallback{
		EventID: "provider-event-failed-001", ProviderMessageID: gateway.receipt.ProviderMessageID,
		Status: notificationmodel.AttemptStatusFailed, FailureCode: notificationmodel.ProviderFailureRejected,
		OccurredAt: fixture.now.Add(2 * time.Second),
	}
	body := mustCallbackBody(t, payload)
	valid := fixture.callbackSignature(config, body, "callback-nonce-0002")
	tampered := append([]byte(nil), body...)
	tampered[len(tampered)-2] ^= 1
	if _, err = fixture.service.ApplyProviderCallback(context.Background(), config.ProviderCode, tampered, valid); notificationCode(err) != notificationmodel.CodeCallbackUnauthorized {
		t.Fatalf("tampered callback should be unauthorized, got %v", err)
	}
	stale := valid
	stale.Timestamp = strconv.FormatInt(fixture.now.Add(-10*time.Minute).Unix(), 10)
	stale.Signature = ProviderSignature([]byte(config.CallbackVerificationSecret), stale.Timestamp, stale.Nonce, body)
	if _, err = fixture.service.ApplyProviderCallback(context.Background(), config.ProviderCode, body, stale); notificationCode(err) != notificationmodel.CodeCallbackUnauthorized {
		t.Fatalf("stale callback should be unauthorized, got %v", err)
	}
	if _, err = fixture.service.ApplyProviderCallback(context.Background(), config.ProviderCode, body, valid); err != nil {
		t.Fatal(err)
	}
	payload.EventID = "provider-event-failed-002"
	secondBody := mustCallbackBody(t, payload)
	reusedNonce := fixture.callbackSignature(config, secondBody, valid.Nonce)
	if _, err = fixture.service.ApplyProviderCallback(context.Background(), config.ProviderCode, secondBody, reusedNonce); notificationCode(err) != notificationmodel.CodeCallbackReplay {
		t.Fatalf("reused nonce should conflict, got %v", err)
	}
	assertNotificationCount(t, fixture.db, &notificationmodel.NotificationProviderCallback{}, "notification_attempt_id = ?", accepted.ResourceID, 1)
}

func TestProviderRateAndCostBoundariesRollbackBlockedAttempt(t *testing.T) {
	tests := []struct {
		name        string
		configure   func(*careconfig.NotificationProvider)
		wantCode    int
		wantRateUse int64
	}{
		{
			name: "rate limit",
			configure: func(config *careconfig.NotificationProvider) {
				config.RateLimitCount = 1
				config.DailyCostLimitMinor = 1000
			},
			wantCode: notificationmodel.CodeRateLimitExceeded, wantRateUse: 1,
		},
		{
			name: "cost limit",
			configure: func(config *careconfig.NotificationProvider) {
				config.RateLimitCount = 10
				config.EstimatedCostMinor = 50
				config.DailyCostLimitMinor = 50
			},
			wantCode: notificationmodel.CodeCostLimitExceeded, wantRateUse: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newNotificationFixture(t)
			config := providerTestConfig()
			tt.configure(&config)
			gateway := &recordingProviderGateway{receipt: ProviderSubmissionReceipt{
				ProviderMessageID: "provider-message-boundary", AcceptedAt: fixture.now.Add(time.Second),
			}}
			fixture.useProviderAdapter(config, gateway)
			systemCtx := datascope.WithSystem(context.Background())
			if _, err := fixture.service.CreateInitial(systemCtx, fixture.task.ID, "boundary-first"); err != nil {
				t.Fatal(err)
			}
			other := fixture.seedTask(t, 202, 302, 99, 101)
			gateway.receipt.ProviderMessageID = "provider-message-boundary-second"
			_, err := fixture.service.CreateInitial(systemCtx, other.ID, "boundary-second")
			if notificationCode(err) != tt.wantCode {
				t.Fatalf("boundary error code = %d, want %d: %v", notificationCode(err), tt.wantCode, err)
			}
			assertNotificationCount(t, fixture.db, &notificationmodel.NotificationRequest{}, "task_id = ?", other.ID, 0)
			if gateway.calls != 1 {
				t.Fatalf("blocked attempt reached gateway: calls=%d", gateway.calls)
			}
			var rateBucket notificationmodel.NotificationQuotaBucket
			if err := fixture.db.WithContext(systemCtx).Where("bucket_kind = ?", notificationmodel.QuotaBucketRate).First(&rateBucket).Error; err != nil {
				t.Fatal(err)
			}
			if rateBucket.UsedValue != tt.wantRateUse {
				t.Fatalf("rate bucket use = %d, want %d", rateBucket.UsedValue, tt.wantRateUse)
			}
		})
	}
}

func TestProviderRetryLimitAndCommandReplayDoNotResubmit(t *testing.T) {
	fixture := newNotificationFixture(t)
	config := providerTestConfig()
	config.MaxAttemptsPerRequest = 2
	gateway := &recordingProviderGateway{receipt: ProviderSubmissionReceipt{
		ProviderMessageID: "provider-message-retry-1", AcceptedAt: fixture.now.Add(time.Second),
	}}
	fixture.useProviderAdapter(config, gateway)
	systemCtx := datascope.WithSystem(context.Background())
	firstResult, err := fixture.service.CreateInitial(systemCtx, fixture.task.ID, "retry-initial")
	if err != nil {
		t.Fatal(err)
	}
	first := fixture.applyFailedCallback(t, config, gateway.receipt.ProviderMessageID, "retry-event-1", "callback-nonce-1001")
	if first.ResourceID != firstResult.ResourceID {
		t.Fatalf("first callback changed attempt: %+v %+v", firstResult, first)
	}
	gateway.receipt = ProviderSubmissionReceipt{
		ProviderMessageID: "provider-message-retry-2", AcceptedAt: fixture.now.Add(3 * time.Second),
	}
	request := notificationreq.Resend{ExpectedVersion: first.Version, Reason: "固定流程重试边界验证"}
	second, err := fixture.service.Resend(fixture.stewardCtx, first.ResourceID, "provider-resend", request)
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != notificationmodel.AttemptStatusAccepted || gateway.calls != 2 {
		t.Fatalf("unexpected second attempt: %+v calls=%d", second, gateway.calls)
	}
	replayed, err := fixture.service.Resend(fixture.stewardCtx, first.ResourceID, "provider-resend", request)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ResourceID != second.ResourceID || gateway.calls != 2 {
		t.Fatalf("idempotent resend invoked gateway again: replay=%+v calls=%d", replayed, gateway.calls)
	}
	secondFailed := fixture.applyFailedCallback(t, config, gateway.receipt.ProviderMessageID, "retry-event-2", "callback-nonce-1002")
	_, err = fixture.service.Resend(fixture.stewardCtx, secondFailed.ResourceID, "provider-resend-third", notificationreq.Resend{
		ExpectedVersion: secondFailed.Version, Reason: "验证最大尝试次数",
	})
	if notificationCode(err) != notificationmodel.CodeRetryLimitExceeded {
		t.Fatalf("third attempt should be blocked, got %v", err)
	}
	assertNotificationCount(t, fixture.db, &notificationmodel.NotificationAttempt{}, "notification_request_id = ?", fixture.requestID(t, first.ResourceID), 2)
}

func TestProviderReadinessRequiresBusinessRoleAndNeverEnablesNetwork(t *testing.T) {
	fixture := newNotificationFixture(t)
	config := providerTestConfig()
	fixture.service.ProviderConfig = &config
	readiness, err := fixture.service.GetProviderReadiness(fixture.stewardCtx)
	if err != nil {
		t.Fatal(err)
	}
	if !readiness.CallbackEndpointEnabled || readiness.NetworkDeliveryEnabled || readiness.FormalDeliveryEnabled || len(readiness.Blockers) != 0 {
		t.Fatalf("unexpected readiness: %+v", readiness)
	}
	if _, err = fixture.service.GetProviderReadiness(context.Background()); err == nil {
		t.Fatal("missing business identity could read provider readiness")
	}
}

type recordingProviderGateway struct {
	calls   int
	last    SignedProviderSubmission
	receipt ProviderSubmissionReceipt
	err     error
}

func (g *recordingProviderGateway) Submit(_ context.Context, submission SignedProviderSubmission) (ProviderSubmissionReceipt, error) {
	g.calls++
	g.last = submission
	return g.receipt, g.err
}

func providerTestConfig() careconfig.NotificationProvider {
	return careconfig.NotificationProvider{
		Mode:         notificationmodel.ProviderModeContractTest,
		ProviderCode: "LOCAL_CONTRACT", PolicyCode: "LOCAL-CONTRACT-POLICY", PolicyVersion: 1,
		TemplateCode:                  "LOCAL-NOTICE-V1",
		RequestSigningSecret:          "request-signing-secret-32-bytes-minimum",
		CallbackVerificationSecret:    "callback-verification-secret-32-bytes",
		CallbackMaxSkewSeconds:        300,
		QualificationEvidenceReviewed: true, TemplateEvidenceReviewed: true,
		ReceiptSemanticsReviewed: true, RetryPolicyReviewed: true,
		FallbackReviewed: true, CostBoundaryReviewed: true,
		MaxAttemptsPerRequest: 3, RateLimitWindowSeconds: 60, RateLimitCount: 20,
		CostCurrency: "CNY", EstimatedCostMinor: 10, DailyCostLimitMinor: 1000,
	}
}

func (f notificationFixture) useProviderAdapter(config careconfig.NotificationProvider, gateway ProviderGateway) {
	f.service.ProviderConfig = &config
	f.service.Adapter = ProviderContractAdapter{
		Policy:               providerDescriptorFromConfig(config),
		RequestSigningSecret: []byte(config.RequestSigningSecret),
		Gateway:              gateway, Clock: f.service.Clock,
		Nonce: func() (string, error) { return "request-nonce-00000001", nil },
	}
}

func (f notificationFixture) callbackSignature(config careconfig.NotificationProvider, body []byte, nonce string) notificationreq.ProviderCallbackSignature {
	timestamp := strconv.FormatInt(f.now.Unix(), 10)
	return notificationreq.ProviderCallbackSignature{
		Timestamp: timestamp, Nonce: nonce,
		Signature: ProviderSignature([]byte(config.CallbackVerificationSecret), timestamp, nonce, body),
	}
}

func (f notificationFixture) applyFailedCallback(
	t *testing.T,
	config careconfig.NotificationProvider,
	messageID string,
	eventID string,
	nonce string,
) caseworkres.ActionResult {
	t.Helper()
	body := mustCallbackBody(t, notificationreq.ProviderCallback{
		EventID: eventID, ProviderMessageID: messageID,
		Status: notificationmodel.AttemptStatusFailed, FailureCode: notificationmodel.ProviderFailureRejected,
		OccurredAt: f.now.Add(4 * time.Second),
	})
	result, err := f.service.ApplyProviderCallback(context.Background(), config.ProviderCode, body, f.callbackSignature(config, body, nonce))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func (f notificationFixture) requestID(t *testing.T, attemptID uint) uint {
	t.Helper()
	var attempt notificationmodel.NotificationAttempt
	if err := f.db.WithContext(datascope.WithSystem(context.Background())).First(&attempt, attemptID).Error; err != nil {
		t.Fatal(err)
	}
	return attempt.NotificationRequestID
}

func mustCallbackBody(t *testing.T, payload notificationreq.ProviderCallback) []byte {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestProviderGatewayErrorLeavesAttemptAndReservationPending(t *testing.T) {
	fixture := newNotificationFixture(t)
	config := providerTestConfig()
	gateway := &recordingProviderGateway{err: errors.New("provider test gateway unavailable")}
	fixture.useProviderAdapter(config, gateway)
	_, err := fixture.service.CreateInitial(datascope.WithSystem(context.Background()), fixture.task.ID, "gateway-error")
	if err == nil {
		t.Fatal("gateway error was ignored")
	}
	var attempt notificationmodel.NotificationAttempt
	if findErr := fixture.db.WithContext(datascope.WithSystem(context.Background())).Where("task_id = ?", fixture.task.ID).First(&attempt).Error; findErr != nil {
		t.Fatal(findErr)
	}
	if attempt.Status != notificationmodel.AttemptStatusPending || attempt.Version != 1 {
		t.Fatalf("gateway error forged provider facts: %+v", attempt)
	}
	var reservation notificationmodel.NotificationDispatchReservation
	if findErr := fixture.db.WithContext(datascope.WithSystem(context.Background())).Where("notification_attempt_id = ?", attempt.ID).First(&reservation).Error; findErr != nil {
		t.Fatal(findErr)
	}
	if reservation.Status != notificationmodel.DispatchReservationReserved {
		t.Fatalf("gateway error changed reservation incorrectly: %+v", reservation)
	}
}
