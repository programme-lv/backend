package srvc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/programme-lv/taskzip/common/etrace"
	"github.com/programme-lv/taskzip/common/zips"
	"github.com/programme-lv/taskzip/taskfs"
)

// createProgLVTempDir creates a temporary directory under /tmp/proglv/
func createProgLVTempDir(pattern string) (string, error) {
	baseDir := "/tmp/proglv"
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create base temp dir: %w", err)
	}
	return os.MkdirTemp(baseDir, pattern)
}

// validateTaskId validates task ID format (same rules as taskfs.Task.Validate)
func validateTaskId(id string) error {
	if len(id) == 0 {
		return fmt.Errorf("task ID cannot be empty")
	}
	if len(id) > 20 {
		return fmt.Errorf("task ID too long, max 20 chars")
	}
	for _, r := range id {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return fmt.Errorf("task ID must contain only lowercase letters and digits")
		}
	}
	return nil
}

func (ts *TaskSrvc) UpdateStatementMd(ctx context.Context, taskId string, statement MarkdownStatement) error {
	err := ts.repo.UpdateStatement(ctx, taskId, statement)
	if err != nil {
		l := ts.logger(ctx)
		l.Error("failed to update statement", "error", err)
		return NewErrorInternalServerError()
	}

	return nil
}

func (ts *TaskSrvc) CreateTask(ctx context.Context, task Task) error {
	err := ts.repo.CreateTask(ctx, task)
	if err != nil {
		l := ts.logger(ctx)
		l.Error("failed to create task", "error", err)
		return NewErrorInternalServerError()
	}
	return nil
}

// S3 bucket: "proglv-public" (as of 2024-09-29)
// S3 key format: "task-pdf-statements/<sha2>.pdf"
func (ts *TaskSrvc) UploadStatementPdf(ctx context.Context, body []byte) (string, error) {
	l := ts.logger(ctx)
	shaHex := ts.Sha2Hex(body)
	s3Key := fmt.Sprintf("%s/%s.pdf", "task-pdf-statements", shaHex)
	url, err := ts.s3PublicBucket.Upload(body, s3Key, "application/pdf")
	if err != nil {
		l.Error("failed to upload PDF to S3", "error", err)
		return "", NewErrorInternalServerError()
	}
	return url, nil
}

// S3 bucket: "proglv-public" (as of 2024-09-29)
// S3 key format: "task-illustrations/<sha2>.<ext>"
func (ts *TaskSrvc) UploadIllustrationImg(ctx context.Context, mimeType string, body []byte) (url string, err error) {
	l := ts.logger(ctx)
	sha2 := ts.Sha2Hex(body)
	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		l.Error("failed to get file extension", "error", err)
		return "", NewErrorImageFileExtFromMimeType(mimeType)
	}
	if len(exts) == 0 {
		l.Error("file extension not found", "mime_type", mimeType)
		return "", NewErrorImageFileExtFromMimeType(mimeType)
	}
	ext := exts[0]
	s3Key := fmt.Sprintf("%s/%s%s", "task-illustrations", sha2, ext)
	url, err = ts.s3PublicBucket.Upload(body, s3Key, mimeType)
	if err != nil {
		l.Error("failed to upload illustration to S3", "error", err)
		return "", NewErrorInternalServerError()
	}
	return url, nil
}

