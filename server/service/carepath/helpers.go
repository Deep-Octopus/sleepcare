package carepath

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	platformoutbox "github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	pathres "github.com/flipped-aurora/gin-vue-admin/server/model/carepath/response"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type loadedTemplate struct {
	Path         pathmodel.PathDefinitionVersion
	Template     pathmodel.PlanTemplateVersion
	Tasks        []pathmodel.PlanTaskDefinition
	Dependencies []pathmodel.PlanTaskDependency
}

const (
	p1PathCode                 = "OSA"
	p1PathVersion              = "1.0.0-synthetic"
	p1PlanTemplateCode         = "SYN-OSA-D1-D5"
	p1PlanTemplateVersion      = "1.0.0-synthetic"
	p1D1QuestionnaireVersionID = uint(9401)
	p1D1AttentionRuleVersionID = uint(9501)
)

func (s *CarePathService) loadTemplate(ctx context.Context, db *gorm.DB, id uint) (loadedTemplate, error) {
	var value loadedTemplate
	if id == 0 {
		return value, pathmodel.NewDomainError(pathmodel.CodeInvalidArgument, "计划模板版本ID必填")
	}
	if err := db.WithContext(ctx).Where("id = ?", id).First(&value.Template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return value, pathmodel.NewDomainError(pathmodel.CodeResourceNotFound, "计划模板版本不存在")
		}
		return value, err
	}
	if err := db.WithContext(ctx).Where("id = ?", value.Template.PathDefinitionVersionID).First(&value.Path).Error; err != nil {
		return value, err
	}
	if err := db.WithContext(ctx).Where("plan_template_version_id = ?", id).Order("sort ASC, id ASC").Find(&value.Tasks).Error; err != nil {
		return value, err
	}
	if err := db.WithContext(ctx).Where("plan_template_version_id = ?", id).Order("id ASC").Find(&value.Dependencies).Error; err != nil {
		return value, err
	}
	return value, nil
}

func (s *CarePathService) validateTemplate(ctx context.Context, value loadedTemplate) error {
	if value.Path.Status == pathmodel.LifecycleDisabled || value.Template.Status == pathmodel.LifecycleDisabled {
		return pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "路径或计划模板版本已禁用")
	}
	if value.Path.Status != pathmodel.LifecyclePublished || value.Template.Status != pathmodel.LifecyclePublished ||
		value.Path.PublishedAt == nil || value.Template.PublishedAt == nil ||
		value.Path.ReviewedAt == nil || value.Template.ReviewedAt == nil ||
		value.Path.ReviewedBy == 0 || value.Template.ReviewedBy == 0 {
		return pathmodel.NewDomainError(pathmodel.CodeContentNotPublished, "路径或计划模板版本未完成发布复核")
	}
	if value.Template.UsageScope != pathmodel.UsageScopeTestOnly || value.Path.UsageScope != pathmodel.UsageScopeTestOnly ||
		!value.Template.Synthetic || !value.Path.Synthetic || value.Template.ProductionEnabled || value.Path.ProductionEnabled ||
		value.Template.ReviewType != pathmodel.ReviewTypeEngineering || value.Path.ReviewType != pathmodel.ReviewTypeEngineering ||
		!s.syntheticFixturesEnabled() {
		return pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "P1-04 只允许启用受控合成计划定义")
	}
	if value.Template.PauseStrategy != pathmodel.PauseStrategyKeepWindows || value.Template.LateSubmissionPolicy != pathmodel.LateSubmissionDeny {
		return pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "计划模板启用了 P1-04 尚未批准的调度策略")
	}
	if value.Path.Code != p1PathCode || value.Path.Version != p1PathVersion ||
		value.Template.Code != p1PlanTemplateCode || value.Template.Version != p1PlanTemplateVersion ||
		value.Template.AnchorDefinition != pathmodel.AnchorFirstValidSyntheticDeviceUse ||
		value.Template.DefinitionSchemaVersion != "v1" {
		return pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "计划模板不符合 P1-04 的固定 OSA 版本、锚点或定义版本约束")
	}
	pathHash, err := pathmodel.HashDefinition(pathmodel.PathDefinitionDocument{
		Code: value.Path.Code, Version: value.Path.Version, Title: value.Path.Title,
		Purpose: value.Path.Purpose, UsageScope: value.Path.UsageScope,
		Synthetic: value.Path.Synthetic, ProductionEnabled: value.Path.ProductionEnabled,
	})
	if err != nil {
		return err
	}
	if pathHash != value.Path.DefinitionHash {
		return pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "路径版本定义哈希校验失败")
	}
	if len(value.Tasks) != 5 {
		return pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "合成 OSA 计划必须严格包含 D1–D5 五项任务")
	}
	if len(value.Dependencies) != 0 {
		return pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "P1-04 合成计划不启用任务依赖")
	}
	document, err := definitionDocument(value)
	if err != nil {
		return err
	}
	hash, err := pathmodel.HashDefinition(document)
	if err != nil {
		return err
	}
	if hash != value.Template.DefinitionHash {
		return pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "计划模板定义哈希校验失败")
	}
	for index, task := range value.Tasks {
		ids, decodeErr := decodeRuleIDs(task.BoundRuleVersionIDsJSON)
		if decodeErr != nil {
			return decodeErr
		}
		expectedDay := fmt.Sprintf("D%d", index+1)
		expectedOpen := int64(index) * 24 * 60 * 60
		expectedDue := expectedOpen + 11*60*60
		if task.DayCode != expectedDay || task.Sort != index+1 || task.OpenOffsetSeconds != expectedOpen ||
			task.DueOffsetSeconds != expectedDue || task.ExpiresOffsetSeconds != nil {
			return pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "P1-04 合成计划的 D1–D5 日序或时间窗无效")
		}
		if task.ExecutionRole != pathmodel.ExecutionRoleCareClient || task.NotificationPolicy != pathmodel.NotificationPolicyDisabled {
			return pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "P1-04 运行夹具只允许 CARE_CLIENT 任务且通知必须禁用")
		}
		if index == 0 {
			if task.QuestionnaireVersionID == nil || *task.QuestionnaireVersionID != p1D1QuestionnaireVersionID ||
				len(ids) != 1 || ids[0] != p1D1AttentionRuleVersionID ||
				!task.ReviewRequired || task.ReviewRole != pathmodel.ExecutionRoleClinician {
				return pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "P1-04 D1 必须冻结问卷 9401、规则 9501 并由医护复核")
			}
		} else if task.QuestionnaireVersionID != nil || len(ids) != 0 || task.ReviewRequired || task.ReviewRole != "" {
			return pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "P1-04 D2–D5 不得绑定问卷、规则或复核动作")
		}
		questionnaireID := uint(0)
		if task.QuestionnaireVersionID != nil {
			questionnaireID = *task.QuestionnaireVersionID
		}
		if bindErr := s.bindingValidator().ValidateFrozenBinding(ctx, questionnaireID, ids, true); bindErr != nil {
			return normalizeBindingError(bindErr)
		}
	}
	return nil
}

