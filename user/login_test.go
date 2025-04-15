package user_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/programme-lv/backend/user/auth"
	userhttp "github.com/programme-lv/backend/user/http"
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

	w := registerUser(t, userHandler, userData)
	require.Equal(t, http.StatusOK, w.Code, "Registration failed: %s", w.Body.String())

	// Now try to login
	loginData := map[string]interface{}{
		"username": "testuser",
		"password": "password123",
	}

	w = loginUser(t, userHandler, loginData)

	// Check status code
	assert.Equal(t, http.StatusOK, w.Code, "Response body: %s", w.Body.String())

	// Parse the response body
	var responseWrapper struct {
		Status string `json:"status"`
		Data   string `json:"data"` // JWT token is a string
	}

	err := json.Unmarshal(w.Body.Bytes(), &responseWrapper)
	require.NoError(t, err, "Failed to unmarshal response body")

	// Verify response structure
	assert.Equal(t, "success", responseWrapper.Status)
	assert.NotEmpty(t, responseWrapper.Data, "JWT token should not be empty")
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

	w := registerUser(t, userHandler, userData)
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
			w := loginUser(t, userHandler, tc.loginData)
			assertErrorInHttpResponse(t, w, tc.errorCode)
		})
	}
}

// loginUser performs a user login request and returns the response
func loginUser(t *testing.T, handler *userhttp.UserHttpHandler, loginData map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	req, err := createTestRequestJson(http.MethodPost, "/login", loginData)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	handler.Login(w, req)
	return w
}

func TestGetRoleHttp(t *testing.T) {
	userHandler := setupUserHttpHandler(t)

	t.Run("Guest Role for Unauthenticated User", func(t *testing.T) {
		// Request without authentication should return guest role
		req := httptest.NewRequest(http.MethodGet, "/role", nil)
		w := httptest.NewRecorder()
		userHandler.GetRole(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)
		var response struct {
			Status string `json:"status"`
			Data   struct {
				Role string `json:"role"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "success", response.Status)
		assert.Equal(t, "guest", response.Data.Role)
	})

	t.Run("User Role for Regular User", func(t *testing.T) {
		// Create request with user claims directly in context
		req := httptest.NewRequest(http.MethodGet, "/role", nil)
		firstname := "Test"
		lastname := "User"
		uuidStr := uuid.New().String()

		// Create claims with regular user role
		claims := &auth.JwtClaims{
			Username:  "testuser",
			Email:     "test@example.com",
			Firstname: &firstname,
			Lastname:  &lastname,
			UUID:      uuidStr,
		}

		// Add claims to request context
		ctx := context.WithValue(req.Context(), auth.CtxJwtClaimsKey, claims)
		req = req.WithContext(ctx)

		// Call endpoint
		w := httptest.NewRecorder()
		userHandler.GetRole(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)
		var response struct {
			Status string `json:"status"`
			Data   struct {
				Role string `json:"role"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "success", response.Status)
		assert.Equal(t, "user", response.Data.Role)
	})

	t.Run("Admin Role for Admin User", func(t *testing.T) {
		// Create request with admin claims directly in context
		req := httptest.NewRequest(http.MethodGet, "/role", nil)
		firstname := "Admin"
		lastname := "User"
		uuidStr := uuid.New().String()

		// Create claims with admin role
		claims := &auth.JwtClaims{
			Username:  "admin",
			Email:     "admin@example.com",
			Firstname: &firstname,
			Lastname:  &lastname,
			UUID:      uuidStr,
		}

		// Add claims to request context
		ctx := context.WithValue(req.Context(), auth.CtxJwtClaimsKey, claims)
		req = req.WithContext(ctx)

		// Call endpoint
		w := httptest.NewRecorder()
		userHandler.GetRole(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)
		var response struct {
			Status string `json:"status"`
			Data   struct {
				Role string `json:"role"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "success", response.Status)
		assert.Equal(t, "admin", response.Data.Role)
	})
}
