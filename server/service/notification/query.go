package notification

import (
	"context"
	"errors"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	notificationmodel "github.com/flipped-aurora/gin-vue-admin/server/model/notification"
	notificationreq "github.com/flipped-aurora/gin-vue-admin/server/model/notification/request"
	notificationres "github.com/flipped-aurora/gin-vue-admin/server/model/notification/response"
	"gorm.io/gorm"
)

func (s *NotificationService) ListDeliveries(ctx context.Context, search notificationreq.DeliverySearch) ([]notificationres.NotificationAttempt, int64, error) {
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return nil, 0, normalizeAccessError(err)
	}
	status := strings.ToUpper(strings.TrimSpace(search.Status))
	if status != "" && !notificationmodel.IsAttemptStatus(status) {
		return nil, 0, notificationmodel.NewDomainError(notificationmodel.CodeInvalidArgument, "通知状态筛选值无效")
	}
	query := scopedAttemptQuery(s.db().WithContext(ctx), decision, s.now())
	if status != "" {
		query = query.Where("notification_attempts.status = ?", status)
	}
	var total int64
	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if search.Page <= 0 {
		search.Page = 1
	}
	if search.PageSize <= 0 {
		search.PageSize = 10
	}
	limit, offset := search.LimitOffset()
	var rows []attemptRow
	err = query.Select("notification_attempts.*, care_clients.display_code AS care_client_display_code, care_clients.display_name AS care_client_display_name").
		Order("notification_attempts.requested_at DESC, notification_attempts.id DESC").
		Limit(limit).Offset(offset).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	if len(rows) == 0 {
		return []notificationres.NotificationAttempt{}, total, nil
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	var events []notificationmodel.DeliveryEvent
	if err = s.db().WithContext(ctx).Where("notification_attempt_id IN ?", ids).
		Order("occurred_at ASC, id ASC").Find(&events).Error; err != nil {
		return nil, 0, err
	}
	eventsByAttempt := make(map[uint][]notificationmodel.DeliveryEvent, len(rows))
	for _, event := range events {
		eventsByAttempt[event.NotificationAttemptID] = append(eventsByAttempt[event.NotificationAttemptID], event)
	}
	result := make([]notificationres.NotificationAttempt, 0, len(rows))
	for _, row := range rows {
		result = append(result, attemptResponse(row, eventsByAttempt[row.ID]))
	}
	return result, total, nil
}

func notFoundAsForbidden(err error, message string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return notificationmodel.NewForbiddenError(message)
	}
	return err
}
