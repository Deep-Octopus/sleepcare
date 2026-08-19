package aiassist

const (
	ShadowModeDisabled          = "DISABLED"
	ShadowSelectionNotSelected  = "NOT_SELECTED"
	ShadowUsageScopeNotEnabled  = "NOT_ENABLED"
	BlockerPhaseTwoNotSelected  = "PHASE_TWO_SHADOW_NOT_SELECTED"
	BlockerKnowledgeNotReviewed = "REVIEWED_KNOWLEDGE_NOT_AVAILABLE"
	BlockerReviewFlowMissing    = "HUMAN_REVIEW_WORKFLOW_NOT_VALIDATED"
	BlockerLineageMissing       = "LINEAGE_PERSISTENCE_NOT_IMPLEMENTED"
	BlockerScenarioPolicy       = "PROHIBITED_SCENARIO_POLICY_NOT_APPROVED"
	BlockerDataPolicy           = "DATA_PROCESSING_REVIEW_NOT_APPROVED"
	BlockerEvaluation           = "EVALUATION_PROTOCOL_NOT_APPROVED"
	BlockerProvider             = "MODEL_PROVIDER_NOT_APPROVED"
)
