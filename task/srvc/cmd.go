package srvc

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/programme-lv/backend/common/srvcerror"
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

func (ts *TaskSrvc) UpdateStatementMd(ctx context.Context, taskId string, statement MarkdownStatement) *srvcerror.Error {
	err := ts.repo.UpdateStatement(ctx, taskId, statement)
	if err != nil {
		l := ts.logger(ctx)
		l.Error("update statement", "error", err)
		return NewErrorInternalServerError()
	}

	return nil
}

func (ts *TaskSrvc) CreateTask(ctx context.Context, task Task) *srvcerror.Error {
	err := ts.repo.CreateTask(ctx, task)
	if err != nil {
		l := ts.logger(ctx)
		l.Error("failed to create task", "error", err)
		return NewErrorInternalServerError()
	}
	return nil
}

// DeleteTask deletes a task and all its related data.
// Note: This only deletes data from the database. S3 cleanup should be handled separately.
func (ts *TaskSrvc) DeleteTask(ctx context.Context, shortId string) *srvcerror.Error {
	l := ts.logger(ctx)
	err := ts.repo.DeleteTask(ctx, shortId)
	if err != nil {
		l.Error("delete task", "task_id", shortId, "error", err)
		return NewErrorInternalServerError()
	}
	l.Info("task deleted successfully", "task_id", shortId)
	return nil
}

// S3 bucket: "proglv-public" (as of 2024-09-29)
// S3 key format: "task-pdf-statements/<sha2>.pdf"
// returns s3 key for the uploaded pdf statement
func (ts *TaskSrvc) UploadStatementPdf(ctx context.Context, body []byte) (s3Key string, err *srvcerror.Error) {
	l := ts.logger(ctx)
	shaHex := Sha2Hex(body)
	s3Key = fmt.Sprintf("%s/%s.pdf", "task-pdf-statements", shaHex)
	_, err = ts.s3PublicBucket.Upload(body, s3Key, "application/pdf")
	if err != nil {
		l.Error("upload PDF to S3", "error", err)
		return "", NewErrorInternalServerError()
	}
	return s3Key, nil
}

// S3 bucket: "proglv-public" (as of 2024-09-29)
// S3 key format: "task-illustrations/<sha2>.<ext>"
// returns s3 key for the uploaded illustration image
func (ts *TaskSrvc) UploadIllustrationImg(ctx context.Context, mimeType string, body []byte) (s3Key string, err *srvcerror.Error) {
	l := ts.logger(ctx)
	sha2 := Sha2Hex(body)
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
	s3Key = fmt.Sprintf("%s/%s%s", "task-illustrations", sha2, ext)
	_, err = ts.s3PublicBucket.Upload(body, s3Key, mimeType)
	if err != nil {
		l.Error("failed to upload illustration to S3", "error", err)
		return "", NewErrorInternalServerError()
	}
	return s3Key, nil
}

// upload and return s3 key. use a new uuid in the filename instead of a sha256 hash
func (ts *TaskSrvc) UploadOgFileArchive(ctx context.Context, zipBytes []byte) (string, *srvcerror.Error) {
	archiveUuid := uuid.New().String()
	s3Key := fmt.Sprintf("og-file-archives/%s.zip", archiveUuid)
	_, err := ts.s3PublicBucket.Upload(zipBytes, s3Key, "application/zip")
	if err != nil {
		ts.logger(ctx).Error("failed to upload og file archive to S3", "error", err)
		return "", NewErrorInternalServerError()
	}
	return s3Key, nil
}

func (ts *TaskSrvc) DownloadOgFileArchive(ctx context.Context, s3Key string) ([]byte, *srvcerror.Error) {
	body, err := ts.s3PublicBucket.Download(s3Key)
	if err != nil {
		ts.logger(ctx).Error("failed to download og file archive from S3", "error", err)
		return nil, NewErrorInternalServerError()
	}
	return body, nil
}

// S3 key format: "task-md-images/<uuid>.<extension>"
// returns s3 uri, e.g. s3://proglv-public/task/<taskId>/md-images/<uuid>.png
func (ts *TaskSrvc) UploadStatementImage(ctx context.Context, taskId string, imgFilename string, imageMimeType string, body []byte) (url string, err *srvcerror.Error) {
	l := ts.logger(ctx)

	// get the file extension from the mime type, e.g. "image/png" -> ".png"
	ext, err := MimeToFileExt(imageMimeType)
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

func MimeToFileExt(mimeType string) (string, error) {
	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return "", fmt.Errorf("failed to get file extension: %w", err)
	}
	if len(exts) == 0 {
		return "", fmt.Errorf("file extension not found")
	}
	return exts[0], nil
}

