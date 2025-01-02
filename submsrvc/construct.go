package submsrvc

import (
	"context"

	"github.com/programme-lv/backend/planglist"
)

func (s *SubmissionSrvc) constructSubm(ctx context.Context, entity SubmissionEntity) (*Submission, error) {
	user, err := s.userSrvc.GetUserByUUID(ctx, entity.AuthorUUID)
	if err != nil {
		return nil, err
	}
	task, err := s.taskSrvc.GetTask(ctx, entity.TaskShortID)
	if err != nil {
		return nil, err
	}
	lang, err := planglist.GetProgrammingLanguageById(entity.LangShortID)
	if err != nil {
		return nil, err
	}
	return &Submission{
		UUID:     entity.UUID,
		Content:  entity.Content,
		Author:   Author{UUID: user.UUID, Username: user.Username},
		Task:     TaskRef{ShortID: task.ShortId, FullName: task.FullName},
		Lang:     PrLang{ShortID: lang.ID, Display: lang.FullName, MonacoID: lang.MonacoId},
		CurrEval: entity.CurrEval,
	}, nil
}
