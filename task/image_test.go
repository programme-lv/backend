package task_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/task/srvc"
	"github.com/programme-lv/backend/user/auth"
	"github.com/stretchr/testify/require"
)

func TestPostStatementImageHttpRequest(t *testing.T) {
	ts := NewTaskSrvc(t)
	taskHttpHandler := NewTaskHttpHandler(t, ts)

	err := ts.CreateTask(context.Background(), srvc.Task{
		ShortId: "aplusb",
	})
	require.NoError(t, err)

	// 1. Try uploading without authentication - should fail
	w := UploadStatementImage(t, taskHttpHandler, "aplusb", "./testdata/seifs.png", "")
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// 2. Generate a valid auth token
	token, err := auth.GenerateJWT(
		"admin",
		"admin@example.com", uuid.Nil,
		[]byte("test"), 24*time.Hour)
	require.NoError(t, err)

	// 3. Upload with authentication - should succeed
	w = UploadStatementImage(t, taskHttpHandler, "aplusb", "./testdata/seifs.png", token)
	require.Equal(t, http.StatusOK, w.Code)

	// 4. Verify the image was uploaded by checking the task
	task, err := ts.GetTask(context.Background(), "aplusb")
	require.NoError(t, err)
	require.Equal(t, 1, len(task.MdImages))

	// 5. Verify image properties
	img := task.MdImages[0]
	require.Equal(t, "seifs.png", img.Filename)
	require.Greater(t, img.WidthPx, 0)
	require.Greater(t, img.HeightPx, 0)
	require.Contains(t, img.S3Uri, "s3://")
	t.Logf("s3 uri: %s", img.S3Uri)

	// 6. Test deleting the image without authentication - should fail
	w = DeleteStatementImage(t, taskHttpHandler, "aplusb", img.S3Uri, "")
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// 7. Test deleting the image with authentication - should succeed
	w = DeleteStatementImage(t, taskHttpHandler, "aplusb", img.S3Uri, token)
	require.Equal(t, http.StatusOK, w.Code)

	// 8. Verify the image was deleted by checking the task
	task, err = ts.GetTask(context.Background(), "aplusb")
	require.NoError(t, err)
	require.Equal(t, 0, len(task.MdImages))
}

func UploadStatementImage(t *testing.T, h http.Handler, taskId string, imagePath string, token string) *httptest.ResponseRecorder {
	// Prepare a multipart form with an image file
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Open the image file
	file, err := os.Open(imagePath)
	require.NoError(t, err)
	defer file.Close()

	// Create a form file field
	formFile, err := w.CreateFormFile("image", filepath.Base(imagePath))
	require.NoError(t, err)

	// Copy the image content to the form file
	_, err = io.Copy(formFile, file)
	require.NoError(t, err)

	// Close the multipart writer
	err = w.Close()
	require.NoError(t, err)

	// Create a request with the form
	url := chi.URLParam(httptest.NewRequest("POST", "/tasks/"+taskId+"/images", nil), "taskId")
	if url == "" {
		url = taskId
	}
	req, err := http.NewRequest("POST", "/tasks/"+url+"/images", &b)
	require.NoError(t, err)

	// Set the content type
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Add auth token if provided
	if token != "" {
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: token,
		})
	}

	// Execute the request
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	return rec
}

func DeleteStatementImage(t *testing.T, h http.Handler, taskId string, s3Uri string, token string) *httptest.ResponseRecorder {
	// URL encode the S3 URI since it may contain characters like '/'
	encodedS3Uri := url.QueryEscape(s3Uri)

	// Create the request URL
	reqURL := "/tasks/" + taskId + "/images/" + encodedS3Uri

	// Create a new request
	req, err := http.NewRequest("DELETE", reqURL, nil)
	require.NoError(t, err)

	// Add auth token if provided
	if token != "" {
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: token,
		})
	}

	// Execute the request
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	return rec
}
