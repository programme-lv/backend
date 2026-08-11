package mail

import (
	"context"
	"errors"
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

var (
	ErrRateLimited = errors.New("email rate limited")
	ErrDisabled    = errors.New("smtp disabled")
)

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
	return ErrDisabled
}

type sendSlot struct {
	id uint64
	at time.Time
}

// RateLimitedMailer wraps a Mailer with a process-local rolling hourly send cap.
// Assumes a single backend instance.
type RateLimitedMailer struct {
	inner Mailer
	limit int
	mu    sync.Mutex
	sent  []sendSlot
	seq   uint64
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
	for _, slot := range m.sent {
		if slot.at.After(cutoff) {
			kept = append(kept, slot)
		}
	}
	m.sent = kept
	if len(m.sent) >= m.limit {
		m.mu.Unlock()
		return ErrRateLimited
	}
	m.seq++
	id := m.seq
	m.sent = append(m.sent, sendSlot{id: id, at: now})
	m.mu.Unlock()

	if err := m.inner.Send(ctx, msg); err != nil {
		m.releaseReservation(id)
		return err
	}
	return nil
}

func (m *RateLimitedMailer) releaseReservation(id uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, slot := range m.sent {
		if slot.id == id {
			m.sent = append(m.sent[:i], m.sent[i+1:]...)
			return
		}
	}
}