// S3 key format: "task-md-images/<uuid>.<extension>"
// returns s3 uri, e.g. s3://proglv-public/task/<taskId>/md-images/<uuid>.png
func (ts *TaskSrvc) UploadStatementImage(ctx context.Context, taskId string, imgFilename string, imageMimeType string, body []byte) (url string, err error) {
	l := ts.logger(ctx)

	// get the file extension from the mime type, e.g. "image/png" -> ".png"
	ext, err := getImgExt(imageMimeType)
	if err != nil {
		l.Error("failed to get image file extension from mime type", "error", err)
		return "", NewErrorImageFileExtFromMimeType(imageMimeType)
	}

	// get the image width and height in pixels
	width, height, err := getImgWidthHeighPx(body, imageMimeType)
	if err != nil {
		l.Error("failed to get image width and height", "error", err)
		return "", NewErrorGetImageWidthAndHeight()
	}

	// verify that the image heas reasonable dimensions
	if width > 2000 || height > 2000 || width == 0 || height == 0 {
		return "", NewErrorImageInadequateDimensions()
	}

	// find the task to verify that it exists and the an image with the same filename does not exist
	t, err := ts.repo.GetTask(ctx, taskId)
	if err != nil {
		l.Error("failed to get corresponding task", "error", err)
		return "", NewErrorFailedToGetTaskFromDb(taskId)
	}
	for _, img := range t.MdImages {
		if img.Filename == imgFilename {
			return "", NewErrorImageAlreadyExists(imgFilename)
		}
	}

	// generate a new UUID for the image (to avoid collision and reduce complexity when renaming semantic filenames), and upload it to S3
	newImgUuid := uuid.New().String()
	s3Key := fmt.Sprintf("task/%s/md-images/%s%s", taskId, newImgUuid, ext)
	s3Uri, err := ts.s3PublicBucket.Upload(body, s3Key, imageMimeType)
	if err != nil {
		l.Error("failed to upload to S3", "error", err)
		return "", NewErrorInternalServerError()
	}

	// update the task with the new image
	err = ts.repo.AddStatementImg(ctx, taskId, StatementImage{
		S3Key:     s3Key,
		Filename:  imgFilename,
		WidthPx:   width,
		HeightPx:  height,
		SzInBytes: len(body),
	})
	if err != nil {
		l.Error("failed to add statement image to db", "error", err)
		return "", NewErrorInternalServerError()
	}
	return s3Uri, nil
}

func getImgExt(mimeType string) (string, error) {
	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return "", fmt.Errorf("failed to get file extension: %w", err)
	}
	if len(exts) == 0 {
		return "", fmt.Errorf("file extension not found")
	}
	return exts[0], nil
}

func getImgWidthHeighPx(body []byte, mimeType string) (int, int, error) {
	reader := bytes.NewReader(body)

	switch mimeType {
	case "image/png":
		img, err := png.Decode(reader)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to decode PNG image: %w", err)
		}
		return img.Bounds().Dx(), img.Bounds().Dy(), nil
	case "image/jpeg", "image/jpg":
		// For JPEGs, use the more efficient DecodeConfig which just reads the header
		config, err := jpeg.DecodeConfig(reader)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to decode JPEG image: %w", err)
		}
		return config.Width, config.Height, nil
	default:
		// Fallback for other image types
		config, _, err := image.DecodeConfig(reader)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to decode image: %w", err)
		}
		return config.Width, config.Height, nil
	}
}

// UploadTestFile uploads a test input or output to S3 after compressing it with Zstandard.
// If The test already exists, it returns no error and does nothing.
//
// The S3 key is the SHA256 hash of the uncompressed body with a .zst extension.
func (ts *TaskSrvc) UploadTestFile(ctx context.Context, body []byte) error {
	l := ts.logger(ctx)
	shaHex := ts.Sha2Hex(body)
	s3Key := fmt.Sprintf("%s.zst", shaHex)
	mediaType := "application/zstd"

	exists, err := ts.s3TestfileBucket.Exists(s3Key)
	if err != nil {
		l.Error("failed to check if object exists in S3", "error", err)
		return NewErrorInternalServerError()
	}

	if exists {
		return nil
	}

	zstdCompressed, err := compressWithZstd(body)
	if err != nil {
		l.Error("failed to compress data", "error", err)
		return NewErrorInternalServerError()
	}

	_, err = ts.s3TestfileBucket.Upload(zstdCompressed, s3Key, mediaType)
	if err != nil {
		l.Error("failed to upload to S3", "error", err)
		return NewErrorInternalServerError()
	}

	return nil
}

// compressWithZstd compresses the given data using Zstandard compression.
// It returns the compressed data or an error if the compression fails.
func compressWithZstd(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Zstd encoder: %w", err)
	}
	defer encoder.Close()

	compressed := encoder.EncodeAll(data, make([]byte, 0, len(data)))
	return compressed, nil
}

// decompressWithZstd decompresses data that was compressed with Zstandard.
// It returns the decompressed data or an error if the decompression fails.
func decompressWithZstd(compressedData []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Zstd decoder: %w", err)
	}
	defer decoder.Close()

	decompressed, err := decoder.DecodeAll(compressedData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}
	return decompressed, nil
}

