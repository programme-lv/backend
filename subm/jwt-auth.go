package subm

import (
	"context"
	"fmt"

	"github.com/programme-lv/backend/auth"
	submgen "github.com/programme-lv/backend/gen/submissions"
	"goa.design/goa/v3/security"
)

var (
	ErrInvalidToken       = submgen.Unauthorized("invalid token")
	ErrInvalidTokenScopes = submgen.Unauthorized("invalid scopes in token")
	ErrMissingScope       = submgen.Unauthorized("missing scope in token")
)

type ClaimsKey string

func (s *submissionssrvc) JWTAuth(ctx context.Context, token string, scheme *security.JWTScheme) (context.Context, error) {
	claims, err := auth.ValidateJWT(token, s.jwtKey)
	if err != nil {
		fmt.Println(err)
		return ctx, ErrInvalidToken
	}

	scopesInToken := claims.Scopes

	if err := scheme.Validate(scopesInToken); err != nil {
		fmt.Println("invalid scopes in token")
		return ctx, ErrMissingScope
	}

	ctx = context.WithValue(ctx, ClaimsKey("claims"), claims)
	return ctx, nil
}
