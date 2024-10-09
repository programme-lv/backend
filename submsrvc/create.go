package submsrvc

import "context"

type CreateSubmissionPayload struct {
	Submission        string
	Username          string
	ProgrammingLangID string
	TaskCodeID        string
	Token             string
}

// -- Table: submissions
// CREATE TABLE submissions (
//     subm_uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
//     content TEXT NOT NULL,
//     author_uuid UUID NOT NULL,
//     task_id TEXT NOT NULL,
//     prog_lang_id TEXT NOT NULL,
//     current_eval_uuid UUID,
//     created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
// );

func (s *SubmissionSrvc) CreateSubmission(ctx context.Context, payload *CreateSubmissionPayload) (*BriefSubmission, error) {
	panic("not implemented")
}
