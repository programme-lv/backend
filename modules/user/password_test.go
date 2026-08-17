//go:build integration

package user_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangePasswordHttp(t *testing.T) {
	handler := newUserHttpHandler(t)
	oldToken := registerAndLogin(t, handler, "pwduser")

	time.Sleep(time.Second)

	w := jsonAuthed(t, handler, http.MethodPost, "/password", map[string]interface{}{
		"current_password": "password123",
		"password":         "newpassword1",
	}, oldToken)
	assert.Equal(t, http.StatusNoContent, w.Code, "Response body: %s", w.Body.String())
	newToken := authCookieValue(t, w)
	assert.NotEqual(t, oldToken, newToken)

	w = whoami(t, handler, newToken)
	assert.Equal(t, http.StatusOK, w.Code, "Response body: %s", w.Body.String())
	assertWhoAmIUsername(t, w, "pwduser")

	w = whoami(t, handler, oldToken)
	assert.Equal(t, http.StatusOK, w.Code, "Response body: %s", w.Body.String())
	assertWhoAmIGuest(t, w)

	w = login(t, handler, map[string]interface{}{
		"username": "pwduser",
		"password": "newpassword1",
	})
	assert.Equal(t, http.StatusOK, w.Code, "Response body: %s", w.Body.String())

	w = login(t, handler, map[string]interface{}{
		"username": "pwduser",
		"password": "password123",
	})
	assertErrorInHttpResponse(t, w, "username_or_password_incorrect")
}

func TestChangePasswordHttpWrongCurrent(t *testing.T) {
	handler := newUserHttpHandler(t)
	token := registerAndLogin(t, handler, "pwdwrong")

	w := jsonAuthed(t, handler, http.MethodPost, "/password", map[string]interface{}{
		"current_password": "not-the-password",
		"password":         "newpassword1",
	}, token)
	assertErrorInHttpResponse(t, w, "username_or_password_incorrect")
}

func TestChangePasswordHttpGuest(t *testing.T) {
	handler := newUserHttpHandler(t)

	w := jsonAuthed(t, handler, http.MethodPost, "/password", map[string]interface{}{
		"current_password": "password123",
		"password":         "newpassword1",
	}, "")
	assertErrorInHttpResponse(t, w, "http_unauthorized")
}

func TestChangePasswordHttpShortNew(t *testing.T) {
	handler := newUserHttpHandler(t)
	token := registerAndLogin(t, handler, "pwdshort")

	w := jsonAuthed(t, handler, http.MethodPost, "/password", map[string]interface{}{
		"current_password": "password123",
		"password":         "short",
	}, token)
	assertErrorInHttpResponse(t, w, "password_too_short")
}

func assertWhoAmIUsername(t *testing.T, w *httptest.ResponseRecorder, username string) {
	t.Helper()
	var response struct {
		Status string          `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, "success", response.Status)

	var user struct {
		Username string `json:"username"`
	}
	require.NoError(t, json.Unmarshal(response.Data, &user))
	assert.Equal(t, username, user.Username)
}

func assertWhoAmIGuest(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	var response struct {
		Status string          `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, "success", response.Status)
	assert.True(t, string(response.Data) == "null" || len(response.Data) == 0)
}
