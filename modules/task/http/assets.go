package http

import (
	"errors"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/filestore"
	"github.com/programme-lv/backend/common/img"
	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/modules/task/storage"
)

// ServePublicAsset serves public task assets from assets/ and disposable WebP
// derivatives from cache/. Legacy .full/.list/.view suffix URLs generate the
// corresponding cache object on demand.
func (h *taskHttpHandler) ServePublicAsset(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "*")
	if strings.HasPrefix(key, "cache/") {
		if !img.ValidCacheKey(key) {
			writeNotFound(w, "asset not found")
			return
		}
		w.Header().Set("Cache-Control", img.DerivedCacheControl())
		if err := h.publicAssetStore.ServeHTTP(w, r, key); err != nil {
			w.Header().Del("Cache-Control")
			writeAssetError(w, err)
		}
		return
	}
	publicKey := path.Join(storage.AssetsRoot, key)
	if !storage.PublicImage(publicKey) {
		writeNotFound(w, "asset not found")
		return
	}
	if orig, variant, ok := img.ParseDerivedKey(publicKey); ok {
		serveKey, err := img.EnsureVariant(r.Context(), h.publicAssetStore, orig, variant)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				h.logger(r.Context()).Error("derive asset variant", "key", publicKey, "error", err)
			}
			writeAssetError(w, err)
			return
		}
		w.Header().Set("Cache-Control", img.DerivedCacheControl())
		publicKey = serveKey
	}
	if err := h.publicAssetStore.ServeHTTP(w, r, publicKey); err != nil {
		w.Header().Del("Cache-Control")
		writeAssetError(w, err)
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
