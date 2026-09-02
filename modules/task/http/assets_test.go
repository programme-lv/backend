package http

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/filestore"
	"github.com/programme-lv/backend/common/img"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/webp"
)

func TestServeIllustrationVariants(t *testing.T) {
	store, err := filestore.NewStore(t.TempDir())
	require.NoError(t, err)

	origPNG := encodeSolidPNG(t, 80, 40)
	_, err = store.Upload(origPNG, "illustrations/abc.png", "image/png")
	require.NoError(t, err)

	h := assetHandler(t, store)

	t.Run("unknown suffix is 404 and writes nothing", func(t *testing.T) {
		rec := getAsset(h, "/assets/illustrations/abc.png.w999.webp")
		require.Equal(t, http.StatusNotFound, rec.Code)
		exists, err := store.Exists("illustrations/abc.png.w999.webp")
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("query w does not create a variant", func(t *testing.T) {
		rec := getAsset(h, "/assets/illustrations/abc.png?w=80")
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "image/png", rec.Header().Get("Content-Type"))
		require.Equal(t, origPNG, rec.Body.Bytes())
		entries, err := os.ReadDir(filepath.Join(store.Root(), "illustrations"))
		require.NoError(t, err)
		require.Len(t, entries, 1)
	})

	t.Run("list variant is created then reused", func(t *testing.T) {
		rec := getAsset(h, "/assets/illustrations/abc.png.list.webp")
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "image/webp", rec.Header().Get("Content-Type"))
		require.Equal(t, img.DerivedCacheControl(), rec.Header().Get("Cache-Control"))
		cfg, err := webp.DecodeConfig(bytes.NewReader(rec.Body.Bytes()))
		require.NoError(t, err)
		require.Equal(t, 80, cfg.Width)
		require.Equal(t, 40, cfg.Height)

		info1, err := os.Stat(filepath.Join(store.Root(), "illustrations", "abc.png.list.webp"))
		require.NoError(t, err)

		rec = getAsset(h, "/assets/illustrations/abc.png.list.webp")
		require.Equal(t, http.StatusOK, rec.Code)
		info2, err := os.Stat(filepath.Join(store.Root(), "illustrations", "abc.png.list.webp"))
		require.NoError(t, err)
		require.Equal(t, info1.ModTime(), info2.ModTime())
	})

	t.Run("original url still original bytes", func(t *testing.T) {
		rec := getAsset(h, "/assets/illustrations/abc.png")
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, origPNG, rec.Body.Bytes())
	})

	t.Run("missing original derived is 404", func(t *testing.T) {
		rec := getAsset(h, "/assets/illustrations/missing.png.full.webp")
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestServeIllustrationFullWebPOriginal(t *testing.T) {
	store, err := filestore.NewStore(t.TempDir())
	require.NoError(t, err)

	origPNG := encodeSolidPNG(t, 32, 32)
	_, err = store.Upload(origPNG, "illustrations/src.png", "image/png")
	require.NoError(t, err)
	derived, err := img.EnsureVariant(t.Context(), store, "illustrations/src.png", img.VariantFull)
	require.NoError(t, err)
	webpBytes, err := store.Download(derived)
	require.NoError(t, err)

	_, err = store.Upload(webpBytes, "illustrations/abc.webp", "image/webp")
	require.NoError(t, err)

	h := assetHandler(t, store)
	rec := getAsset(h, "/assets/illustrations/abc.webp.full.webp")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, webpBytes, rec.Body.Bytes())
	exists, err := store.Exists("illustrations/abc.webp.full.webp")
	require.NoError(t, err)
	require.False(t, exists)
}

func assetHandler(t *testing.T, store *filestore.Store) http.Handler {
	t.Helper()
	h := NewTaskHttpHandler(nil, WithFileStores(store, store, []byte("test-signing-key")))
	r := chi.NewRouter()
	h.RegisterRoutes(r, []byte("jwt"), []byte("admin"), false, nil)
	return r
}

func getAsset(h http.Handler, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func encodeSolidPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			m.SetRGBA(x, y, color.RGBA{R: 200, G: 10, B: 10, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, m))
	return buf.Bytes()
}
