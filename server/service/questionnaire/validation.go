package questionnaire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
)

type validationRules struct {
	MinLength *int     `json:"minLength"`
	MaxLength *int     `json:"maxLength"`
	Min       *float64 `json:"min"`
	Max       *float64 `json:"max"`
}

func canonicalAnswers(answers map[string]any) (map[string]any, []byte, error) {
	if answers == nil {
		answers = map[string]any{}
	}
	raw, err := json.Marshal(answers)
	if err != nil {
		return nil, nil, qmodel.NewDomainError(qmodel.CodeSubmissionInvalid, "答案不是有效的 JSON 结构")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	normalized := map[string]any{}
	if err = decoder.Decode(&normalized); err != nil {
		return nil, nil, qmodel.NewDomainError(qmodel.CodeSubmissionInvalid, "答案不是有效的 JSON 对象")
	}
	canonical, err := json.Marshal(normalized)
	if err != nil {
		return nil, nil, err
	}
	return normalized, canonical, nil
}

func validateAnswers(questions []qmodel.QuestionnaireQuestion, options []qmodel.QuestionnaireOption, answers map[string]any) error {
	byCode := make(map[string]qmodel.QuestionnaireQuestion, len(questions))
	optionsByQuestion := make(map[uint]map[string]struct{})
	for _, question := range questions {
		if !qmodel.IsQuestionType(question.Type) {
			return qmodel.NewDomainError(qmodel.CodeSubmissionInvalid, fmt.Sprintf("问题 %s 使用了不支持的题型", question.Code))
		}
		byCode[question.Code] = question
	}
	for _, option := range options {
		if optionsByQuestion[option.QuestionID] == nil {
			optionsByQuestion[option.QuestionID] = map[string]struct{}{}
		}
		optionsByQuestion[option.QuestionID][option.Code] = struct{}{}
	}
	for code := range answers {
		if _, ok := byCode[code]; !ok {
			return qmodel.NewDomainError(qmodel.CodeSubmissionInvalid, fmt.Sprintf("答案包含未知问题 %s", code))
		}
	}
	for _, question := range questions {
		value, exists := answers[question.Code]
		if !exists || isEmptyAnswer(value) {
			if question.Required {
				return qmodel.NewDomainError(qmodel.CodeSubmissionInvalid, fmt.Sprintf("问题 %s 必答", question.Code))
			}
			continue
		}
		if err := validateAnswerValue(question, optionsByQuestion[question.ID], value); err != nil {
			return err
		}
	}
	return nil
}

func validateAnswerValue(question qmodel.QuestionnaireQuestion, allowed map[string]struct{}, value any) error {
	invalid := func(reason string) error {
		return qmodel.NewDomainError(qmodel.CodeSubmissionInvalid, fmt.Sprintf("问题 %s %s", question.Code, reason))
	}
	switch question.Type {
	case qmodel.QuestionTypeSingleChoice:
		choice, ok := value.(string)
		if !ok {
			return invalid("必须是单个选项编码")
		}
		if _, ok = allowed[choice]; !ok {
			return invalid("包含无效选项")
		}
	case qmodel.QuestionTypeMultipleChoice:
		items, ok := value.([]any)
		if !ok {
			return invalid("必须是选项编码数组")
		}
		seen := map[string]struct{}{}
		for _, item := range items {
			choice, ok := item.(string)
			if !ok {
				return invalid("包含非字符串选项")
			}
			if _, ok = allowed[choice]; !ok {
				return invalid("包含无效选项")
			}
			if _, duplicate := seen[choice]; duplicate {
				return invalid("包含重复选项")
			}
			seen[choice] = struct{}{}
		}
	case qmodel.QuestionTypeText:
		text, ok := value.(string)
		if !ok {
			return invalid("必须是文本")
		}
		rules, err := decodeValidation(question)
		if err != nil {
			return err
		}
		length := len([]rune(text))
		if rules.MinLength != nil && length < *rules.MinLength {
			return invalid("短于最小长度")
		}
		if rules.MaxLength != nil && length > *rules.MaxLength {
			return invalid("超过最大长度")
		}
	case qmodel.QuestionTypeNumber:
		number, ok := numberValue(value)
		if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
			return invalid("必须是有限数字")
		}
		rules, err := decodeValidation(question)
		if err != nil {
			return err
		}
		if rules.Min != nil && number < *rules.Min {
			return invalid("小于最小值")
		}
		if rules.Max != nil && number > *rules.Max {
			return invalid("大于最大值")
		}
	case qmodel.QuestionTypeDate:
		date, ok := value.(string)
		if !ok {
			return invalid("必须是 YYYY-MM-DD 日期")
		}
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return invalid("必须是 YYYY-MM-DD 日期")
		}
	case qmodel.QuestionTypeBoolean:
		if _, ok := value.(bool); !ok {
			return invalid("必须是布尔值")
		}
	}
	return nil
}

func decodeValidation(question qmodel.QuestionnaireQuestion) (validationRules, error) {
	rules := validationRules{}
	if len(question.ValidationJSON) == 0 {
		return rules, nil
	}
	if err := json.Unmarshal(question.ValidationJSON, &rules); err != nil {
		return rules, qmodel.NewDomainError(qmodel.CodeSubmissionInvalid, fmt.Sprintf("问题 %s 校验定义无效", question.Code))
	}
	return rules, nil
}

func isEmptyAnswer(value any) bool {
	if value == nil {
		return true
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	}
	return false
}

func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case json.Number:
		result, err := typed.Float64()
		return result, err == nil
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	default:
		return 0, false
	}
}

func jsonEqual(left, right any) bool {
	leftRaw, leftErr := json.Marshal(left)
	rightRaw, rightErr := json.Marshal(right)
	if leftErr == nil && rightErr == nil {
		var leftValue, rightValue any
		leftDecoder := json.NewDecoder(bytes.NewReader(leftRaw))
		rightDecoder := json.NewDecoder(bytes.NewReader(rightRaw))
		leftDecoder.UseNumber()
		rightDecoder.UseNumber()
		if leftDecoder.Decode(&leftValue) == nil && rightDecoder.Decode(&rightValue) == nil {
			return reflect.DeepEqual(leftValue, rightValue)
		}
	}
	return false
}
