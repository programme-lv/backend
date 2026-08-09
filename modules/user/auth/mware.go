package auth

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/programme-lv/backend/common/jsonresp"
)

// HttpAllowOnlyAdmins allows requests authenticated either by an admin JWT or
// by the server-to-server admin API key.
func HttpAllowOnlyAdmins(adminAPIKey []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if IsAdmin(r.Context()) || hasAdminAPIKey(r, adminAPIKey) {
				next.ServeHTTP(w, r)
				return
			}

			claims, _ := r.Context().Value(CtxJwtClaimsKey).(*JwtClaims)
			if claims == nil {
				jsonresp.Unauthorized(w, "admin authentication required")
				return
			}
			jsonresp.Forbidden(w, "access is restricted to admins only")
		})
	}
}

func hasAdminAPIKey(r *http.Request, adminAPIKey []byte) bool {
	const bearerPrefix = "Bearer "

	authorization := r.Header.Get("Authorization")
	if len(adminAPIKey) == 0 || !strings.HasPrefix(authorization, bearerPrefix) {
		return false
	}
	provided := []byte(strings.TrimPrefix(authorization, bearerPrefix))
	return subtle.ConstantTimeCompare(provided, adminAPIKey) == 1
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
