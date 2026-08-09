package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHttpAllowOnlyAdmins(t *testing.T) {
	adminAPIKey := []byte("a-long-random-admin-api-key")

	tests := []struct {
		name          string
		claims        *JwtClaims
		authorization string
		wantStatus    int
	}{
		{
			name:       "admin JWT",
			claims:     &JwtClaims{Username: "admin"},
			wantStatus: http.StatusNoContent,
		},
		{
			name:          "admin API key",
			authorization: "Bearer a-long-random-admin-api-key",
			wantStatus:    http.StatusNoContent,
		},
		{
			name:          "wrong API key",
			authorization: "Bearer wrong-key",
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "wrong authorization scheme",
			authorization: "Basic a-long-random-admin-api-key",
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:       "non-admin JWT",
			claims:     &JwtClaims{Username: "user"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "no authentication",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})
			handler := HttpAllowOnlyAdmins(adminAPIKey)(next)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tt.authorization)
			ctx := context.WithValue(req.Context(), CtxJwtClaimsKey, tt.claims)
			res := httptest.NewRecorder()

			handler.ServeHTTP(res, req.WithContext(ctx))

			assert.Equal(t, tt.wantStatus, res.Code)
		})
	}
}

func TestHttpAllowOnlyAdminsRejectsEmptyConfiguredKey(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := HttpAllowOnlyAdmins(nil)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	assert.Equal(t, http.StatusUnauthorized, res.Code)
}
