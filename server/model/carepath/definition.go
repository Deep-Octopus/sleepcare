package carepath

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// PathDefinitionDocument preserves the canonical field names used by the
// immutable path-version hash. Keep the declaration order stable.
type PathDefinitionDocument struct {
	Code, Version, Title, Purpose, UsageScope string
	Synthetic, ProductionEnabled              bool
}

type TaskDefinitionDocument struct {
	DayCode                string `json:"dayCode"`
	Title                  string `json:"title"`
	Sort                   int    `json:"sort"`
	ExecutionRole          string `json:"executionRole"`
	OpenOffsetSeconds      int64  `json:"openOffsetSeconds"`
	DueOffsetSeconds       int64  `json:"dueOffsetSeconds"`
	ExpiresOffsetSeconds   *int64 `json:"expiresOffsetSeconds,omitempty"`
	QuestionnaireVersionID *uint  `json:"questionnaireVersionId,omitempty"`
	BoundRuleVersionIDs    []uint `json:"boundRuleVersionIds"`
	ReviewRequired         bool   `json:"reviewRequired"`
	ReviewRole             string `json:"reviewRole,omitempty"`
	NotificationPolicy     string `json:"notificationPolicy"`
}

type PlanDefinitionDocument struct {
	PathCode                string                   `json:"pathCode"`
	Code                    string                   `json:"code"`
	Version                 string                   `json:"version"`
	Title                   string                   `json:"title"`
	Purpose                 string                   `json:"purpose"`
	UsageScope              string                   `json:"usageScope"`
	Synthetic               bool                     `json:"synthetic"`
	ProductionEnabled       bool                     `json:"productionEnabled"`
	AnchorDefinition        string                   `json:"anchorDefinition"`
	LateSubmissionPolicy    string                   `json:"lateSubmissionPolicy"`
	PauseStrategy           string                   `json:"pauseStrategy"`
	DefinitionSchemaVersion string                   `json:"definitionSchemaVersion"`
	Tasks                   []TaskDefinitionDocument `json:"tasks"`
}

func HashDefinition(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}
