package evalsrvc

import (
	"net/http"

	"github.com/programme-lv/backend/srvcerror"
)

const ErrCodeInvalidTesterParams = "invalid_tester_params"

func ErrInvalidTesterParams() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeInvalidTesterParams,
		"Invalid tester parameters",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeConstraintTooLose = "constraint_too_loose"

func ErrCpuConstraintTooLose() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeConstraintTooLose,
		"CPU time limit too long",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeMemConstraintTooLose = "mem_constraint_too_loose"

func ErrMemConstraintTooLose() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeMemConstraintTooLose,
		"Memory limit too large",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeCheckerTooLarge = "checker_too_large"

func ErrCheckerTooLarge() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeCheckerTooLarge,
		"Checker program too large",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeInteractorTooLarge = "interactor_too_large"

func ErrInteractorTooLarge() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeInteractorTooLarge,
		"Interactor program too large",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeEvalNotFound = "eval_not_found"

func ErrEvalNotFound() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeEvalNotFound,
		"Evaluation not found",
	).SetHttpStatusCode(http.StatusNotFound)
}

const ErrCodeInvalidTestFile = "invalid_test_file"

func ErrInvalidTestFile() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeInvalidTestFile,
		"Invalid test file",
	).SetHttpStatusCode(http.StatusBadRequest)
}
