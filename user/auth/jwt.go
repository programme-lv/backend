package auth

import (
	"context"
	"errors"
	"net/http"
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

func GenerateJWT(username, email string, uuid uuid.UUID, firstname, lastname *string, jwtKey []byte, validFor time.Duration) (string, error) {
	expirationTime := time.Now().Add(validFor)

	claims := &JwtClaims{
		Username:         username,
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
			// Get token from cookie instead of Authorization header
			cookie, err := r.Cookie("auth_token")
			if err != nil {
				// No cookie found, continue as unauthenticated user
				ctx := context.WithValue(r.Context(), CtxJwtClaimsKey, (*JwtClaims)(nil))
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			token := cookie.Value
			claims, err := ValidateJWT(token, jwtKey)
			if err != nil {
				// Invalid token, continue as unauthenticated user
				// Optionally, clear the invalid cookie
				http.SetCookie(w, &http.Cookie{
					Name:     "auth_token",
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				ctx := context.WithValue(r.Context(), CtxJwtClaimsKey, (*JwtClaims)(nil))
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			ctx := context.WithValue(r.Context(), CtxJwtClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(hfn)
	}
}
