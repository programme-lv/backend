//go:build integration

package user_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUsernameChangeContinuityAndValidation(t *testing.T) {
	h, pg := newUserHttpHandlerWithPool(t)
	token := registerAndLogin(t, h, "rename_old")
	registerAndLogin(t, h, "rename_taken")
	change := func(name, pass, cookie string) *httptest.ResponseRecorder {
		return jsonAuthed(t, h, http.MethodPatch, "/users/me/username", map[string]interface{}{"username": name, "current_password": pass}, cookie)
	}
	check := func(w *httptest.ResponseRecorder, status int, code string) {
		t.Helper()
		require.Equal(t, status, w.Code, w.Body.String())
		assertErrorInHttpResponse(t, w, code)
	}
	check(change("newname", "password123", ""), 401, "http_unauthorized")
	check(change("rename_old", "wrong", token), 401, "username_or_password_incorrect")
	check(change("rename_taken", "password123", token), 409, "username_exists")
	check(change("Admin", "password123", token), 400, "username_reserved")
	check(change("..", "password123", token), 400, "username_invalid")
	check(change("ab\x00cd", "password123", token), 400, "username_invalid")
	check(change(strings.Repeat("ā", 13), "password123", token), 400, "username_too_long")
	for _, name := range []string{"..", "ab\x00cd"} {
		w := register(t, h, map[string]interface{}{
			"username": name, "email": "unused@example.com", "password": "password123",
		})
		check(w, 400, "username_invalid")
	}
	require.Equal(t, 200, change("rename_old", "password123", token).Code)
	before := whoami(t, h, token)
	var original struct {
		Data struct{ UUID, Username, Email, Firstname, Lastname string } `json:"data"`
	}
	require.NoError(t, json.Unmarshal(before.Body.Bytes(), &original))
	target := strings.Repeat("ā", 12)
	w := change(target, "password123", token)
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Empty(t, w.Result().Cookies())
	require.JSONEq(t, `{"status":"success","data":{"username":"`+target+`"}}`, w.Body.String())
	var after struct {
		Data struct{ UUID, Username, Email, Firstname, Lastname string } `json:"data"`
	}
	w = whoami(t, h, token)
	require.Equal(t, 200, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &after))
	require.Equal(t, original.Data.UUID, after.Data.UUID)
	require.Equal(t, original.Data.Email, after.Data.Email)
	require.Equal(t, original.Data.Firstname, after.Data.Firstname)
	require.Equal(t, original.Data.Lastname, after.Data.Lastname)
	require.Equal(t, target, after.Data.Username)
	for _, identifier := range []string{target, "rename_old@example.com"} {
		w = login(t, h, map[string]interface{}{"username": identifier, "password": "password123"})
		require.Equal(t, 200, w.Code, w.Body.String())
	}
	w = login(t, h, map[string]interface{}{"username": "rename_old", "password": "password123"})
	check(w, 401, "username_or_password_incorrect")
	// The cookie still says rename_old; the database row, not claims, identifies admin.
	_, err := pg.Exec(context.Background(), `UPDATE users SET username = 'admin' WHERE uuid = $1`, original.Data.UUID)
	require.NoError(t, err)
	w = change("nonadmin", "password123", token)
	check(w, 403, "username_change_forbidden")
	require.Contains(t, w.Body.String(), "administratora lietotājvārdu nevar mainīt")
}

func TestUsernameChangeConcurrentConflict(t *testing.T) {
	h := newUserHttpHandler(t)
	a := registerAndLogin(t, h, "race_a")
	b := registerAndLogin(t, h, "race_b")
	start := make(chan struct{})
	results := make(chan *httptest.ResponseRecorder, 2)
	var wg sync.WaitGroup
	for _, token := range []string{a, b} {
		wg.Add(1)
		go func(token string) {
			defer wg.Done()
			<-start
			results <- jsonAuthed(t, h, http.MethodPatch, "/users/me/username", map[string]interface{}{"username": "race_target", "current_password": "password123"}, token)
		}(token)
	}
	close(start)
	wg.Wait()
	close(results)
	counts := map[int]int{}
	for w := range results {
		counts[w.Code]++
		if w.Code == 409 {
			assertErrorInHttpResponse(t, w, "username_exists")
		}
	}
	require.Equal(t, 1, counts[200])
	require.Equal(t, 1, counts[409])
}
