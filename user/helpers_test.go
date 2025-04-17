package user_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	"github.com/programme-lv/backend/user"
	userhttp "github.com/programme-lv/backend/user/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPgDb(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	conf := pgtestdb.Config{
		DriverName: "pgx",
		User:       "proglv", // local dev pg user
		Password:   "proglv", // local dev pg password
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}
	gm := golangmigrator.New("../migrate")
	config := pgtestdb.Custom(t, conf, gm)

	pool, err := pgxpool.New(ctx, config.URL())
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func newUserHttpHandler(t *testing.T) http.Handler {
	pg := newTestPgDb(t)
	userSrvc := user.NewUserService(pg)
	userHandler := userhttp.NewUserHttpHandler(userSrvc, []byte("test"))
	chi := chi.NewRouter()
	userHandler.RegisterRoutes(chi)
	return chi
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
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func assertErrorInHttpResponse(t *testing.T, w *httptest.ResponseRecorder, expectedCode string) {
	t.Helper()

	// Check the response status code is not OK
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
	assert.Equal(t, expectedCode, errorResponse.Code, "Incorrect error code")
	assert.NotEmpty(t, errorResponse.Message, "Expected non-empty error message")
}

// login performs a user login request and returns the response
func login(t *testing.T, handler http.Handler, loginData map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	req, err := newJsonReq(http.MethodPost, "/login", loginData)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

// getRole sends a request to the role endpoint and returns the role from the response
func getRole(t *testing.T, handler http.Handler, token string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/role", nil)

	// Add token as cookie instead of Authorization header
	if token != "" {
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: token,
		})
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var response struct {
		Status string `json:"status"`
		Data   struct {
			Role string `json:"role"`
		} `json:"data"`
	}

	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v. Response body: %s", err, w.Body.String())
	}

	require.Equal(t, "success", response.Status)
	return response.Data.Role
}

// registerAndLogin registers a new user with the given username and logs them in,
// returning the JWT token from the cookie
func registerAndLogin(t *testing.T, userHttpHandler http.Handler, username string) string {
	t.Helper()
	// Register user
	userData := map[string]interface{}{
		"username":  username,
		"email":     username + "@example.com",
		"firstname": "Test",
		"lastname":  "User",
		"password":  "password123",
	}
	w := register(t, userHttpHandler, userData)
	require.Equal(t, http.StatusOK, w.Code)

	// Login user
	loginData := map[string]interface{}{
		"username": username,
		"password": "password123",
	}
	w = login(t, userHttpHandler, loginData)
	require.Equal(t, http.StatusOK, w.Code)

	// Extract token from cookie
	cookies := w.Result().Cookies()
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "auth_token" {
			authCookie = cookie
			break
		}
	}
	require.NotNil(t, authCookie, "No auth_token cookie found in response")
	return authCookie.Value
}