func (ts *TaskSrvc) Sha2Hex(body []byte) (sha2 string) {
	hash := sha256.Sum256(body)
	sha2 = fmt.Sprintf("%x", hash[:])
	return
}

// DeleteStatementImage implements TaskSrvcClient.
// It deletes an image from both S3 and the database.
func (ts *TaskSrvc) DeleteStatementImage(ctx context.Context, taskId string, filename string) error {
	l := ts.logger(ctx)

	// First, get the task to find the image with the specified filename
	t, err := ts.repo.GetTask(ctx, taskId)
	if err != nil {
		l.Error("failed to get task", "error", err)
		return NewErrorFailedToGetTaskFromDb(taskId)
	}

	// Find the image with the specified filename
	var targetImage *StatementImage
	for _, img := range t.MdImages {
		if img.Filename == filename {
			targetImage = &img
			break
		}
	}

	if targetImage == nil {
		l.Error("image with filename not found", "filename", filename)
		return fmt.Errorf("image with filename %s does not exist for task %s", filename, taskId)
	}

	// The S3 key is already stored directly in the database
	s3Key := targetImage.S3Key

	// Check if the image exists in S3
	exists, err := ts.s3PublicBucket.Exists(s3Key)
	if err != nil {
		l.Error("failed to check if image exists in S3", "error", err)
		return NewErrorInternalServerError()
	}
	if !exists {
		l.Error("image does not exist in S3", "s3_key", s3Key)
		return NewErrorInternalServerError()
	}

	// Delete the image from the database first
	err = ts.repo.DeleteStatementImg(ctx, taskId, filename)
	if err != nil {
		l.Error("failed to delete image from database", "error", err)
		return NewErrorInternalServerError()
	}

	// Delete the image from S3
	err = ts.s3PublicBucket.Delete(s3Key)
	if err != nil {
		l.Error("failed to delete image from S3", "error", err)
		return NewErrorInternalServerError()
	}

	return nil
}

// DeleteIllustrationImg implements TaskSrvcClient.
// It deletes an illustration image from both S3 and the database.
func (ts *TaskSrvc) DeleteIllustrationImg(ctx context.Context, taskId string) error {
	l := ts.logger(ctx)

	// First, get the task to find the illustration image S3 key
	t, err := ts.repo.GetTask(ctx, taskId)
	if err != nil {
		l.Error("failed to get task", "error", err)
		return NewErrorFailedToGetTaskFromDb(taskId)
	}

	// Check if there's an illustration image to delete
	if t.IllustrImg.S3Key == "" {
		l.Error("no illustration image found for task", "task_id", taskId)
		return fmt.Errorf("no illustration image found for task %s", taskId)
	}

	s3Key := t.IllustrImg.S3Key

	// Check if the image exists in S3
	exists, err := ts.s3PublicBucket.Exists(s3Key)
	if err != nil {
		l.Error("failed to check if image exists in S3", "error", err)
		return NewErrorInternalServerError()
	}
	if !exists {
		l.Error("image does not exist in S3", "s3_key", s3Key)
		return NewErrorInternalServerError()
	}

	// Update the database to remove illustration image fields
	emptyImg := IllustrationImage{
		S3Key:     "",
		WidthPx:   0,
		HeightPx:  0,
		SzInBytes: 0,
	}
	err = ts.repo.UpdateIllustrationImg(ctx, taskId, emptyImg)
	if err != nil {
		l.Error("failed to update illustration image in database", "error", err)
		return NewErrorInternalServerError()
	}

	// Delete the image from S3
	err = ts.s3PublicBucket.Delete(s3Key)
	if err != nil {
		l.Error("failed to delete image from S3", "error", err)
		return NewErrorInternalServerError()
	}

	return nil
}

// UpdateIllustrationImg implements TaskSrvcClient.
// It updates the illustration image information in the database.
func (ts *TaskSrvc) UpdateIllustrationImg(ctx context.Context, taskId string, img IllustrationImage) error {
	l := ts.logger(ctx)

	err := ts.repo.UpdateIllustrationImg(ctx, taskId, img)
	if err != nil {
		l.Error("failed to update illustration image in database", "error", err)
		return NewErrorInternalServerError()
	}

	return nil
}

