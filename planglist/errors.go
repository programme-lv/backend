package planglist

import (
	"net/http"

	"github.com/programme-lv/backend/srvcerror"
)

const ErrCodeInvalidProgLang = "invalid_programming_language"

func ErrInvalidProgLang() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeInvalidProgLang,
		"Nederīga programmēšanas valoda",
	).SetHttpStatusCode(http.StatusBadRequest)
}
