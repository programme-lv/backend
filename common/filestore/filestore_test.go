package filestore

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStoreUploadDownloadAndServe(t *testing.T) {
	store, err := NewStore(t.TempDir())
	require.NoError(t, err)

	_, err = store.Upload([]byte("hello"), "task/a/file.txt", "text/plain")
	require.NoError(t, err)

	got, err := store.Download("task/a/file.txt")
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), got)

	req := httptest.NewRequest("GET", "/assets/task/a/file.txt", nil)
	rec := httptest.NewRecorder()
	require.NoError(t, store.ServeHTTP(rec, req, "task/a/file.txt"))
	require.Equal(t, "hello", rec.Body.String())
}

func TestCleanKeyRejectsUnsafePaths(t *testing.T) {
	for _, key := range []string{"", "/absolute", "../escape", "a/../escape", `a\escape`} {
		_, err := CleanKey(key)
		require.Error(t, err, key)
	}

	key, err := CleanKey("task/a/file.txt")
	require.NoError(t, err)
	require.Equal(t, "task/a/file.txt", key)
}

func TestSignedTestfileURLValidation(t *testing.T) {
	key := []byte("secret")
	expires := time.Now().Add(time.Hour).Unix()
	sig := SignTestfile("abc123", expires, key)

	require.True(t, ValidTestfileSignature("abc123", expires, sig, key, time.Now()))
	require.False(t, ValidTestfileSignature("abc123", expires, sig+"x", key, time.Now()))
	require.False(t, ValidTestfileSignature("abc123", time.Now().Add(-time.Hour).Unix(), sig, key, time.Now()))
}

func TestAssetURLEscapesPathSegments(t *testing.T) {
	url, err := AssetURL("https://api.programme.lv/", "task/a b/file.png")
	require.NoError(t, err)
	require.Equal(t, "https://api.programme.lv/assets/task/a%20b/file.png", url)
}
