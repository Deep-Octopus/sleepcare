package carepath

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	pathreq "github.com/flipped-aurora/gin-vue-admin/server/model/carepath/request"
	pathres "github.com/flipped-aurora/gin-vue-admin/server/model/carepath/response"
	"gorm.io/gorm"
)

func (s *CarePathService) ListClientPlans(ctx context.Context, careClientID uint) ([]pathres.PlanInstanceSummary, error) {
	_, client, err := s.decisionAndClient(ctx, careClientID, false)
	if err != nil {
		return nil, err
	}
	var plans []pathmodel.PlanInstance
	if err = s.db().WithContext(ctx).Where("care_client_id = ? AND synthetic = ?", careClientID, true).Order("id DESC").Find(&plans).Error; err != nil {
		return nil, err
	}
	result := make([]pathres.PlanInstanceSummary, 0, len(plans))
	for _, plan := range plans {
		item, buildErr := s.planSummary(ctx, plan, client)
		if buildErr != nil {
			return nil, buildErr
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *CarePathService) planSummary(ctx context.Context, plan pathmodel.PlanInstance, client caremodel.CareClient) (pathres.PlanInstanceSummary, error) {
	var template pathmodel.PlanTemplateVersion
	if err := s.db().WithContext(ctx).Where("id = ? AND synthetic = ?", plan.PlanTemplateVersionID, true).First(&template).Error; err != nil {
		return pathres.PlanInstanceSummary{}, err
	}
	var enrollment pathmodel.Enrollment
	if err := s.db().WithContext(ctx).Where("id = ? AND synthetic = ?", plan.EnrollmentID, true).First(&enrollment).Error; err != nil {
		return pathres.PlanInstanceSummary{}, err
	}
	var tasks []pathmodel.TaskInstance
	if err := s.db().WithContext(ctx).Where("plan_instance_id = ? AND synthetic = ?", plan.ID, true).Order("sort ASC, id ASC").Find(&tasks).Error; err != nil {
		return pathres.PlanInstanceSummary{}, err
	}
	items := make([]pathres.TaskSummary, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, s.taskSummary(task, plan, client))
	}
	timeline, err := s.planTimeline(ctx, plan.ID)
	if err != nil {
		return pathres.PlanInstanceSummary{}, err
	}
	return pathres.PlanInstanceSummary{
		ID: plan.ID, EnrollmentID: plan.EnrollmentID, CareClientID: plan.CareClientID,
		PlanTemplateVersionID: plan.PlanTemplateVersionID, TemplateTitle: template.Title,
		PathCode: enrollment.PathCode, AnchorAt: plan.AnchorAt, Status: plan.Status,
		PauseStrategy: plan.PauseStrategy, PausedAt: plan.PausedAt, Version: plan.Version,
		Synthetic: plan.Synthetic, Tasks: items, Timeline: timeline,
	}, nil
}

func (s *CarePathService) ListTasks(ctx context.Context, req pathreq.TaskSearch) ([]pathres.TaskSummary, int64, error) {
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return nil, 0, normalizeCareClientError(err)
	}
	clientQuery := decision.Scope(s.db().WithContext(ctx).Model(&caremodel.CareClient{}), s.now()).
		Where("care_clients.synthetic = ? AND care_clients.status = ?", true, caremodel.ClientStatusActive)
	if req.CareClientID != 0 {
		clientQuery = clientQuery.Where("care_clients.id = ?", req.CareClientID)
	}
	var clients []caremodel.CareClient
	if err = clientQuery.Order("care_clients.id ASC").Find(&clients).Error; err != nil {
		return nil, 0, err
	}
	if len(clients) == 0 {
		return []pathres.TaskSummary{}, 0, nil
	}
	clientIDs := make([]uint, 0, len(clients))
	clientByID := make(map[uint]caremodel.CareClient, len(clients))
	for _, client := range clients {
		clientIDs = append(clientIDs, client.ID)
		clientByID[client.ID] = client
	}
	query := s.db().WithContext(ctx).Model(&pathmodel.TaskInstance{}).
		Where("care_client_id IN ? AND synthetic = ?", clientIDs, true).
		Where("plan_instance_id IN (SELECT id FROM care_plan_instances WHERE synthetic = ? AND deleted_at IS NULL)", true)
	if req.PlanInstanceID != 0 {
		query = query.Where("plan_instance_id = ?", req.PlanInstanceID)
	}
	if req.ExecutionStatus != "" {
		query = query.Where("execution_status = ?", req.ExecutionStatus)
	}
	if req.DayCode != "" {
		query = query.Where("day_code = ?", strings.TrimSpace(req.DayCode))
	}
	query = applyTimingFilter(query, req.TimingStatus, s.now())
	var total int64
	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit, offset := req.LimitOffset()
	if limit == 0 {
		limit = 10
	}
	var tasks []pathmodel.TaskInstance
	if err = query.Order("open_at ASC, sort ASC, id ASC").Limit(limit).Offset(offset).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	planIDs := make([]uint, 0, len(tasks))
	for _, task := range tasks {
		planIDs = append(planIDs, task.PlanInstanceID)
	}
	var plans []pathmodel.PlanInstance
	if err = s.db().WithContext(ctx).Where("id IN ? AND synthetic = ?", nonEmptyIDs(planIDs), true).Find(&plans).Error; err != nil {
		return nil, 0, err
	}
	planByID := make(map[uint]pathmodel.PlanInstance, len(plans))
	for _, plan := range plans {
		planByID[plan.ID] = plan
	}
	items := make([]pathres.TaskSummary, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, s.taskSummary(task, planByID[task.PlanInstanceID], clientByID[task.CareClientID]))
	}
	return items, total, nil
}