// ExportTaskAsZip exports a task as a ZIP file and returns the ZIP bytes.
func (ts *TaskSrvc) ExportTaskAsZip(ctx context.Context, taskId string) ([]byte, error) {
	logger := ts.logger(ctx).With("task_id", taskId)
	logger.Info("starting task export")

	// fetch full task via narrow interface
	t, err := ts.GetTask(ctx, taskId)
	if err != nil {
		logger.Error("failed to fetch task data", "error", err)
		return nil, err
	}
	logger.Info("task data fetched successfully", "full_name", t.FullName)

	logger.Info("mapping task to archive format")
	task, err := ts.mapToArchiveFormat(ctx, t, logger)
	if err != nil {
		logger.Error("failed to map task to archive format", "error", err)
		return nil, fmt.Errorf("failed to map task: %w", err)
	}
	logger.Info("task mapped to archive format successfully")

	logger.Info("generating task ZIP")
	zipBytes, err := ts.createTaskZipBytes(task)
	if err != nil {
		logger.Error("failed to generate task ZIP", "error", err)
		return nil, fmt.Errorf("failed to generate ZIP: %w", err)
	}

	logger.Info("task export completed successfully", "zip_size_bytes", len(zipBytes))
	return zipBytes, nil
}

func (ts *TaskSrvc) mapToArchiveFormat(ctx context.Context, t Task, logger *slog.Logger) (taskfs.Task, error) {
	stories := make(map[string]taskfs.StoryMd)
	for _, story := range t.MdStatements {
		key := story.LangIso639
		stories[key] = taskfs.StoryMd{
			Story:   story.Story,
			Input:   story.Input,
			Output:  story.Output,
			Notes:   story.Notes,
			Scoring: story.Scoring,
			Talk:    story.Talk,
			Example: story.Example,
		}
	}
	visInpSubtasks := make(map[int]bool)
	for _, visibleInputSubtask := range t.VisInpSubtasks {
		visInpSubtasks[visibleInputSubtask.SubtaskId] = true
	}
	subtasks := []taskfs.Subtask{}
	for i, subtask := range t.Subtasks {
		descriptions := make(map[string]string)
		for lang, desc := range subtask.Descriptions {
			descriptions[lang] = desc
		}
		subtasks = append(subtasks, taskfs.Subtask{
			Desc:     descriptions,
			Points:   subtask.Score,
			VisInput: visInpSubtasks[i+1],
		})
	}
	examples := []taskfs.Example{}
	for _, example := range t.Examples {
		examples = append(examples, taskfs.Example{
			Input:  example.Input,
			Output: example.Output,
			MdNote: taskfs.I18N[string]{
				"lv": example.MdNote,
			},
		})
	}
	images := []taskfs.Image{}
	if len(t.MdImages) > 0 {
		logger.Info("downloading statement images", "count", len(t.MdImages))
	}
	for i, image := range t.MdImages {
		logger.Debug("downloading statement image", "filename", image.Filename, "progress", fmt.Sprintf("%d/%d", i+1, len(t.MdImages)))
		url, err := ts.GetPublicUrlForStatementImage(ctx, image.S3Key)
		if err != nil {
			logger.Error("failed to get public URL for statement image", "filename", image.Filename, "s3_key", image.S3Key, "error", err)
			return taskfs.Task{}, err
		}
		response, err := http.Get(url)
		if err != nil {
			logger.Error("failed to download statement image", "filename", image.Filename, "url", url, "error", err)
			return taskfs.Task{}, err
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			logger.Error("failed to read statement image content", "filename", image.Filename, "error", err)
			return taskfs.Task{}, err
		}
		images = append(images, taskfs.Image{
			Fname:   image.Filename,
			Content: body,
		})
	}
	if len(t.MdImages) > 0 {
		logger.Info("statement images downloaded successfully", "count", len(t.MdImages))
	}
	notes := make(map[string]string)
	for _, note := range t.OriginNotes {
		notes[note.Lang] = note.Info
	}
	testingType := "simple"
	if t.Checker != "" {
		testingType = "checker"
	} else if t.Interactor != "" {
		testingType = "interactor"
	}
	tests := []taskfs.Test{}
	if len(t.Tests) > 0 {
		logger.Info("downloading test files with caching", "count", len(t.Tests))

		// Download test files in parallel
		type testResult struct {
			index      int
			inputData  []byte
			answerData []byte
			err        error
		}

		resultCh := make(chan testResult, len(t.Tests))

		// Worker goroutines for parallel downloads
		for i, test := range t.Tests {
			go func(idx int, testData Test) {
				// Get download URLs
				inpUrl, err := ts.GetTestDownlUrl(ctx, testData.InpSha2)
				if err != nil {
					resultCh <- testResult{index: idx, err: fmt.Errorf("failed to get input URL: %w", err)}
					return
				}

				ansUrl, err := ts.GetTestDownlUrl(ctx, testData.AnsSha2)
				if err != nil {
					resultCh <- testResult{index: idx, err: fmt.Errorf("failed to get answer URL: %w", err)}
					return
				}

				// Download input and answer files using cache
				inpContent, err := ts.testCache.GetTestFile(ctx, testData.InpSha2, inpUrl, logger)
				if err != nil {
					resultCh <- testResult{index: idx, err: fmt.Errorf("failed to get input file: %w", err)}
					return
				}

				ansContent, err := ts.testCache.GetTestFile(ctx, testData.AnsSha2, ansUrl, logger)
				if err != nil {
					resultCh <- testResult{index: idx, err: fmt.Errorf("failed to get answer file: %w", err)}
					return
				}

				resultCh <- testResult{
					index:      idx,
					inputData:  inpContent,
					answerData: ansContent,
					err:        nil,
				}
			}(i, test)
		}

		// Collect results
		testResults := make([]testResult, len(t.Tests))
		for i := 0; i < len(t.Tests); i++ {
			result := <-resultCh
			if result.err != nil {
				logger.Error("failed to download test file", "test_index", result.index, "error", result.err)
				return taskfs.Task{}, result.err
			}
			testResults[result.index] = result
		}

		// Convert to taskfs.Test in original order
		for _, result := range testResults {
			tests = append(tests, taskfs.Test{
				Input:  string(result.inputData),
				Answer: string(result.answerData),
			})
		}

		logger.Info("test files downloaded successfully", "count", len(t.Tests))
	}

	scoringType := "test-sum"
	if len(t.TestGroups) > 0 {
		scoringType = "min-groups"
	}

	totalPoints := 0
	if scoringType == "test-sum" {
		totalPoints = len(t.Tests)
	} else {
		for _, testGroup := range t.TestGroups {
			totalPoints += testGroup.Points
		}
	}

	tGroups := []taskfs.TestGroup{}
	for i, testGroup := range t.TestGroups {
		min := testGroup.TestIDs[0]
		max := testGroup.TestIDs[len(testGroup.TestIDs)-1]
		subtasks := t.FindTestgroupSubtasks(i + 1)
		if len(subtasks) == 0 {
			tGroups = append(tGroups, taskfs.TestGroup{
				Points:  testGroup.Points,
				Range:   [2]int{min, max},
				Public:  testGroup.Public,
				Subtask: 0,
			})
		} else if len(subtasks) == 1 {
			tGroups = append(tGroups, taskfs.TestGroup{
				Points:  testGroup.Points,
				Range:   [2]int{min, max},
				Public:  testGroup.Public,
				Subtask: subtasks[0],
			})
		} else {
			return taskfs.Task{}, fmt.Errorf("test group %d has multiple subtasks", i+1)
		}
	}

	res := taskfs.Task{
		ShortID: t.ShortId,
		FullName: taskfs.I18N[string]{
			"lv": t.FullName,
		},
		ReadMe: "",
		Statement: taskfs.Statement{
			Stories:  stories,
			Subtasks: subtasks,
			Examples: examples,
			Images:   images,
		},
		Origin: taskfs.Origin{
			Olympiad: t.OriginOlympiad,
			OlyStage: "",
			Org:      "",
			Notes:    notes,
			Authors:  []string{},
			Year:     "",
		},
		Testing: taskfs.Testing{
			TestingT:   testingType,
			MemLimMiB:  t.MemLimMegabytes,
			CpuLimMs:   t.CpuMillis(),
			Tests:      tests,
			Checker:    t.Checker,
			Interactor: t.Interactor,
		},
		Scoring: taskfs.Scoring{
			ScoringT: scoringType,
			TotalP:   totalPoints,
			Groups:   tGroups,
		},
		Archive:   taskfs.Archive{},
		Solutions: []taskfs.Solution{},
		Metadata: taskfs.Metadata{
			ProblemTags: []string{},
			Difficulty:  t.DifficultyRating,
		},
	}

	err := res.Validate()
	if err != nil && etrace.IsCritical(err) {
		return taskfs.Task{}, fmt.Errorf("task is invalid: %w", err)
	}
	return res, nil
}

