package subm

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/guregu/dynamo/v2"
	"github.com/programme-lv/backend/auth"
	submgen "github.com/programme-lv/backend/gen/submissions"
	taskgen "github.com/programme-lv/backend/gen/tasks"
	usergen "github.com/programme-lv/backend/gen/users"
	"goa.design/clue/log"
)

// CreateSubmission implements submissions.Service.
func (s *submissionssrvc) CreateSubmission(ctx context.Context, p *submgen.CreateSubmissionPayload) (res *submgen.Submission, err error) {
	submContent := SubmissionContent{Value: p.Submission}

	for _, v := range []Validatable{&submContent} {
		err := v.IsValid()
		if err != nil {
			return nil, err
		}
	}

	userSrvcUser, err := s.userSrvc.GetUserByUsername(ctx, &usergen.GetUserByUsernamePayload{Username: p.Username})
	if err != nil {
		log.Errorf(ctx, err, "error getting user: %+v", err.Error())
		if e, ok := err.(usergen.NotFound); ok {
			return nil, submgen.InvalidSubmissionDetails(string(e))
		}
		return nil, submgen.InternalError("error getting user")
	}

	claims := ctx.Value(ClaimsKey("claims")).(*auth.Claims)
	log.Printf(ctx, "%+v", claims)

	if claims.UUID != userSrvcUser.UUID {
		return nil, submgen.Unauthorized("jwt claims uuid does not match username's user's uuid")
	}

	taskSrvTask, err := s.taskSrvc.GetTask(ctx, &taskgen.GetTaskPayload{
		TaskID: p.TaskCodeID,
	})
	if err != nil {
		log.Errorf(ctx, err, "error getting task: %+v", err.Error())
		if e, ok := err.(taskgen.TaskNotFound); ok {
			return nil, submgen.InvalidSubmissionDetails(string(e))
		}
		return nil, submgen.InternalError("error getting task")
	}

	langs := getHardcodedLanguageList()
	var foundPLang *ProgrammingLang = nil
	for _, lang := range langs {
		if lang.ID == p.ProgrammingLangID {
			foundPLang = &lang
		}
	}

	if foundPLang == nil {
		return nil, submgen.InvalidSubmissionDetails("invalid programming language")
	}

	uuid := uuid.New()
	createdAt := time.Now()
	row := &SubmissionRow{
		Uuid:       uuid.String(),
		UnixTime:   createdAt.Unix(),
		Content:    submContent.String(),
		Version:    0,
		AuthorUuid: userSrvcUser.UUID,
		ProgLangId: foundPLang.ID,
		TaskId:     taskSrvTask.PublishedTaskID,
	}

	err = s.ddbSubmTable.Save(ctx, row)
	if err != nil {
		// TODO: automatically retry with exponential backoff on version conflict
		if dynamo.IsCondCheckFailed(err) {
			log.Errorf(ctx, err, "version conflict saving user")
			return nil, submgen.InternalError("version conflict saving user")
		} else {
			log.Errorf(ctx, err, "error saving user")
			return nil, submgen.InternalError("error saving user")
		}
	}

	messageBody := fmt.Sprintf("Message %s", row.Uuid)
	_, err = s.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.submQueueUrl),
		MessageBody: aws.String(messageBody),
	})
	if err != nil {
		fmt.Printf("failed to send message %s, %v\n", row.Uuid, err)
	}

	createdAtRfc3339 := createdAt.Format(time.RFC3339)

	res = &submgen.Submission{
		UUID:       row.Uuid,
		Submission: row.Content,
		Username:   userSrvcUser.Username,
		CreatedAt:  createdAtRfc3339,
		Evaluation: nil,
		Language: &submgen.SubmProgrammingLang{
			ID:       foundPLang.ID,
			FullName: foundPLang.FullName,
			MonacoID: foundPLang.MonacoId,
		},
		Task: &submgen.SubmTask{
			Name: taskSrvTask.TaskFullName,
			Code: taskSrvTask.PublishedTaskID,
		},
	}

	return res, nil
}
