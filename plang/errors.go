package plang

import (
	"net/http"

	"github.com/programme-lv/backend/common/srvcerror"
)

var ErrInvalidProgLang = srvcerror.New(
	"invalid_programming_language",
	"Norādīta programmēšanas valoda nav pieejama.",
).SetHttpStatusCode(http.StatusBadRequest)
