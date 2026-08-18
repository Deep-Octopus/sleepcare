package casework

import (
	"context"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
)

var workbenchLocation = func() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err == nil {
		return location
	}
	return time.FixedZone("Asia/Shanghai", 8*60*60)
}()

func (s *CaseWorkService) GetWorkbench(ctx context.Context) (caseworkres.WorkbenchData, error) {
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.WorkbenchData{}, normalizeAccessError(err)
	}

	clientQuery := decision.Scope(s.db().WithContext(ctx).Model(&caremodel.CareClient{}), s.now()).
		Where("care_clients.synthetic = ? AND care_clients.status = ?", true, caremodel.ClientStatusActive)
	var clientIDs []uint
	if err = clientQuery.Order("care_clients.id ASC").Pluck("care_clients.id", &clientIDs).Error; err != nil {
		return caseworkres.WorkbenchData{}, err
	}
	if len(clientIDs) == 0 {
		return caseworkres.WorkbenchData{}, nil
	}

	data := caseworkres.WorkbenchData{}
	dayStart, dayEnd := workbenchDayBounds(s.now())
	if err = s.db().WithContext(ctx).Model(&pathmodel.TaskInstance{}).
		Where("care_client_id IN ? AND synthetic = ?", clientIDs, true).
		Where("due_at >= ? AND due_at < ?", dayStart, dayEnd).
		Where("execution_status NOT IN ?", []string{pathmodel.ExecutionSubmitted, pathmodel.ExecutionCancelled}).
		Count(&data.DueToday).Error; err != nil {
		return caseworkres.WorkbenchData{}, err
	}
	if err = s.db().WithContext(ctx).Model(&pathmodel.TaskInstance{}).
		Where("care_client_id IN ? AND synthetic = ?", clientIDs, true).
		Where("execution_role = ? AND execution_status IN ?", pathmodel.ExecutionRoleCareClient,
			[]string{pathmodel.ExecutionOpen, pathmodel.ExecutionInProgress}).
		Count(&data.WaitingClient).Error; err != nil {
		return caseworkres.WorkbenchData{}, err
	}
	if err = s.db().WithContext(ctx).Model(&pathmodel.TaskInstance{}).
		Where("care_client_id IN ? AND synthetic = ?", clientIDs, true).
		Where("review_status = ?", pathmodel.ReviewPending).
		Count(&data.ReviewRequired).Error; err != nil {
		return caseworkres.WorkbenchData{}, err
	}

	caseQuery := decision.ScopeAttentionCases(
		s.db().WithContext(ctx).Model(&caseworkmodel.AttentionCase{}), s.now(),
	).Where("attention_cases.synthetic = ? AND attention_cases.status <> ?", true, caseworkmodel.CaseStatusClosed)
	if err = caseQuery.Count(&data.AttentionCases).Error; err != nil {
		return caseworkres.WorkbenchData{}, err
	}

	if err = s.db().WithContext(ctx).Model(&caseworkmodel.TodoItem{}).
		Where("care_client_id IN ? AND synthetic = ? AND status = ?", clientIDs, true, caseworkmodel.TodoStatusOpen).
		Where("category = ?", caseworkmodel.TodoCategoryDeliveryIssue).
		Count(&data.DeliveryIssues).Error; err != nil {
		return caseworkres.WorkbenchData{}, err
	}
	if err = s.db().WithContext(ctx).Model(&caseworkmodel.TodoItem{}).
		Where("care_client_id IN ? AND synthetic = ? AND status = ?", clientIDs, true, caseworkmodel.TodoStatusOpen).
		Where("assignee_id = ?", decision.Identity.UserID).
		Count(&data.AssignedToMe).Error; err != nil {
		return caseworkres.WorkbenchData{}, err
	}

	return data, nil
}

func workbenchDayBounds(now time.Time) (time.Time, time.Time) {
	local := now.In(workbenchLocation)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, workbenchLocation)
	return start, start.AddDate(0, 0, 1)
}
