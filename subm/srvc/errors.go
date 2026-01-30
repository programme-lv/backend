package srvc

import (
	"fmt"
	"net/http"

	"github.com/programme-lv/backend/common/srvcerror"
)

var ErrSubmTooLong = srvcerror.New(
	"subm_too_long",
	fmt.Sprintf("Maksimālais koda garums ir %d KB.", MaxSubmLengthKB),
).SetHttpStatusCode(http.StatusBadRequest)

var ErrUserNotFound = srvcerror.New(
	"user_not_found",
	"Norādītais lietotājs netika atrasts.",
).SetHttpStatusCode(http.StatusNotFound)

var ErrInvalidProgLang = srvcerror.New(
	"invalid_programming_language",
	"Norādīta programmēšanas valoda nav iespējota.",
).SetHttpStatusCode(http.StatusBadRequest)

var ErrSubmissionNotFound = srvcerror.New(
	"submission_not_found",
	"Atbilstošais iesūtījums netika atrasts",
).SetHttpStatusCode(http.StatusNotFound)

var ErrInternal = srvcerror.ErrInternal
