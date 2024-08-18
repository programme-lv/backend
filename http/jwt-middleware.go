package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/golang-jwt/jwt/v5/request"
	"github.com/programme-lv/backend/auth"
)

func getJwtAuthMiddleware(jwtKey []byte) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			token, err := request.BearerExtractor{}.ExtractToken(r)
			if err != nil {
				if errors.Is(err, request.ErrNoTokenInRequest) {
					ctx := context.WithValue(r.Context(), auth.CtxJwtClaimsKey, (*auth.JwtClaims)(nil))
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			claims, err := auth.ValidateJWT(token, jwtKey)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), auth.CtxJwtClaimsKey, (*auth.JwtClaims)(claims))
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(hfn)
	}
}
