package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/modules/task/highlight"
	"github.com/programme-lv/backend/modules/user/auth"
	"github.com/stretchr/testify/require"
)

type fakeHighlightStore struct {
	tags    map[string][]highlight.Label
	err     error
	calls   int
	aliases highlight.Aliases
}

func (f *fakeHighlightStore) List(context.Context) ([]highlight.Tag, error) {
	return []highlight.Tag{}, f.err
}
func (f *fakeHighlightStore) Create(_ context.Context, _, _ string, aliases ...highlight.Aliases) (highlight.Tag, error) {
	if len(aliases) > 0 {
		f.aliases = aliases[0]
	}
	return highlight.Tag{ID: "a", Revision: 1}, f.err
}
func (f *fakeHighlightStore) Edit(_ context.Context, _, _ string, _ bool, _ int64, aliases ...highlight.Aliases) (highlight.Tag, error) {
	if len(aliases) > 0 {
		f.aliases = aliases[0]
	}
	return highlight.Tag{}, f.err
}
func (f *fakeHighlightStore) Get(context.Context, string) (highlight.Assignment, error) {
	return highlight.Assignment{HighlightedTags: []highlight.Label{}}, f.err
}
func (f *fakeHighlightStore) Replace(context.Context, string, []string, int64) (highlight.Assignment, error) {
	return highlight.Assignment{HighlightedTags: []highlight.Label{}, Revision: 1}, f.err
}
func (f *fakeHighlightStore) Batch(context.Context, []string) (map[string][]highlight.Label, error) {
	f.calls++
	return f.tags, f.err
}
func TestHighlightRoutesAuthenticationAndValidation(t *testing.T) {
	key := []byte("jwt-secret")
	apiKey := []byte("admin-key")
	f := &fakeHighlightStore{}
	h := NewTaskHttpHandler(nil, WithHighlightStore(f))
	r := chi.NewRouter()
	h.RegisterRoutes(r, key, apiKey, false, nil)
	user, err := auth.GenerateJWT("user", "", uuid.New(), key, time.Hour)
	require.NoError(t, err)
	admin, err := auth.GenerateJWT("admin", "", uuid.New(), key, time.Hour)
	require.NoError(t, err)
	routes := []struct{ method, path, body string }{{"GET", "/task-highlight-tags", ""}, {"POST", "/task-highlight-tags", `{"id":"a","label":"A"}`}, {"PATCH", "/task-highlight-tags/a", `{"label":"A","archived":false,"expected_revision":1}`}, {"GET", "/tasks/a/highlighted-tags", ""}, {"PATCH", "/tasks/a/highlighted-tags", `{"highlighted_tags":[],"expected_revision":0}`}}
	for _, route := range routes {
		t.Run(route.method+route.path, func(t *testing.T) {
			for _, c := range []struct {
				cookie, bearer string
				want           int
			}{{"", "", 401}, {user, "", 403}, {admin, "", 200}, {"", "admin-key", 200}} {
				req := httptest.NewRequest(route.method, route.path, bytes.NewBufferString(route.body))
				if c.cookie != "" {
					req.AddCookie(&http.Cookie{Name: "auth_token", Value: c.cookie})
				}
				if c.bearer != "" {
					req.Header.Set("Authorization", "Bearer "+c.bearer)
				}
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				require.Equal(t, c.want, w.Code, w.Body.String())
			}
		})
	}
	for _, body := range []string{`{}`, `{"highlighted_tags":null,"expected_revision":0}`, `{"highlighted_tags":[]}`, `{"highlighted_tags":[],"expected_revision":null}`} {
		req := httptest.NewRequest("PATCH", "/tasks/a/highlighted-tags", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer admin-key")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, 400, w.Code, body)
	}
	f.err = highlight.ErrConflict
	req := httptest.NewRequest("PATCH", "/tasks/a/highlighted-tags", bytes.NewBufferString(`{"highlighted_tags":[],"expected_revision":0}`))
	req.Header.Set("Authorization", "Bearer admin-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 409, w.Code)
}
func TestHighlightsRefreshAfterCacheAndNoMutation(t *testing.T) {
	svc := &fakeTaskPreviewService{previews: taskListWithTwoPreviews()}
	f := &fakeHighlightStore{tags: map[string][]highlight.Label{"task-a": {{ID: "a", Label: "Old"}}}}
	h := NewTaskHttpHandler(svc, WithHighlightStore(f))
	ctx := context.Background()
	first, e := h.GetTaskList(ctx)
	require.NoError(t, e)
	require.Equal(t, "Old", first[0].HighlightedTags[0].Label)
	f.tags = map[string][]highlight.Label{"task-a": {{ID: "a", Label: "New"}}}
	second, e := h.GetTaskList(ctx)
	require.NoError(t, e)
	require.Equal(t, "New", second[0].HighlightedTags[0].Label)
	require.Equal(t, "Old", first[0].HighlightedTags[0].Label)
	require.Equal(t, 1, svc.calls)
	require.Equal(t, 2, f.calls)
	raw, err := json.Marshal(second)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"highlighted_tags"`)
	h.getTaskViewCache.Set("task-a", Task{ShortTaskID: "task-a", TaskFullName: "A"}, time.Minute)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("taskId", "task-a")
	viewCtx := context.WithValue(ctx, chi.RouteCtxKey, routeCtx)
	view, ve := h.GetTaskView(viewCtx)
	require.NoError(t, ve)
	require.Equal(t, "New", view.HighlightedTags[0].Label)
	f.tags = map[string][]highlight.Label{}
	view, ve = h.GetTaskView(viewCtx)
	require.NoError(t, ve)
	require.Empty(t, view.HighlightedTags)
}
func TestHighlightReadFailure(t *testing.T) {
	h := NewTaskHttpHandler(&fakeTaskPreviewService{previews: taskListWithTwoPreviews()}, WithHighlightStore(&fakeHighlightStore{err: errors.New("db unavailable")}))
	_, e := h.GetTaskList(context.Background())
	require.Equal(t, 500, e.HttpStatusCode())
}

func TestHighlightAliasRequestsAndPublicJSON(t *testing.T) {
	f := &fakeHighlightStore{}
	h := NewTaskHttpHandler(nil, WithHighlightStore(f))
	r := chi.NewRouter()
	h.RegisterRoutes(r, []byte("secret"), []byte("admin-key"), false, nil)
	call := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer admin-key")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	w := call("POST", "/task-highlight-tags", `{"id":"a","label":"A","abbreviation":"DP","english_label":"dynamic programming"}`)
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Equal(t, "DP", *f.aliases.Abbreviation)
	require.Equal(t, "dynamic programming", *f.aliases.EnglishLabel)
	w = call("PATCH", "/task-highlight-tags/a", `{"label":"A","archived":false,"expected_revision":1,"abbreviation":""}`)
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Equal(t, "", *f.aliases.Abbreviation)
	require.Nil(t, f.aliases.EnglishLabel)
	for _, value := range []string{`null`, `123`, `true`} {
		w = call("POST", "/task-highlight-tags", `{"id":"a","label":"A","abbreviation":`+value+`}`)
		require.Equal(t, 400, w.Code, w.Body.String())
	}
	raw, err := json.Marshal(highlight.Label{ID: "a", Label: "A", Abbreviation: "DP", EnglishLabel: "dynamic programming"})
	require.NoError(t, err)
	require.JSONEq(t, `{"id":"a","label":"A","abbreviation":"DP","english_label":"dynamic programming"}`, string(raw))
	raw, err = json.Marshal(highlight.Label{ID: "a", Label: "A"})
	require.NoError(t, err)
	require.JSONEq(t, `{"id":"a","label":"A"}`, string(raw))
}
