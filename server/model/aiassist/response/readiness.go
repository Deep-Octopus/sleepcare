package response

type ShadowReadiness struct {
	Mode                          string   `json:"mode"`
	SelectionStatus               string   `json:"selectionStatus"`
	UsageScope                    string   `json:"usageScope"`
	StaffShadowEnabled            bool     `json:"staffShadowEnabled"`
	SuggestionGenerationEnabled   bool     `json:"suggestionGenerationEnabled"`
	KnowledgeRetrievalEnabled     bool     `json:"knowledgeRetrievalEnabled"`
	ExternalModelEnabled          bool     `json:"externalModelEnabled"`
	UserFacingAIEnabled           bool     `json:"userFacingAiEnabled"`
	DirectSendEnabled             bool     `json:"directSendEnabled"`
	ReviewedKnowledgeReady        bool     `json:"reviewedKnowledgeReady"`
	HumanReviewWorkflowReady      bool     `json:"humanReviewWorkflowReady"`
	LineagePersistenceReady       bool     `json:"lineagePersistenceReady"`
	ProhibitedScenarioPolicyReady bool     `json:"prohibitedScenarioPolicyReady"`
	DataProcessingReviewReady     bool     `json:"dataProcessingReviewReady"`
	EvaluationProtocolReady       bool     `json:"evaluationProtocolReady"`
	ModelProviderReviewReady      bool     `json:"modelProviderReviewReady"`
	Blockers                      []string `json:"blockers"`
}
