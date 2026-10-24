package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskAliasRedirectPreservesMethodAndQuery(t *testing.T) {
	h := taskAliases(func(_ context.Context, id string) (string, error) {
		if id == "old" {
			return "new", nil
		}
		return id, nil
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, method := range []string{"GET", "POST", "PATCH", "DELETE"} {
		r := httptest.NewRequest(method, "/tasks/old/archive/source.pdf?download=1", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		require.Equal(t, http.StatusPermanentRedirect, w.Code)
		require.Equal(t, "/tasks/new/archive/source.pdf?download=1", w.Header().Get("Location"))
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/assets/tasks/old/images/hash/1.png.full.webp", nil))
	require.Equal(t, "/assets/tasks/new/images/hash/1.png.full.webp", w.Header().Get("Location"))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/tasks/new", nil))
	require.Equal(t, http.StatusNoContent, w.Code)
}
