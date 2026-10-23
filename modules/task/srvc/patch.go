package srvc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/programme-lv/backend/common/srvcerror"
	taskzipv1 "github.com/programme-lv/backend/modules/task/taskzip"
)

// TaskManifest describes the remote state used to negotiate a sparse update.
type TaskManifest struct {
	TaskID       string            `json:"task_id"`
	TestsVersion int               `json:"tests_version"`
	TestsSHA256  string            `json:"tests_sha256"`
	TestsCount   int               `json:"tests_count"`
	ContentETag  string            `json:"content_etag"`
	Revision     int               `json:"revision"`
	Sections     map[string]string `json:"sections"`
	Files        map[string]string `json:"files"`
}

// PatchMeta is the JSON manifest sent with a sparse TaskZip update.
type PatchMeta struct {
	BaseContentETag  string   `json:"base_content_etag,omitempty"`
	BaseTestsVersion *int     `json:"base_tests_version,omitempty"`
	BaseRevision     *int     `json:"base_revision"`
	Sections         []string `json:"sections"`
	AllowTests       bool     `json:"allow_test_changes"`
}

// PatchTaskFromZip applies a sparse update. Omitted sections keep their stored
// value; sections named in PatchMeta.Sections replace the stored section.
func (ts *taskSrvc) PatchTaskFromZip(
	ctx context.Context, taskID string, patchBytes []byte, meta PatchMeta,
) (UpdateResult, srvcerror.E) {
	existing, getErr := ts.GetTask(ctx, taskID)
	if getErr != nil {
		return UpdateResult{}, getErr
	}
	if meta.BaseRevision == nil {
		return UpdateResult{}, ErrTaskConflict.WithMsg("base revision required")
	}
	if meta.BaseContentETag != "" && contentETag(existing) != meta.BaseContentETag {
		return UpdateResult{}, ErrTaskConflict.WithMsg("task changed since the manifest was read")
	}
	if meta.BaseTestsVersion != nil && *meta.BaseTestsVersion != existing.TestsVersion {
		return UpdateResult{}, ErrTaskConflict.WithMsg("tests changed since the manifest was read")
	}
	patch, err := taskzipv1.ReadPatch(patchBytes)
	if err != nil {
		return UpdateResult{}, taskZipReadError(err)
	}
	if patch.ID != taskID {
		return UpdateResult{}, errInvalidTaskZip("archive ID does not match task ID")
	}
	mapped, mapErr := mapFromTaskZip(patch, taskID)
	if mapErr != nil {
		return UpdateResult{}, errInvalidTaskZip(mapErr.Error())
	}
	sections := sectionSet(meta.Sections)
	if sections["solutions"] {
		for _, solution := range patch.Solutions {
			if solution.Data == nil {
				return UpdateResult{}, errInvalidTaskZip(
					fmt.Sprintf("missing solutions/%s", solution.Filename))
			}
		}
	}
	if sections["statement"] {
		if prepErr := prepareTaskZipImages(patch, &mapped); prepErr != nil {
			return UpdateResult{}, errInvalidTaskZip(prepErr.Error())
		}
	}
	hasTests := sections["tests"]
	if hasTests {
		mapped.Tests = taskZipTests(patch)
	}
	next := mergeTaskPatch(existing, mapped, sections, hasTests, patch.Testing.Type)
	if typeErr := patchTestingComplete(patch, next); typeErr != nil {
		return UpdateResult{}, errInvalidTaskZip(typeErr.Error())
	}
	if verifyErr := validateMergedTask(next); verifyErr != nil {
		return UpdateResult{}, errInvalidTaskZip(verifyErr.Error())
	}
	visible, visErr := ts.visibleInputs(ctx, patch, mapped, next, existing, hasTests)
	if visErr != nil {
		return UpdateResult{}, errInvalidTaskZip(visErr.Error())
	}
	if visErr := validateVisibleInputs(next, visible); visErr != nil {
		return UpdateResult{}, errInvalidTaskZip(visErr.Error())
	}
	next.VisInpSubtasks = visible
	testsChanged := taskTestsChanged(existing, next)
	if testsChanged && !meta.AllowTests {
		return UpdateResult{}, ErrTaskTestsChanged
	}
	if sections["statement"] {
		if uploadErr := ts.uploadTaskZipImages(ctx, patch, &mapped); uploadErr != nil {
			return UpdateResult{}, uploadErr
		}
	}
	if hasTests && testsChanged {
		if uploadErr := ts.uploadTaskZipTests(ctx, patch, &mapped); uploadErr != nil {
			return UpdateResult{}, uploadErr
		}
	}
	if testsChanged {
		next.TestsVersion++
	}
	if repoErr := ts.repo.UpdateTask(ctx, next, testsChanged, *meta.BaseRevision); repoErr != nil {
		if errors.Is(repoErr, ErrRevisionConflict) {
			return UpdateResult{}, ErrTaskConflict.WithMsg("task changed while applying the patch")
		}
		ts.logger(ctx).Error("patch task", "task_id", taskID, "error", repoErr)
		return UpdateResult{}, srvcerror.InternalServerError()
	}
	return UpdateResult{
		TaskID: next.ShortId, TestsChanged: testsChanged,
		TestsVersion: next.TestsVersion, Revision: existing.Revision + 1,
	}, nil
}

