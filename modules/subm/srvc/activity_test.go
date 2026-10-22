package srvc

import (
	"context"
	"testing"
	"time"

	"github.com/programme-lv/backend/modules/subm/domain"
	"github.com/programme-lv/backend/modules/subm/notify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldNotify(t *testing.T) {
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	quietPeriod := 6 * time.Hour

	tests := []struct {
		name       string
		lastSubmAt time.Time
		want       bool
	}{
		{name: "no previous submission", lastSubmAt: time.Time{}, want: true},
		{name: "just submitted", lastSubmAt: now.Add(-time.Minute), want: false},
		{name: "still inside quiet period", lastSubmAt: now.Add(-5 * time.Hour), want: false},
		{name: "exactly at the boundary", lastSubmAt: now.Add(-quietPeriod), want: true},
		{name: "after a long quiet period", lastSubmAt: now.Add(-3 * 24 * time.Hour), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldNotify(tt.lastSubmAt, now, quietPeriod))
		})
	}
}

func TestShouldNotifyDisabledForNonPositivePeriod(t *testing.T) {
	now := time.Now()
	assert.False(t, shouldNotify(now.Add(-24*time.Hour), now, 0))
	assert.False(t, shouldNotify(now.Add(-24*time.Hour), now, -time.Hour))
}

type recordingNotifier struct {
	ch chan notify.Submission
}

func (r *recordingNotifier) NotifySubmission(_ context.Context, s notify.Submission) error {
	r.ch <- s
	return nil
}

func TestNotifySubmissionSendsOncePerQuietPeriod(t *testing.T) {
	notifier := &recordingNotifier{ch: make(chan notify.Submission, 4)}
	service := &submSrvc{notifier: notifier, quietPeriod: time.Hour}
	subm := domain.Subm{ShortID: "abc123", TaskShortID: "task", LangShortID: "cpp17"}

	service.notifySubmission(context.Background(), subm, "alise", 7*time.Hour)

	select {
	case got := <-notifier.ch:
		assert.Equal(t, "alise", got.Username)
		assert.Equal(t, "abc123", got.ShortID)
		assert.Equal(t, 7*time.Hour, got.QuietFor)
	case <-time.After(time.Second):
		t.Fatal("expected a notification")
	}

	// A second submission in the same burst must not notify again.
	service.notifySubmission(context.Background(), subm, "bob", 7*time.Hour)
	select {
	case got := <-notifier.ch:
		t.Fatalf("unexpected duplicate notification for %s", got.Username)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestNotifySubmissionSkipsAdmin(t *testing.T) {
	notifier := &recordingNotifier{ch: make(chan notify.Submission, 1)}
	service := &submSrvc{notifier: notifier, quietPeriod: time.Hour}

	service.notifySubmission(context.Background(), domain.Subm{ShortID: "abc123"}, "admin", time.Hour)

	select {
	case got := <-notifier.ch:
		t.Fatalf("unexpected notification for %s", got.Username)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestShouldNotifyRequiresTime(t *testing.T) {
	// Guard against accidental future timestamps or zero periods.
	now := time.Now()
	require.False(t, shouldNotify(now.Add(time.Minute), now, time.Hour))
}
