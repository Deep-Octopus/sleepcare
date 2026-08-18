package questionnaire

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type OptionDefinition struct {
	Code  string `json:"code"`
	Label string `json:"label"`
	Sort  int    `json:"sort"`
}

type QuestionDefinition struct {
	Code                    string             `json:"code"`
	Type                    string             `json:"type"`
	Title                   string             `json:"title"`
	Required                bool               `json:"required"`
	Sort                    int                `json:"sort"`
	ValidationSchemaVersion string             `json:"validationSchemaVersion"`
	Validation              json.RawMessage    `json:"validation"`
	Options                 []OptionDefinition `json:"options"`
}

type VersionDefinition struct {
	Code                    string               `json:"code"`
	Version                 string               `json:"version"`
	Title                   string               `json:"title"`
	Purpose                 string               `json:"purpose"`
	UsageScope              string               `json:"usageScope"`
	Synthetic               bool                 `json:"synthetic"`
	ProductionEnabled       bool                 `json:"productionEnabled"`
	ExpectedMinutes         uint                 `json:"expectedMinutes"`
	DefinitionSchemaVersion string               `json:"definitionSchemaVersion"`
	Questions               []QuestionDefinition `json:"questions"`
}

type RuleDefinition struct {
	QuestionnaireVersionID  uint            `json:"questionnaireVersionId"`
	Code                    string          `json:"code"`
	Version                 string          `json:"version"`
	Title                   string          `json:"title"`
	UsageScope              string          `json:"usageScope"`
	Synthetic               bool            `json:"synthetic"`
	ProductionEnabled       bool            `json:"productionEnabled"`
	ConditionSchemaVersion  string          `json:"conditionSchemaVersion"`
	Condition               json.RawMessage `json:"condition"`
	AttentionLevel          string          `json:"attentionLevel"`
	ReasonSnapshot          string          `json:"reasonSnapshot"`
	Recipients              json.RawMessage `json:"recipients"`
	SLAFirstResponseMinutes *uint           `json:"slaFirstResponseMinutes"`
	SLAResolutionMinutes    *uint           `json:"slaResolutionMinutes"`
	DedupKeyTemplate        string          `json:"dedupKeyTemplate"`
}

func HashDefinition(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

// CanonicalJSON normalizes object key order while preserving JSON number text.
// MySQL's JSON storage may return object keys in a different order than the
// original insert, so raw JSON must be canonicalized before definition hashes
// are recomputed.
func CanonicalJSON(raw json.RawMessage) json.RawMessage {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return raw
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return raw
	}
	return canonical
}
