package exec

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestParseFileRequest(t *testing.T) {
	id := uuid.New()
	sha := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	req, err := parseFileRequest([]byte(`{"eval_uuid":"` + id.String() + `","sha256":"` + sha + `"}`))
	require.NoError(t, err)
	require.Equal(t, id.String(), req.EvalUUID)
	require.Equal(t, sha, req.SHA256)

	for _, data := range []string{
		`{`,
		`{"eval_uuid":"nope","sha256":"` + sha + `"}`,
		`{"eval_uuid":"` + id.String() + `","sha256":"ABCDEF0123456789abcdef0123456789abcdef0123456789abcdef0123456789"}`,
		`{"eval_uuid":"` + id.String() + `","sha256":"short"}`,
	} {
		_, err := parseFileRequest([]byte(data))
		require.Error(t, err, data)
	}
}

func TestFileAuthorization(t *testing.T) {
	id := uuid.New()
	allowed := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	srvc := &execSrvc{
		fileHashes: map[uuid.UUID]map[string]struct{}{
			id: {allowed: {}},
		},
	}

	require.True(t, srvc.fileAllowed(id.String(), allowed))
	require.False(t, srvc.fileAllowed(id.String(), "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"))
	require.False(t, srvc.fileAllowed(uuid.NewString(), allowed))
}

func TestStreamFileFrames(t *testing.T) {
	type frame struct {
		status   string
		sequence int
		data     []byte
	}
	var frames []frame
	err := streamFileFrames(bytes.NewBufferString("abcdefgh"), 3, func(status string, sequence int, data []byte) error {
		frames = append(frames, frame{status, sequence, bytes.Clone(data)})
		return nil
	}, func(err error) {
		t.Fatalf("unexpected terminal error: %v", err)
	})
	require.NoError(t, err)
	require.Equal(t, []frame{
		{"chunk", 0, []byte("abc")},
		{"chunk", 1, []byte("def")},
		{"chunk", 2, []byte("gh")},
		{"done", 0, nil},
	}, frames)
}

func TestStreamFileFramesReadError(t *testing.T) {
	readErr := errors.New("broken")
	var terminalErr error
	err := streamFileFrames(errorReader{err: readErr}, 8, func(string, int, []byte) error {
		t.Fatal("unexpected data frame")
		return nil
	}, func(err error) {
		terminalErr = err
	})
	require.ErrorIs(t, err, readErr)
	require.ErrorContains(t, terminalErr, "read test file")
}

func TestStreamFileFramesPublishError(t *testing.T) {
	for _, status := range []string{"chunk", "done"} {
		t.Run(status, func(t *testing.T) {
			publishErr := errors.New("publish unavailable")
			var terminalErr error
			err := streamFileFrames(bytes.NewBufferString("data"), 8, func(got string, _ int, _ []byte) error {
				if got == status {
					return publishErr
				}
				return nil
			}, func(err error) {
				terminalErr = err
			})
			require.ErrorIs(t, err, publishErr)
			require.ErrorIs(t, terminalErr, publishErr)
			require.ErrorContains(t, terminalErr, "publish test file "+status)
		})
	}
}

func TestFileChunkSize(t *testing.T) {
	size, err := fileChunkSize(1024)
	require.NoError(t, err)
	require.Equal(t, 512, size)

	size, err = fileChunkSize(1024 * 1024)
	require.NoError(t, err)
	require.Equal(t, maxFileChunkSize, size)

	_, err = fileChunkSize(fileHeaderReserve)
	require.Error(t, err)
}

func TestFileFrameHeaders(t *testing.T) {
	chunk := fileFrameMsg("reply", "chunk", 3, []byte("data"), nil)
	require.Equal(t, "reply", chunk.Subject)
	require.Equal(t, "chunk", chunk.Header.Get(fileStatusHeader))
	require.Equal(t, "3", chunk.Header.Get(fileSequenceHeader))
	require.Equal(t, []byte("data"), chunk.Data)

	done := fileFrameMsg("reply", "done", 0, nil, nil)
	require.Equal(t, "done", done.Header.Get(fileStatusHeader))
	require.Empty(t, done.Header.Get(fileSequenceHeader))
	require.Empty(t, done.Data)

	terminalErr := errors.New("not allowed")
	errorMsg := fileFrameMsg("reply", "error", 0, nil, terminalErr)
	require.Equal(t, "error", errorMsg.Header.Get(fileStatusHeader))
	require.Equal(t, terminalErr.Error(), errorMsg.Header.Get(fileErrorHeader))
	require.Empty(t, errorMsg.Data)
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

var _ io.Reader = errorReader{}
