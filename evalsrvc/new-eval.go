package evalsrvc

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/planglist"
)

// user submitted solution
type CodeWithLang struct {
	Code   string // user submitted solution source code
	LangId string // short compiler, interpreter id
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

// Enqueues code for evaluation by tester, returns eval uuid.
// 1. validates programming language;
// 2. validates cpu, memory constraints, checker and interactor size;
// 3. generates an
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

	// Enqueue for processing
	// err = e.enqueue(&req, evalUuid, e.resSqsUrl, lang)
	// if err != nil {
	// 	return uuid.Nil, err
	// }

	return evalUuid, nil
}

func getPrLangById(id string) (PrLang, error) {
	lang, err := planglist.GetProgrammingLanguageById(id)
	if err != nil {
		return PrLang{}, err
	}
	return PrLang{
		CodeFname: lang.CodeFilename,
		CompCmd:   lang.CompileCmd,
		CompFname: lang.CompiledFilename,
		ExecCmd:   lang.ExecuteCmd,
	}, nil
}
