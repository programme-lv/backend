package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/httpjson"
	"github.com/programme-lv/backend/user/auth"
)

func (h *UserHttpHandler) WhoAmI(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.CtxJwtClaimsKey).(*auth.JwtClaims)
	if !ok || claims == nil {
		httpjson.HandleError(slog.Default(), w, errors.New("not authenticated"))
		return
	}

	uuid, err := uuid.Parse(claims.UUID)
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	user, err := h.userSrvc.GetUserByUUID(r.Context(), uuid)
	if err != nil {
		httpjson.HandleError(slog.Default(), w, err)
		return
	}

	httpjson.WriteSuccessJson(w, User{
		UUID:      user.UUID.String(),
		Username:  user.Username,
		Email:     user.Email,
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
	})
}
