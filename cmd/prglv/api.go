package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// apiClient allows long uploads and exports of large test sets.
var apiClient = &http.Client{Timeout: 10 * time.Minute}

// resolveAPIURL picks the backend base URL: flag, then PRGLV_API_URL,
// then API_PUBLIC_BASE_URL, then the local default.
func resolveAPIURL(flagValue string) string {
	if v := strings.TrimSpace(flagValue); v != "" {
		return v
	}
	for _, name := range []string{"PRGLV_API_URL", "API_PUBLIC_BASE_URL"} {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	return defaultAPIURL
}

// resolveAPIKey picks the admin key: flag, then PRGLV_API_KEY, then ADMIN_API_KEY.
func resolveAPIKey(flagValue string) string {
	if v := strings.TrimSpace(flagValue); v != "" {
		return v
	}
	for _, name := range []string{"PRGLV_API_KEY", "ADMIN_API_KEY"} {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	return ""
}

// apiEndpoint joins base with escaped path segments.
func apiEndpoint(base string, segments ...string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid API URL %q", base)
	}
	endpoint, err := url.JoinPath(base, segments...)
	if err != nil {
		return "", fmt.Errorf("build API URL from %q: %w", base, err)
	}
	return endpoint, nil
}

// apiResponse is the JSON envelope written by common/jsonresp.
type apiResponse struct {
	Status  string          `json:"status"`
	Data    json.RawMessage `json:"data"`
	Code    string          `json:"code"`
	Message string          `json:"message"`
}

// decodeAPIResponse extracts the data payload or turns an error envelope into
// a readable error.
func decodeAPIResponse(endpoint string, statusCode int, body []byte) (json.RawMessage, error) {
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("%s: unexpected response (HTTP %d): %s",
			endpoint, statusCode, strings.TrimSpace(string(body)))
	}
	if statusCode == http.StatusOK && resp.Status == "success" {
		return resp.Data, nil
	}

	detail := strings.TrimSpace(resp.Message)
	if detail == "" && resp.Code != "" {
		detail = resp.Code
	}
	switch {
	case detail != "" && resp.Code != "" && resp.Message != "":
		return nil, fmt.Errorf("%s: %s (%s, HTTP %d)", endpoint, detail, resp.Code, statusCode)
	case detail != "":
		return nil, fmt.Errorf("%s: %s (HTTP %d)", endpoint, detail, statusCode)
	default:
		return nil, fmt.Errorf("%s: request rejected (HTTP %d)", endpoint, statusCode)
	}
}

// uploadTask posts zipBytes as multipart field task_zip and returns the
// created task ID. overrideID, when non-empty, replaces the archive ID.
func uploadTask(ctx context.Context, base, apiKey string, zipBytes []byte, overrideID string) (string, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("task_zip", "task.zip")
	if err != nil {
		return "", fmt.Errorf("build upload body: %w", err)
	}
	if _, err := part.Write(zipBytes); err != nil {
		return "", fmt.Errorf("build upload body: %w", err)
	}
	if err := mw.Close(); err != nil {
		return "", fmt.Errorf("build upload body: %w", err)
	}

	endpoint, err := apiEndpoint(base, "tasks", "upload")
	if err != nil {
		return "", err
	}
	if overrideID != "" {
		parsed, parseErr := url.Parse(endpoint)
		if parseErr != nil {
			return "", fmt.Errorf("build upload URL: %w", parseErr)
		}
		query := parsed.Query()
		query.Set("override_id", overrideID)
		parsed.RawQuery = query.Encode()
		endpoint = parsed.String()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", fmt.Errorf("build upload request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := apiClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("POST %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read %s response: %w", endpoint, err)
	}
	payload, err := decodeAPIResponse(endpoint, resp.StatusCode, data)
	if err != nil {
		return "", err
	}

	var result struct {
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return "", fmt.Errorf("%s: decode response: %w", endpoint, err)
	}
	if result.TaskID == "" {
		return "", fmt.Errorf("%s: response is missing task_id", endpoint)
	}
	return result.TaskID, nil
}

// exportTask downloads the TaskZip archive of taskID.
func exportTask(ctx context.Context, base, apiKey, taskID string) ([]byte, error) {
	endpoint, err := apiEndpoint(base, "tasks", taskID, "export")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build export request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", endpoint, err)
	}
	if resp.StatusCode != http.StatusOK {
		if _, decodeErr := decodeAPIResponse(endpoint, resp.StatusCode, data); decodeErr != nil {
			return nil, decodeErr
		}
		return nil, fmt.Errorf("%s: unexpected HTTP %d", endpoint, resp.StatusCode)
	}
	return data, nil
}
