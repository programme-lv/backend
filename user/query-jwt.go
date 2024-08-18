package user

import (
	"context"
	"fmt"

	"github.com/programme-lv/backend/auth"
)

type JwtClaims struct {
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

// QueryCurrentJWT implements users.Service.
func (s *UserService) QueryCurrentJWT(ctx context.Context) (res *JwtClaims, err error) {
	claims := ctx.Value(auth.CtxJwtClaimsKey).(*auth.JwtClaims)
	if claims == nil {
		return nil, fmt.Errorf("no claims found in context")
	}

	var expiresAt *string = nil
	if claims.ExpiresAt != nil {
		expiresAt = new(string)
		*expiresAt = claims.ExpiresAt.String()
	}

	var issuedAt *string = nil
	if claims.IssuedAt != nil {
		issuedAt = new(string)
		*issuedAt = claims.IssuedAt.String()
	}

	var notBefore *string = nil
	if claims.NotBefore != nil {
		notBefore = new(string)
		*notBefore = claims.NotBefore.String()
	}

	res = &JwtClaims{
		Username:  &claims.Username,
		Firstname: claims.Firstname,
		Lastname:  claims.Lastname,
		Email:     &claims.Email,
		UUID:      &claims.UUID,
		Scopes:    claims.Scopes,
		Issuer:    &claims.Issuer,
		Subject:   &claims.Subject,
		Audience:  claims.Audience,
		ExpiresAt: expiresAt,
		IssuedAt:  issuedAt,
		NotBefore: notBefore,
	}

	return res, nil
}
