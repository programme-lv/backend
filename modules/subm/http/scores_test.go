package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/modules/subm/domain"
	submsrvc "github.com/programme-lv/backend/modules/subm/srvc"
	usersrvc "github.com/programme-lv/backend/modules/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type scoresUserService struct {
	usersrvc.UserService
	gotUsernames []string
}

func (f *scoresUserService) GetUserByUsername(ctx context.Context, username string) (usersrvc.User, srvcerror.E) {
	f.gotUsernames = append(f.gotUsernames, username)
	return usersrvc.User{UUID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}, nil
}

type scoresSubmService struct {
	submsrvc.SubmissionService
}

func (f *scoresSubmService) GetMaxScorePerTask(ctx context.Context, userUUID uuid.UUID) (map[string]domain.MaxScore, srvcerror.E) {
	return map[string]domain.MaxScore{}, nil
}

func newScoresTestRouter(t *testing.T) (*chi.Mux, *scoresUserService) {
	t.Helper()
	userSrvc := &scoresUserService{}
	h := NewSubmHttpHandler(&scoresSubmService{}, nil, nil, userSrvc)
	router := chi.NewRouter()
	h.RegisterRoutes(router, []byte("test-jwt-key"), []byte("test-admin-key"), false, nil)
	return router, userSrvc
}

func TestGetMaxScorePerTaskDecodesUsernameOnce(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   string
	}{
		{
			name:   "raw at sign",
			target: "/subm/scores/legacy@example.com",
			want:   "legacy@example.com",
		},
		{
			name:   "encoded at sign",
			target: "/subm/scores/legacy%40example.com",
			want:   "legacy@example.com",
		},
		{
			name:   "space",
			target: "/subm/scores/j%C4%81nis%20b%C4%93rzi%C5%86%C5%A1",
			want:   "jānis bērziņš",
		},
		{
			name:   "hash",
			target: "/subm/scores/user%23tag",
			want:   "user#tag",
		},
		{
			name:   "raw plus is not a space",
			target: "/subm/scores/a+b",
			want:   "a+b",
		},
		{
			name:   "encoded plus is not a space",
			target: "/subm/scores/a%2Bb",
			want:   "a+b",
		},
		{
			name:   "raw unicode",
			target: "/subm/scores/jānis",
			want:   "jānis",
		},
		{
			name:   "encoded unicode",
			target: "/subm/scores/j%C4%81nis",
			want:   "jānis",
		},
		{
			name:   "literal percent 40 is not decoded twice",
			target: "/subm/scores/user%2540example.com",
			want:   "user%40example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, userSrvc := newScoresTestRouter(t)
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.Len(t, userSrvc.gotUsernames, 1)
			assert.Equal(t, tt.want, userSrvc.gotUsernames[0])
		})
	}
}

func TestGetMaxScorePerTaskRejectsMalformedUsernameEscape(t *testing.T) {
	router, userSrvc := newScoresTestRouter(t)
	// RawPath is normally validated by net/url, so build the malformed state
	// explicitly and drive it through the real chi route and handler.
	req := httptest.NewRequest(http.MethodGet, "/subm/scores/placeholder", nil)
	req.URL.RawPath = "/subm/scores/%zz"
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Empty(t, userSrvc.gotUsernames)

	var resp jsonresp.JsonResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, jsonresp.ErrHttpBadRequest.ErrCode, resp.ErrCode)
	assert.Equal(t, jsonresp.ErrHttpBadRequest.ErrMsg, resp.ErrMsg)
}
