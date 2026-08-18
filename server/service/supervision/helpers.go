package supervision

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func summaryDate(value string, fallback time.Time) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		local := fallback.In(summaryLocation)
		return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, summaryLocation), nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, summaryLocation)
	if err != nil {
		return time.Time{}, supervisionmodel.NewDomainError(supervisionmodel.CodeInvalidArgument, "businessDate 必须使用 YYYY-MM-DD")
	}
	return parsed, nil
}

func summaryBounds(businessDate, now time.Time) (time.Time, time.Time, error) {
	start := time.Date(
		businessDate.In(summaryLocation).Year(), businessDate.In(summaryLocation).Month(), businessDate.In(summaryLocation).Day(),
		0, 0, 0, 0, summaryLocation,
	)
	current := now.In(summaryLocation)
	currentStart := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, summaryLocation)
	if start.After(currentStart) {
		return time.Time{}, time.Time{}, supervisionmodel.NewDomainError(supervisionmodel.CodeInvalidArgument, "不能汇总未来业务日期")
	}
	cutoff := start.AddDate(0, 0, 1)
	if start.Equal(currentStart) {
		cutoff = now
	}
	return start, cutoff, nil
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

func duplicateError(err error) bool {
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "duplicate") || strings.Contains(text, "unique")
}

func normalizeCaseWorkError(err error) error {
	var domainErr *caseworkmodel.DomainError
	if !errors.As(err, &domainErr) {
		return err
	}
	switch domainErr.Code {
	case caseworkmodel.CodeAccessScopeDenied:
		return supervisionmodel.NewForbiddenError(supervisionmodel.CodeReviewScopeDenied, "复核事项不存在或不在当前管理范围")
	case caseworkmodel.CodeCaseResponsibilityRequired:
		return supervisionmodel.NewDomainError(supervisionmodel.CodeGuidanceResultRequired, "责任医护无效或活动待办不完整")
	case caseworkmodel.CodeVersionConflict:
		return supervisionmodel.NewDomainError(supervisionmodel.CodeVersionConflict, domainErr.Message)
	default:
		return &supervisionmodel.DomainError{Code: domainErr.Code, Message: domainErr.Message, HTTPStatus: domainErr.HTTPStatus}
	}
}

func summaryLocking(db *gorm.DB) *gorm.DB {
	if db.Dialector.Name() == "mysql" || db.Dialector.Name() == "postgres" {
		return db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return db
}
