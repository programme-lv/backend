//go:build integration

package user_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/testutil"
	"github.com/programme-lv/backend/modules/user"
	userhttp "github.com/programme-lv/backend/modules/user/http"
	"github.com/programme-lv/backend/modules/user/mail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUserHttpHandler(t *testing.T) http.Handler {
	t.Helper()
	pg := testutil.MustGetMigratedTestPostgresDb(t)
	userSrvc := user.NewUserService(pg, mail.NewNoopMailer(), user.EmailFlowConfig{
		WebsiteBaseURL:  "http://localhost:3000",
		ResetTokenTTL:   time.Hour,
		VerifyTokenTTL:  24 * time.Hour,
		PerUserCooldown: 5 * time.Minute,
	})
	userHandler := userhttp.NewUserHttpHandler(
		userSrvc,
		[]byte("test"),
		userhttp.WithSecureCookie(true),
	)
	r := chi.NewRouter()
	userHandler.RegisterRoutes(r)
	return r
}

func newJsonReq(method, path string, body map[string]interface{}) (*http.Request, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req := httptest.NewRequest(method, path, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func register(t *testing.T, handler http.Handler, userData map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	req, err := newJsonReq(http.MethodPost, "/users", userData)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func assertErrorInHttpResponse(t *testing.T, w *httptest.ResponseRecorder, expectedCode string) {
	t.Helper()
	assert.NotEqual(t, http.StatusOK, w.Code, "Expected error status code")

	var errorResponse struct {
		Status  string `json:"status"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResponse))
	assert.Equal(t, "error", errorResponse.Status)
	assert.Equal(t, expectedCode, errorResponse.Code)
	assert.NotEmpty(t, errorResponse.Message)
}

func login(t *testing.T, handler http.Handler, loginData map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	req, err := newJsonReq(http.MethodPost, "/login", loginData)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func jsonAuthed(t *testing.T, handler http.Handler, method, path string, body map[string]interface{}, token string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	var err error
	if body == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req, err = newJsonReq(method, path, body)
		require.NoError(t, err)
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func authCookieValue(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "auth_token" && cookie.Value != "" {
			return cookie.Value
		}
	}
	t.Fatal("No auth_token cookie found in response")
	return ""
}

func whoami(t *testing.T, handler http.Handler, token string) *httptest.ResponseRecorder {
	t.Helper()
	return jsonAuthed(t, handler, http.MethodGet, "/whoami", nil, token)
}

func registerAndLogin(t *testing.T, userHttpHandler http.Handler, username string) string {
	t.Helper()
	userData := map[string]interface{}{
		"username":  username,
		"email":     username + "@example.com",
		"firstname": "Test",
		"lastname":  "User",
		"password":  "password123",
	}
	w := register(t, userHttpHandler, userData)
	require.Equal(t, http.StatusOK, w.Code)

	w = login(t, userHttpHandler, map[string]interface{}{
		"username": username,
		"password": "password123",
	})
	require.Equal(t, http.StatusOK, w.Code)

	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "auth_token" {
			return cookie.Value
		}
	}
	t.Fatal("No auth_token cookie found in response")
	return ""
}
