//go:build integration

package user_test

import (
	"encoding/json"
	"net/http"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

func TestPreferencesPersistAndAreAccountScoped(t *testing.T) {
	handler := newUserHttpHandler(t)
	a := registerAndLogin(t, handler, "preferences_a")
	b := registerAndLogin(t, handler, "preferences_b")
	value := func(wBody []byte) bool {
		t.Helper()
		var r struct {
			Data struct {
				HideTaskTags *bool `json:"hide_task_tags"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(wBody, &r))
		require.NotNil(t, r.Data.HideTaskTags)
		return *r.Data.HideTaskTags
	}
	require.False(t, value(whoami(t, handler, a).Body.Bytes()))
	require.False(t, value(whoami(t, handler, b).Body.Bytes()))
	w := jsonAuthed(t, handler, http.MethodPatch, "/users/me/preferences", map[string]interface{}{"hide_task_tags": true}, a)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.True(t, value(w.Body.Bytes()))
	require.True(t, value(whoami(t, handler, a).Body.Bytes()))
	require.False(t, value(whoami(t, handler, b).Body.Bytes()))
	w = jsonAuthed(t, handler, http.MethodPatch, "/users/me", map[string]interface{}{"firstname": "Anna", "lastname": "Test"}, a)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.True(t, value(w.Body.Bytes()))
	w = login(t, handler, map[string]interface{}{"username": "preferences_a", "password": "password123"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.True(t, value(w.Body.Bytes()))
	w = jsonAuthed(t, handler, http.MethodPatch, "/users/me/preferences", map[string]interface{}{"hide_task_tags": false}, a)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.False(t, value(whoami(t, handler, a).Body.Bytes()))
}
