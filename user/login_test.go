package user_test

import (
	"encoding/json"
	"net/http"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginHttp(t *testing.T) {
	userHandler := setupUserHttpHandler(t)

	// Register a user first
	userData := map[string]interface{}{
		"username":  "testuser",
		"email":     "test@example.com",
		"firstname": "Test",
		"lastname":  "User",
		"password":  "password123",
	}

	w := register(t, userHandler, userData)
	require.Equal(t, http.StatusOK, w.Code, "Registration failed: %s", w.Body.String())

	// Now try to login
	loginData := map[string]interface{}{
		"username": "testuser",
		"password": "password123",
	}

	w = login(t, userHandler, loginData)

	// Check status code
	assert.Equal(t, http.StatusOK, w.Code, "Response body: %s", w.Body.String())

	// Check for auth_token cookie
	cookies := w.Result().Cookies()
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "auth_token" {
			authCookie = cookie
			break
		}
	}
	require.NotNil(t, authCookie, "No auth_token cookie found in response")
	assert.True(t, authCookie.HttpOnly, "Cookie should be HttpOnly")
	assert.NotEmpty(t, authCookie.Value, "Cookie value should not be empty")

	// Parse the response body
	var responseWrapper struct {
		Status string            `json:"status"`
		Data   map[string]string `json:"data"`
	}

	err := json.Unmarshal(w.Body.Bytes(), &responseWrapper)
	require.NoError(t, err, "Failed to unmarshal response body")

	// Verify response structure
	assert.Equal(t, "success", responseWrapper.Status)
	assert.Equal(t, "Login successful", responseWrapper.Data["message"])
}

func TestLoginHttpInvalidCredentials(t *testing.T) {
	userHandler := setupUserHttpHandler(t)

	// Register a user first
	userData := map[string]interface{}{
		"username":  "testuser",
		"email":     "test@example.com",
		"firstname": "Test",
		"lastname":  "User",
		"password":  "password123",
	}

	w := register(t, userHandler, userData)
	require.Equal(t, http.StatusOK, w.Code, "Registration failed: %s", w.Body.String())

	// Test cases for invalid login attempts
	testCases := []struct {
		name      string
		loginData map[string]interface{}
		errorCode string
	}{
		{
			name: "Wrong Password",
			loginData: map[string]interface{}{
				"username": "testuser",
				"password": "wrongpassword",
			},
			errorCode: "username_or_password_incorrect",
		},
		{
			name: "Non-existent Username",
			loginData: map[string]interface{}{
				"username": "nonexistentuser",
				"password": "password123",
			},
			errorCode: "username_or_password_incorrect",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := login(t, userHandler, tc.loginData)
			assertErrorInHttpResponse(t, w, tc.errorCode)
		})
	}
}
