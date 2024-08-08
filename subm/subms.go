package subm

import (
	"context"

	submgen "github.com/programme-lv/backend/gen/submissions"
	"goa.design/clue/log"
)

// submissions service example implementation.
// The example methods log the requests and return zero values.
type submissionssrvc struct{}

// NewSubmissions returns the submissions service implementation.
func NewSubmissions() submgen.Service {
	return &submissionssrvc{}
}

// CreateSubmission implements submissions.Service.
func (s *submissionssrvc) CreateSubmission(context.Context, *submgen.CreateSubmissionPayload) (res *submgen.Submission, err error) {
	panic("unimplemented")
}

// List all submissions
func (s *submissionssrvc) ListSubmissions(ctx context.Context) (res []*submgen.Submission, err error) {
	log.Printf(ctx, "submissions.listSubmissions")
	return
}

// Get a submission by UUID
func (s *submissionssrvc) GetSubmission(ctx context.Context, p *submgen.GetSubmissionPayload) (res *submgen.Submission, err error) {
	res = &submgen.Submission{}
	log.Printf(ctx, "submissions.getSubmission")
	return
}
