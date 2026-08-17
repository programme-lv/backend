package exec

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

const (
	fileSubjectHeader  = "Proglv-File-Subject"
	fileStatusHeader   = "Proglv-File-Status"
	fileSequenceHeader = "Proglv-File-Sequence"
	fileErrorHeader    = "Proglv-File-Error"
	maxFileChunkSize   = 512 * 1024
	fileHeaderReserve  = 512
)

type TestFileStore interface {
	Open(key string) (io.ReadCloser, error)
}

type fileRequest struct {
	EvalUUID string `json:"eval_uuid"`
	SHA256   string `json:"sha256"`
}

func (e *execSrvc) handleFileRequest(msg *nats.Msg) {
	if msg.Reply == "" {
		e.logger.Error("file request reply is empty")
		return
	}
	req, err := parseFileRequest(msg.Data)
	if err != nil {
		e.publishFileError(msg.Reply, err)
		return
	}
	if !e.fileAllowed(req.EvalUUID, req.SHA256) {
		e.publishFileError(msg.Reply, errors.New("test file not allowed"))
		return
	}
	if e.testfileStore == nil {
		e.publishFileError(msg.Reply, errors.New("test file store unavailable"))
		return
	}
	file, err := e.testfileStore.Open(req.SHA256 + ".zst")
	if err != nil {
		e.publishFileError(msg.Reply, errors.New("open test file"))
		return
	}
	defer file.Close()
	if err := e.streamFile(msg.Reply, file); err != nil {
		e.logger.Error("stream test file", "error", err)
	}
}

func parseFileRequest(data []byte) (fileRequest, error) {
	var req fileRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return req, fmt.Errorf("decode file request: %w", err)
	}
	if _, err := uuid.Parse(req.EvalUUID); err != nil {
		return req, errors.New("invalid eval_uuid")
	}
	if !validSHA256(req.SHA256) {
		return req, errors.New("invalid sha256")
	}
	return req, nil
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func (e *execSrvc) fileAllowed(evalUUID, sha256 string) bool {
	id, err := uuid.Parse(evalUUID)
	if err != nil {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, allowed := e.fileHashes[id][sha256]
	return allowed
}

func allowedFileHashes(tests []TestFile) map[string]struct{} {
	hashes := make(map[string]struct{}, len(tests)*2)
	for _, test := range tests {
		if test.InSha256 != nil {
			hashes[*test.InSha256] = struct{}{}
		}
		if test.AnsSha256 != nil {
			hashes[*test.AnsSha256] = struct{}{}
		}
	}
	return hashes
}

func (e *execSrvc) streamFile(reply string, reader io.Reader) error {
	chunkSize, err := fileChunkSize(e.natsConn.MaxPayload())
	if err != nil {
		e.publishFileError(reply, err)
		return err
	}
	return streamFileFrames(reader, chunkSize, func(status string, sequence int, data []byte) error {
		return e.publishFileFrame(reply, status, sequence, data)
	}, func(fileErr error) {
		e.publishFileError(reply, fileErr)
	})
}

func streamFileFrames(
	reader io.Reader,
	chunkSize int,
	publish func(status string, sequence int, data []byte) error,
	publishError func(error),
) error {
	buf := make([]byte, chunkSize)
	for sequence := 0; ; sequence++ {
		n, readErr := reader.Read(buf)
		if n > 0 {
			if err := publish("chunk", sequence, buf[:n]); err != nil {
				return reportPublishError("chunk", err, publishError)
			}
		}
		if errors.Is(readErr, io.EOF) {
			if err := publish("done", 0, nil); err != nil {
				return reportPublishError("done", err, publishError)
			}
			return nil
		}
		if readErr != nil {
			publishError(fmt.Errorf("read test file: %w", readErr))
			return readErr
		}
	}
}

func reportPublishError(status string, err error, publishError func(error)) error {
	wrapped := fmt.Errorf("publish test file %s: %w", status, err)
	publishError(wrapped)
	return wrapped
}

func fileChunkSize(maxPayload int64) (int, error) {
	size := min(int64(maxFileChunkSize), maxPayload-fileHeaderReserve)
	if size <= 0 {
		return 0, errors.New("NATS max payload too small")
	}
	return int(size), nil
}

func (e *execSrvc) publishFileFrame(reply, status string, sequence int, data []byte) error {
	return e.natsConn.PublishMsg(fileFrameMsg(reply, status, sequence, data, nil))
}

func fileFrameMsg(reply, status string, sequence int, data []byte, fileErr error) *nats.Msg {
	msg := nats.NewMsg(reply)
	msg.Header.Set(fileStatusHeader, status)
	if status == "chunk" {
		msg.Header.Set(fileSequenceHeader, fmt.Sprint(sequence))
	}
	if fileErr != nil {
		msg.Header.Set(fileErrorHeader, fileErr.Error())
	}
	msg.Data = data
	return msg
}

func (e *execSrvc) publishFileError(reply string, fileErr error) {
	msg := fileFrameMsg(reply, "error", 0, nil, fileErr)
	if err := e.natsConn.PublishMsg(msg); err != nil {
		e.logger.Error("publish test file error", "error", err)
	}
}
