package http

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/modules/user/auth"
)

func (h *UserHttpHandler) WhoAmI(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.CtxJwtClaimsKey).(*auth.JwtClaims)
	if !ok || claims == nil {
		jsonresp.Success(w, nil)
		return
	}

	uuid, err := uuid.Parse(claims.UUID)
	if err != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, err)
		return
	}

	user, err := h.userSrvc.GetUserByUUID(r.Context(), uuid)
	if err != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, err)
		return
	}

	jsonresp.Success(w, toHTTPUserValue(user))
}
