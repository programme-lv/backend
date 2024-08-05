package user

import (
	"fmt"

	usergen "github.com/programme-lv/backend/gen/users"
)

var (
	ErrInvalidToken       = usergen.Unauthorized("invalid token")
	ErrInvalidTokenScopes = usergen.Unauthorized("invalid scopes in token")
	ErrMissingScope       = usergen.Unauthorized("missing scope in token")
)

func ErrUsernameExists(username string) usergen.UsernameExistsConflict {
	return usergen.UsernameExistsConflict(fmt.Sprintf("lietotājvārds %v jau eksistē", username))
}

func ErrEmailExists(email string) usergen.EmailExistsConflict {
	return usergen.EmailExistsConflict(fmt.Sprintf("lietotājs ar epastu %v jau eksistē", email))
}
