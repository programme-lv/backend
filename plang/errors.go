package plang

import (
	"net/http"

	"github.com/programme-lv/backend/common/srvcerror"
)

const ErrCodeInvalidProgLang = "invalid_programming_language"

func ErrInvalidProgLang() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeInvalidProgLang,
		"Nederīga programmēšanas valoda",
	).SetHttpStatusCode(http.StatusBadRequest)
}
