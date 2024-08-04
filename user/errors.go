package user

import (
	"fmt"

	usergen "github.com/programme-lv/backend/gen/users"
)

func ErrorUsernameAlreadyExists(username string) usergen.UsernameExists {
	return usergen.UsernameExists(mustToJson(map[string]string{
		"lv": fmt.Sprintf("lietotājvārds %s jau eksistē", username),
		"en": fmt.Sprintf("username %s already exists", username),
	}))
}

var (
	ErrInvalidToken       = usergen.Unauthorized("invalid token")
	ErrInvalidTokenScopes = usergen.Unauthorized("invalid scopes in token")
	ErrMissingScope       = usergen.Unauthorized("missing scope in token")
)
