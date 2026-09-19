package srvc

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArchiveAddsExceedQuota(t *testing.T) {
	require.NoError(t, archiveAddsExceedQuota(nil, 1, 1))
	require.NoError(t, archiveAddsExceedQuota(nil, maxArchiveFiles, maxArchiveTotalBytes))

	full := make([]ArchiveFile, maxArchiveFiles)
	require.ErrorIs(t, archiveAddsExceedQuota(full, 1, 1), ErrArchiveTooManyFiles)
	require.ErrorIs(t, archiveAddsExceedQuota(nil, maxArchiveFiles+1, 1), ErrArchiveTooManyFiles)

	require.ErrorIs(t,
		archiveAddsExceedQuota([]ArchiveFile{{SzInBytes: maxArchiveTotalBytes}}, 1, 1),
		ErrArchiveTotalTooLarge,
	)
	require.ErrorIs(t, archiveAddsExceedQuota(nil, 1, maxArchiveTotalBytes+1), ErrArchiveTotalTooLarge)
}

func TestExtractLegacyArchiveZipTaskZipKeepsArchiveDir(t *testing.T) {
	data := zipBytes(t, map[string][]byte{
		"task.toml":          []byte("taskzip = 1\n"),
		"tests/001i.txt":     []byte("1\n"),
		"archive/source.pdf": []byte("pdf"),
		"archive/old/a.txt":  []byte("a"),
	})
	files, err := extractLegacyArchiveZip(data)
	require.NoError(t, err)
	require.Equal(t, map[string][]byte{
		"source.pdf": []byte("pdf"),
		"old/a.txt":  []byte("a"),
	}, files)
}

func TestExtractLegacyArchiveZipOriginalTreeSkipsTests(t *testing.T) {
	data := zipBytes(t, map[string][]byte{
		"statement.pdf":    []byte("pdf"),
		"tests/001.in":     []byte("1"),
		"testspec/gen.cpp": []byte("c"),
		"src/main.cpp":     []byte("c++"),
	})
	files, err := extractLegacyArchiveZip(data)
	require.NoError(t, err)
	require.Equal(t, map[string][]byte{
		"statement.pdf": []byte("pdf"),
		"src/main.cpp":  []byte("c++"),
	}, files)
}

func zipBytes(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write(body)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}
