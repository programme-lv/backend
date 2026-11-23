package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/modules/user"
	"github.com/programme-lv/backend/modules/user/auth"
	"github.com/stretchr/testify/require"
)

type preferenceService struct {
	user.UserService
	values map[uuid.UUID]bool
}

func (s *preferenceService) PasswordChangedAt(context.Context, uuid.UUID) (time.Time, error) {
	return time.Unix(0, 0), nil
}
func (s *preferenceService) UpdatePreferences(_ context.Context, id uuid.UUID, value bool) srvcerror.E {
	s.values[id] = value
	return nil
}

func TestPreferencesAuthenticationValidationAndOwnership(t *testing.T) {
	svc := &preferenceService{values: map[uuid.UUID]bool{}}
	key := []byte("test")
	router := chi.NewRouter()
	NewUserHttpHandler(svc, key).RegisterRoutes(router)
	first, second := uuid.New(), uuid.New()
	token, err := auth.GenerateJWT("first", "first@example.com", first, key, time.Hour)
	require.NoError(t, err)
	request := func(body, cookie string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/users/me/preferences", strings.NewReader(body))
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: "auth_token", Value: cookie})
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	require.Equal(t, http.StatusUnauthorized, request(`{"hide_task_tags":true}`, "").Code)
	require.Equal(t, http.StatusUnauthorized, request(`{"hide_task_tags":true}`, "invalid").Code)
	for _, body := range []string{`{}`, `null`, `{"hide_task_tags":null}`, `{"hide_task_tags":"true"}`, `{"hide_task_tags":1}`, `{"hide_task_tags":true,"uuid":"` + second.String() + `"}`, `{"hide_task_tags":true} {}`} {
		require.Equal(t, http.StatusBadRequest, request(body, token).Code, body)
	}
	require.Empty(t, svc.values)
	w := request(`{"hide_task_tags":true}`, token)
	require.Equal(t, http.StatusOK, w.Code)
	var response struct {
		Data struct {
			HideTaskTags bool `json:"hide_task_tags"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.True(t, response.Data.HideTaskTags)
	require.True(t, svc.values[first])
	require.False(t, svc.values[second])
	require.Equal(t, http.StatusOK, request(`{"hide_task_tags":false}`, token).Code)
	require.False(t, svc.values[first])
}
