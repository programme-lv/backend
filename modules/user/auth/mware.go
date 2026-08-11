package auth

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
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

// PasswordChangedChecker returns when the user's password was last changed.
// Returns nil if the password has never been changed or user doesn't exist.
type PasswordChangedChecker func(ctx context.Context, userUUID uuid.UUID) (*time.Time, error)

// JwtAuthOptions configures HttpJwtAuthentication behavior.
type JwtAuthOptions struct {
	CookieSecure      bool
	PwdChangedChecker PasswordChangedChecker
}

// HttpJwtAuthentication validates JWT token and adds the claims to the request context.
// Optional cookieSecure parameter controls the Secure flag when clearing invalid tokens.
func HttpJwtAuthentication(jwtKey []byte, cookieSecure ...bool) func(next http.Handler) http.Handler {
	secure := true
	if len(cookieSecure) > 0 {
		secure = cookieSecure[0]
	}
	return HttpJwtAuthenticationWithOptions(jwtKey, JwtAuthOptions{CookieSecure: secure})
}

// HttpJwtAuthenticationWithOptions validates JWT token with extended options.
func HttpJwtAuthenticationWithOptions(jwtKey []byte, opts JwtAuthOptions) func(next http.Handler) http.Handler {
	clearCookie := func(w http.ResponseWriter) {
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   opts.CookieSecure,
		})
	}
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
				clearCookie(w)
				// continue as unauthenticated user
				ctx := context.WithValue(r.Context(), CtxJwtClaimsKey, (*JwtClaims)(nil))
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// check if password was changed after token was issued
			if opts.PwdChangedChecker != nil && claims.IssuedAt != nil {
				userUUID, parseErr := uuid.Parse(claims.UUID)
				if parseErr == nil {
					pwdChangedAt, checkErr := opts.PwdChangedChecker(r.Context(), userUUID)
					if checkErr == nil && pwdChangedAt != nil && pwdChangedAt.After(claims.IssuedAt.Time) {
						clearCookie(w)
						ctx := context.WithValue(r.Context(), CtxJwtClaimsKey, (*JwtClaims)(nil))
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}

			// add jwt claims to context
			ctx := context.WithValue(r.Context(), CtxJwtClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(handlerFunc)
	}
}