func MimeFromFname(fname string) (string, error) {
	ext := filepath.Ext(fname)
	if ext == "" {
		return "", fmt.Errorf("file extension not found")
	}
	return mime.TypeByExtension(ext), nil
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

func GetTestfileS3Key(body []byte) string {
	shaHex := Sha2Hex(body)
	return fmt.Sprintf("%s.zst", shaHex)
}

// UploadTestFile uploads a test input or output to S3 after compressing it with Zstandard.
// If The test already exists, it returns no error and does nothing.
//
// The S3 key is the SHA256 hash of the uncompressed body with a .zst extension.
func (ts *TaskSrvc) UploadTestFile(ctx context.Context, body []byte) *srvcerror.Error {
	l := ts.logger(ctx)
	s3Key := GetTestfileS3Key(body)
	mediaType := "application/zstd"

	exists, err := ts.s3TestfileBucket.Exists(s3Key)
	if err != nil {
		l.Error("failed to check if object exists in S3", "error", err)
		return NewErrorInternalServerError()
	}

	if exists {
		return nil
	}

	zstdCompressed, err := CompressWithZstd(body)
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

// CompressWithZstd compresses the given data using Zstandard compression.
// It returns the compressed data or an error if the compression fails.
func CompressWithZstd(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Zstd encoder: %w", err)
	}
	defer encoder.Close()

	compressed := encoder.EncodeAll(data, make([]byte, 0, len(data)))
	return compressed, nil
}

func MustCompressWithZstd(data []byte) []byte {
	compressed, err := CompressWithZstd(data)
	if err != nil {
		panic(err)
	}
	return compressed
}

// DecompressWithZstd decompresses data that was compressed with Zstandard.
// It returns the decompressed data or an error if the decompression fails.
func DecompressWithZstd(compressedData []byte) ([]byte, error) {
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

func Sha2Hex(body []byte) (sha2 string) {
	hash := sha256.Sum256(body)
	sha2 = fmt.Sprintf("%x", hash[:])
	return
}

// DeleteStatementImage implements TaskSrvcClient.
// It deletes an image from both S3 and the database.
func (ts *TaskSrvc) DeleteStatementImage(ctx context.Context, taskId string, filename string) *srvcerror.Error {
	l := ts.logger(ctx)

	// First, get the task to find the image with the specified filename
	t, err := ts.repo.GetTask(ctx, taskId)
	if err != nil {
		l.Error("get task", "error", err)
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
		return NewErrorImageNotFound(filename)
	}

	// The S3 key is already stored directly in the database
	s3Key := targetImage.S3Key

	// Check if the image exists in S3
	exists, err := ts.s3PublicBucket.Exists(s3Key)
	if err != nil {
		l.Error("check if image exists in S3", "error", err)
		return NewErrorInternalServerError()
	}
	if !exists {
		l.Error("image does not exist in S3", "s3_key", s3Key)
		return NewErrorInternalServerError()
	}

	// Delete the image from the database first
	err = ts.repo.DeleteStatementImg(ctx, taskId, filename)
	if err != nil {
		l.Error("delete image from database", "error", err)
		return NewErrorInternalServerError()
	}

	// Delete the image from S3
	err = ts.s3PublicBucket.Delete(s3Key)
	if err != nil {
		l.Error("delete image from S3", "error", err)
		return NewErrorInternalServerError()
	}

	return nil
}

// DeleteIllustrationImg implements TaskSrvcClient.
// It deletes an illustration image from both S3 and the database.
func (ts *TaskSrvc) DeleteIllustrationImg(ctx context.Context, taskId string) *srvcerror.Error {
	l := ts.logger(ctx)

	// First, get the task to find the illustration image S3 key
	t, err := ts.repo.GetTask(ctx, taskId)
	if err != nil {
		l.Error("get task", "error", err)
		return NewErrorFailedToGetTaskFromDb(taskId)
	}

	// Check if there's an illustration image to delete
	if t.IllustrImg.S3Key == "" {
		l.Error("no illustration image found for task", "task_id", taskId)
		return NewErrorImageNotFound(taskId)
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
func (ts *TaskSrvc) UpdateIllustrationImg(ctx context.Context, taskId string, img IllustrationImage) *srvcerror.Error {
	l := ts.logger(ctx)

	err := ts.repo.UpdateIllustrationImg(ctx, taskId, img)
	if err != nil {
		l.Error("failed to update illustration image in database", "error", err)
		return NewErrorInternalServerError()
	}

	return nil
}

// ExportTaskAsZip exports a task as a ZIP file and returns the ZIP bytes.
func (ts *TaskSrvc) ExportTaskAsZip(ctx context.Context, taskId string) ([]byte, *srvcerror.Error) {
	l := ts.logger(ctx).With("task_id", taskId)
	l.Info("starting task export")

	// fetch full task via narrow interface
	t, getTaskErr := ts.GetTask(ctx, taskId)
	if getTaskErr != nil {
		l.Error("fetch task data", "error", getTaskErr)
		return nil, NewErrorInternalServerError()
	}
	l.Info("task data fetched successfully", "full_name", t.FullName)

	l.Info("mapping task to archive format")
	task, mapToTaskfsErr := ts.mapToTaskfs(ctx, t)
	if mapToTaskfsErr != nil {
		l.Error("map task to archive format", "error", mapToTaskfsErr)
		return nil, NewErrorInternalServerError()
	}
	l.Info("task mapped to archive format successfully")

	l.Info("generating task ZIP")
	zipBytes, createTaskZipBytesErr := ts.createTaskZipBytes(task)
	if createTaskZipBytesErr != nil {
		l.Error("generate task ZIP", "error", createTaskZipBytesErr)
		return nil, NewErrorInternalServerError()
	}

	l.Info("task export completed successfully", "zip_size_bytes", len(zipBytes))
	return zipBytes, nil
}

func (ts *TaskSrvc) mapToTaskfs(ctx context.Context, t Task) (taskfs.Task, error) {
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
			MdNote: taskfs.I18N[string](example.MdNote),
		})
	}
	images := []taskfs.Image{}
	for _, image := range t.MdImages {
		content, err := ts.s3PublicBucket.Download(image.S3Key)
		if err != nil {
			prefix := "download statement image from S3"
			return taskfs.Task{}, fmt.Errorf("%s: %w", prefix, err)
		}
		images = append(images, taskfs.Image{
			Fname:   image.Filename,
			Content: content,
		})
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
				// Download input and answer files via cached downloader
				inpContent, err := ts.DownloadTestFile(ctx, testData.InpSha2)
				if err != nil {
					resultCh <- testResult{index: idx, err: fmt.Errorf("failed to download input file: %w", err)}
					return
				}

				ansContent, err := ts.DownloadTestFile(ctx, testData.AnsSha2)
				if err != nil {
					resultCh <- testResult{index: idx, err: fmt.Errorf("failed to download answer file: %w", err)}
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
				prefix := "download test file"
				return taskfs.Task{}, fmt.Errorf("%s: %w", prefix, result.err)
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

	var taskfsArchive taskfs.Archive
	if t.OgFilesZipS3Key != "" {
		taskFsArchiveBytes, err := ts.DownloadOgFileArchive(ctx, t.OgFilesZipS3Key)
		if err != nil {
			prefix := "download og file archive"
			return taskfs.Task{}, fmt.Errorf("%s: %w", prefix, err)
		} else {
			taskfsArchive, err = TaskfsArchiveFromZip(taskFsArchiveBytes)
			if err != nil {
				prefix := "parse og file archive"
				return taskfs.Task{}, fmt.Errorf("%s: %w", prefix, err)
			}
		}
	} else {
		taskfsArchive = taskfs.Archive{}
	}

	solutions := []taskfs.Solution{}
	for _, solution := range t.Solutions {
		solutions = append(solutions, taskfs.Solution{
			Fname:    solution.Fname,
			Content:  solution.Content,
			Subtasks: solution.Subtasks,
		})
	}

	res := taskfs.Task{
		ShortID:  t.ShortId,
		FullName: t.FullName,
		ReadMe:   t.Readme,
		Statement: taskfs.Statement{
			Stories:  stories,
			Subtasks: subtasks,
			Examples: examples,
			Images:   images,
		},
		Origin: taskfs.Origin{
			Lang:     t.OrigLang,
			Olympiad: t.OriginOlympiad,
			OlyStage: t.OlympStage,
			Org:      t.OriginOrg,
			Notes:    notes,
			Authors:  t.Authors,
			Year:     t.OriginYear,
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
		Archive:   taskfsArchive,
		Solutions: solutions,
		Metadata: taskfs.Metadata{
			ProblemTags: t.ProblemTags,
			Difficulty:  t.DifficultyRating,
		},
	}

	if err := res.Validate(); err != nil && etrace.IsCritical(err) {
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

// ImportTaskFromZip imports a task from a taskfs ZIP archive with optional ID override.
// If overrideId is empty, uses the original task ID from the ZIP.
// If overrideId is provided, validates and uses it instead of the original ID.
func (ts *TaskSrvc) ImportTaskFromZip(ctx context.Context, zipBytes []byte, overrideId string) (string, *srvcerror.Error) {
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
	serviceTask, err := ts.mapFromTaskfs(archTask, overrideId)
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
	err = ts.uploadTaskArchiveAssets(ctx, archTask, &serviceTask)
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

// mapFromTaskfs maps the basic task structure without uploading heavy assets
func (ts *TaskSrvc) mapFromTaskfs(t taskfs.Task, overrideId string) (Task, error) {
	res := Task{}
	// Use override ID if provided, otherwise use original from ZIP
	if overrideId != "" {
		// Validate override ID format (same rules as taskfs)
		if err := validateTaskId(overrideId); err != nil {
			return Task{}, fmt.Errorf("invalid override ID '%s': %w", overrideId, err)
		}
		res.ShortId = overrideId
	} else {
		res.ShortId = t.ShortID
	}
	// ensure illustration struct is non-nil for DB insert
	res.IllustrImg = &IllustrationImage{}
	res.FullName = t.FullName
	res.OrigLang = t.Origin.Lang
	res.OlympStage = t.Origin.OlyStage
	res.OriginOrg = t.Origin.Org
	res.OriginYear = t.Origin.Year
	res.Authors = t.Origin.Authors

	// Readme
	res.Readme = t.ReadMe

	// Constraints
	res.MemLimMegabytes = t.Testing.MemLimMiB
	res.CpuTimeLimSecs = float64(t.Testing.CpuLimMs) / 1000.0

	// Origin and metadata
	res.DifficultyRating = t.Metadata.Difficulty
	res.OriginOlympiad = t.Origin.Olympiad
	res.ProblemTags = t.Metadata.ProblemTags
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
		res.Examples = append(res.Examples, Example{Input: ex.Input, Output: ex.Output, MdNote: map[string]string(ex.MdNote)})
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
	res.VisInpSubtasks = make([]VisibleInputSubtask, 0)
	for i, st := range t.Statement.Subtasks {
		stId := i + 1
		if st.VisInput {
			tests := make([]VisInpSubtaskTest, 0)
			for _, group := range t.Scoring.Groups {
				if group.Subtask == stId {
					for testId := group.Range[0]; testId <= group.Range[1]; testId++ {
						tests = append(tests, VisInpSubtaskTest{TestId: testId, Input: t.Testing.Tests[testId-1].Input})
					}
				}
			}
			res.VisInpSubtasks = append(res.VisInpSubtasks, VisibleInputSubtask{
				SubtaskId: stId,
				Tests:     tests,
			})
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
	res.Solutions = make([]Solution, len(t.Solutions))
	for i, solution := range t.Solutions {
		res.Solutions[i] = Solution{
			Fname:    solution.Fname,
			Content:  solution.Content,
			Subtasks: solution.Subtasks,
		}
	}

	return res, nil
}

// uploadTaskArchiveAssets handles heavy uploads (images, PDFs, test files) for task import
func (ts *TaskSrvc) uploadTaskArchiveAssets(ctx context.Context, t taskfs.Task, res *Task) error {
	// Upload statement images to S3 and record metadata
	for i, img := range t.Statement.Images {
		// Detect mime type by extension
		mimeType, err := MimeFromFname(img.Fname)
		if err != nil {
			return fmt.Errorf("determine mime type for %s: %w", img.Fname, err)
		}
		width, height, err := getImgWidthHeighPx(img.Content, mimeType)
		if err != nil {
			return fmt.Errorf("get image dims for %s: %w", img.Fname, err)
		}
		newUuid := uuid.New().String()
		ext, err := MimeToFileExt(mimeType)
		if err != nil {
			return fmt.Errorf("get file ext for %s: %w", img.Fname, err)
		}
		s3Key := fmt.Sprintf("task/%s/md-images/%s%s", t.ShortID, newUuid, ext)
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
		illMime, err := MimeFromFname(ill.Fname)
		if err != nil {
			return fmt.Errorf("determine mime type for %s: %w", ill.Fname, err)
		}
		width, height, err := getImgWidthHeighPx(ill.Content, illMime)
		if err != nil {
			return fmt.Errorf("get illustr dims: %w", err)
		}
		shaHex := Sha2Hex(ill.Content)
		ext, err := MimeToFileExt(illMime)
		if err != nil {
			return fmt.Errorf("get file ext for %s: %w", ill.Fname, err)
		}
		s3Key := fmt.Sprintf("%s/%s%s", "task-illustrations", shaHex, ext)
		if _, err := ts.s3PublicBucket.Upload(ill.Content, s3Key, illMime); err != nil {
			return fmt.Errorf("upload illustration: %w", err)
		}
		res.IllustrImg = &IllustrationImage{S3Key: s3Key, WidthPx: width, HeightPx: height, SzInBytes: len(ill.Content)}
	}

	// Original PDFs (optional)
	for _, pdf := range t.Archive.GetOgStatementPdfs() {
		s3Key, err := ts.UploadStatementPdf(ctx, pdf.Content)
		if err != nil && etrace.IsCritical(err) {
			return fmt.Errorf("upload pdf: %w", err)
		}
		res.PdfStatements = append(res.PdfStatements, PdfStatement{LangIso639: pdf.Language, S3Key: s3Key})
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
		res.Tests[i] = Test{InpSha2: Sha2Hex(inpBytes), AnsSha2: Sha2Hex(ansBytes)}
	}

	// Original file archive
	zipBytes, err := TaskfsArchiveToZip(t.Archive)
	if err != nil {
		return fmt.Errorf("create zip from archive: %w", err)
	}
	s3Key, err := ts.UploadOgFileArchive(ctx, zipBytes)
	if err != nil {
		return fmt.Errorf("upload og file archive: %w", err)
	}
	res.OgFilesZipS3Key = s3Key

	return nil
}

func TaskfsArchiveToZip(archive taskfs.Archive) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	zipWriter := zip.NewWriter(buf)
	defer zipWriter.Close()
	for _, file := range archive.Files {
		writer, err := zipWriter.Create(file.RelPath)
		if err != nil {
			return nil, fmt.Errorf("create zip entry for %s: %w", file.RelPath, err)
		}
		if _, err := writer.Write(file.Content); err != nil {
			return nil, fmt.Errorf("write file content to zip: %w", err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("close zip writer: %w", err)
	}
	return buf.Bytes(), nil
}

func TaskfsArchiveFromZip(zipBytes []byte) (taskfs.Archive, error) {
	if len(zipBytes) < 4 {
		return taskfs.Archive{}, fmt.Errorf("open zip reader: data too short (%d bytes)", len(zipBytes))
	}
	// Check if it looks like a ZIP file (starts with "PK")
	if zipBytes[0] != 0x50 || zipBytes[1] != 0x4B {
		return taskfs.Archive{}, fmt.Errorf("open zip reader: not a ZIP file (starts with %02x%02x, expected 504B)", zipBytes[0], zipBytes[1])
	}
	reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return taskfs.Archive{}, fmt.Errorf("open zip reader: %w", err)
	}

	var files []taskfs.ArchiveFile
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return taskfs.Archive{}, fmt.Errorf("open file %s in zip: %w", file.Name, err)
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return taskfs.Archive{}, fmt.Errorf("read file %s content: %w", file.Name, err)
		}

		files = append(files, taskfs.ArchiveFile{
			RelPath: file.Name,
			Content: content,
		})
	}

	return taskfs.Archive{Files: files}, nil
}

func (ts *TaskSrvc) DownloadTestFile(ctx context.Context, testFileSha256 string) ([]byte, *srvcerror.Error) {
	logger := ts.logger(ctx)

	// Try cache first
	if content, found := ts.testCache.getFromCache(testFileSha256); found {
		logger.Debug("test file cache hit (direct)", "sha256", testFileSha256)
		return content, nil
	}

	// Use singleflight to prevent thundering herd for the same key
	v, err, _ := ts.dlGroup.Do(testFileSha256, func() (any, error) {
		// Re-check cache inside the singleflight window
		if content, found := ts.testCache.getFromCache(testFileSha256); found {
			return content, nil
		}

		// Download compressed from S3 by key <sha>.zst
		s3Key := fmt.Sprintf("%s.zst", testFileSha256)
		compressed, err := ts.s3TestfileBucket.Download(s3Key)
		if err != nil {
			logger.Error("failed to download test file from S3", "sha256", testFileSha256, "s3_key", s3Key, "error", err)
			return nil, NewErrorInternalServerError()
		}

		// Decompress
		content, err := DecompressWithZstd(compressed)
		if err != nil {
			logger.Error("failed to decompress test file", "sha256", testFileSha256, "error", err)
			return nil, NewErrorInternalServerError()
		}

		// Store in cache
		ts.testCache.storeInCache(testFileSha256, content, logger)
		return content, nil
	})
	if err != nil {
		if se, ok := err.(*srvcerror.Error); ok {
			return nil, se
		}
		logger.Error("unexpected error type from singleflight", "error", err)
		return nil, NewErrorInternalServerError()
	}
	return v.([]byte), nil
}
