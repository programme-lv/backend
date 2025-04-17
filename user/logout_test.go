package user_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogout(t *testing.T) {
	userHandler := newUserHttpHandler(t)

	// Register and login a user to get a token
	token := registerAndLogin(t, userHandler, "testuser")

	// Create a request to logout
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)

	// Add auth token cookie
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: token,
	})

	w := httptest.NewRecorder()
	userHandler.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Check that the cookie was cleared
	cookies := w.Result().Cookies()
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "auth_token" {
			authCookie = cookie
			break
		}
	}
	require.NotNil(t, authCookie, "No auth_token cookie found in response")
	assert.Empty(t, authCookie.Value, "Cookie value should be empty")
	assert.True(t, authCookie.MaxAge < 0, "Cookie should be set to expire")
}
