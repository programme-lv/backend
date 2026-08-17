package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORSAllowsPatchPreflight(t *testing.T) {
	h := corsHandler()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("preflight should not reach the next handler")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/users/me", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "PATCH")
	req.Header.Set("Access-Control-Request-Headers", "content-type")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d, body %q", w.Code, w.Body.String())
	}
	allow := w.Header().Get("Access-Control-Allow-Methods")
	if !strings.Contains(allow, http.MethodPatch) {
		t.Fatalf("Access-Control-Allow-Methods %q missing PATCH", allow)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin %q", got)
	}
}
