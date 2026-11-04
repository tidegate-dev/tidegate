package service

import "context"

// Logins signs users in with a local password.
type Logins struct {
	authenticator *Authenticator
	tokens        *LoginTokens
}

// NewLogins returns Logins that checks passwords with authenticator and
// hands out tokens from tokens.
func NewLogins(authenticator *Authenticator, tokens *LoginTokens) *Logins {
	return &Logins{authenticator: authenticator, tokens: tokens}
}

// Login returns a new login token if plaintext is the local password of the
// user called name. It returns the errors of Authenticator.Authenticate.
func (l *Logins) Login(ctx context.Context, name, plaintext string) (string, error) {
	userID, err := l.authenticator.Authenticate(ctx, name, plaintext)
	if err != nil {
		return "", err
	}

	return l.tokens.Issue(ctx, userID)
}
