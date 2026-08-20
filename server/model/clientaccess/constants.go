package clientaccess

const (
	AccountStatusActive  = "ACTIVE"
	AccountStatusRevoked = "REVOKED"

	CredentialStatusActive   = "ACTIVE"
	CredentialStatusDisabled = "DISABLED"

	GrantStatusIssued   = "ISSUED"
	GrantStatusRedeemed = "REDEEMED"
	GrantStatusRevoked  = "REVOKED"

	SessionStatusActive  = "ACTIVE"
	SessionStatusRevoked = "REVOKED"

	SessionAuthGrant   = "TASK_GRANT"
	SessionAuthAccount = "ACCOUNT"

	InteractionOpened    = "OPENED"
	InteractionConsented = "CONSENTED"
	InteractionStarted   = "STARTED"
)

func IsInteractionType(value string) bool {
	return value == InteractionOpened || value == InteractionConsented || value == InteractionStarted
}
