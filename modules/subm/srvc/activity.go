package srvc

import (
	"context"
	"time"

	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/modules/subm/domain"
	"github.com/programme-lv/backend/modules/subm/notify"
)

// shouldNotify reports whether a submission at now resumes activity after at
// least quietPeriod. A zero lastSubmAt means there is no earlier submission to
// compare against, so the first submission also counts as resumed activity.
func shouldNotify(lastSubmAt, now time.Time, quietPeriod time.Duration) bool {
	if quietPeriod <= 0 {
		return false
	}
	if lastSubmAt.IsZero() {
		return true
	}
	return now.Sub(lastSubmAt) >= quietPeriod
}

// notifySubmission publishes a best-effort notification in the background. A
// per-process guard prevents a burst of concurrent submissions from producing
// duplicate alerts for the same quiet period.
func (s *submSrvc) notifySubmission(ctx context.Context, subm domain.Subm, username string, quietFor time.Duration) {
	if s.notifier == nil || username == "" || username == "admin" {
		return
	}

	now := time.Now()
	s.notifyMu.Lock()
	if !s.lastNotifiedAt.IsZero() && now.Sub(s.lastNotifiedAt) < s.quietPeriod {
		s.notifyMu.Unlock()
		return
	}
	s.lastNotifiedAt = now
	s.notifyMu.Unlock()

	log := ctxlog.FromContext(ctx).With("module", "subm", "layer", "srvc")
	notification := notify.Submission{
		Username:    username,
		TaskShortID: subm.TaskShortID,
		LangShortID: subm.LangShortID,
		ShortID:     subm.ShortID,
		QuietFor:    quietFor,
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.notifier.NotifySubmission(ctx, notification); err != nil {
			log.Error("send submission notification", "subm_short_id", subm.ShortID, "error", err)
			return
		}
		log.Info("sent submission notification",
			"subm_short_id", subm.ShortID,
			"username", username,
			"quiet_for", quietFor,
		)
	}()
}
