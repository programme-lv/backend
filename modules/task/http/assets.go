package http

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/filestore"
	"github.com/programme-lv/backend/common/jsonresp"
)

// ServePublicAsset serves a file from the public asset store.
func (h *taskHttpHandler) ServePublicAsset(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "*")
	if err := h.publicAssetStore.ServeHTTP(w, r, key); err != nil {
		writeAssetError(w, err)
		return
	}
}

// ServeTestfile serves a .zst test file after checking the HMAC query signature.
func (h *taskHttpHandler) ServeTestfile(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if !strings.HasSuffix(filename, ".zst") {
		writeNotFound(w, "test file not found")
		return
	}

	sha256Hex := strings.TrimSuffix(filename, ".zst")
	expiresUnix, err := strconv.ParseInt(r.URL.Query().Get("expires"), 10, 64)
	if err != nil {
		jsonresp.Forbidden(w, "invalid test file signature")
		return
	}
	sig := r.URL.Query().Get("sig")
	if !filestore.ValidTestfileSignature(sha256Hex, expiresUnix, sig, h.testfileDownloadSigningKey, time.Now()) {
		jsonresp.Forbidden(w, "invalid test file signature")
		return
	}

	if err := h.testfileStore.ServeHTTP(w, r, filename); err != nil {
		writeAssetError(w, err)
		return
	}
}

func writeAssetError(w http.ResponseWriter, err error) {
	if strings.Contains(err.Error(), "object key") {
		jsonresp.BadRequest(w, err.Error())
		return
	}
	if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "no such file") {
		writeNotFound(w, "asset not found")
		return
	}
	jsonresp.InternalError(w)
}

func writeNotFound(w http.ResponseWriter, msg string) {
	_ = jsonresp.WriteCustom(w, msg, http.StatusNotFound, "http_not_found")
}
