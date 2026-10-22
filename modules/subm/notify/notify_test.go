package notify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNtfyNotifierPublishesSubmission(t *testing.T) {
	var gotPath, gotBody, gotTitle, gotTags, gotClick, gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotTitle = r.Header.Get("Title")
		gotTags = r.Header.Get("Tags")
		gotClick = r.Header.Get("Click")
		gotAuth = r.Header.Get("Authorization")
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewNtfyNotifier(NtfyConfig{
		ServerURL:  server.URL + "/",
		Topic:      "test-topic",
		Token:      "access-token",
		WebsiteURL: "https://programme.lv/",
	})

	err := notifier.NotifySubmission(context.Background(), Submission{
		Username:    "alise",
		TaskShortID: "lio2026dusa",
		LangShortID: "cpp17",
		ShortID:     "abc123",
		QuietFor:    9*time.Hour + 20*time.Minute,
	})
	require.NoError(t, err)

	assert.Equal(t, "/test-topic", gotPath)
	assert.Equal(t, "programme.lv is awake", gotTitle)
	assert.Equal(t, "bell", gotTags)
	assert.Equal(t, "Bearer access-token", gotAuth)
	assert.Equal(t, "https://programme.lv/submissions/abc123", gotClick)
	assert.Equal(t,
		"alise submitted code for lio2026dusa (cpp17) after 9 hours without submissions.",
		gotBody,
	)
}

func TestNtfyNotifierReturnsErrorForNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	notifier := NewNtfyNotifier(NtfyConfig{
		ServerURL: server.URL,
		Topic:     "test-topic",
	})

	err := notifier.NotifySubmission(context.Background(), Submission{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestFormatQuiet(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{name: "no previous submission", in: 0, want: "a long time"},
		{name: "minutes", in: 45 * time.Minute, want: "45 minutes"},
		{name: "one hour", in: time.Hour, want: "60 minutes"},
		{name: "hours", in: 9*time.Hour + 20*time.Minute, want: "9 hours"},
		{name: "one day", in: 24 * time.Hour, want: "24 hours"},
		{name: "days", in: 50 * time.Hour, want: "2 days"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatQuiet(tt.in))
		})
	}
}
