package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JwtClaims struct {
	Username string   `json:"username,omitempty"`
	UUID     string   `json:"uuid,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

type ClaimsKeyType string

var CtxJwtClaimsKey ClaimsKeyType = "jwtClaims"

func GenerateJWT(username, email string, uuid uuid.UUID, jwtKey []byte, validFor time.Duration) (string, error) {
	now := time.Now()
	expirationTime := now.Add(validFor)

	claims := &JwtClaims{
		Username: username,
		UUID:     uuid.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func ValidateJWT(tokenStr string, jwtKey []byte) (*JwtClaims, error) {
	claims := &JwtClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected JWT signing method")
		}
		return jwtKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			return nil, errors.New("invalid token signature")
		}
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

var ErrNoJwtClaims = errors.New("no jwt claims found in context")
var ErrEmptyJwtClaims = errors.New("empty jwt claims found in context")

func GetUserUuidFromCtx(ctx context.Context) (uuid.UUID, error) {
	claims, ok := ctx.Value(CtxJwtClaimsKey).(*JwtClaims)
	if !ok {
		return uuid.Nil, ErrNoJwtClaims
	}
	if claims == nil {
		return uuid.Nil, ErrEmptyJwtClaims
	}
	u, err := uuid.Parse(claims.UUID)
	if err != nil {
		return uuid.Nil, err
	}
	if u == uuid.Nil {
		return uuid.Nil, ErrEmptyJwtClaims
	}
	return u, nil
}

func IsAdmin(ctx context.Context) bool {
	claims, ok := ctx.Value(CtxJwtClaimsKey).(*JwtClaims)
	if !ok || claims == nil {
		return false
	}
	return claims.Username == "admin"
}
