package http

type CreateUserRequestBody struct {
	Username  *string `json:"username,omitempty"`
	Email     *string `json:"email,omitempty"`
	Firstname *string `json:"firstname,omitempty"`
	Lastname  *string `json:"lastname,omitempty"`
	Password  *string `json:"password,omitempty"`
}

type LoginRequestBody struct {
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
}

type GetUserResponseBody struct {
	UUID      string `json:"uuid"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
}

type CreateUserResponseBody struct {
	UUID      string `json:"uuid"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
}

type QueryCurrentJWTResponseBody struct {
	Username  *string  `json:"username,omitempty"`
	Firstname *string  `json:"firstname,omitempty"`
	Lastname  *string  `json:"lastname,omitempty"`
	Email     *string  `json:"email,omitempty"`
	UUID      *string  `json:"uuid,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
	Issuer    *string  `json:"issuer,omitempty"`
	Subject   *string  `json:"subject,omitempty"`
	Audience  []string `json:"audience,omitempty"`
	ExpiresAt *string  `json:"expires_at,omitempty"`
	IssuedAt  *string  `json:"issued_at,omitempty"`
	NotBefore *string  `json:"not_before,omitempty"`
}