func definitionDocument(value loadedTemplate) (pathmodel.PlanDefinitionDocument, error) {
	tasks := make([]pathmodel.TaskDefinitionDocument, 0, len(value.Tasks))
	for _, task := range value.Tasks {
		ids, err := decodeRuleIDs(task.BoundRuleVersionIDsJSON)
		if err != nil {
			return pathmodel.PlanDefinitionDocument{}, err
		}
		tasks = append(tasks, pathmodel.TaskDefinitionDocument{
			DayCode: task.DayCode, Title: task.Title, Sort: task.Sort, ExecutionRole: task.ExecutionRole,
			OpenOffsetSeconds: task.OpenOffsetSeconds, DueOffsetSeconds: task.DueOffsetSeconds,
			ExpiresOffsetSeconds: task.ExpiresOffsetSeconds, QuestionnaireVersionID: task.QuestionnaireVersionID,
			BoundRuleVersionIDs: ids, ReviewRequired: task.ReviewRequired, ReviewRole: task.ReviewRole,
			NotificationPolicy: task.NotificationPolicy,
		})
	}
	return pathmodel.PlanDefinitionDocument{
		PathCode: value.Path.Code, Code: value.Template.Code, Version: value.Template.Version,
		Title: value.Template.Title, Purpose: value.Template.Purpose, UsageScope: value.Template.UsageScope,
		Synthetic: value.Template.Synthetic, ProductionEnabled: value.Template.ProductionEnabled,
		AnchorDefinition: value.Template.AnchorDefinition, LateSubmissionPolicy: value.Template.LateSubmissionPolicy,
		PauseStrategy: value.Template.PauseStrategy, DefinitionSchemaVersion: value.Template.DefinitionSchemaVersion,
		Tasks: tasks,
	}, nil
}

func decodeRuleIDs(value []byte) ([]uint, error) {
	result := []uint{}
	if len(value) == 0 {
		return result, pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "任务冻结规则列表缺失")
	}
	if err := json.Unmarshal(value, &result); err != nil {
		return nil, pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "任务冻结规则列表无效")
	}
	return result, nil
}

func normalizeBindingError(err error) error {
	var domainErr *qmodel.DomainError
	if errors.As(err, &domainErr) {
		return &pathmodel.DomainError{Code: domainErr.Code, Message: domainErr.Message, HTTPStatus: domainErr.HTTPStatus}
	}
	return err
}

func (s *CarePathService) decisionAndClient(ctx context.Context, clientID uint, requireWrite bool) (*accesspolicy.CareClientDecision, caremodel.CareClient, error) {
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return nil, caremodel.CareClient{}, normalizeCareClientError(err)
	}
	if requireWrite && decision.RoleType != caremodel.AuthorityRoleCareSteward && decision.RoleType != caremodel.AuthorityRoleClinician {
		return nil, caremodel.CareClient{}, pathmodel.NewForbiddenError("仅当前责任管家或医护可操作计划")
	}
	var client caremodel.CareClient
	err = decision.Scope(s.db().WithContext(ctx).Model(&caremodel.CareClient{}), s.now()).
		Where("care_clients.id = ?", clientID).First(&client).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, client, pathmodel.NewForbiddenError("康养用户不存在或不在当前责任范围")
	}
	if err != nil {
		return nil, client, err
	}
	if client.Status != caremodel.ClientStatusActive || !client.Synthetic {
		return nil, client, pathmodel.NewDomainError(pathmodel.CodeCareClientUnavailable, "P1-04 只允许为活动的合成康养用户操作计划")
	}
	return decision, client, nil
}

