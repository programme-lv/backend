package auth

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/jsonresp"
)

// PasswordChangedAtLookup returns when the user's password was last changed.
type PasswordChangedAtLookup func(ctx context.Context, userUUID uuid.UUID) (time.Time, error)

type jwtAuthConfig struct {
	cookieSecure       bool
	passwordChangedAt  PasswordChangedAtLookup
}

type JwtAuthOption func(*jwtAuthConfig)

func WithSecureCookie(secure bool) JwtAuthOption {
	return func(c *jwtAuthConfig) {
		c.cookieSecure = secure
	}
}

func WithPasswordChangedAtLookup(lookup PasswordChangedAtLookup) JwtAuthOption {
	return func(c *jwtAuthConfig) {
		c.passwordChangedAt = lookup
	}
}

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

// HttpJwtAuthentication validates JWT token and adds the claims to the request context.
// Pass WithSecureCookie and optionally WithPasswordChangedAtLookup.
func HttpJwtAuthentication(jwtKey []byte, opts ...JwtAuthOption) func(next http.Handler) http.Handler {
	cfg := jwtAuthConfig{cookieSecure: true}
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(next http.Handler) http.Handler {
		handlerFunc := func(w http.ResponseWriter, r *http.Request) {
			clearAndGuest := func() {
				http.SetCookie(w, &http.Cookie{
					Name:     "auth_token",
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
					Secure:   cfg.cookieSecure,
				})
				ctx := context.WithValue(r.Context(), CtxJwtClaimsKey, (*JwtClaims)(nil))
				next.ServeHTTP(w, r.WithContext(ctx))
			}

			cookie, err := r.Cookie("auth_token")
			if err != nil {
				ctx := context.WithValue(r.Context(), CtxJwtClaimsKey, (*JwtClaims)(nil))
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			claims, err := ValidateJWT(cookie.Value, jwtKey)
			if err != nil {
				clearAndGuest()
				return
			}

			if cfg.passwordChangedAt != nil && claims.UUID != "" {
				userUUID, parseErr := uuid.Parse(claims.UUID)
				if parseErr != nil {
					clearAndGuest()
					return
				}
				changedAt, lookupErr := cfg.passwordChangedAt(r.Context(), userUUID)
				if lookupErr != nil {
					slog.Error("lookup password changed at", "error", lookupErr, "uuid", claims.UUID)
					clearAndGuest()
					return
				}
				if claims.IssuedAt == nil || claims.IssuedAt.Time.Unix() < changedAt.Unix() {
					clearAndGuest()
					return
				}
			}

			ctx := context.WithValue(r.Context(), CtxJwtClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(handlerFunc)
	}
}
