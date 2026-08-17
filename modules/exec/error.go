package exec

import (
	"net/http"

	"github.com/programme-lv/backend/common/srvcerror"
)

var ErrInvalidTesterParams = srvcerror.New(
	"invalid_tester_params",
	"nederīgi testera parametri",
).SetHttpStatusCode(http.StatusBadRequest)

var ErrCpuConstraintTooLose = srvcerror.New(
	"constraint_too_loose",
	"CPU laika limits ir pārāk liels",
).SetHttpStatusCode(http.StatusBadRequest)

var ErrMemConstraintTooLose = srvcerror.New(
	"mem_constraint_too_loose",
	"atmiņas limits ir pārāk liels",
).SetHttpStatusCode(http.StatusBadRequest)

var ErrCheckerTooLarge = srvcerror.New(
	"checker_too_large",
	"čekera programma ir pārāk liela",
).SetHttpStatusCode(http.StatusBadRequest)

var ErrInteractorTooLarge = srvcerror.New(
	"interactor_too_large",
	"interaktora programma ir pārāk liela",
).SetHttpStatusCode(http.StatusBadRequest)

var ErrTooManyTests = srvcerror.New(
	"too_many_tests",
	"pārāk daudz testu",
).SetHttpStatusCode(http.StatusBadRequest)

var ErrEvalNotFound = srvcerror.New(
	"eval_not_found",
	"izpilde netika atrasta",
).SetHttpStatusCode(http.StatusNotFound)

var ErrInvalidTestFile = srvcerror.New(
	"invalid_test_file",
	"nederīgs testa fails",
).SetHttpStatusCode(http.StatusBadRequest)
