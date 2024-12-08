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

// user submitted solution
type CodeWithLang struct {
	SrcCode string // user submitted solution source code
	LangId  string // short compiler, interpreter id
}

// input and expected output
type TestFile struct {
	InSha256   *string // SHA256 hash of input for caching
	InDownlUrl *string // URL to download input
	InContent  *string // input content as alternative to URL

	AnsSha256   *string // SHA256 hash of answer for caching
	AnsDownlUrl *string // URL to download answer
	AnsContent  *string // answer content as alternative to URL
}

// Enqueues code for evaluation by tester, returns eval uuid:
// 1. validates programming language;
// 2. validates cpu, mem constraints & checker, interactor size;
// 3. new empty eval record, store in memory;
// 4. convert tests to tester format, marshall request to json;
// 5. enqueue request to sqs queue, return eval uuid.
func (e *EvalSrvc) NewEvaluation(
	code CodeWithLang,
	tests []TestFile,
	params TesterParams,
) (uuid.UUID, error) {

	lang, err := getPrLangById(code.LangId)
	if err != nil {
		return uuid.Nil, err
	}

	err = params.IsValid() // validate tester parameters
	if err != nil {
		return uuid.Nil, err
	}

	// initialize test results array
	// in the future we can also maybe resolve in & ans text here
	testRes := make([]TestRes, len(tests))
	for i := range tests {
		testRes[i] = TestRes{
			ID: i + 1,
		}
	}

	evalUuid, err := uuid.NewV7()
	if err != nil {
		format := "failed to generate UUID: %w"
		errMsg := fmt.Errorf(format, err)
		return uuid.Nil, errMsg
	}

	// create an initial evaluation record
	eval := Evaluation{
		UUID:      evalUuid,
		Stage:     "waiting",
		TestRes:   testRes,
		PrLang:    lang,
		ErrorMsg:  nil,
		Params:    params,
		CreatedAt: time.Now(),
	}

	e.evalsLock.Lock()
	e.evals = append(e.evals, eval)
	e.evalsLock.Unlock()

	testsTester := make([]tester.ReqTest, len(tests))
	for i, test := range tests {
		testsTester[i] = tester.ReqTest{
			ID: i + 1,

			InSha256:  test.InSha256,
			InUrl:     test.InDownlUrl,
			InContent: test.InContent,

			AnsSha256:  test.AnsSha256,
			AnsUrl:     test.AnsDownlUrl,
			AnsContent: test.AnsContent,
		}
	}

	// prepare evaluation request
	jsonReq, err := json.Marshal(tester.EvalReq{
		EvalUuid:  evalUuid.String(),
		ResSqsUrl: e.responseSqsUrl,
		Code:      code.SrcCode,
		Language: tester.Language{
			LangID:        lang.ShortId,
			LangName:      lang.Display,
			CodeFname:     lang.CodeFname,
			CompileCmd:    lang.CompCmd,
			CompiledFname: lang.CompFname,
			ExecCmd:       lang.ExecCmd,
		},
		Tests:      testsTester,
		Checker:    params.Checker,
		Interactor: params.Interactor,
		CpuMillis:  params.CpuMs,
		MemoryKiB:  params.MemKiB,
	})
	if err != nil {
		format := "failed to marshal eval request: %w"
		errMsg := fmt.Errorf(format, err)
		return uuid.Nil, errMsg
	}

	_, err = e.sqsClient.SendMessage(context.TODO(),
		&sqs.SendMessageInput{
			QueueUrl:    aws.String(e.submSqsUrl),
			MessageBody: aws.String(string(jsonReq)),
		})
	if err != nil {
		format := "failed to send message to eval queue: %w"
		errMsg := fmt.Errorf(format, err)
		return uuid.Nil, errMsg
	}

	return evalUuid, nil
}

func getPrLangById(id string) (PrLang, error) {
	lang, err := planglist.GetProgrammingLanguageById(id)
	if err != nil {
		return PrLang{}, err
	}
	return PrLang{
		ShortId:   lang.ID,
		Display:   lang.FullName,
		CodeFname: lang.CodeFilename,
		CompCmd:   lang.CompileCmd,
		CompFname: lang.CompiledFilename,
		ExecCmd:   lang.ExecuteCmd,
	}, nil
}
