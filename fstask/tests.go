package fstask

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (dir TaskDir) ReadTests() ([]Test, error) {
	testDirPath := filepath.Join(dir.AbsPath, "tests")
	entries, err := os.ReadDir(testDirPath)
	if err != nil {
		return nil, fmt.Errorf("error reading tests directory: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	tests := make([]Test, 0, len(entries)/2)

	for i := 0; i < len(entries); i += 2 {
		inPath := filepath.Join(testDirPath, entries[i].Name())
		ansPath := filepath.Join(testDirPath, entries[i+1].Name())

		inFilename := entries[i].Name()
		ansFilename := entries[i+1].Name()

		inFilenameBase := strings.TrimSuffix(inFilename, filepath.Ext(inFilename))
		ansFilenameBase := strings.TrimSuffix(ansFilename, filepath.Ext(ansFilename))

		if inFilenameBase != ansFilenameBase {
			return nil, fmt.Errorf("input and answer file base names do not match: %s, %s", inFilenameBase, ansFilenameBase)
		}

		if strings.Contains(inFilename, ".ans") || strings.Contains(ansFilename, ".in") {
			inPath, ansPath = ansPath, inPath
		}

		input, err := os.ReadFile(inPath)
		if err != nil {
			return nil, fmt.Errorf("error reading input file: %w", err)
		}

		answer, err := os.ReadFile(ansPath)
		if err != nil {
			return nil, fmt.Errorf("error reading answer file: %w", err)
		}

		tests = append(tests, Test{
			Input:  input,
			Answer: answer,
		})
	}

	return tests, nil
}

func (task *Task) LoadTests(dir TaskDir) error {
	tests, err := dir.ReadTests()
	if err != nil {
		return fmt.Errorf("failed to read tests: %w", err)
	}
	task.Tests = tests
	return nil
}
