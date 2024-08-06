package user

import (
	"context"

	"github.com/programme-lv/backend/auth"
	usergen "github.com/programme-lv/backend/gen/users"
)

// QueryCurrentJWT implements users.Service.
func (s *userssrvc) QueryCurrentJWT(ctx context.Context, p *usergen.QueryCurrentJWTPayload) (res *usergen.JWTClaims, err error) {
	claims := ctx.Value(ClaimsKey("claims")).(*auth.Claims)

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

	res = &usergen.JWTClaims{
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
