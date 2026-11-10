package service

import (
	"context"
	"errors"

	"github.com/tidegate-dev/tidegate/internal/secret"
)

// ErrNotSignedIn means a call has no login token, or one that is unknown or
// expired.
var ErrNotSignedIn = errors.New("not signed in")

// Logins signs users in with a local password and logs them out.
type Logins struct {
	authenticator *Authenticator
	tokens        *LoginTokens
}

// NewLogins returns Logins.
func NewLogins(authenticator *Authenticator, tokens *LoginTokens) *Logins {
	return &Logins{authenticator: authenticator, tokens: tokens}
}

// Login returns a new login token if plaintext is the user's password.
func (l *Logins) Login(ctx context.Context, name string, plaintext secret.String) (secret.String, error) {
	userID, err := l.authenticator.Authenticate(ctx, name, plaintext)
	if err != nil {
		return secret.String{}, err
	}

	return l.tokens.Issue(ctx, userID)
}

// Logout revokes the login token of the session. With allSessions it revokes
// every login token of the user.
func (l *Logins) Logout(ctx context.Context, session Session, allSessions bool) error {
	if allSessions {
		return l.tokens.RevokeAll(ctx, session.UserID)
	}

	return l.tokens.Revoke(ctx, session.ID)
}
