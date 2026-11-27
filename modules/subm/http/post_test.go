package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/modules/subm/domain"
	"github.com/programme-lv/backend/modules/subm/srvc"
	tasksrvc "github.com/programme-lv/backend/modules/task/srvc"
	user "github.com/programme-lv/backend/modules/user"
	"github.com/programme-lv/backend/modules/user/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type postSubmService struct {
	srvc.SubmissionService
	submitted []srvc.SubmitSolParams
}

func (f *postSubmService) SubmitSol(_ context.Context, p srvc.SubmitSolParams) srvcerror.E {
	f.submitted = append(f.submitted, p)
	return nil
}
func (f *postSubmService) ViewSubm(_ context.Context, id uuid.UUID) (domain.Subm, srvcerror.E) {
	p := f.submitted[len(f.submitted)-1]
	return domain.Subm{UUID: id, AuthorUUID: p.AuthorUUID, TaskShortID: p.TaskShortID, LangShortID: p.ProgrLangID}, nil
}

type postTaskService struct{ tasksrvc.TaskService }

func (*postTaskService) ResolveNames(_ context.Context, ids []string) ([]string, srvcerror.E) {
	return ids, nil
}

type postUserService struct {
	user.UserService
	lookedUp []uuid.UUID
}

func (f *postUserService) GetUserByUUID(_ context.Context, id uuid.UUID) (user.User, srvcerror.E) {
	f.lookedUp = append(f.lookedUp, id)
	return user.User{UUID: id, Username: "current-name"}, nil
}

func postRequest(h *SubmHttpHandler, claims any, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/subm", strings.NewReader(body))
	if claims != nil {
		req = req.WithContext(context.WithValue(req.Context(), auth.CtxJwtClaimsKey, claims))
	}
	rec := httptest.NewRecorder()
	h.PostSubm(rec, req)
	return rec
}

func TestPostSubmIdentityAndRateLimitByUUID(t *testing.T) {
	submissions := &postSubmService{}
	users := &postUserService{}
	h := NewSubmHttpHandler(submissions, nil, &postTaskService{}, users)
	first, second := uuid.New(), uuid.New()
	body := `{"submission":"code","username":"reused-name","programming_lang_id":"cpp17","task_code_id":"task"}`
	rec := postRequest(h, &auth.JwtClaims{UUID: first.String(), Username: "old-name"}, body)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.Len(t, submissions.submitted, 1)
	assert.Equal(t, first, submissions.submitted[0].AuthorUUID)
	assert.Equal(t, []uuid.UUID{first}, users.lookedUp)
	assert.Contains(t, rec.Body.String(), `"username":"current-name"`)

	// A changed username and different legacy payload cannot bypass the same UUID's limit.
	rec = postRequest(h, &auth.JwtClaims{UUID: first.String(), Username: "new-name"}, `{"username":"another-name"}`)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, rec.Body.String())
	require.Len(t, submissions.submitted, 1)

	// Reuse of the old name by a different UUID does not inherit the limit.
	rec = postRequest(h, &auth.JwtClaims{UUID: second.String(), Username: "old-name"}, body)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.Len(t, submissions.submitted, 2)
	assert.Equal(t, second, submissions.submitted[1].AuthorUUID)
	assert.Equal(t, []uuid.UUID{first, second}, users.lookedUp)

	// Legacy username may be omitted entirely.
	rec = postRequest(h, &auth.JwtClaims{UUID: uuid.NewString(), Username: "stale"}, `{"submission":"code","programming_lang_id":"cpp17","task_code_id":"task"}`)
	assert.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
}

func TestPostSubmRejectsInvalidIdentity(t *testing.T) {
	tests := []struct {
		name   string
		claims any
	}{
		{"absent", nil}, {"nil claims", (*auth.JwtClaims)(nil)}, {"wrong type", "claims"},
		{"empty UUID", &auth.JwtClaims{}}, {"malformed UUID", &auth.JwtClaims{UUID: "not-a-uuid"}},
		{"nil UUID", &auth.JwtClaims{UUID: uuid.Nil.String()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			submissions := &postSubmService{}
			h := NewSubmHttpHandler(submissions, nil, nil, nil)
			rec := postRequest(h, tt.claims, `{}`)
			assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
			assert.Empty(t, submissions.submitted)
			assert.Empty(t, h.lastSubmTime)
		})
	}
}
