package questionnaire

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

// QuestionnaireVersion is an immutable, published definition. Definitions are
// global configuration and deliberately do not carry row-level ownership.
type QuestionnaireVersion struct {
	global.GVA_MODEL
	Code                    string     `json:"code" gorm:"type:varchar(64);uniqueIndex:idx_questionnaire_code_version,priority:1;not null"`
	Version                 string     `json:"version" gorm:"type:varchar(32);uniqueIndex:idx_questionnaire_code_version,priority:2;not null"`
	Title                   string     `json:"title" gorm:"type:varchar(160);not null"`
	Purpose                 string     `json:"purpose" gorm:"type:varchar(500);not null"`
	Status                  string     `json:"status" gorm:"type:varchar(24);index;not null"`
	UsageScope              string     `json:"usageScope" gorm:"type:varchar(24);index;not null"`
	Synthetic               bool       `json:"synthetic" gorm:"index;not null;default:false"`
	ProductionEnabled       bool       `json:"productionEnabled" gorm:"not null;default:false"`
	ReviewType              string     `json:"reviewType" gorm:"type:varchar(24);not null"`
	ReviewedBy              uint       `json:"reviewedBy" gorm:"index"`
	ReviewedAt              *time.Time `json:"reviewedAt"`
	ReviewNote              string     `json:"reviewNote" gorm:"type:varchar(500)"`
	ExpectedMinutes         uint       `json:"expectedMinutes" gorm:"not null;default:0"`
	ReplacesVersionID       *uint      `json:"replacesVersionId" gorm:"index"`
	PublishedAt             *time.Time `json:"publishedAt" gorm:"index"`
	DefinitionSchemaVersion string     `json:"definitionSchemaVersion" gorm:"type:varchar(16);not null;default:v1"`
	DefinitionHash          string     `json:"definitionHash" gorm:"type:char(64);not null"`
	RowVersion              uint       `json:"rowVersion" gorm:"not null;default:1"`
}

func (QuestionnaireVersion) TableName() string { return "questionnaire_versions" }

type QuestionnaireQuestion struct {
	global.GVA_MODEL
	QuestionnaireVersionID  uint           `json:"questionnaireVersionId" gorm:"uniqueIndex:idx_questionnaire_question_code,priority:1;index;not null"`
	Code                    string         `json:"code" gorm:"type:varchar(64);uniqueIndex:idx_questionnaire_question_code,priority:2;not null"`
	Type                    string         `json:"type" gorm:"type:varchar(32);not null"`
	Title                   string         `json:"title" gorm:"type:varchar(300);not null"`
	Required                bool           `json:"required" gorm:"not null;default:false"`
	Sort                    int            `json:"sort" gorm:"index;not null"`
	ValidationSchemaVersion string         `json:"validationSchemaVersion" gorm:"type:varchar(16);not null;default:v1"`
	ValidationJSON          datatypes.JSON `json:"validation" gorm:"type:json;not null" swaggertype:"object"`
}

func (QuestionnaireQuestion) TableName() string { return "questionnaire_questions" }

type QuestionnaireOption struct {
	global.GVA_MODEL
	QuestionID uint   `json:"questionId" gorm:"uniqueIndex:idx_questionnaire_option_code,priority:1;index;not null"`
	Code       string `json:"code" gorm:"type:varchar(64);uniqueIndex:idx_questionnaire_option_code,priority:2;not null"`
	Label      string `json:"label" gorm:"type:varchar(200);not null"`
	Sort       int    `json:"sort" gorm:"index;not null"`
}

func (QuestionnaireOption) TableName() string { return "questionnaire_options" }

