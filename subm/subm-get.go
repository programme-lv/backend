package subm

import (
	"context"

	submgen "github.com/programme-lv/backend/gen/submissions"
	"goa.design/clue/log"
)

// Get a submission by UUID
func (s *submissionssrvc) GetSubmission(ctx context.Context, p *submgen.GetSubmissionPayload) (res *submgen.Submission, err error) {
	res = &submgen.Submission{}
	log.Printf(ctx, "submissions.getSubmission")
	return
}
