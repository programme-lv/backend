package subm

import (
	"context"

	submgen "github.com/programme-lv/backend/gen/submissions"
	"goa.design/clue/log"
)

type GetSubmissionPayload struct {
	UUID string
}

// Get a submission by UUID
func (s *SubmissionsService) GetSubmission(ctx context.Context, p *GetSubmissionPayload) (res *submgen.Submission, err error) {
	res = &submgen.Submission{}
	log.Printf(ctx, "submissions.getSubmission")
	return
}
