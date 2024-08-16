package subm

import (
	"context"

	submgen "github.com/programme-lv/backend/gen/submissions"
)

// List all submissions
func (s *submissionssrvc) ListSubmissions(ctx context.Context) (res []*submgen.Submission, err error) {
	panic("not implemented")
}
