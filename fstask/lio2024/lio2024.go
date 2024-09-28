package lio2024

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/programme-lv/backend/fstask"
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

	if parsedYaml.CheckerPathRelToYaml != nil {
		// TODO: implement
		log.Fatalf("found checker %s", *parsedYaml.CheckerPathRelToYaml)
		return nil, fmt.Errorf("checkers are not implemented yet")
	}

	if parsedYaml.InteractorPathRelToYaml != nil {
		// TODO: implement
		return nil, fmt.Errorf("interactors are not implemented yet")
	}

	task, err := fstask.NewTask(parsedYaml.FullTaskName)
	if err != nil {
		return nil, fmt.Errorf("failed to create new task: %v", err)
	}

	testZipAbsolutePath := filepath.Join(dirPath, parsedYaml.TestZipPathRelToYaml)

	tests, err := ReadLioTestsFromZip(testZipAbsolutePath)
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
			task.AddExample(t.Input, t.Answer, []byte(""))
			continue
		}
		id := task.AddTest(t.Input, t.Answer)
		name := fmt.Sprintf("%03d_%s", t.TestGroup, string(t.NoInTestGroup+int('a')-1))
		task.AssignFilenameToTest(name, id)
		mapTestsToTestGroups[t.TestGroup] = append(mapTestsToTestGroups[t.TestGroup], id)
	}

	for _, g := range parsedYaml.TestGroups {
		if g.GroupID == 0 {
			continue // examples
		}
		err := task.AddTestGroupWithID(g.GroupID, g.Points,
			g.Public, mapTestsToTestGroups[g.GroupID],
			g.Subtask)
		if err != nil {
			return nil, fmt.Errorf("failed to add test group: %v", err)
		}
	}

	/*
			type ParsedLio2024Yaml struct {
			CheckerPathRelToYaml    string
			InteractorPathRelToYaml string
		}
	*/
	task.CpuTimeLimInSeconds = parsedYaml.CpuTimeLimitInSeconds
	task.MemoryLimInMegabytes = parsedYaml.MemoryLimitInMegabytes

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

	task.AddVisibleInputSubtask(1)
	task.OriginOlympiad = "LIO"

	// TODO: implement adding checker and interactor if present

	return task, nil
}
