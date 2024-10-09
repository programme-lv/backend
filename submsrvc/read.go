package submsrvc

import "context"

func (s *SubmissionSrvc) GetSubmission(ctx context.Context, submUuid string) (*FullSubmission, error) {
	panic("not implemented")
}

func (s *SubmissionSrvc) ListSubmissions(ctx context.Context) ([]*BriefSubmission, error) {
	panic("not implemented")
}
