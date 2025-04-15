package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-jwt/jwt/v5/request"
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

// GetJwtAuthMiddleware validates JWT token and adds the claims to the request context
func GetJwtAuthMiddleware(jwtKey []byte) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			token, err := request.BearerExtractor{}.ExtractToken(r)
			if err != nil {
				if errors.Is(err, request.ErrNoTokenInRequest) {
					ctx := context.WithValue(r.Context(), CtxJwtClaimsKey, (*JwtClaims)(nil))
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			claims, err := ValidateJWT(token, jwtKey)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), CtxJwtClaimsKey, (*JwtClaims)(claims))
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(hfn)
	}
}
