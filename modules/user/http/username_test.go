package http

import (
	"context"
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

type usernameService struct {
	user.UserService
	id             uuid.UUID
	name, password string
}

func (s *usernameService) PasswordChangedAt(context.Context, uuid.UUID) (time.Time, error) {
	return time.Unix(0, 0), nil
}
func (s *usernameService) ChangeUsername(_ context.Context, id uuid.UUID, name, password string) (string, srvcerror.E) {
	s.id, s.name, s.password = id, name, password
	return name, nil
}

func TestChangeUsernameRequest(t *testing.T) {
	svc := &usernameService{}
	key := []byte("test")
	router := chi.NewRouter()
	NewUserHttpHandler(svc, key).RegisterRoutes(router)
	id := uuid.New()
	token, err := auth.GenerateJWT("old", "old@example.com", id, key, time.Hour)
	require.NoError(t, err)
	request := func(body, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/users/me/username", strings.NewReader(body))
		if token != "" {
			req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	require.Equal(t, 401, request(`{"username":"new","current_password":"pass"}`, "").Code)
	for _, body := range []string{`null`, `{}`, `{"username":null,"current_password":"p"}`, `{"username":"new"}`, `{"username":"new","current_password":null}`, `{"username":"new","current_password":"p","uuid":"x"}`, `{"username":"new","current_password":"p"} {}`, `{"username":"new","current_password":"` + strings.Repeat("x", 4096) + `"}`} {
		require.Equal(t, 400, request(body, token).Code, body)
	}
	require.Equal(t, uuid.Nil, svc.id)
	w := request(`{"username":"new","current_password":"pass"}`, token)
	require.Equal(t, 200, w.Code)
	require.JSONEq(t, `{"status":"success","data":{"username":"new"}}`, w.Body.String())
	require.Equal(t, id, svc.id)
	require.Equal(t, "new", svc.name)
	require.Equal(t, "pass", svc.password)
	require.Empty(t, w.Result().Cookies())
}
