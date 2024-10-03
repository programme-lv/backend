package fstask

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

type TaskDir struct {
	AbsPath       string  // absolute path to the task
	ProblemToml   []byte  // problem.toml
	Specification Version // specification version in problem.toml
}

func Read(dir string) (*Task, error) {
	t := &Task{}

	problemToml, err := os.ReadFile(filepath.Join(dir, "problem.toml"))
	if err != nil {
		return nil, fmt.Errorf("error reading problem.toml: %w", err)
	}

	specVersion, err := getSpecVersionFromToml(problemToml)
	if err != nil {
		return nil, fmt.Errorf("error reading specification: %w", err)
	}

	absPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("error getting absolute path: %w", err)
	}

	taskDir := TaskDir{
		AbsPath:       absPath,
		Specification: specVersion,
		ProblemToml:   problemToml,
	}

	err = t.LoadConstraintsFromDir(taskDir)
	if err != nil {
		return nil, fmt.Errorf("error reading constraints: %w", err)
	}

	err = t.LoadOriginInformation(taskDir)
	if err != nil {
		return nil, fmt.Errorf("error reading origin: %w", err)
	}

	err = t.LoadTests(taskDir)
	if err != nil {
		return nil, fmt.Errorf("error reading tests: %w", err)
	}

	t.Examples, err = readExamplesDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error reading examples directory: %w", err)
	}

	err = t.LoadTestGroups(taskDir)
	if err != nil {
		return nil, fmt.Errorf("error reading test groups: %w", err)
	}

	err = t.LoadPDFStatements(taskDir)
	if err != nil {
		return nil, fmt.Errorf("error reading PDF statements: %w", err)
	}

	err = t.LoadMarkdownStatements(taskDir)
	if err != nil {
		return nil, fmt.Errorf("error reading markdown statements: %w", err)
	}

	err = t.LoadAssetFiles(taskDir)
	if err != nil {
		return nil, fmt.Errorf("error reading assets: %w", err)
	}

	err = t.LoadSolutions(taskDir)
	if err != nil {
		return nil, fmt.Errorf("error reading solutions: %w", err)
	}

	err = t.LoadArchiveFiles(taskDir)
	if err != nil {
		return nil, fmt.Errorf("error reading archive files: %w", err)
	}

	err = t.LoadCheckerAndInteractor(taskDir)
	if err != nil {
		return nil, fmt.Errorf("error reading evaluation: %w", err)
	}

	return t, nil
}
