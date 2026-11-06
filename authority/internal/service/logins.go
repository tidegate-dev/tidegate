package service

import (
	"context"

	"github.com/tidegate-dev/tidegate/internal/secret"
)

// Logins signs users in with a local password.
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
