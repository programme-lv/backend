package http

import (
	"encoding/json"
	"net/http"
)

type JsonResponse struct {
	Status  string `json:"status"` // "success" or "error"
	Data    any    `json:"data"`
	ErrCode string `json:"code,omitempty"`
	ErrMsg  string `json:"message,omitempty"`
}

func writeJsonSuccessResponse(w http.ResponseWriter, data any) {
	resp := JsonResponse{
		Status: "success",
		Data:   data,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func writeJsonErrorResponse(w http.ResponseWriter, errMsg string, statusCode int, errCode string) {
	resp := JsonResponse{
		Status:  "error",
		ErrMsg:  errMsg,
		ErrCode: errCode,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

func writeJsonInternalServerError(w http.ResponseWriter) {
	writeJsonErrorResponse(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError, "internal_server_error")
}
