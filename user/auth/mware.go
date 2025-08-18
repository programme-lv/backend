package auth

import (
	"context"
	"net/http"

	"github.com/programme-lv/backend/common/httpjson"
)

// HttpJwtAllowOnlyAdmins is a middleware that ensures only admin users can access the route
func HttpJwtAllowOnlyAdmins(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(CtxJwtClaimsKey).(*JwtClaims)
		if !ok || claims == nil {
			httpjson.Unauthorized(w, "failed to authenticate user via jwt")
			return
		}
		if claims.Username != "admin" {
			httpjson.Forbidden(w, "access is restricted to admins only")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// HttpJwtAuthentication validates JWT token and adds the claims to the request context
func HttpJwtAuthentication(jwtKey []byte) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		handlerFunc := func(w http.ResponseWriter, r *http.Request) {
			// get token from cookie instead of Authorization header
			cookie, err := r.Cookie("auth_token")
			if err != nil {
				// no cookie found; continue as unauthenticated user
				ctx := context.WithValue(r.Context(), CtxJwtClaimsKey, (*JwtClaims)(nil))
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// jwt cookie found; validate token inside
			token := cookie.Value
			claims, err := ValidateJWT(token, jwtKey)
			if err != nil {
				// invalid token; clear the invalid jwt cookie
				http.SetCookie(w, &http.Cookie{
					Name:     "auth_token",
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				// continue as unauthenticated user
				ctx := context.WithValue(r.Context(), CtxJwtClaimsKey, (*JwtClaims)(nil))
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// add jwt claims to context
			ctx := context.WithValue(r.Context(), CtxJwtClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(handlerFunc)
	}
}