type QuestionnaireRuleVersion struct {
	global.GVA_MODEL
	QuestionnaireVersionID  uint           `json:"questionnaireVersionId" gorm:"index;not null"`
	Code                    string         `json:"code" gorm:"type:varchar(64);uniqueIndex:idx_questionnaire_rule_code_version,priority:1;not null"`
	Version                 string         `json:"version" gorm:"type:varchar(32);uniqueIndex:idx_questionnaire_rule_code_version,priority:2;not null"`
	Title                   string         `json:"title" gorm:"type:varchar(160);not null"`
	Status                  string         `json:"status" gorm:"type:varchar(24);index;not null"`
	UsageScope              string         `json:"usageScope" gorm:"type:varchar(24);index;not null"`
	Synthetic               bool           `json:"synthetic" gorm:"index;not null;default:false"`
	ProductionEnabled       bool           `json:"productionEnabled" gorm:"not null;default:false"`
	ReviewType              string         `json:"reviewType" gorm:"type:varchar(24);not null"`
	ReviewedBy              uint           `json:"reviewedBy" gorm:"index"`
	ReviewedAt              *time.Time     `json:"reviewedAt"`
	ReviewNote              string         `json:"reviewNote" gorm:"type:varchar(500)"`
	ConditionSchemaVersion  string         `json:"conditionSchemaVersion" gorm:"type:varchar(16);not null;default:v1"`
	ConditionJSON           datatypes.JSON `json:"condition" gorm:"type:json;not null" swaggertype:"object"`
	AttentionLevel          string         `json:"attentionLevel" gorm:"type:varchar(64);not null"`
	ReasonSnapshot          string         `json:"reasonSnapshot" gorm:"type:varchar(500);not null"`
	RecipientsJSON          datatypes.JSON `json:"recipients" gorm:"type:json;not null" swaggertype:"array,string"`
	SLAFirstResponseMinutes *uint          `json:"slaFirstResponseMinutes"`
	SLAResolutionMinutes    *uint          `json:"slaResolutionMinutes"`
	DedupKeyTemplate        string         `json:"dedupKeyTemplate" gorm:"type:varchar(200);not null"`
	ReplacesVersionID       *uint          `json:"replacesVersionId" gorm:"index"`
	PublishedAt             *time.Time     `json:"publishedAt" gorm:"index"`
	DefinitionHash          string         `json:"definitionHash" gorm:"type:char(64);not null"`
	RowVersion              uint           `json:"rowVersion" gorm:"not null;default:1"`
}

func (QuestionnaireRuleVersion) TableName() string { return "questionnaire_rule_versions" }

type QuestionnaireSubmission struct {
	global.GVA_MODEL
	TaskID                  uint64         `json:"taskId" gorm:"uniqueIndex;not null"`
	CareClientID            uint           `json:"careClientId" gorm:"index;not null"`
	QuestionnaireVersionID  uint           `json:"questionnaireVersionId" gorm:"index;not null"`
	BoundRuleVersionIDsJSON datatypes.JSON `json:"boundRuleVersionIds" gorm:"type:json;not null" swaggertype:"array,integer"`
	Source                  string         `json:"source" gorm:"type:varchar(32);index;not null"`
	ActorKind               string         `json:"actorKind" gorm:"type:varchar(16);not null"`
	ActorID                 uint           `json:"actorId" gorm:"index;not null"`
	SourceReason            string         `json:"sourceReason" gorm:"type:varchar(500)"`
	ConfirmationMethod      string         `json:"confirmationMethod" gorm:"type:varchar(64)"`
	RequestHash             string         `json:"requestHash" gorm:"type:char(64);not null"`
	SubmittedAt             time.Time      `json:"submittedAt" gorm:"index;not null"`
	ClientOccurredAt        *time.Time     `json:"clientOccurredAt"`
	CurrentRevisionNo       uint           `json:"currentRevisionNo" gorm:"not null;default:1"`
	Synthetic               bool           `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId                  uint           `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy               uint           `json:"createdBy" gorm:"column:created_by;index"`
}

func (QuestionnaireSubmission) TableName() string { return "questionnaire_submissions" }

