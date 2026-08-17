package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/common/jsonresp"
)

func (h *UserHttpHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := h.userSrvc.RequestPasswordReset(r.Context(), request.Login); err != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, err)
		return
	}

	jsonresp.Success(w, map[string]string{
		"message": "ja konts eksistē, e-pasts ar norādījumiem ir nosūtīts",
	})
}

func (h *UserHttpHandler) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := h.userSrvc.ConfirmPasswordReset(r.Context(), request.Token, request.Password); err != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, err)
		return
	}

	jsonresp.Success(w, map[string]string{
		"message": "parole atjaunota",
	})
}

func (h *UserHttpHandler) RequestEmailVerification(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := requireUserUUID(w, r)
	if !ok {
		return
	}

	if reqErr := h.userSrvc.RequestEmailVerification(r.Context(), userUUID); reqErr != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, reqErr)
		return
	}

	jsonresp.Success(w, map[string]string{
		"message": "ja e-pasts vēl nav apstiprināts, nosūtīts apstiprinājuma e-pasts",
	})
}

func (h *UserHttpHandler) ConfirmEmailVerification(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := h.userSrvc.ConfirmEmailVerification(r.Context(), request.Token); err != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, err)
		return
	}

	jsonresp.Success(w, map[string]string{
		"message": "e-pasts apstiprināts",
	})
}
