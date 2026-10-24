package img

import (
	"context"
	"image/color"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDerivedKeyAllowlist(t *testing.T) {
	orig, v, ok := ParseDerivedKey("illustrations/abc.png.list.webp")
	require.True(t, ok)
	require.Equal(t, "illustrations/abc.png", orig)
	require.Equal(t, VariantList, v)

	orig, v, ok = ParseDerivedKey("illustrations/abc.png.view.webp")
	require.True(t, ok)
	require.Equal(t, VariantView, v)
	require.Equal(t, "illustrations/abc.png", orig)

	orig, v, ok = ParseDerivedKey("illustrations/abc.png.full.webp")
	require.True(t, ok)
	require.Equal(t, VariantFull, v)

	orig, v, ok = ParseDerivedKey("md-images/task/abc.png.full.webp")
	require.True(t, ok)
	require.Equal(t, "md-images/task/abc.png", orig)
	require.Equal(t, VariantFull, v)

	orig, v, ok = ParseDerivedKey("task-md-images/abc.png.full.webp")
	require.True(t, ok)
	require.Equal(t, "task-md-images/abc.png", orig)
	require.Equal(t, VariantFull, v)

	orig, v, ok = ParseDerivedKey("task/abc.png.full.webp")
	require.True(t, ok)
	require.Equal(t, "task/abc.png", orig)
	require.Equal(t, VariantFull, v)

	orig, v, ok = ParseDerivedKey("assets/tasks/t/illustration/abc.png.full.webp")
	require.True(t, ok)
	require.Equal(t, "assets/tasks/t/illustration/abc.png", orig)
	require.Equal(t, VariantFull, v)

	_, _, ok = ParseDerivedKey("illustrations/abc.png")
	require.False(t, ok)
	_, _, ok = ParseDerivedKey("illustrations/abc.png.w999.webp")
	require.False(t, ok)
	_, _, ok = ParseDerivedKey("testfiles/abc.png.full.webp")
	require.False(t, ok)
	_, _, ok = ParseDerivedKey("illustrations/nested/abc.png.list.webp")
	require.False(t, ok)
	_, _, ok = ParseDerivedKey("md-images/.full.webp")
	require.False(t, ok)
	_, _, ok = ParseDerivedKey("md-images/aplusb/abc.svg.full.webp")
	require.False(t, ok)
}

func TestIsDerivable(t *testing.T) {
	for _, key := range []string{
		"md-images/task/abc.png",
		"md-images/task/abc.JPG",
		"md-images/task/abc.jpeg",
		"md-images/task/abc.webp",
	} {
		require.True(t, IsDerivable(key), key)
	}
	for _, key := range []string{
		"md-images/task/abc.svg",
		"md-images/task/abc.gif",
		"md-images/task/abc",
	} {
		require.False(t, IsDerivable(key), key)
	}
}

func TestServeKeyAliasesWebPOriginalForFull(t *testing.T) {
	require.Equal(t, "illustrations/abc.webp", ServeKey("illustrations/abc.webp", VariantFull))
	require.True(t, ValidCacheKey(ServeKey("illustrations/abc.webp", VariantList)))
	require.True(t, ValidCacheKey(ServeKey("illustrations/abc.png", VariantFull)))
	require.Equal(t, DerivedKey("illustrations/abc.png", VariantFull), ServeKey("illustrations/abc.png", VariantFull))
}

func TestEnsureVariantWritesOnce(t *testing.T) {
	store := newMemStore()
	orig := "illustrations/abc.png"
	_, err := store.Upload(encodePNG(solid(80, 40, color.RGBA{R: 30, G: 60, B: 90, A: 255})), orig, "image/png")
	require.NoError(t, err)

	want := ServeKey(orig, VariantList)
	key, err := EnsureVariant(context.Background(), store, orig, VariantList)
	require.NoError(t, err)
	require.Equal(t, want, key)
	require.Equal(t, int64(2), store.uploads.Load()) // original + derived

	key, err = EnsureVariant(context.Background(), store, orig, VariantList)
	require.NoError(t, err)
	require.Equal(t, want, key)
	require.Equal(t, int64(2), store.uploads.Load())
}

func TestEnsureVariantMissingOriginal(t *testing.T) {
	store := newMemStore()
	_, err := EnsureVariant(context.Background(), store, "illustrations/missing.png", VariantFull)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Equal(t, int64(0), store.uploads.Load())
}

func TestEnsureVariantConcurrent(t *testing.T) {
	store := newMemStore()
	orig := "illustrations/abc.png"
	_, err := store.Upload(encodePNG(solid(64, 64, color.RGBA{R: 30, G: 60, B: 90, A: 255})), orig, "image/png")
	require.NoError(t, err)
	uploadsAfterOrig := store.uploads.Load()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := EnsureVariant(context.Background(), store, orig, VariantFull)
			require.NoError(t, err)
		}()
	}
	wg.Wait()
	require.Equal(t, uploadsAfterOrig+1, store.uploads.Load())
}

func TestEnsureVariantFullWebPDoesNotWrite(t *testing.T) {
	store := newMemStore()
	orig := "illustrations/abc.webp"
	src := solid(32, 32, color.RGBA{R: 30, G: 60, B: 90, A: 255})
	webpBytes, err := rasterWebP(src, 0)
	require.NoError(t, err)
	_, err = store.Upload(webpBytes, orig, "image/webp")
	require.NoError(t, err)

	key, err := EnsureVariant(context.Background(), store, orig, VariantFull)
	require.NoError(t, err)
	require.Equal(t, orig, key)
	require.Equal(t, int64(1), store.uploads.Load())
}

type memStore struct {
	mu      sync.Mutex
	m       map[string][]byte
	uploads atomic.Int64
}

func newMemStore() *memStore {
	return &memStore{m: map[string][]byte{}}
}

func (s *memStore) Upload(content []byte, key string, mediaType string) (string, error) {
	s.uploads.Add(1)
	cp := append([]byte(nil), content...)
	s.mu.Lock()
	s.m[key] = cp
	s.mu.Unlock()
	return "file://" + key, nil
}

func (s *memStore) Download(key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.m[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), b...), nil
}

func (s *memStore) Exists(key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.m[key]
	return ok, nil
}
