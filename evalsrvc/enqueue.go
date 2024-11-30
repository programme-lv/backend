package evalsrvc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/planglist"
	"github.com/programme-lv/tester"
)

// NewEvaluation creates a new evaluation request with external API key validation.
func (e *EvalSrvc) NewEvaluation(apiKey string, req NewEvalParams) (uuid.UUID, error) {
	if apiKey != e.extEvalKey {
		return uuid.Nil, ErrInvalidApiKey()
	}

	lang, err := planglist.GetProgrammingLanguageById(req.LangId)
	if err != nil {
		return uuid.Nil, err
	}

	evalUuid, err := uuid.NewV7()
	if err != nil {
		errMsg := fmt.Errorf("failed to generate UUID: %w", err)
		return uuid.Nil, errMsg
	}

	// Initialize test results array
	tests := make([]TestRes, len(req.Tests))
	for i, test := range req.Tests {
		tests[i] = TestRes{
			ID:     i,
			InpUrl: test.InUrl,
			AnsUrl: test.AnsUrl,
		}
	}

	// Create initial evaluation record
	eval := &Evaluation{
		UUID:  evalUuid,
		Stage: "waiting",
		Tests: tests,
		PrLang: PrLang{
			ShortId:   lang.ID,
			FullName:  lang.FullName,
			CodeFname: lang.CodeFilename,
			CompCmd:   lang.CompileCmd,
			CompFname: lang.CompiledFilename,
			ExecCmd:   lang.ExecuteCmd,
		},
		ErrorMsg:   nil,
		Checker:    req.Checker,
		Interactor: req.Interactor,
		CpuMsLim:   req.CpuMs,
		MemKiBLim:  req.MemKiB,
		CreatedAt:  time.Now(),
	}

	// Save evaluation state
	err = e.repo.Save(eval)
	if err != nil {
		return uuid.Nil, err
	}

	// Enqueue for processing
	err = e.enqueue(&req, evalUuid, e.resSqsUrl, lang)
	if err != nil {
		return uuid.Nil, err
	}

	return evalUuid, nil
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
	if apiKey != e.extEvalKey {
		return uuid.Nil, ErrInvalidApiKey()
	}

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
			ID:         int(test.ID),
			InSha256:   test.InSha256,
			InUrl:      test.InUrl,
			InContent:  test.InContent,
			AnsSha256:  test.AnsSha256,
			AnsUrl:     test.AnsUrl,
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
