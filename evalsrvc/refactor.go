package evalsrvc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/planglist"
	"github.com/programme-lv/tester"
)

// parameters needed to create a new evaluation request
type NewEvalParams struct {
	Code   string // user submitted solution source code
	LangId string // short compiler, interpreter id

	Tests []TestFile // test cases to run against the code

	CpuMs  int // maximum user-mode CPU time in milliseconds
	MemKiB int // maximum resident set size in kibibytes

	// optional testlib.h checker program. If not provided,
	// only output of the user's solution is returned from tester
	// and is not viable for grading. "run program" use case.
	Checker *string

	// optional testlib.h interactor program.
	Interactor *string
}

// Enqueue adds an evaluation request to the processing queue using a pre-generated UUID
func (e *EvalSrvc) Enqueue(req NewEvalParams, evalUuid uuid.UUID) (uuid.UUID, error) {
	lang, err := planglist.GetProgrammingLanguageById(req.LangId)
	if err != nil {
		return uuid.Nil, err
	}
	err = e.enqueue(&req, evalUuid, e.resSqsUrl, lang)
	if err != nil {
		return uuid.Nil, err
	}
	return evalUuid, nil
}

// EnqueueExternal adds an external evaluation request to a separate queue after API key validation
func (e *EvalSrvc) EnqueueExternal(apiKey string, req NewEvalParams) (uuid.UUID, error) {
	// check validity of programming language before creating a new queue
	lang, err := planglist.GetProgrammingLanguageById(req.LangId)
	if err != nil {
		return uuid.Nil, err
	}

	evalUuid, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to generate UUID: %w", err)
	}

	return evalUuid, e.enqueue(&req, evalUuid, e.extEvalSqsUrl, lang)
}

// enqueue handles the common logic of sending an evaluation request to an SQS queue.
// It converts the request into the tester's format and sends it as a JSON message.
func (e *EvalSrvc) enqueue(req *NewEvalParams,
	evalUuid uuid.UUID,
	resSqsUrl string,
	lang *planglist.ProgrammingLang,
) error {
	// Convert tests to tester format
	tests := make([]tester.ReqTest, len(req.Tests))
	for i, test := range req.Tests {
		tests[i] = tester.ReqTest{
			ID:         i + 1,
			InSha256:   test.InSha256,
			InUrl:      test.InDownlUrl,
			InContent:  test.InContent,
			AnsSha256:  test.AnsSha256,
			AnsUrl:     test.AnsDownlUrl,
			AnsContent: test.AnsContent,
		}
	}

	// Prepare evaluation request
	jsonReq, err := json.Marshal(tester.EvalReq{
		EvalUuid:  evalUuid.String(),
		ResSqsUrl: resSqsUrl,
		Code:      req.Code,
		Language: tester.Language{
			LangID:        lang.ID,
			LangName:      lang.FullName,
			CodeFname:     lang.CodeFilename,
			CompileCmd:    lang.CompileCmd,
			CompiledFname: lang.CompiledFilename,
			ExecCmd:       lang.ExecuteCmd,
		},
		Tests:      tests,
		Checker:    req.Checker,
		Interactor: req.Interactor,
		CpuMillis:  req.CpuMs,
		MemoryKiB:  req.MemKiB,
	})
	if err != nil {
		format := "failed to marshal evaluation request: %w"
		errMsg := fmt.Errorf(format, err)
		return errMsg
	}

	// Send to SQS queue
	_, err = e.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(e.submSqsUrl),
		MessageBody: aws.String(string(jsonReq)),
	})
	if err != nil {
		format := "failed to send message to evaluation queue: %w"
		errMsg := fmt.Errorf(format, err)
		return errMsg
	}

	return nil
}