func normalizeCareClientError(err error) error {
	var domainErr *caremodel.DomainError
	if errors.As(err, &domainErr) {
		return &pathmodel.DomainError{Code: domainErr.Code, Message: domainErr.Message, HTTPStatus: domainErr.HTTPStatus}
	}
	return err
}

func withDepartment(ctx context.Context, deptID uint) context.Context {
	identity, ok := datascope.FromContext(ctx)
	if !ok || identity == nil {
		return ctx
	}
	copyIdentity := *identity
	copyIdentity.DeptID = deptID
	return datascope.WithIdentity(ctx, &copyIdentity)
}

func actorID(ctx context.Context) uint {
	if identity, ok := datascope.FromContext(ctx); ok && identity != nil {
		return identity.UserID
	}
	return 0
}

func sourceForRole(role string) string {
	switch role {
	case caremodel.AuthorityRoleCareSteward:
		return pathmodel.EventSourceCareSteward
	case caremodel.AuthorityRoleClinician:
		return pathmodel.EventSourceClinician
	case caremodel.AuthorityRoleSupervisor:
		return pathmodel.EventSourceSupervisor
	default:
		return pathmodel.EventSourceSystem
	}
}

func lockQuery(db *gorm.DB) *gorm.DB {
	if db.Dialector.Name() == "mysql" || db.Dialector.Name() == "postgres" {
		return db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return db
}

func duplicateError(err error) bool {
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "duplicate") || strings.Contains(text, "unique")
}

func requestHash(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func runIdempotent[T any](s *CarePathService, ctx context.Context, operation, key string, request any, fn func(*gorm.DB) (T, error)) (T, error) {
	var result T
	key = strings.TrimSpace(key)
	actor := actorID(ctx)
	if key == "" || len(key) > 128 || actor == 0 {
		return result, pathmodel.NewDomainError(pathmodel.CodeInvalidArgument, "Idempotency-Key 必填且不超过 128 字符，并要求有效操作人")
	}
	hash, err := requestHash(request)
	if err != nil {
		return result, err
	}
	err = s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var receipt pathmodel.CommandReceipt
		receiptErr := tx.Where("actor_id = ? AND operation = ? AND idempotency_key = ?", actor, operation, key).First(&receipt).Error
		if receiptErr == nil {
			if receipt.RequestHash != hash {
				return pathmodel.NewDomainError(pathmodel.CodeIdempotencyConflict, "相同 Idempotency-Key 对应了不同请求")
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
		receipt = pathmodel.CommandReceipt{ActorID: actor, Operation: operation, IdempotencyKey: key, RequestHash: hash, ResultJSON: string(encoded)}
		if err = tx.Create(&receipt).Error; err != nil {
			if duplicateError(err) {
				return pathmodel.NewDomainError(pathmodel.CodeIdempotencyConflict, "幂等请求发生并发冲突，请重试")
			}
			return err
		}
		return nil
	})
	return result, err
}

func taskDefinitionResponse(task pathmodel.PlanTaskDefinition) (pathres.PlanTaskDefinition, error) {
	ids, err := decodeRuleIDs(task.BoundRuleVersionIDsJSON)
	if err != nil {
		return pathres.PlanTaskDefinition{}, err
	}
	return pathres.PlanTaskDefinition{
		ID: task.ID, DayCode: task.DayCode, Title: task.Title, Sort: task.Sort,
		ExecutionRole: task.ExecutionRole, OpenOffsetSeconds: task.OpenOffsetSeconds,
		DueOffsetSeconds: task.DueOffsetSeconds, ExpiresOffsetSeconds: task.ExpiresOffsetSeconds,
		QuestionnaireVersionID: task.QuestionnaireVersionID, BoundRuleVersionIDs: ids,
		ReviewRequired: task.ReviewRequired, ReviewRole: task.ReviewRole,
		NotificationPolicy: task.NotificationPolicy,
	}, nil
}

func appendDomainEvent(tx *gorm.DB, event pathmodel.CarePathEvent) error {
	if event.EventID == "" {
		event.EventID = uuid.NewString()
	}
	return tx.Create(&event).Error
}

func appendOutbox(tx *gorm.DB, eventType, aggregateType string, aggregateID uint, payload any, occurredAt time.Time, causationID string, deptID uint) error {
	return platformoutbox.Append(tx, platformoutbox.AppendInput{
		EventType: eventType, AggregateType: aggregateType, AggregateID: aggregateID,
		Payload: payload, OccurredAt: occurredAt, CausationID: causationID, Synthetic: true,
		DeptID: deptID,
	})
}

func operation(resource string, id uint) string {
	return fmt.Sprintf("%s:%d", resource, id)
}
