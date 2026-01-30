package exec

import (
	"net/http"

	"github.com/programme-lv/backend/common/srvcerror"
)

const ErrCodeInvalidTesterParams = "invalid_tester_params"

var ErrInvalidTesterParams = srvcerror.New(
	ErrCodeInvalidTesterParams,
	"invalid tester parameters",
).SetHttpStatusCode(http.StatusBadRequest)

const ErrCodeConstraintTooLose = "constraint_too_loose"

var ErrCpuConstraintTooLose = srvcerror.New(
	ErrCodeConstraintTooLose,
	"CPU time limit too long",
).SetHttpStatusCode(http.StatusBadRequest)

const ErrCodeMemConstraintTooLose = "mem_constraint_too_loose"

var ErrMemConstraintTooLose = srvcerror.New(
	ErrCodeMemConstraintTooLose,
	"Memory limit too large",
).SetHttpStatusCode(http.StatusBadRequest)

const ErrCodeCheckerTooLarge = "checker_too_large"

var ErrCheckerTooLarge = srvcerror.New(
	ErrCodeCheckerTooLarge,
	"Checker program too large",
).SetHttpStatusCode(http.StatusBadRequest)

const ErrCodeInteractorTooLarge = "interactor_too_large"

var ErrInteractorTooLarge = srvcerror.New(
	ErrCodeInteractorTooLarge,
	"Interactor program too large",
).SetHttpStatusCode(http.StatusBadRequest)

const ErrCodeTooManyTests = "too_many_tests"

var ErrTooManyTests = srvcerror.New(
	ErrCodeTooManyTests,
	"Too many tests",
).SetHttpStatusCode(http.StatusBadRequest)

const ErrCodeEvalNotFound = "eval_not_found"

var ErrEvalNotFound = srvcerror.New(
	ErrCodeEvalNotFound,
	"Evaluation not found",
).SetHttpStatusCode(http.StatusNotFound)

const ErrCodeInvalidTestFile = "invalid_test_file"

var ErrInvalidTestFile = srvcerror.New(
	ErrCodeInvalidTestFile,
	"Invalid test file",
).SetHttpStatusCode(http.StatusBadRequest)