// createTaskZipBytes creates a ZIP archive for a task and returns the bytes
func (ts *TaskSrvc) createTaskZipBytes(task taskfs.Task) ([]byte, error) {
	// Create temporary directory
	tempDir, err := createProgLVTempDir("task-export-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	// Create ZIP file path
	zipPath := filepath.Join(tempDir, task.ShortID+".zip")

	// Use taskfs.WriteZip to write directly to ZIP
	err = taskfs.WriteZip(task, zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to write task as ZIP: %w", err)
	}

	// Read the ZIP file bytes
	zipBytes, err := os.ReadFile(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ZIP file: %w", err)
	}

	return zipBytes, nil
}

// GetCacheStats returns test file cache statistics
func (ts *TaskSrvc) GetCacheStats() (totalSizeMB int64, fileCount int, err error) {
	totalSize, count, err := ts.testCache.GetCacheStats()
	if err != nil {
		return 0, 0, err
	}
	return totalSize / (1024 * 1024), count, nil
}

// ImportTaskFromZip imports a task from a taskfs ZIP archive using the original task ID from the ZIP.
func (ts *TaskSrvc) ImportTaskFromZip(ctx context.Context, zipBytes []byte) (string, error) {
	return ts.ImportTaskFromZipWithId(ctx, zipBytes, "")
}

// ImportTaskFromZipWithId imports a task from a taskfs ZIP archive with optional ID override.
// If overrideId is empty, uses the original task ID from the ZIP.
// If overrideId is provided, validates and uses it instead of the original ID.
func (ts *TaskSrvc) ImportTaskFromZipWithId(ctx context.Context, zipBytes []byte, overrideId string) (string, error) {
	l := ts.logger(ctx)
	l.Info("starting task import")

	// Create temp working dir
	tempDir, err := createProgLVTempDir("task-import-")
	if err != nil {
		l.Error("failed to create temp dir", "error", err)
		return "", NewErrorInternalServerError()
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "task.zip")
	if err := os.WriteFile(zipPath, zipBytes, 0600); err != nil {
		l.Error("failed to write zip file", "error", err)
		return "", NewErrorInternalServerError()
	}

	// Unzip to a subdir
	unzipDir := filepath.Join(tempDir, "unzipped")
	if err := os.MkdirAll(unzipDir, 0755); err != nil {
		l.Error("failed to create unzip dir", "error", err)
		return "", NewErrorInternalServerError()
	}
	if err := zips.Unzip(zipPath, unzipDir); err != nil {
		l.Error("failed to unzip archive", "error", err)
		return "", NewErrorInternalServerError()
	}

	// Many zips contain a single top-level folder. If so, dive into it.
	rootDir := unzipDir
	entries, err := os.ReadDir(unzipDir)
	if err == nil && len(entries) == 1 && entries[0].IsDir() {
		rootDir = filepath.Join(unzipDir, entries[0].Name())
	}

	archTask, err := taskfs.Read(rootDir)
	if err != nil && etrace.IsCritical(err) {
		l.Error("failed to parse taskfs directory", "error", err)
		return "", NewErrorInternalServerError()
	}

	// Map task structure first (without heavy uploads)
	serviceTask, err := ts.mapTaskStructureFromArchive(ctx, archTask, overrideId, l)
	if err != nil {
		l.Error("failed to map task structure", "error", err)
		return "", NewErrorInternalServerError()
	}

	// Check if task already exists before doing heavy uploads
	exists, err := ts.repo.Exists(ctx, serviceTask.ShortId)
	if err != nil {
		l.Error("failed to check task existence", "error", err)
		return "", NewErrorInternalServerError()
	}
	if exists {
		return "", NewErrorTaskAlreadyExists(serviceTask.ShortId)
	}

	// Now do the heavy uploads (images, PDFs, test files)
	err = ts.uploadTaskAssets(ctx, archTask, &serviceTask, l)
	if err != nil {
		l.Error("failed to upload task assets", "error", err)
		return "", NewErrorInternalServerError()
	}

	// Persist in DB
	if err := ts.repo.CreateTask(ctx, serviceTask); err != nil {
		l.Error("failed to create task in db", "error", err)
		return "", NewErrorInternalServerError()
	}

	l.Info("task import completed successfully", "task_id", serviceTask.ShortId)
	return serviceTask.ShortId, nil
}

// mapTaskStructureFromArchive maps the basic task structure without uploading heavy assets
func (ts *TaskSrvc) mapTaskStructureFromArchive(ctx context.Context, t taskfs.Task, overrideId string, logger *slog.Logger) (Task, error) {
	res := Task{}
	// Use override ID if provided, otherwise use original from ZIP
	if overrideId != "" {
		// Validate override ID format (same rules as taskfs)
		if err := validateTaskId(overrideId); err != nil {
			return Task{}, fmt.Errorf("invalid override ID '%s': %w", overrideId, err)
		}
		res.ShortId = overrideId
		logger.Info("using override task ID", "original_id", t.ShortID, "override_id", overrideId)
	} else {
		res.ShortId = t.ShortID
	}
	// ensure illustration struct is non-nil for DB insert
	res.IllustrImg = &IllustrationImage{}

	// Choose best full name (prefer lv, then en, then any)
	if name, ok := t.FullName["lv"]; ok && name != "" {
		res.FullName = name
	} else if name, ok := t.FullName["en"]; ok && name != "" {
		res.FullName = name
	} else {
		for _, name := range t.FullName {
			res.FullName = name
			break
		}
	}

	// Constraints
	res.MemLimMegabytes = t.Testing.MemLimMiB
	res.CpuTimeLimSecs = float64(t.Testing.CpuLimMs) / 1000.0

	// Origin and metadata
	res.DifficultyRating = t.Metadata.Difficulty
	res.OriginOlympiad = t.Origin.Olympiad
	for lang, note := range t.Origin.Notes {
		res.OriginNotes = append(res.OriginNotes, OriginNote{Lang: lang, Info: note})
	}

	// Statements (MD)
	for lang, story := range t.Statement.Stories {
		res.MdStatements = append(res.MdStatements, MarkdownStatement{
			LangIso639: lang,
			Story:      story.Story,
			Input:      story.Input,
			Output:     story.Output,
			Notes:      story.Notes,
			Scoring:    story.Scoring,
			Talk:       story.Talk,
			Example:    story.Example,
		})
	}

	// Examples
	for _, ex := range t.Statement.Examples {
		mdNote := ""
		if note, ok := ex.MdNote["lv"]; ok {
			mdNote = note
		} else if note, ok := ex.MdNote["en"]; ok {
			mdNote = note
		} else {
			for _, note := range ex.MdNote {
				mdNote = note
				break
			}
		}
		res.Examples = append(res.Examples, Example{Input: ex.Input, Output: ex.Output, MdNote: mdNote})
	}

	// Heavy uploads will be done separately

	// Checker/Interactor
	if t.Testing.TestingT == "checker" {
		res.Checker = t.Testing.Checker
	}
	if t.Testing.TestingT == "interactor" {
		res.Interactor = t.Testing.Interactor
	}

	// Scoring groups
	for _, group := range t.Scoring.Groups {
		testIDs := make([]int, 0, group.Range[1]-group.Range[0]+1)
		for id := group.Range[0]; id <= group.Range[1]; id++ {
			testIDs = append(testIDs, id)
		}
		res.TestGroups = append(res.TestGroups, TestGroup{Points: group.Points, Public: group.Public, TestIDs: testIDs})
	}

	// Subtasks and descriptions; derive test links from scoring groups where possible
	res.Subtasks = make([]Subtask, len(t.Statement.Subtasks))
	for i, st := range t.Statement.Subtasks {
		res.Subtasks[i] = Subtask{Score: st.Points, Descriptions: map[string]string{}}
		for lang, desc := range st.Desc {
			if res.Subtasks[i].Descriptions == nil {
				res.Subtasks[i].Descriptions = make(map[string]string)
			}
			res.Subtasks[i].Descriptions[lang] = desc
		}
	}
	// Build subtask->tests mapping from scoring groups
	for _, group := range t.Scoring.Groups {
		if group.Subtask < 1 || group.Subtask > len(res.Subtasks) {
			continue
		}
		for id := group.Range[0]; id <= group.Range[1]; id++ {
			res.Subtasks[group.Subtask-1].TestIDs = append(res.Subtasks[group.Subtask-1].TestIDs, id)
		}
	}

	return res, nil
}

// uploadTaskAssets handles heavy uploads (images, PDFs, test files) for task import
func (ts *TaskSrvc) uploadTaskAssets(ctx context.Context, t taskfs.Task, res *Task, logger *slog.Logger) error {
	// Upload statement images to S3 and record metadata
	for i, img := range t.Statement.Images {
		// Detect mime type by extension
		mimeType := "image/png"
		lower := strings.ToLower(filepath.Ext(img.Fname))
		if lower == ".jpg" || lower == ".jpeg" {
			mimeType = "image/jpeg"
		}
		width, height, err := getImgWidthHeighPx(img.Content, mimeType)
		if err != nil {
			return fmt.Errorf("get image dims for %s: %w", img.Fname, err)
		}
		newUuid := uuid.New().String()
		s3Key := fmt.Sprintf("task/%s/md-images/%s%s", t.ShortID, newUuid, filepath.Ext(img.Fname))
		if _, err := ts.s3PublicBucket.Upload(img.Content, s3Key, mimeType); err != nil {
			return fmt.Errorf("upload statement image %d: %w", i+1, err)
		}
		res.MdImages = append(res.MdImages, StatementImage{
			S3Key:     s3Key,
			Filename:  img.Fname,
			WidthPx:   width,
			HeightPx:  height,
			SzInBytes: len(img.Content),
		})
	}

	// Illustration image (optional) from reserved archive
	illustrImgs := t.Archive.GetIllustrImgs()
	if len(illustrImgs) > 0 {
		ill := illustrImgs[0]
		illMime := "image/png"
		ext := strings.ToLower(filepath.Ext(ill.Fname))
		if ext == ".jpg" || ext == ".jpeg" {
			illMime = "image/jpeg"
		}
		width, height, err := getImgWidthHeighPx(ill.Content, illMime)
		if err != nil {
			return fmt.Errorf("get illustr dims: %w", err)
		}
		shaHex := ts.Sha2Hex(ill.Content)
		s3Key := fmt.Sprintf("%s/%s%s", "task-illustrations", shaHex, ext)
		if _, err := ts.s3PublicBucket.Upload(ill.Content, s3Key, illMime); err != nil {
			return fmt.Errorf("upload illustration: %w", err)
		}
		res.IllustrImg = &IllustrationImage{S3Key: s3Key, WidthPx: width, HeightPx: height, SzInBytes: len(ill.Content)}
	}

	// Original PDFs (optional)
	for _, pdf := range t.Archive.GetOgStatementPdfs() {
		url, err := ts.UploadStatementPdf(ctx, pdf.Content)
		if err != nil {
			return fmt.Errorf("upload pdf: %w", err)
		}
		res.PdfStatements = append(res.PdfStatements, PdfStatement{LangIso639: pdf.Language, ObjectUrl: url})
	}

	// Tests: upload to testfile bucket and record sha256
	res.Tests = make([]Test, len(t.Testing.Tests))
	for i, test := range t.Testing.Tests {
		inpBytes := []byte(test.Input)
		ansBytes := []byte(test.Answer)
		if err := ts.UploadTestFile(ctx, inpBytes); err != nil {
			return fmt.Errorf("upload test input %d: %w", i+1, err)
		}
		if err := ts.UploadTestFile(ctx, ansBytes); err != nil {
			return fmt.Errorf("upload test answer %d: %w", i+1, err)
		}
		res.Tests[i] = Test{InpSha2: ts.Sha2Hex(inpBytes), AnsSha2: ts.Sha2Hex(ansBytes)}
	}

	return nil
}
