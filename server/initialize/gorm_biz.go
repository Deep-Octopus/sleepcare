package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	"github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	"github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	"github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess"
	"github.com/flipped-aurora/gin-vue-admin/server/model/notification"
	"github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
)

func bizModel() error {
	db := global.GVA_DB
	err := db.AutoMigrate(
		&carepath.PathDefinitionVersion{},
		&carepath.PlanTemplateVersion{},
		&carepath.PlanTaskDefinition{},
		&carepath.PlanTaskDependency{},
		&carepath.Enrollment{},
		&carepath.PlanPreview{},
		&carepath.PlanInstance{},
		&carepath.TaskInstance{},
		&carepath.CarePathEvent{},
		&carepath.CommandReceipt{},
		&careclient.CareClient{},
		&careclient.CareAssignment{},
		&careclient.ConsentRecord{},
		&careclient.CareOrgUnitProfile{},
		&careclient.CareAuthorityProfile{},
		&careclient.CareClientCommandReceipt{},
		&clientaccess.CareClientAccount{},
		&clientaccess.ClientAccessGrant{},
		&clientaccess.ClientSession{},
		&clientaccess.ClientTaskCommandReceipt{},
		&casework.AttentionCase{},
		&casework.CaseAction{},
		&casework.TodoItem{},
		&casework.CommandReceipt{},
		&casework.Consultation{},
		&casework.ConsultationInteraction{},
		&questionnaire.QuestionnaireVersion{},
		&questionnaire.QuestionnaireQuestion{},
		&questionnaire.QuestionnaireOption{},
		&questionnaire.QuestionnaireRuleVersion{},
		&questionnaire.QuestionnaireSubmission{},
		&questionnaire.QuestionnaireTaskDraft{},
		&questionnaire.QuestionnaireAnswerRevision{},
		&questionnaire.QuestionnaireRuleHit{},
		&questionnaire.QuestionnaireCommandReceipt{},
		&questionnaire.OutboxEvent{},
		&supervision.DailySummaryVersion{},
		&supervision.SupervisorGuidance{},
		&supervision.SatisfactionPolicy{},
		&supervision.SatisfactionRequest{},
		&supervision.SatisfactionResponse{},
		&supervision.SatisfactionFollowUp{},
		&supervision.SatisfactionFollowUpAction{},
		&notification.NotificationRequest{},
		&notification.NotificationAttempt{},
		&notification.DeliveryEvent{},
	)
	if err != nil {
		return err
	}
	return nil
}