// GetTaskManifest returns hashes for update negotiation without test payloads.
func (ts *taskSrvc) GetTaskManifest(ctx context.Context, taskID string) (TaskManifest, srvcerror.E) {
	task, getErr := ts.GetTask(ctx, taskID)
	if getErr != nil {
		return TaskManifest{}, getErr
	}
	sections, files, err := ts.sectionDigests(task)
	if err != nil {
		ts.logger(ctx).Error("hash task sections", "task_id", taskID, "error", err)
		return TaskManifest{}, srvcerror.InternalServerError()
	}
	return TaskManifest{
		TaskID:       task.ShortId,
		TestsVersion: task.TestsVersion,
		TestsSHA256:  testsDigest(task.Tests),
		TestsCount:   len(task.Tests),
		ContentETag:  contentETag(task),
		Revision:     task.Revision,
		Sections:     sections,
		Files:        files,
	}, nil
}

func patchTestingComplete(patch taskzipv1.Task, merged Task) error {
	switch patch.Testing.Type {
	case "checker":
		if merged.Checker == "" {
			return fmt.Errorf("checker.cpp missing")
		}
	case "interactor":
		if merged.Interactor == "" {
			return fmt.Errorf("interactor.cpp missing")
		}
	}
	return nil
}

func validateVisibleInputs(task Task, visible []VisibleInputSubtask) error {
	seen := map[int]bool{}
	for _, vis := range visible {
		if vis.SubtaskId < 1 || vis.SubtaskId > len(task.Subtasks) || seen[vis.SubtaskId] {
			return fmt.Errorf("invalid visible subtask %d", vis.SubtaskId)
		}
		seen[vis.SubtaskId] = true
		for _, test := range vis.Tests {
			if test.TestId < 1 || test.TestId > len(task.Tests) {
				return fmt.Errorf("visible test %d out of range", test.TestId)
			}
		}
	}
	return nil
}

func validateMergedTask(task Task) error {
	mapped, err := mapToTaskZip(task)
	if err != nil {
		return err
	}
	for _, image := range task.MdImages {
		mapped.StatementImages[image.Filename] = nil
	}
	mapped.Tests = make([]taskzipv1.Test, len(task.Tests))
	for i := range mapped.Tests {
		mapped.Tests[i] = taskzipv1.Test{Input: []byte("x\n"), Output: []byte{}}
	}
	return taskzipv1.ValidateMerged(&mapped)
}

