//go:build integration

package user_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateProfileHttp(t *testing.T) {
	handler := newUserHttpHandler(t)
	token := registerAndLogin(t, handler, "profileuser")

	w := jsonAuthed(t, handler, http.MethodPatch, "/users/me", map[string]interface{}{
		"firstname": "Anna",
		"lastname":  "Bērziņa",
	}, token)
	assert.Equal(t, http.StatusOK, w.Code, "Response body: %s", w.Body.String())

	var response struct {
		Status string `json:"status"`
		Data   struct {
			Username      string  `json:"username"`
			Email         string  `json:"email"`
			Firstname     *string `json:"firstname"`
			Lastname      *string `json:"lastname"`
			EmailVerified bool    `json:"email_verified"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, "success", response.Status)
	assert.Equal(t, "profileuser", response.Data.Username)
	assert.Equal(t, "profileuser@example.com", response.Data.Email)
	require.NotNil(t, response.Data.Firstname)
	assert.Equal(t, "Anna", *response.Data.Firstname)
	require.NotNil(t, response.Data.Lastname)
	assert.Equal(t, "Bērziņa", *response.Data.Lastname)

	w = whoami(t, handler, token)
	assert.Equal(t, http.StatusOK, w.Code, "Response body: %s", w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.NotNil(t, response.Data.Firstname)
	assert.Equal(t, "Anna", *response.Data.Firstname)
	require.NotNil(t, response.Data.Lastname)
	assert.Equal(t, "Bērziņa", *response.Data.Lastname)
}

func TestUpdateProfileHttpGuest(t *testing.T) {
	handler := newUserHttpHandler(t)

	w := jsonAuthed(t, handler, http.MethodPatch, "/users/me", map[string]interface{}{
		"firstname": "Anna",
		"lastname":  "Bērziņa",
	}, "")
	assertErrorInHttpResponse(t, w, "http_unauthorized")
}

func TestUpdateProfileHttpNameTooLong(t *testing.T) {
	handler := newUserHttpHandler(t)
	token := registerAndLogin(t, handler, "namelong")
	tooLong := strings.Repeat("a", 36)

	w := jsonAuthed(t, handler, http.MethodPatch, "/users/me", map[string]interface{}{
		"firstname": tooLong,
		"lastname":  "Bērziņa",
	}, token)
	assertErrorInHttpResponse(t, w, "firstname_too_long")

	w = jsonAuthed(t, handler, http.MethodPatch, "/users/me", map[string]interface{}{
		"firstname": "Anna",
		"lastname":  tooLong,
	}, token)
	assertErrorInHttpResponse(t, w, "lastname_too_long")
}
