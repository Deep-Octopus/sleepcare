package response

import "time"

type ReviewRecord struct {
	ReviewType            string     `json:"reviewType"`
	ReviewedBy            string     `json:"reviewedBy"`
	ReviewedAt            *time.Time `json:"reviewedAt"`
	FormalMedicalApproval bool       `json:"formalMedicalApproval"`
	Note                  string     `json:"note"`
}

type QuestionnaireVersionSummary struct {
	ID                uint         `json:"id"`
	Code              string       `json:"code"`
	Version           string       `json:"version"`
	Title             string       `json:"title"`
	Purpose           string       `json:"purpose"`
	LifecycleStatus   string       `json:"lifecycleStatus"`
	UsageScope        string       `json:"usageScope"`
	Synthetic         bool         `json:"synthetic"`
	ProductionEnabled bool         `json:"productionEnabled"`
	ReviewRecord      ReviewRecord `json:"reviewRecord"`
	ExpectedMinutes   uint         `json:"expectedMinutes"`
	QuestionCount     int64        `json:"questionCount"`
	RuleCount         int64        `json:"ruleCount"`
	PublishedAt       *time.Time   `json:"publishedAt"`
	DefinitionHash    string       `json:"definitionHash"`
}

type QuestionnaireOption struct {
	ID    uint   `json:"id"`
	Code  string `json:"code"`
	Label string `json:"label"`
	Order int    `json:"order"`
}

type QuestionnaireQuestion struct {
	ID                      uint                  `json:"id"`
	Code                    string                `json:"code"`
	Type                    string                `json:"type"`
	Title                   string                `json:"title"`
	Required                bool                  `json:"required"`
	Order                   int                   `json:"order"`
	ValidationSchemaVersion string                `json:"validationSchemaVersion"`
	Validation              map[string]any        `json:"validation" swaggertype:"object"`
	Options                 []QuestionnaireOption `json:"options"`
}

type QuestionnaireRuleVersion struct {
	ID                      uint           `json:"id"`
	Code                    string         `json:"code"`
	Version                 string         `json:"version"`
	Title                   string         `json:"title"`
	LifecycleStatus         string         `json:"lifecycleStatus"`
	UsageScope              string         `json:"usageScope"`
	Synthetic               bool           `json:"synthetic"`
	ProductionEnabled       bool           `json:"productionEnabled"`
	ReviewRecord            ReviewRecord   `json:"reviewRecord"`
	ConditionSchemaVersion  string         `json:"conditionSchemaVersion"`
	Condition               map[string]any `json:"condition" swaggertype:"object"`
	AttentionLevel          string         `json:"attentionLevel"`
	ReasonSnapshot          string         `json:"reasonSnapshot"`
	Recipients              []string       `json:"recipients"`
	SLAFirstResponseMinutes *uint          `json:"slaFirstResponseMinutes"`
	SLAResolutionMinutes    *uint          `json:"slaResolutionMinutes"`
	DedupKeyTemplate        string         `json:"dedupKeyTemplate"`
	PublishedAt             *time.Time     `json:"publishedAt"`
	DefinitionHash          string         `json:"definitionHash"`
}

type QuestionnaireVersionDetail struct {
	QuestionnaireVersionSummary
	DefinitionSchemaVersion string                     `json:"definitionSchemaVersion"`
	ReplacesVersionID       *uint                      `json:"replacesVersionId"`
	Questions               []QuestionnaireQuestion    `json:"questions"`
	Rules                   []QuestionnaireRuleVersion `json:"rules"`
}