func (ts *taskSrvc) visibleInputs(
	ctx context.Context, patch taskzipv1.Task, mapped, next Task,
	existing Task, hasTests bool,
) ([]VisibleInputSubtask, error) {
	if hasTests {
		return mapped.VisInpSubtasks, nil
	}
	inputs := map[int]string{}
	for _, vis := range existing.VisInpSubtasks {
		for _, test := range vis.Tests {
			inputs[test.TestId] = test.Input
		}
	}
	if len(patch.Subtasks) != len(next.Subtasks) {
		return nil, fmt.Errorf("subtask count changed without tests")
	}
	var out []VisibleInputSubtask
	for i, subtask := range patch.Subtasks {
		if !subtask.VisibleInput {
			continue
		}
		vis := VisibleInputSubtask{SubtaskId: i + 1}
		for _, testID := range next.Subtasks[i].TestIDs {
			input, ok := inputs[testID]
			if !ok {
				if testID < 1 || testID > len(next.Tests) {
					return nil, fmt.Errorf("visible test %d out of range", testID)
				}
				if ts.testfileStore == nil {
					return nil, fmt.Errorf("test file store unavailable")
				}
				data, err := ts.testfileStore.Download(next.Tests[testID-1].InpSha2)
				if err != nil {
					return nil, fmt.Errorf("download visible test %d: %w", testID, err)
				}
				input = string(data)
			}
			vis.Tests = append(vis.Tests, VisInpSubtaskTest{
				TestId: testID, Input: input,
			})
		}
		out = append(out, vis)
	}
	return out, nil
}

func (ts *taskSrvc) sectionDigests(task Task) (map[string]string, map[string]string, error) {
	mapped, err := mapToTaskZip(task)
	if err != nil {
		return nil, nil, err
	}
	if len(mapped.Readme) == 0 {
		mapped.Readme = nil
	}
	for _, image := range task.MdImages {
		data, err := ts.publicStore.Download(taskStatementImageObjectKey(image.ObjectKey))
		if err != nil {
			return nil, nil, fmt.Errorf("download image %s: %w", image.Filename, err)
		}
		mapped.StatementImages[image.Filename] = data
	}
	files, err := taskzipv1.FileMap(mapped)
	if err != nil {
		return nil, nil, err
	}
	hashes := map[string]string{}
	for name, data := range files {
		if managedPath(name) {
			hashes[name] = sha256Hex(data)
		}
	}
	for name, sha := range testFileHashes(task.Tests) {
		hashes[name] = sha
	}
	out := map[string]string{
		"task":       digestTask(files),
		"statement":  digestSection(files, "statement/"),
		"examples":   digestSection(files, "examples/"),
		"solutions":  digestSection(files, "solutions/"),
		"tests":      testsDigest(task.Tests),
		"readme":     digestOptional(files, "readme.md"),
		"checker":    digestOptional(files, "checker.cpp"),
		"interactor": digestOptional(files, "interactor.cpp"),
	}
	return out, hashes, nil
}

