package carepath

import (
	"context"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	pathreq "github.com/flipped-aurora/gin-vue-admin/server/model/carepath/request"
	pathres "github.com/flipped-aurora/gin-vue-admin/server/model/carepath/response"
)

func (s *CarePathService) ListPlanVersions(ctx context.Context, req pathreq.PlanVersionSearch) ([]pathres.PlanVersionSummary, int64, error) {
	if _, err := accesspolicy.ResolvePlanContent(ctx, s.db()); err != nil {
		return nil, 0, normalizeCareClientError(err)
	}
	query := s.db().WithContext(ctx).Model(&pathmodel.PlanTemplateVersion{}).
		Joins("JOIN care_path_definition_versions path ON path.id = care_plan_template_versions.path_definition_version_id AND path.deleted_at IS NULL").
		Where("care_plan_template_versions.synthetic = ? AND care_plan_template_versions.usage_scope = ? AND care_plan_template_versions.production_enabled = ?", true, pathmodel.UsageScopeTestOnly, false).
		Where("path.synthetic = ? AND path.usage_scope = ? AND path.production_enabled = ?", true, pathmodel.UsageScopeTestOnly, false).
		Where("care_plan_template_versions.code = ? AND care_plan_template_versions.version = ?", p1PlanTemplateCode, p1PlanTemplateVersion).
		Where("path.code = ? AND path.version = ?", p1PathCode, p1PathVersion)
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("care_plan_template_versions.code LIKE ? OR care_plan_template_versions.version LIKE ? OR care_plan_template_versions.title LIKE ?", like, like, like)
	}
	if req.Status != "" {
		query = query.Where("care_plan_template_versions.status = ?", req.Status)
	}
	if req.UsageScope != "" {
		query = query.Where("care_plan_template_versions.usage_scope = ?", req.UsageScope)
	}
	if req.Synthetic != nil {
		query = query.Where("care_plan_template_versions.synthetic = ?", *req.Synthetic)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit, offset := req.LimitOffset()
	if limit == 0 {
		limit = 10
	}
	type row struct {
		pathmodel.PlanTemplateVersion
		PathCode  string
		TaskCount int64
	}
	var rows []row
	err := query.Select("care_plan_template_versions.*, path.code AS path_code, (SELECT COUNT(1) FROM care_plan_task_definitions td WHERE td.plan_template_version_id = care_plan_template_versions.id AND td.deleted_at IS NULL) AS task_count").
		Order("care_plan_template_versions.id DESC").Limit(limit).Offset(offset).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	items := make([]pathres.PlanVersionSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, planVersionSummary(row.PlanTemplateVersion, row.PathCode, row.TaskCount))
	}
	return items, total, nil
}

func (s *CarePathService) GetPlanVersion(ctx context.Context, id uint) (pathres.PlanVersionDetail, error) {
	if _, err := accesspolicy.ResolvePlanContent(ctx, s.db()); err != nil {
		return pathres.PlanVersionDetail{}, normalizeCareClientError(err)
	}
	value, err := s.loadTemplate(ctx, s.db(), id)
	if err != nil {
		return pathres.PlanVersionDetail{}, err
	}
	if err = s.validateTemplate(ctx, value); err != nil {
		return pathres.PlanVersionDetail{}, err
	}
	tasks := make([]pathres.PlanTaskDefinition, 0, len(value.Tasks))
	for _, task := range value.Tasks {
		item, itemErr := taskDefinitionResponse(task)
		if itemErr != nil {
			return pathres.PlanVersionDetail{}, itemErr
		}
		tasks = append(tasks, item)
	}
	return pathres.PlanVersionDetail{
		PlanVersionSummary:      planVersionSummary(value.Template, value.Path.Code, int64(len(value.Tasks))),
		PathDefinitionVersionID: value.Path.ID, DefinitionSchemaVersion: value.Template.DefinitionSchemaVersion,
		Tasks: tasks,
	}, nil
}

func planVersionSummary(template pathmodel.PlanTemplateVersion, pathCode string, taskCount int64) pathres.PlanVersionSummary {
	return pathres.PlanVersionSummary{
		ID: template.ID, PathCode: pathCode, Code: template.Code, Version: template.Version,
		Title: template.Title, Purpose: template.Purpose, LifecycleStatus: template.Status,
		UsageScope: template.UsageScope, Synthetic: template.Synthetic, ProductionEnabled: template.ProductionEnabled,
		ReviewRecord: pathres.ReviewRecord{
			ReviewType: template.ReviewType, ReviewedBy: template.ReviewedBy, ReviewedAt: template.ReviewedAt,
			FormalMedicalApproval: template.ReviewType == pathmodel.ReviewTypeFormal, Note: template.ReviewNote,
		},
		AnchorDefinition: template.AnchorDefinition, LateSubmissionPolicy: template.LateSubmissionPolicy,
		PauseStrategy: template.PauseStrategy, TaskCount: taskCount, PublishedAt: template.PublishedAt,
		DefinitionHash: template.DefinitionHash,
	}
}
