//go:build integration

package user_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func insertLegacyUser(t *testing.T, pg *pgxpool.Pool, username, email, password string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	_, err = pg.Exec(context.Background(), `
		INSERT INTO users (firstname, lastname, username, email, bcrypt_pwd)
		VALUES ('Legacy', 'User', $1, $2, $3)
	`, username, email, string(hash))
	require.NoError(t, err)
}

func loginResponseUsername(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	require.Equal(t, http.StatusOK, w.Code, "Response body: %s", w.Body.String())

	var response struct {
		Status string `json:"status"`
		Data   struct {
			Username string `json:"username"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "success", response.Status)
	return response.Data.Username
}

func TestRegisterHttpRejectsAtSignInUsername(t *testing.T) {
	userHandler := newUserHttpHandler(t)

	w := register(t, userHandler, map[string]interface{}{
		"username":  "user@example.com",
		"email":     "user@example.net",
		"firstname": "Test",
		"lastname":  "User",
		"password":  "password123",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertErrorInHttpResponse(t, w, "username_contains_at")
}

func TestRegisterHttpAllowsExistingUsernameSyntax(t *testing.T) {
	userHandler := newUserHttpHandler(t)

	w := register(t, userHandler, map[string]interface{}{
		"username":  "User.Name-123_ok",
		"email":     "syntax@example.com",
		"firstname": "Test",
		"lastname":  "User",
		"password":  "password123",
	})
	require.Equal(t, http.StatusOK, w.Code, "Registration failed: %s", w.Body.String())

	w = login(t, userHandler, map[string]interface{}{
		"username": "User.Name-123_ok",
		"password": "password123",
	})
	assert.Equal(t, "User.Name-123_ok", loginResponseUsername(t, w))
}

func TestLoginWithEmailIsCaseInsensitive(t *testing.T) {
	userHandler := newUserHttpHandler(t)

	w := register(t, userHandler, map[string]interface{}{
		"username":  "emailuser",
		"email":     "Email.User@Example.COM",
		"firstname": "Test",
		"lastname":  "User",
		"password":  "password123",
	})
	require.Equal(t, http.StatusOK, w.Code, "Registration failed: %s", w.Body.String())

	w = login(t, userHandler, map[string]interface{}{
		"username": "email.user@example.com",
		"password": "password123",
	})
	assert.Equal(t, "emailuser", loginResponseUsername(t, w))
}

func TestLoginUnknownEmailAndWrongEmailPasswordAreGeneric(t *testing.T) {
	userHandler := newUserHttpHandler(t)

	w := register(t, userHandler, map[string]interface{}{
		"username":  "genericuser",
		"email":     "generic@example.com",
		"firstname": "Test",
		"lastname":  "User",
		"password":  "password123",
	})
	require.Equal(t, http.StatusOK, w.Code, "Registration failed: %s", w.Body.String())

	testCases := []struct {
		name      string
		loginData map[string]interface{}
	}{
		{
			name: "Wrong Password For Email",
			loginData: map[string]interface{}{
				"username": "generic@example.com",
				"password": "wrongpassword",
			},
		},
		{
			name: "Unknown Email",
			loginData: map[string]interface{}{
				"username": "missing@example.com",
				"password": "password123",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := login(t, userHandler, tc.loginData)
			assertErrorInHttpResponse(t, w, "username_or_password_incorrect")
		})
	}
}

func TestLoginLegacyAtUsername(t *testing.T) {
	userHandler, pg := newUserHttpHandlerWithPool(t)
	insertLegacyUser(t, pg, "legacy@example.com", "legacy-account@example.net", "legacy-pass-1")

	w := login(t, userHandler, map[string]interface{}{
		"username": "legacy@example.com",
		"password": "legacy-pass-1",
	})
	assert.Equal(t, "legacy@example.com", loginResponseUsername(t, w))
}

func TestLoginLegacyAtUsernameAndEmailDistinctPasswords(t *testing.T) {
	userHandler, pg := newUserHttpHandlerWithPool(t)
	insertLegacyUser(t, pg, "shadow@example.com", "shadow-legacy@example.net", "legacy-pass-1")

	w := register(t, userHandler, map[string]interface{}{
		"username":  "shadowemail",
		"email":     "shadow@example.com",
		"firstname": "Test",
		"lastname":  "User",
		"password":  "email-pass-1",
	})
	require.Equal(t, http.StatusOK, w.Code, "Registration failed: %s", w.Body.String())

	w = login(t, userHandler, map[string]interface{}{
		"username": "shadow@example.com",
		"password": "legacy-pass-1",
	})
	assert.Equal(t, "shadow@example.com", loginResponseUsername(t, w))

	w = login(t, userHandler, map[string]interface{}{
		"username": "shadow@example.com",
		"password": "email-pass-1",
	})
	assert.Equal(t, "shadowemail", loginResponseUsername(t, w))
}

func TestLoginAmbiguousIdentifierSharedPassword(t *testing.T) {
	userHandler, pg := newUserHttpHandlerWithPool(t)
	insertLegacyUser(t, pg, "shared@example.com", "shared-legacy@example.net", "same-pass-1")

	w := register(t, userHandler, map[string]interface{}{
		"username":  "sharedemail",
		"email":     "shared@example.com",
		"firstname": "Test",
		"lastname":  "User",
		"password":  "same-pass-1",
	})
	require.Equal(t, http.StatusOK, w.Code, "Registration failed: %s", w.Body.String())

	w = login(t, userHandler, map[string]interface{}{
		"username": "shared@example.com",
		"password": "same-pass-1",
	})
	assertErrorInHttpResponse(t, w, "username_or_password_incorrect")
}

func TestLoginDuplicateEmailCaseDistinctPasswords(t *testing.T) {
	userHandler := newUserHttpHandler(t)

	for _, userData := range []map[string]interface{}{
		{
			"username":  "dupone",
			"email":     "dup@example.com",
			"firstname": "Test",
			"lastname":  "User",
			"password":  "dup-one-pass",
		},
		{
			"username":  "duptwo",
			"email":     "DUP@example.com",
			"firstname": "Test",
			"lastname":  "User",
			"password":  "dup-two-pass",
		},
	} {
		w := register(t, userHandler, userData)
		require.Equal(t, http.StatusOK, w.Code, "Registration failed: %s", w.Body.String())
	}

	w := login(t, userHandler, map[string]interface{}{
		"username": "Dup@Example.com",
		"password": "dup-one-pass",
	})
	assert.Equal(t, "dupone", loginResponseUsername(t, w))

	w = login(t, userHandler, map[string]interface{}{
		"username": "Dup@Example.com",
		"password": "dup-two-pass",
	})
	assert.Equal(t, "duptwo", loginResponseUsername(t, w))
}

func TestLoginDuplicateEmailCaseSharedPassword(t *testing.T) {
	userHandler := newUserHttpHandler(t)

	for _, userData := range []map[string]interface{}{
		{
			"username":  "shareddupone",
			"email":     "shareddup@example.com",
			"firstname": "Test",
			"lastname":  "User",
			"password":  "same-dup-pass",
		},
		{
			"username":  "sharedduptwo",
			"email":     "SharedDup@example.com",
			"firstname": "Test",
			"lastname":  "User",
			"password":  "same-dup-pass",
		},
	} {
		w := register(t, userHandler, userData)
		require.Equal(t, http.StatusOK, w.Code, "Registration failed: %s", w.Body.String())
	}

	w := login(t, userHandler, map[string]interface{}{
		"username": "SHAREDDUP@example.com",
		"password": "same-dup-pass",
	})
	assertErrorInHttpResponse(t, w, "username_or_password_incorrect")
}
