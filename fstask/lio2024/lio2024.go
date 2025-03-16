package lio2024

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/programme-lv/backend/fstask"
	"github.com/programme-lv/backend/fstask/lio"
)

func ParseLio2024TaskDir(dirPath string) (*fstask.Task, error) {
	taskYamlPath := filepath.Join(dirPath, "task.yaml")

	taskYamlContent, err := os.ReadFile(taskYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read task.yaml: %v", err)
	}

	parsedYaml, err := ParseLio2024Yaml(taskYamlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task.yaml: %v", err)
	}

	task := &fstask.Task{}
	task.FullName = parsedYaml.FullTaskName

	if parsedYaml.CheckerPathRelToYaml != nil {
		relativePath := *parsedYaml.CheckerPathRelToYaml
		checkerPath := filepath.Join(dirPath, relativePath)
		checkerBytes, err := os.ReadFile(checkerPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read checker file: %v", err)
		}
		task.TestlibChecker = string(checkerBytes)
	}

	if parsedYaml.InteractorPathRelToYaml != nil {
		relativePath := *parsedYaml.InteractorPathRelToYaml
		interactorPath := filepath.Join(dirPath, relativePath)
		interactorBytes, err := os.ReadFile(interactorPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read interactor file: %v", err)
		}
		task.TestlibInteractor = string(interactorBytes)
	}

	testZipRelPath := parsedYaml.TestZipPathRelToYaml
	testZipAbsPath := filepath.Join(dirPath, testZipRelPath)
	tests, err := lio.ReadLioTestsFromZip(testZipAbsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tests from zip: %v", err)
	}

	sort.Slice(tests, func(i, j int) bool {
		if tests[i].TestGroup == tests[j].TestGroup {
			return tests[i].NoInTestGroup < tests[j].NoInTestGroup
		}
		return tests[i].TestGroup < tests[j].TestGroup
	})

	mapTestsToTestGroups := map[int][]int{}

	for _, t := range tests {
		if t.TestGroup == 0 {
			task.Examples = append(task.Examples, fstask.Example{
				Input:  t.Input,
				Output: t.Answer,
				MdNote: []byte{},
			})
			continue
		}
		task.Tests = append(task.Tests, fstask.Test{
			Input:  t.Input,
			Answer: t.Answer,
		})
		id := len(task.Tests)
		mapTestsToTestGroups[t.TestGroup] = append(mapTestsToTestGroups[t.TestGroup], id)
	}

	for _, g := range parsedYaml.TestGroups {
		if g.GroupID == 0 {
			continue // examples
		}
		task.TestGroups = append(task.TestGroups, fstask.TestGroup{
			Points:  g.Points,
			Public:  g.Public,
			TestIDs: mapTestsToTestGroups[g.GroupID],
		})
	}

	subtaskToTestIDs := map[int][]int{}
	subtaskToPoints := map[int]int{}
	minSubtaskID := 9999
	maxSubtaskID := -1
	for _, g := range parsedYaml.TestGroups {
		if g.GroupID == 0 {
			continue // examples
		}
		if g.Subtask == 0 {
			panic("only group id 0 (examples) can have subtask id 0")
		}
		if _, ok := subtaskToTestIDs[g.Subtask]; !ok {
			subtaskToTestIDs[g.Subtask] = []int{}
		}
		if _, ok := subtaskToPoints[g.Subtask]; !ok {
			subtaskToPoints[g.Subtask] = 0
		}
		subtaskToTestIDs[g.Subtask] = append(subtaskToTestIDs[g.Subtask], mapTestsToTestGroups[g.GroupID]...)
		subtaskToPoints[g.Subtask] += g.Points
		if g.Subtask < minSubtaskID {
			minSubtaskID = g.Subtask
		}
		if g.Subtask > maxSubtaskID {
			maxSubtaskID = g.Subtask
		}
	}

	for i := minSubtaskID; i <= maxSubtaskID; i++ {
		if _, ok := subtaskToTestIDs[i]; !ok {
			return nil, fmt.Errorf("subtask id %d does not have any tests", i)
		}
		if _, ok := subtaskToPoints[i]; !ok {
			return nil, fmt.Errorf("subtask id %d does not have any points", i)
		}
		task.Subtasks = append(task.Subtasks, fstask.Subtask{
			Points:  subtaskToPoints[i],
			TestIDs: subtaskToTestIDs[i],
			Descriptions: map[string]string{
				"lv": "",
			},
		})
	}

	// verify that subtask points sum up to 100
	totalPoints := 0
	for _, subtask := range task.Subtasks {
		totalPoints += subtask.Points
	}
	if totalPoints != 100 {
		return nil, fmt.Errorf("subtask points do not sum up to 100")
	}

	task.CPUTimeLimitSeconds = parsedYaml.CpuTimeLimitInSeconds
	task.MemoryLimitMegabytes = parsedYaml.MemoryLimitInMegabytes

	pdfFilePath := filepath.Join(dirPath, "teksts")
	pdfFiles, err := filepath.Glob(filepath.Join(pdfFilePath, "*.pdf"))
	if err != nil {
		return nil, fmt.Errorf("failed to find PDF files: %w", err)
	}

	if len(pdfFiles) == 0 {
		return nil, fmt.Errorf("no PDF files found in the directory %s", pdfFilePath)
	}
	if len(pdfFiles) > 1 {
		return nil, fmt.Errorf("more than one PDF file found in the directory (%d)", len(pdfFiles))
	}

	pdfStatementPath := pdfFiles[0]
	pdfBytes, err := os.ReadFile(pdfStatementPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF file: %w", err)
	}
	task.PdfStatements = append(task.PdfStatements,
		fstask.PdfStatement{Language: "lv", Content: pdfBytes})

	task.VisibleInputSubtasks = append(task.VisibleInputSubtasks, 1)
	task.OriginOlympiad = "LIO"

	solutionsDirPath := filepath.Join(dirPath, "risin")
	if _, err := os.Stat(solutionsDirPath); err == nil {
		filepath.WalkDir(solutionsDirPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			fileBytes, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", path, err)
			}
			task.Solutions = append(task.Solutions, fstask.Solution{
				Filename: filepath.Base(path),
				Content:  fileBytes,
			})
			return nil
		})
	}

	ignore := []string{"testi/tests.zip"}

	// get all files recursively
	filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// make the path relative to the task directory
		relativePath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return fmt.Errorf("failed to make path relative to task directory: %w", err)
		}
		for _, ignore := range ignore {
			matched, err := filepath.Match(ignore, relativePath)
			if err != nil {
				return fmt.Errorf("failed to match ignore pattern %s: %w", ignore, err)
			}
			if matched {
				return nil
			}
		}
		fileBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}
		task.ArchiveFiles = append(task.ArchiveFiles, fstask.ArchiveFile{
			RelativePath: relativePath,
			Content:      fileBytes,
		})
		return nil
	})

	return task, nil
}
