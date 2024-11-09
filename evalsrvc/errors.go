package evalsrvc

import (
	"net/http"

	"github.com/programme-lv/backend/srvcerror"
)

const ErrCodeInvalidApiKey = "invalid_api_key"

func ErrInvalidApiKey() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeInvalidApiKey,
		"Nederīga API atslēga",
	).SetHttpStatusCode(http.StatusUnauthorized)
}
