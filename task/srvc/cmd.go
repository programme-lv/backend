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

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/programme-lv/taskzip/common/etrace"
	"github.com/programme-lv/taskzip/taskfs"
)

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
		logger.Info("downloading test files", "count", len(t.Tests))
	}
	for i, test := range t.Tests {
		testLogger := logger.With("test_progress", fmt.Sprintf("%d/%d", i+1, len(t.Tests)))
		testLogger.Debug("downloading and decompressing test files", "inp_sha2", test.InpSha2[:8]+"...", "ans_sha2", test.AnsSha2[:8]+"...")

		// Download input file
		inpUrl, err := ts.GetTestDownlUrl(ctx, test.InpSha2)
		if err != nil {
			testLogger.Error("failed to get input download URL", "inp_sha2", test.InpSha2, "error", err)
			return taskfs.Task{}, fmt.Errorf("failed to get input download URL: %w", err)
		}
		inpResp, err := http.Get(inpUrl)
		if err != nil {
			testLogger.Error("failed to download input file", "url", inpUrl, "error", err)
			return taskfs.Task{}, fmt.Errorf("failed to download input file: %w", err)
		}
		defer inpResp.Body.Close()
		inpCompressed, err := io.ReadAll(inpResp.Body)
		if err != nil {
			testLogger.Error("failed to read input content", "error", err)
			return taskfs.Task{}, fmt.Errorf("failed to read input content: %w", err)
		}

		// Decompress the zstd-compressed input data
		inpContent, err := decompressWithZstd(inpCompressed)
		if err != nil {
			testLogger.Error("failed to decompress input content", "error", err)
			return taskfs.Task{}, fmt.Errorf("failed to decompress input content: %w", err)
		}

		// Download answer file
		ansUrl, err := ts.GetTestDownlUrl(ctx, test.AnsSha2)
		if err != nil {
			testLogger.Error("failed to get answer download URL", "ans_sha2", test.AnsSha2, "error", err)
			return taskfs.Task{}, fmt.Errorf("failed to get answer download URL: %w", err)
		}
		ansResp, err := http.Get(ansUrl)
		if err != nil {
			testLogger.Error("failed to download answer file", "url", ansUrl, "error", err)
			return taskfs.Task{}, fmt.Errorf("failed to download answer file: %w", err)
		}
		defer ansResp.Body.Close()
		ansCompressed, err := io.ReadAll(ansResp.Body)
		if err != nil {
			testLogger.Error("failed to read answer content", "error", err)
			return taskfs.Task{}, fmt.Errorf("failed to read answer content: %w", err)
		}

		// Decompress the zstd-compressed answer data
		ansContent, err := decompressWithZstd(ansCompressed)
		if err != nil {
			testLogger.Error("failed to decompress answer content", "error", err)
			return taskfs.Task{}, fmt.Errorf("failed to decompress answer content: %w", err)
		}

		tests = append(tests, taskfs.Test{
			Input:  string(inpContent),
			Answer: string(ansContent),
		})
	}
	if len(t.Tests) > 0 {
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
	tempDir, err := os.MkdirTemp("", "proglv-task-export-")
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
