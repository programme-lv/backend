package mail

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Message is a transactional email payload.
type Message struct {
	To       string
	Subject  string
	TextBody string
	HTMLBody string
}

// Mailer sends transactional email.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

type noopMailer struct{}

func NewNoopMailer() Mailer {
	return noopMailer{}
}

func (noopMailer) Send(ctx context.Context, msg Message) error {
	slog.InfoContext(ctx, "smtp disabled; skipping email",
		"to", msg.To,
		"subject", msg.Subject,
	)
	return nil
}

// RateLimitedMailer wraps a Mailer with a process-local rolling hourly send cap.
// Assumes a single backend instance.
type RateLimitedMailer struct {
	inner Mailer
	limit int
	mu    sync.Mutex
	sent  []time.Time
}

func NewRateLimitedMailer(inner Mailer, hourlyLimit int) *RateLimitedMailer {
	if hourlyLimit <= 0 {
		hourlyLimit = 60
	}
	return &RateLimitedMailer{inner: inner, limit: hourlyLimit}
}

func (m *RateLimitedMailer) Send(ctx context.Context, msg Message) error {
	now := time.Now()
	cutoff := now.Add(-time.Hour)

	m.mu.Lock()
	kept := m.sent[:0]
	for _, t := range m.sent {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	m.sent = kept
	if len(m.sent) >= m.limit {
		m.mu.Unlock()
		return fmt.Errorf("global email hourly limit reached (%d)", m.limit)
	}
	m.sent = append(m.sent, now)
	m.mu.Unlock()

	return m.inner.Send(ctx, msg)
}
