package notification

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	notificationmodel "github.com/flipped-aurora/gin-vue-admin/server/model/notification"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var notificationLocation = loadNotificationLocation()

func loadNotificationLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err == nil {
		return location
	}
	return time.FixedZone("Asia/Shanghai", 8*60*60)
}

func reserveDispatch(tx *gorm.DB, attempt notificationmodel.NotificationAttempt, descriptor AdapterDescriptor, now time.Time) error {
	if !descriptor.requiresReservation() {
		return nil
	}
	if err := descriptor.validate(); err != nil {
		return err
	}
	if attempt.AttemptNo > descriptor.MaxAttemptsPerRequest {
		return notificationmodel.NewDomainError(notificationmodel.CodeRetryLimitExceeded, "通知尝试已达到当前策略上限")
	}
	var existing notificationmodel.NotificationDispatchReservation
	err := tx.Where("notification_attempt_id = ?", attempt.ID).First(&existing).Error
	if err == nil {
		if existing.ProviderCode != descriptor.ProviderCode || existing.PolicyCode != descriptor.PolicyCode ||
			existing.PolicyVersion != descriptor.PolicyVersion || existing.EstimatedCostMinor != descriptor.EstimatedCostMinor {
			return notificationmodel.NewDomainError(notificationmodel.CodeProviderConfigInvalid, "通知发送预留与策略快照不一致")
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	windowSeconds := int64(descriptor.RateLimitWindowSeconds)
	windowStart := time.Unix(now.Unix()/windowSeconds*windowSeconds, 0).In(now.Location())
	windowEnd := windowStart.Add(time.Duration(descriptor.RateLimitWindowSeconds) * time.Second)
	rateBucketID, err := consumeQuotaBucket(tx, quotaBucketInput{
		Descriptor: descriptor,
		Kind:       notificationmodel.QuotaBucketRate,
		Key:        strconv.FormatInt(windowStart.Unix(), 10),
		Start:      windowStart,
		End:        windowEnd,
		Amount:     1,
		Limit:      descriptor.RateLimitCount,
		ErrorCode:  notificationmodel.CodeRateLimitExceeded,
		ErrorText:  "通知发送达到当前策略限流上限",
	})
	if err != nil {
		return err
	}
	businessNow := now.In(notificationLocation)
	dayStart := time.Date(businessNow.Year(), businessNow.Month(), businessNow.Day(), 0, 0, 0, 0, notificationLocation)
	costBucketID, err := consumeQuotaBucket(tx, quotaBucketInput{
		Descriptor: descriptor,
		Kind:       notificationmodel.QuotaBucketCost,
		Key:        dayStart.Format("2006-01-02"),
		Start:      dayStart,
		End:        dayStart.AddDate(0, 0, 1),
		Amount:     descriptor.EstimatedCostMinor,
		Limit:      descriptor.DailyCostLimitMinor,
		ErrorCode:  notificationmodel.CodeCostLimitExceeded,
		ErrorText:  "通知发送达到当前策略日费用上限",
	})
	if err != nil {
		return err
	}
	reservation := notificationmodel.NotificationDispatchReservation{
		NotificationRequestID: attempt.NotificationRequestID,
		NotificationAttemptID: attempt.ID,
		ProviderCode:          descriptor.ProviderCode,
		PolicyCode:            descriptor.PolicyCode,
		PolicyVersion:         descriptor.PolicyVersion,
		TemplateCode:          descriptor.TemplateCode,
		RateBucketID:          rateBucketID,
		CostBucketID:          costBucketID,
		EstimatedCostMinor:    descriptor.EstimatedCostMinor,
		CostCurrency:          descriptor.CostCurrency,
		Status:                notificationmodel.DispatchReservationReserved,
		ReservedAt:            now,
		Synthetic:             attempt.Synthetic,
		DeptId:                attempt.DeptId,
		CreatedBy:             attempt.CreatedBy,
	}
	if err = tx.Create(&reservation).Error; err != nil {
		if duplicateError(err) {
			return notificationmodel.NewDomainError(notificationmodel.CodeIdempotencyConflict, "通知发送预留发生并发冲突")
		}
		return err
	}
	return nil
}

type quotaBucketInput struct {
	Descriptor AdapterDescriptor
	Kind       string
	Key        string
	Start      time.Time
	End        time.Time
	Amount     int64
	Limit      int64
	ErrorCode  int
	ErrorText  string
}

func consumeQuotaBucket(tx *gorm.DB, input quotaBucketInput) (uint, error) {
	bucket := notificationmodel.NotificationQuotaBucket{
		ProviderCode:  input.Descriptor.ProviderCode,
		PolicyCode:    input.Descriptor.PolicyCode,
		PolicyVersion: input.Descriptor.PolicyVersion,
		BucketKind:    input.Kind,
		BucketKey:     input.Key,
		WindowStart:   input.Start,
		WindowEnd:     input.End,
		LimitValue:    input.Limit,
		Version:       1,
	}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&bucket).Error; err != nil {
		return 0, err
	}
	if err := locking(tx).Where(
		"provider_code = ? AND policy_code = ? AND policy_version = ? AND bucket_kind = ? AND bucket_key = ?",
		input.Descriptor.ProviderCode, input.Descriptor.PolicyCode, input.Descriptor.PolicyVersion, input.Kind, input.Key,
	).First(&bucket).Error; err != nil {
		return 0, err
	}
	if bucket.LimitValue != input.Limit || !bucket.WindowStart.Equal(input.Start) || !bucket.WindowEnd.Equal(input.End) {
		return 0, notificationmodel.NewDomainError(notificationmodel.CodeProviderConfigInvalid, "通知额度桶与不可变策略版本不一致")
	}
	if input.Amount <= 0 || input.Limit < input.Amount {
		return 0, notificationmodel.NewDomainError(input.ErrorCode, input.ErrorText)
	}
	updated := tx.Model(&notificationmodel.NotificationQuotaBucket{}).
		Where("id = ? AND used_value <= ?", bucket.ID, input.Limit-input.Amount).
		Updates(map[string]any{
			"used_value": gorm.Expr("used_value + ?", input.Amount),
			"version":    gorm.Expr("version + 1"),
		})
	if updated.Error != nil {
		return 0, updated.Error
	}
	if updated.RowsAffected != 1 {
		return 0, notificationmodel.NewDomainError(input.ErrorCode, input.ErrorText)
	}
	return bucket.ID, nil
}

func updateReservationStatus(tx *gorm.DB, attemptID uint, deliveryStatus string) error {
	status := ""
	switch deliveryStatus {
	case notificationmodel.AttemptStatusSubmittedToProvider:
		status = notificationmodel.DispatchReservationSubmitted
	case notificationmodel.AttemptStatusDelivered,
		notificationmodel.AttemptStatusFailed,
		notificationmodel.AttemptStatusUnknown:
		status = notificationmodel.DispatchReservationFinalized
	default:
		return nil
	}
	result := tx.Model(&notificationmodel.NotificationDispatchReservation{}).
		Where("notification_attempt_id = ?", attemptID).
		Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("update notification dispatch reservation: %w", result.Error)
	}
	return nil
}
