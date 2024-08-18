package user

type User struct {
	UUID      string
	Username  string
	Email     string
	Firstname *string
	Lastname  *string
}

type JWTClaims struct {
	Username  *string
	Firstname *string
	Lastname  *string
	Email     *string
	UUID      *string
	Scopes    []string
	Issuer    *string
	Subject   *string
	Audience  []string
	ExpiresAt *string
	IssuedAt  *string
	NotBefore *string
}

type LoginPayload struct {
	Username string
	Password string
}

type NotFound string

type QueryCurrentJWTPayload struct {
	Token string
}

type SecureUUIDPayload struct {
	Token string
	UUID  string
}

type UserPayload struct {
	Username  string
	Email     string
	Firstname *string
	Lastname  *string
	Password  string
}

type GetUserByUsernamePayload struct {
	Username string
}
