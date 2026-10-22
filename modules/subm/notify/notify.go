// Package notify publishes best-effort phone notifications about submission
// activity. Notifications are optional and must never affect submission
// handling.
package notify

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Submission is the information included in an activity notification.
// It deliberately excludes source code and anything else that could leak
// a solution.
type Submission struct {
	Username    string
	TaskShortID string
	LangShortID string
	ShortID     string

	// QuietFor is the time since the previous submission. Zero means there
	// was no previous submission to compare against.
	QuietFor time.Duration
}

// SubmissionNotifier delivers activity notifications.
type SubmissionNotifier interface {
	NotifySubmission(ctx context.Context, s Submission) error
}

// NtfyConfig configures the ntfy.sh publisher.
type NtfyConfig struct {
	// ServerURL is the base URL, for example https://ntfy.sh.
	ServerURL string

	// Topic is the ntfy topic. Treat it as a secret: anyone who knows it can
	// read notifications published to it.
	Topic string

	// Token is an optional ntfy access token for protected topics.
	Token string

	// WebsiteURL is the public website base URL used for the notification's
	// click target. Empty disables the link.
	WebsiteURL string

	// HTTPClient is optional; a bounded default client is used when nil.
	HTTPClient *http.Client
}

// NtfyNotifier publishes notifications through the ntfy HTTP API.
type NtfyNotifier struct {
	serverURL  string
	topic      string
	token      string
	websiteURL string
	client     *http.Client
}

// NewNtfyNotifier returns a notifier that publishes to an ntfy topic.
func NewNtfyNotifier(cfg NtfyConfig) *NtfyNotifier {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &NtfyNotifier{
		serverURL:  strings.TrimRight(cfg.ServerURL, "/"),
		topic:      cfg.Topic,
		token:      cfg.Token,
		websiteURL: strings.TrimRight(cfg.WebsiteURL, "/"),
		client:     client,
	}
}

// NotifySubmission publishes one activity notification.
func (n *NtfyNotifier) NotifySubmission(ctx context.Context, s Submission) error {
	body := fmt.Sprintf(
		"%s submitted code for %s (%s) after %s without submissions.",
		s.Username,
		s.TaskShortID,
		s.LangShortID,
		formatQuiet(s.QuietFor),
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		n.serverURL+"/"+url.PathEscape(n.topic),
		strings.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create ntfy request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	req.Header.Set("Title", "programme.lv is awake")
	req.Header.Set("Tags", "bell")
	if n.token != "" {
		req.Header.Set("Authorization", "Bearer "+n.token)
	}
	if n.websiteURL != "" && s.ShortID != "" {
		req.Header.Set("Click", n.websiteURL+"/submissions/"+url.PathEscape(s.ShortID))
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("publish ntfy notification: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("publish ntfy notification: unexpected status %s", resp.Status)
	}
	return nil
}

func formatQuiet(d time.Duration) string {
	d = d.Round(time.Minute)
	switch {
	case d <= 0:
		return "a long time"
	case d >= 48*time.Hour:
		return plural(int(d.Hours())/24, "day")
	case d >= 2*time.Hour:
		return plural(int(d.Hours()), "hour")
	default:
		return plural(int(d.Minutes()), "minute")
	}
}

func plural(n int, unit string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", unit)
	}
	return fmt.Sprintf("%d %ss", n, unit)
}
