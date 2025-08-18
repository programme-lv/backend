package http

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/klauspost/compress/zstd"
	"github.com/programme-lv/backend/common/httpjson"
	"github.com/programme-lv/backend/task/srvc"
	"github.com/programme-lv/taskzip/common/etrace"
	"github.com/programme-lv/taskzip/taskfs"
)

// ExportTask writes the specified task into a hardcoded directory in a simple JSON form.
// For now we do not zip; this is a placeholder for using taskzip later.
// Concurrency: only one export request is processed at a time.
func (h *taskHttpHandler) ExportTask(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	taskId := chi.URLParam(r, "taskId")

	logger := h.logger(r.Context()).With(
		"handler", "ExportTask",
		"task_id", taskId,
	)

	logger.Info("starting task export")

	// serialize access
	h.exportMu.Lock()
	defer h.exportMu.Unlock()

	// If client is already gone, stop immediately to avoid doing any work.
	select {
	case <-r.Context().Done():
		logger.Warn("client disconnected before export started")
		return
	default:
	}

	// fetch full task via narrow interface
	logger.Info("fetching task data")
	t, err := h.taskSrvc.GetTask(r.Context(), taskId)
	if err != nil {
		logger.Error("failed to fetch task data", "error", err)
		httpjson.HandleSrvcError(logger, w, err)
		return
	}
	logger.Info("task data fetched successfully", "full_name", t.FullName)

	// export dir (configurable on handler; default set in constructor)
	baseDir := h.exportDir
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		logger.Error("failed to create export directory", "base_dir", baseDir, "error", err)
		httpjson.Error(w, fmt.Sprintf("failed to create export dir: %v", err), http.StatusInternalServerError, "export_mkdir_failed")
		return
	}

	dstDir := filepath.Join(baseDir, t.ShortId)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		logger.Error("failed to create task directory", "dst_dir", dstDir, "error", err)
		httpjson.Error(w, fmt.Sprintf("failed to create task dir: %v", err), http.StatusInternalServerError, "export_taskdir_failed")
		return
	}

	logger.Info("mapping task to archive format")
	task, err := mapToArchiveFormat(r.Context(), t, h.taskSrvc, logger)
	if err != nil {
		logger.Error("failed to map task to archive format", "error", err)
		httpjson.Error(w, fmt.Sprintf("failed to map task: %v", err), http.StatusInternalServerError, "export_map_failed")
		return
	}
	logger.Info("task mapped to archive format successfully")

	// write task to temporary directory
	logger.Info("creating temporary directory for export")
	tempDir, err := os.MkdirTemp("", "proglv-task-export")
	if err != nil {
		logger.Error("failed to create temporary directory", "error", err)
		httpjson.Error(w, fmt.Sprintf("failed to create temp dir: %v", err), http.StatusInternalServerError, "export_tempdir_failed")
		return
	}

	taskZipPath := filepath.Join(tempDir, t.ShortId+".zip")
	// defer os.RemoveAll(tempDir)
	// make dir inside the tempDir
	logger.Info("writing task archive to filesystem", "zip_path", taskZipPath)
	err = taskfs.WriteZip(task, taskZipPath)
	if err != nil {
		logger.Error("failed to write task archive", "zip_path", taskZipPath, "error", err)
		httpjson.Error(w, fmt.Sprintf("failed to write task: %v", err), http.StatusInternalServerError, "export_write_failed")
		return
	}

	duration := time.Since(startTime)
	logger.Info("task export completed successfully",
		"exported_path", taskZipPath,
		"duration_ms", duration.Milliseconds(),
	)

	_ = httpjson.Success(w, map[string]string{
		"status":   "ok",
		"exported": taskZipPath,
	})
}

func mapToArchiveFormat(ctx context.Context, t srvc.Task, srvc srvc.TaskSrvcClient, logger *slog.Logger) (taskfs.Task, error) {
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
		url, err := srvc.GetPublicUrlForStatementImage(ctx, image.S3Key)
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
		inpUrl, err := srvc.GetTestDownlUrl(ctx, test.InpSha2)
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
		ansUrl, err := srvc.GetTestDownlUrl(ctx, test.AnsSha2)
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