type QuestionnaireAnswerRevision struct {
	global.GVA_MODEL
	SubmissionID uint           `json:"submissionId" gorm:"uniqueIndex:idx_answer_revision,priority:1;index;not null"`
	RevisionNo   uint           `json:"revisionNo" gorm:"uniqueIndex:idx_answer_revision,priority:2;not null"`
	AnswersJSON  datatypes.JSON `json:"answers" gorm:"type:json;not null" swaggertype:"object"`
	Reason       string         `json:"reason" gorm:"type:varchar(500);not null"`
	ActorKind    string         `json:"actorKind" gorm:"type:varchar(16);not null"`
	ActorID      uint           `json:"actorId" gorm:"index;not null"`
	OccurredAt   time.Time      `json:"occurredAt" gorm:"index;not null"`
	Synthetic    bool           `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId       uint           `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy    uint           `json:"createdBy" gorm:"column:created_by;index"`
}

func (QuestionnaireAnswerRevision) TableName() string { return "questionnaire_answer_revisions" }

type QuestionnaireRuleHit struct {
	global.GVA_MODEL
	SubmissionID          uint           `json:"submissionId" gorm:"uniqueIndex:idx_rule_hit_dedup,priority:1;index;not null"`
	AnswerRevisionID      uint           `json:"answerRevisionId" gorm:"index;not null"`
	RuleVersionID         uint           `json:"ruleVersionId" gorm:"uniqueIndex:idx_rule_hit_dedup,priority:2;index;not null"`
	ConditionSnapshotJSON datatypes.JSON `json:"conditionSnapshot" gorm:"type:json;not null" swaggertype:"object"`
	AttentionLevel        string         `json:"attentionLevel" gorm:"type:varchar(64);index;not null"`
	ReasonSnapshot        string         `json:"reasonSnapshot" gorm:"type:varchar(500);not null"`
	RecipientsJSON        datatypes.JSON `json:"recipients" gorm:"type:json;not null" swaggertype:"array,string"`
	DedupKey              string         `json:"dedupKey" gorm:"type:varchar(200);uniqueIndex:idx_rule_hit_dedup,priority:3;not null"`
	OccurredAt            time.Time      `json:"occurredAt" gorm:"index;not null"`
	Synthetic             bool           `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId                uint           `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy             uint           `json:"createdBy" gorm:"column:created_by;index"`
}

func (QuestionnaireRuleHit) TableName() string { return "questionnaire_rule_hits" }

type QuestionnaireCommandReceipt struct {
	global.GVA_MODEL
	Operation      string `json:"operation" gorm:"type:varchar(64);uniqueIndex:idx_questionnaire_receipt_actor_op_key,priority:2;not null"`
	ActorID        uint   `json:"actorId" gorm:"uniqueIndex:idx_questionnaire_receipt_actor_op_key,priority:1;not null"`
	IdempotencyKey string `json:"idempotencyKey" gorm:"type:varchar(128);uniqueIndex:idx_questionnaire_receipt_actor_op_key,priority:3;not null"`
	RequestHash    string `json:"requestHash" gorm:"type:char(64);not null"`
	ResultJSON     string `json:"-" gorm:"type:text;not null"`
	DeptId         uint   `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy      uint   `json:"createdBy" gorm:"column:created_by;index"`
}

func (QuestionnaireCommandReceipt) TableName() string { return "questionnaire_command_receipts" }

type OutboxEvent struct {
	global.GVA_MODEL
	EventID        string         `json:"eventId" gorm:"type:char(36);uniqueIndex;not null"`
	EventType      string         `json:"eventType" gorm:"type:varchar(64);index;not null"`
	PayloadVersion string         `json:"payloadVersion" gorm:"type:varchar(16);not null"`
	AggregateType  string         `json:"aggregateType" gorm:"type:varchar(64);index;not null"`
	AggregateID    string         `json:"aggregateId" gorm:"type:varchar(64);index;not null"`
	PayloadJSON    datatypes.JSON `json:"payload" gorm:"type:json;not null" swaggertype:"object"`
	OccurredAt     time.Time      `json:"occurredAt" gorm:"index;not null"`
	CorrelationID  string         `json:"correlationId" gorm:"type:varchar(64);index"`
	CausationID    string         `json:"causationId" gorm:"type:varchar(64);index"`
	Synthetic      bool           `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId         uint           `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy      uint           `json:"createdBy" gorm:"column:created_by;index"`
}

func (OutboxEvent) TableName() string { return "outbox_events" }
