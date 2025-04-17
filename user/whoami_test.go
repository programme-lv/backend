package user_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhoAmIHttpAuthenticated(t *testing.T) {
	userHandler := newUserHttpHandler(t)

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

	// Login to get auth token
	loginData := map[string]interface{}{
		"username": "testuser",
		"password": "password123",
	}

	w = login(t, userHandler, loginData)
	require.Equal(t, http.StatusOK, w.Code, "Login failed: %s", w.Body.String())

	// Extract auth token from cookie
	cookies := w.Result().Cookies()
	var authToken string
	for _, cookie := range cookies {
		if cookie.Name == "auth_token" {
			authToken = cookie.Value
			break
		}
	}
	require.NotEmpty(t, authToken, "No auth_token cookie found in response")

	// Make whoami request with auth token
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: authToken,
	})

	w = httptest.NewRecorder()
	userHandler.ServeHTTP(w, req)

	// Check status code
	assert.Equal(t, http.StatusOK, w.Code, "Response body: %s", w.Body.String())

	// Parse the response body
	var responseWrapper struct {
		Status string          `json:"status"`
		Data   json.RawMessage `json:"data"`
	}

	err := json.Unmarshal(w.Body.Bytes(), &responseWrapper)
	require.NoError(t, err, "Failed to unmarshal response body")

	// Verify response structure
	assert.Equal(t, "success", responseWrapper.Status)

	// Parse the user data
	var userData2 struct {
		UUID      string  `json:"uuid"`
		Username  string  `json:"username"`
		Email     string  `json:"email"`
		Firstname *string `json:"firstname"`
		Lastname  *string `json:"lastname"`
	}

	err = json.Unmarshal(responseWrapper.Data, &userData2)
	require.NoError(t, err, "Failed to unmarshal user data")

	// Verify user data
	assert.Equal(t, "testuser", userData2.Username)
	assert.Equal(t, "test@example.com", userData2.Email)
	assert.NotNil(t, userData2.Firstname)
	assert.Equal(t, "Test", *userData2.Firstname)
	assert.NotNil(t, userData2.Lastname)
	assert.Equal(t, "User", *userData2.Lastname)
	assert.NotEmpty(t, userData2.UUID)
}

func TestWhoAmIHttpUnauthenticated(t *testing.T) {
	userHandler := newUserHttpHandler(t)

	// Make whoami request without auth token
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	w := httptest.NewRecorder()
	userHandler.ServeHTTP(w, req)

	// Check that the status code is not OK
	assert.NotEqual(t, http.StatusOK, w.Code, "Expected error status code")

	// Parse the error response
	var errorResponse struct {
		Status  string `json:"status"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(t, err, "Failed to unmarshal error response body")

	// Check error response fields
	assert.Equal(t, "error", errorResponse.Status, "Expected status to be 'error'")
	assert.Equal(t, "Internal Server Error", errorResponse.Message)
}
