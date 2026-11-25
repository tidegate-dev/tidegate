package service

import (
	"context"
	"errors"

	"github.com/phemera-dev/phemera/authority/internal/model"
	"github.com/phemera-dev/phemera/internal/secret"
)

// ErrNotSignedIn means a call has no session token, or one that is unknown or
// expired.
var ErrNotSignedIn = errors.New("not signed in")

// Logins signs users in with a local password and logs them out.
type Logins struct {
	authenticator *Authenticator
	sessions      *Sessions
}

// NewLogins returns Logins.
func NewLogins(authenticator *Authenticator, sessions *Sessions) *Logins {
	return &Logins{authenticator: authenticator, sessions: sessions}
}

// Login returns a new session token if plaintext is the user's password.
func (l *Logins) Login(ctx context.Context, name string, plaintext secret.String) (secret.String, error) {
	userID, err := l.authenticator.Authenticate(ctx, name, plaintext)
	if err != nil {
		return secret.String{}, err
	}

	return l.sessions.Issue(ctx, userID)
}

// Logout revokes the session token of the session. With allSessions it revokes
// every session token of the user.
func (l *Logins) Logout(ctx context.Context, session model.Session, allSessions bool) error {
	if allSessions {
		return l.sessions.RevokeAll(ctx, session.UserID)
	}

	return l.sessions.Revoke(ctx, session)
}
