package example

type Token string

type Login struct {
	Name     string
	Password string // want `Login.Password holds a secret as string, use secret.String`
	APIKey   []byte // want `Login.APIKey holds a secret as \[\]byte, use secret.String`
	Token    Token  // want `Login.Token holds a secret as Token, use secret.String`
	password string
	Lifetime int
}

type hidden struct {
	Password string
}

type Signer interface {
	SignIn(name, password string)     // want `Signer.SignIn's password holds a secret as string, use secret.String`
	Issue() (token string, err error) // want `Signer.Issue's token holds a secret as string, use secret.String`
	signOut(token string)
}

type hiddenSigner interface {
	SignIn(name, password string)
}

func SignIn(name, password string) {} // want `SignIn's password holds a secret as string, use secret.String`

func Issue() (token string, err error) { return "", nil } // want `Issue's token holds a secret as string, use secret.String`

func Authenticate(name, plaintext string) {} // want `Authenticate's plaintext holds a secret as string, use secret.String`

func TokenLifetime(tokenCount int) {}

func check(password string) {}
