package clientaccess

const (
	AccountStatusActive  = "ACTIVE"
	AccountStatusRevoked = "REVOKED"

	GrantStatusIssued   = "ISSUED"
	GrantStatusRedeemed = "REDEEMED"
	GrantStatusRevoked  = "REVOKED"

	SessionStatusActive  = "ACTIVE"
	SessionStatusRevoked = "REVOKED"

	InteractionOpened    = "OPENED"
	InteractionConsented = "CONSENTED"
	InteractionStarted   = "STARTED"
)

func IsInteractionType(value string) bool {
	return value == InteractionOpened || value == InteractionConsented || value == InteractionStarted
}
