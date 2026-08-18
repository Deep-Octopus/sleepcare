package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	"github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
)

func bizModel() error {
	db := global.GVA_DB
	err := db.AutoMigrate(
		&careclient.CareClient{},
		&careclient.CareAssignment{},
		&careclient.ConsentRecord{},
		&careclient.CareOrgUnitProfile{},
		&careclient.CareAuthorityProfile{},
		&careclient.CareClientCommandReceipt{},
		&questionnaire.QuestionnaireVersion{},
		&questionnaire.QuestionnaireQuestion{},
		&questionnaire.QuestionnaireOption{},
		&questionnaire.QuestionnaireRuleVersion{},
		&questionnaire.QuestionnaireSubmission{},
		&questionnaire.QuestionnaireAnswerRevision{},
		&questionnaire.QuestionnaireRuleHit{},
		&questionnaire.QuestionnaireCommandReceipt{},
		&questionnaire.OutboxEvent{},
	)
	if err != nil {
		return err
	}
	return nil
}
