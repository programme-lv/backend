package user_test

import (
	"encoding/json"
	"net/http"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterHttp(t *testing.T) {
	userHandler := setupUserHttpHandler(t)

	userData := map[string]interface{}{
		"username":  "testuser",
		"email":     "test@example.com",
		"firstname": "Test",
		"lastname":  "User",
		"password":  "password123",
	}

	w := register(t, userHandler, userData)

	// Check status code
	assert.Equal(t, http.StatusOK, w.Code, "Response body: %s", w.Body.String())

	// Parse the response body
	var responseWrapper struct {
		Status string                 `json:"status"`
		Data   map[string]interface{} `json:"data"`
	}

	err := json.Unmarshal(w.Body.Bytes(), &responseWrapper)
	require.NoError(t, err, "Failed to unmarshal response body")

	// Verify response structure and content
	assert.Equal(t, "success", responseWrapper.Status)
	assert.Contains(t, responseWrapper.Data, "uuid")
	assert.Equal(t, "testuser", responseWrapper.Data["username"])
	assert.Equal(t, "test@example.com", responseWrapper.Data["email"])
}

func TestRegisterHttpDuplicateUsername(t *testing.T) {
	userHandler := setupUserHttpHandler(t)

	// Create first user
	firstUserData := map[string]interface{}{
		"username":  "testuser",
		"email":     "test@example.com",
		"firstname": "Test",
		"lastname":  "User",
		"password":  "password123",
	}

	w := register(t, userHandler, firstUserData)
	require.Equal(t, http.StatusOK, w.Code, "First registration failed: %s", w.Body.String())

	// Try to register a second user with the same username
	secondUserData := map[string]interface{}{
		"username":  "testuser", // Same username
		"email":     "different@example.com",
		"firstname": "Another",
		"lastname":  "User",
		"password":  "password456",
	}

	w = register(t, userHandler, secondUserData)
	assertErrorInHttpResponse(t, w, "username_exists")
}

func TestRegisterHttpDuplicateEmail(t *testing.T) {
	userHandler := setupUserHttpHandler(t)

	// Create first user
	firstUserData := map[string]interface{}{
		"username":  "firstuser",
		"email":     "test@example.com", // We'll reuse this email
		"firstname": "Test",
		"lastname":  "User",
		"password":  "password123",
	}

	w := register(t, userHandler, firstUserData)
	require.Equal(t, http.StatusOK, w.Code, "First registration failed: %s", w.Body.String())

	// Try to register a second user with the same email
	secondUserData := map[string]interface{}{
		"username":  "seconduser",       // Different username
		"email":     "test@example.com", // Same email
		"firstname": "Another",
		"lastname":  "User",
		"password":  "password456",
	}

	w = register(t, userHandler, secondUserData)
	assertErrorInHttpResponse(t, w, "email_exists")
}
