package subm

import (
	"context"

	submgen "github.com/programme-lv/backend/gen/submissions"
)

// List all submissions
func (s *submissionssrvc) ListSubmissions(ctx context.Context) (res []*submgen.Submission, err error) {
	panic("not implemented")
	// subms, err := s.ddbSubmTable.List(ctx)
	// if err != nil {
	// 	return nil, submgen.InternalError("error retrieving submission list")
	// }

	// users, err := s.userSrvc.ListUsers(ctx)
	// if err != nil {
	// 	return nil, submgen.InternalError("error retrieving users")
	// }

	// userUuidToUsername := make(map[string]string)
	// for _, user := range users {
	// 	userUuidToUsername[user.UUID] = user.Username
	// }

	// pLangIdToDetails := make(map[string]struct {
	// 	fullName string
	// 	monacoId string
	// })
	// pLangs := getHardcodedLanguageList()
	// for _, lang := range pLangs {
	// 	pLangIdToDetails[lang.ID] = struct {
	// 		fullName string
	// 		monacoId string
	// 	}{
	// 		fullName: lang.FullName,
	// 		monacoId: lang.MonacoId,
	// 	}
	// }

	// tasks, err := s.taskSrvc.ListTasks(ctx)
	// if err != nil {
	// 	return nil, submgen.InternalError("error retrieving tasks")
	// }

	// taskIdToDetailsMap := make(map[string]*taskgen.Task)
	// for _, task := range tasks {
	// 	taskIdToDetailsMap[task.PublishedTaskID] = task
	// }

	// res = make([]*submgen.Submission, 0)
	// for _, subm := range subms {
	// 	author := subm.AuthorUuid
	// 	username, ok := userUuidToUsername[author]
	// 	if !ok {
	// 		log.Printf(ctx, "user %v not found for submission %v", subm.AuthorUuid, subm.Uuid)
	// 		continue
	// 	}
	// 	createdAt := time.Unix(subm.UnixTime, 0)
	// 	createdAtRfc3339 := createdAt.Format(time.RFC3339)
	// 	pLangDetails, ok := pLangIdToDetails[subm.ProgLangId]
	// 	if !ok {
	// 		log.Printf(ctx, "programming language %v not found for submission %v", subm.ProgLangId, subm.Uuid)
	// 		continue
	// 	}

	// 	submTask, ok := taskIdToDetailsMap[subm.TaskId]
	// 	if !ok {
	// 		log.Printf(ctx, "task %v not found for submission %v", subm.TaskId, subm.Uuid)
	// 		continue
	// 	}

	// 	res = append(res, &submgen.Submission{
	// 		UUID:       subm.Uuid,
	// 		Submission: subm.Content,
	// 		Username:   username,
	// 		CreatedAt:  createdAtRfc3339,
	// 		Evaluation: nil,
	// 		Language: &submgen.SubmProgrammingLang{
	// 			ID:       subm.ProgLangId,
	// 			FullName: pLangDetails.fullName,
	// 			MonacoID: pLangDetails.fullName,
	// 		},
	// 		Task: &submgen.SubmTask{
	// 			Name: submTask.TaskFullName,
	// 			Code: submTask.PublishedTaskID,
	// 		},
	// 	})
	// }

	// return res, nil
}
