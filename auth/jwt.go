package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JwtClaims struct {
	Username  string   `json:"username,omitempty"`
	Firstname *string  `json:"firstname,omitempty"`
	Lastname  *string  `json:"lastname,omitempty"`
	Email     string   `json:"email,omitempty"`
	UUID      string   `json:"uuid,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

type ClaimsKeyType string

var CtxJwtClaimsKey ClaimsKeyType = "jwtClaims"

func GenerateJWT(username, email string, uuid uuid.UUID, firstname, lastname *string, jwtKey []byte) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &JwtClaims{
		Username:         username,
		Firstname:        firstname,
		Lastname:         lastname,
		Email:            email,
		UUID:             uuid.String(),
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(expirationTime)},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func ValidateJWT(tokenStr string, jwtKey []byte) (*JwtClaims, error) {
	claims := &JwtClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return nil, errors.New("invalid token signature")
		}
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
