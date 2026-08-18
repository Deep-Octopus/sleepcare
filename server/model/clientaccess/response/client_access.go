package response

import "time"

type RedeemResult struct {
	ExpiresAt        time.Time `json:"expiresAt"`
	AllowedTaskCount int       `json:"allowedTaskCount"`
}

type TaskSummary struct {
	ID               uint       `json:"id"`
	DayCode          string     `json:"dayCode"`
	Title            string     `json:"title"`
	ExecutionStatus  string     `json:"executionStatus"`
	TimingStatus     string     `json:"timingStatus"`
	ReviewStatus     string     `json:"reviewStatus"`
	OpenAt           time.Time  `json:"openAt"`
	DueAt            time.Time  `json:"dueAt"`
	ExpiresAt        *time.Time `json:"expiresAt"`
	SubmittedAt      *time.Time `json:"submittedAt"`
	Version          uint       `json:"version"`
	Accessible       bool       `json:"accessible"`
	HasQuestionnaire bool       `json:"hasQuestionnaire"`
}

type TaskDetail struct {
	TaskSummary
	Opened       bool `json:"opened"`
	Consented    bool `json:"consented"`
	Started      bool `json:"started"`
	CanSaveDraft bool `json:"canSaveDraft"`
	CanSubmit    bool `json:"canSubmit"`
}

type QuestionnaireOption struct {
	Code  string `json:"code"`
	Label string `json:"label"`
	Order int    `json:"order"`
}

type QuestionnaireQuestion struct {
	Code       string                `json:"code"`
	Type       string                `json:"type"`
	Title      string                `json:"title"`
	Required   bool                  `json:"required"`
	Order      int                   `json:"order"`
	Validation map[string]any        `json:"validation" swaggertype:"object"`
	Options    []QuestionnaireOption `json:"options"`
}

type Draft struct {
	Version uint           `json:"version"`
	Answers map[string]any `json:"answers" swaggertype:"object"`
	SavedAt time.Time      `json:"savedAt"`
}

type Questionnaire struct {
	ID              uint                    `json:"id"`
	Title           string                  `json:"title"`
	Purpose         string                  `json:"purpose"`
	ExpectedMinutes uint                    `json:"expectedMinutes"`
	TaskVersion     uint                    `json:"taskVersion"`
	Questions       []QuestionnaireQuestion `json:"questions"`
	Draft           *Draft                  `json:"draft"`
}

type InteractionResult struct {
	TaskID          uint   `json:"taskId"`
	InteractionType string `json:"interactionType"`
	ExecutionStatus string `json:"executionStatus"`
	TaskVersion     uint   `json:"taskVersion"`
}

type DraftResult struct {
	TaskID  uint      `json:"taskId"`
	Version uint      `json:"version"`
	SavedAt time.Time `json:"savedAt"`
}

type SubmitResult struct {
	TaskID           uint   `json:"taskId"`
	SubmissionID     uint   `json:"submissionId"`
	ExecutionStatus  string `json:"executionStatus"`
	ReviewStatus     string `json:"reviewStatus"`
	TaskVersion      uint   `json:"taskVersion"`
	RuleHitIDs       []uint `json:"ruleHitIds"`
	AttentionCaseIDs []uint `json:"attentionCaseIds"`
}
