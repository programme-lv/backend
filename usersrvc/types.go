package usersrvc

import "github.com/google/uuid"

type User struct {
	UUID      uuid.UUID
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
