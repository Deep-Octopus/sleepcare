package clientaccess

import (
	"context"

	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
)

type SessionIdentity struct {
	SessionID      string
	AccountID      uint
	CareClientID   uint
	DeptID         uint
	AllowedTaskIDs []uint
	Synthetic      bool
}

type sessionIdentityKey struct{}

func ContextWithSessionIdentity(ctx context.Context, identity SessionIdentity) context.Context {
	ctx = context.WithValue(ctx, sessionIdentityKey{}, identity)
	return datascope.WithIdentity(ctx, &datascope.Identity{
		UserID:         identity.CareClientID,
		DeptID:         identity.DeptID,
		DeptIDs:        []uint{identity.DeptID},
		VisibleDeptIDs: []uint{identity.DeptID},
		Scope:          datascope.ScopeDept,
	})
}

func SessionIdentityFromContext(ctx context.Context) (SessionIdentity, bool) {
	if ctx == nil {
		return SessionIdentity{}, false
	}
	value, ok := ctx.Value(sessionIdentityKey{}).(SessionIdentity)
	return value, ok
}