func managedPath(name string) bool {
	if name == "task.toml" || name == "tgroups.txt" {
		return true
	}
	return strings.HasPrefix(name, "statement/") ||
		strings.HasPrefix(name, "examples/") ||
		strings.HasPrefix(name, "solutions/") ||
		strings.HasPrefix(name, "tests/") ||
		name == "readme.md" || name == "checker.cpp" || name == "interactor.cpp"
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func digestSection(files map[string][]byte, prefix string) string {
	keys := make([]string, 0)
	for name := range files {
		if strings.HasPrefix(name, prefix) {
			keys = append(keys, name)
		}
	}
	return digestFiles(files, keys)
}

func digestTask(files map[string][]byte) string {
	var keys []string
	for _, name := range []string{"task.toml", "tgroups.txt"} {
		if _, ok := files[name]; ok {
			keys = append(keys, name)
		}
	}
	return digestFiles(files, keys)
}

func digestOptional(files map[string][]byte, name string) string {
	if _, ok := files[name]; !ok {
		return digestFiles(files, nil)
	}
	return digestFiles(files, []string{name})
}

func digestFiles(files map[string][]byte, keys []string) string {
	sort.Strings(keys)
	h := sha256.New()
	for _, name := range keys {
		sum := sha256.Sum256(files[name])
		h.Write([]byte(name))
		h.Write([]byte{0})
		h.Write([]byte(hex.EncodeToString(sum[:])))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func mergeTaskPatch(existing, mapped Task, sections map[string]bool, hasTests bool, kind string) Task {
	next := existing
	next.FullName = mapped.FullName
	next.OrigLang = mapped.OrigLang
	next.MemLimMegabytes = mapped.MemLimMegabytes
	next.CpuTimeLimSecs = mapped.CpuTimeLimSecs
	next.OriginOlympiad = mapped.OriginOlympiad
	next.OriginOrg = mapped.OriginOrg
	next.OriginYear = mapped.OriginYear
	next.OlympStage = mapped.OlympStage
	next.OriginDivisions = mapped.OriginDivisions
	next.Authors = mapped.Authors
	next.ProblemTags = mapped.ProblemTags
	next.DifficultyRating = mapped.DifficultyRating
	switch kind {
	case "checker":
		next.Interactor = ""
		if sections["checker"] {
			next.Checker = mapped.Checker
		}
	case "interactor":
		next.Checker = ""
		if sections["interactor"] {
			next.Interactor = mapped.Interactor
		}
	default:
		next.Checker = ""
		next.Interactor = ""
	}
	next.TestGroups = mapped.TestGroups
	next.Subtasks = mapped.Subtasks
	if sections["readme"] {
		next.Readme = mapped.Readme
	}
	if sections["statement"] {
		next.MdStatements = mapped.MdStatements
		next.MdImages = mapped.MdImages
	}
	if sections["examples"] {
		next.Examples = mapped.Examples
	}
	if sections["solutions"] {
		next.Solutions = mapped.Solutions
	}
	if hasTests {
		next.Tests = mapped.Tests
	}
	return next
}

func sectionSet(sections []string) map[string]bool {
	out := make(map[string]bool, len(sections))
	for _, section := range sections {
		out[section] = true
	}
	return out
}

func testsDigest(tests []Test) string {
	hashes := testFileHashes(tests)
	keys := make([]string, 0, len(hashes))
	for name := range hashes {
		keys = append(keys, name)
	}
	return digestHashes(hashes, keys)
}

func testFileHashes(tests []Test) map[string]string {
	hashes := make(map[string]string, len(tests)*2)
	for i, test := range tests {
		n := i + 1
		hashes[fmt.Sprintf("tests/%03di.txt", n)] = test.InpSha2
		hashes[fmt.Sprintf("tests/%03do.txt", n)] = test.AnsSha2
	}
	return hashes
}

func digestHashes(hashes map[string]string, keys []string) string {
	sort.Strings(keys)
	h := sha256.New()
	for _, name := range keys {
		h.Write([]byte(name))
		h.Write([]byte{0})
		h.Write([]byte(hashes[name]))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func contentETag(task Task) string {
	task.OriginNotes = append([]OriginNote(nil), task.OriginNotes...)
	sort.Slice(task.OriginNotes, func(i, j int) bool {
		return task.OriginNotes[i].Lang < task.OriginNotes[j].Lang
	})
	task.MdStatements = append([]MarkdownStatement(nil), task.MdStatements...)
	sort.Slice(task.MdStatements, func(i, j int) bool {
		return task.MdStatements[i].LangIso639 < task.MdStatements[j].LangIso639
	})
	task.MdImages = append([]StatementImage(nil), task.MdImages...)
	sort.Slice(task.MdImages, func(i, j int) bool {
		return task.MdImages[i].Filename < task.MdImages[j].Filename
	})
	task.Solutions = append([]Solution(nil), task.Solutions...)
	sort.Slice(task.Solutions, func(i, j int) bool {
		return task.Solutions[i].Fname < task.Solutions[j].Fname
	})
	payload, err := json.Marshal(task)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