func applyTimingFilter(db *gorm.DB, status string, now time.Time) *gorm.DB {
	switch status {
	case pathmodel.TimingNotOpen:
		return db.Where("open_at > ?", now)
	case pathmodel.TimingWithinWindow:
		return db.Where("open_at <= ? AND due_at > ?", now, now).
			Where("expires_at IS NULL OR expires_at > ?", now)
	case pathmodel.TimingOverdue:
		return db.Where("open_at <= ? AND due_at <= ?", now, now).
			Where("late_submission_policy <> ?", pathmodel.LateSubmissionDeny).
			Where("expires_at IS NULL OR expires_at > ?", now)
	case pathmodel.TimingExpired:
		return db.Where("(expires_at IS NOT NULL AND expires_at <= ?) OR (late_submission_policy = ? AND due_at <= ?)", now, pathmodel.LateSubmissionDeny, now)
	default:
		return db
	}
}

func (s *CarePathService) GetTask(ctx context.Context, id uint) (pathres.TaskDetail, error) {
	if id == 0 {
		return pathres.TaskDetail{}, pathmodel.NewDomainError(pathmodel.CodeInvalidArgument, "任务ID无效")
	}
	var task pathmodel.TaskInstance
	if err := s.db().WithContext(ctx).Where("id = ? AND synthetic = ?", id, true).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pathres.TaskDetail{}, pathmodel.NewForbiddenError("任务不存在或不在当前数据范围")
		}
		return pathres.TaskDetail{}, err
	}
	_, client, err := s.decisionAndClient(ctx, task.CareClientID, false)
	if err != nil {
		return pathres.TaskDetail{}, err
	}
	var plan pathmodel.PlanInstance
	if err = s.db().WithContext(ctx).Where("id = ? AND synthetic = ?", task.PlanInstanceID, true).First(&plan).Error; err != nil {
		return pathres.TaskDetail{}, err
	}
	ruleIDs, err := decodeRuleIDs(task.BoundRuleVersionIDsJSON)
	if err != nil {
		return pathres.TaskDetail{}, err
	}
	timeline, err := s.taskTimeline(ctx, plan.ID, task.ID)
	if err != nil {
		return pathres.TaskDetail{}, err
	}
	return pathres.TaskDetail{
		TaskSummary: s.taskSummary(task, plan, client), QuestionnaireVersionID: task.QuestionnaireVersionID,
		RuleVersionIDs: ruleIDs, LateSubmissionPolicy: task.LateSubmissionPolicy,
		NotificationPolicy: task.NotificationPolicy, ReviewRole: task.ReviewRole, Timeline: timeline,
	}, nil
}

func (s *CarePathService) taskSummary(task pathmodel.TaskInstance, plan pathmodel.PlanInstance, client caremodel.CareClient) pathres.TaskSummary {
	return pathres.TaskSummary{
		ID: task.ID, PlanInstanceID: task.PlanInstanceID, CareClientID: task.CareClientID,
		CareClientDisplayCode: client.DisplayCode, CareClientDisplayName: client.DisplayName,
		PlanStatus: plan.Status, Title: task.Title, DayCode: task.DayCode,
		ExecutionRole: task.ExecutionRole, ExecutionStatus: task.ExecutionStatus,
		TimingStatus: task.TimingStatus(s.now()), ReviewStatus: task.ReviewStatus,
		OpenAt: task.OpenAt, DueAt: task.DueAt, ExpiresAt: task.ExpiresAt,
		SubmittedAt: task.SubmittedAt, Version: task.Version,
	}
}

func (s *CarePathService) planTimeline(ctx context.Context, planID uint) ([]pathres.TimelineEvent, error) {
	var events []pathmodel.CarePathEvent
	if err := s.db().WithContext(ctx).Where("plan_instance_id = ? AND synthetic = ?", planID, true).Order("occurred_at ASC, id ASC").Find(&events).Error; err != nil {
		return nil, err
	}
	return timelineResponses(events), nil
}

func (s *CarePathService) taskTimeline(ctx context.Context, planID, taskID uint) ([]pathres.TimelineEvent, error) {
	var events []pathmodel.CarePathEvent
	if err := s.db().WithContext(ctx).
		Where("plan_instance_id = ? AND synthetic = ? AND (task_instance_id IS NULL OR task_instance_id = ?)", planID, true, taskID).
		Order("occurred_at ASC, id ASC").Find(&events).Error; err != nil {
		return nil, err
	}
	return timelineResponses(events), nil
}

func timelineResponses(events []pathmodel.CarePathEvent) []pathres.TimelineEvent {
	result := make([]pathres.TimelineEvent, 0, len(events))
	for _, event := range events {
		summary := event.FromStatus + " → " + event.ToStatus
		if event.EventType == pathmodel.EventTaskContactRecorded {
			summary = "联系渠道：" + event.Channel + "；结果：" + event.Reason
		} else if event.Reason != "" {
			summary += "；原因：" + event.Reason
		}
		result = append(result, pathres.TimelineEvent{
			EventType: event.EventType, OccurredAt: event.OccurredAt, Source: event.Source, Summary: summary,
		})
	}
	return result
}

func nonEmptyIDs(ids []uint) []uint {
	if len(ids) == 0 {
		return []uint{0}
	}
	return ids
}
