package lio2023

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/programme-lv/backend/fstask"
	"github.com/programme-lv/backend/fstask/lio"
)

func ParseLio2023TaskDir(dirPath string) (*fstask.Task, error) {
	taskYamlPath := filepath.Join(dirPath, "task.yaml")

	taskYamlContent, err := os.ReadFile(taskYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read task.yaml: %w", err)
	}

	taskYaml, err := ParseLio2023Yaml(taskYamlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task.yaml: %w", err)
	}

	task, err := fstask.NewTask(taskYaml.Title)
	if err != nil {
		return nil, fmt.Errorf("failed to create new task: %w", err)
	}

	checkerPath := filepath.Join(dirPath, "riki", "checker.cpp")
	if _, err := os.Stat(checkerPath); !errors.Is(err, fs.ErrNotExist) {
		content, err := os.ReadFile(checkerPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read checker: %w", err)
		}
		task.TestlibChecker = string(content)
	}

	interactorPath := filepath.Join(dirPath, "riki", "interactor.cpp")
	if _, err := os.Stat(interactorPath); !errors.Is(err, fs.ErrNotExist) {
		content, err := os.ReadFile(interactorPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read interactor: %w", err)
		}
		task.TestlibInteractor = string(content)
	}

	solutionsPath := filepath.Join(dirPath, "risin")
	if _, err := os.Stat(solutionsPath); !errors.Is(err, fs.ErrNotExist) {
		// loop through all files in risin using filepath.Walk
		err = filepath.Walk(solutionsPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			relativePath, err := filepath.Rel(solutionsPath, path)
			if err != nil {
				return err
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			task.Solutions = append(task.Solutions, fstask.Solution{
				Filename: filepath.Base(relativePath),
				Content:  content,
			})

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to read solutions: %w", err)
		}
	}

	testZipAbsolutePath := filepath.Join(dirPath, taskYaml.TestArchive)
	tests, err := lio.ReadLioTestsFromZip(testZipAbsolutePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tests from zip: %v", err)
	}

	testGroupTestIds := make(map[int][]int)
	for _, test := range tests {
		filename := fmt.Sprintf("%02d%c", test.TestGroup,
			test.NoInTestGroup+int('a')-1)
		if test.TestGroup == 0 {
			exampleId := task.AddExample(test.Input, test.Answer, nil)
			task.AssignFilenameToExample(filename, int(exampleId))
		} else {
			testId := task.AddTest(test.Input, test.Answer)
			task.AssignFilenameToTest(filename, int(testId))

			if testGroupTestIds[test.TestGroup] == nil {
				testGroupTestIds[test.TestGroup] = make([]int, 0)
			}
			testGroupTestIds[test.TestGroup] = append(testGroupTestIds[test.TestGroup], int(testId))
		}
	}

	punktiTxtPath := filepath.Join(dirPath, "punkti.txt")
	punktiTxtContent, err := os.ReadFile(punktiTxtPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read punkti.txt: %w", err)
	}
	// split by "\n"
	parts := strings.Split(string(punktiTxtContent), "\n")
	for _, line := range parts {
		if line == "" {
			continue
		}
		// split by space
		parts := strings.Split(line, " ")
		testInterval := strings.Split(parts[0], "-")

		if len(testInterval) != 2 {
			return nil, fmt.Errorf("failed to parse test interval: %s", line)
		}

		start, err := strconv.Atoi(testInterval[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse test interval: %w", err)
		}
		end, err := strconv.Atoi(testInterval[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse test interval: %w", err)
		}

		points, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse points: %w", err)
		}

		for i := start; i <= end; i++ {
			if i == 0 {
				continue // example test group
			}
			task.AddTestGroup(points, false, testGroupTestIds[i], 69)
		}
	}

	task.CpuTimeLimInSeconds = taskYaml.TimeLimit
	task.MemoryLimInMegabytes = taskYaml.MemoryLimit

	excludePrefixFromArchive := []string{
		punktiTxtPath,
		testZipAbsolutePath,
		solutionsPath,
		taskYamlPath,
		checkerPath,
		interactorPath,
	}

	// look at all the paths in the directory.
	// if it starts with one of the prefixes in excludePrefixFromArchive
	// then it should be excluded from the archive
	// otherwise, it should be included

	// task.ArchiveFiles = append(task.ArchiveFiles, fstask.ArchiveFile{
	// 	RelativePath: "",
	// 	Content:      []byte{},
	// })

	err = filepath.Walk(dirPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relativePath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}
		relativePath = "./" + relativePath
		for _, prefix := range excludePrefixFromArchive {
			prefixAbs, err := filepath.Abs(prefix)
			if err != nil {
				return err
			}
			pathAbs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			if pathAbs == prefixAbs {
				return nil
			}
			if strings.HasPrefix(pathAbs, prefixAbs+string(filepath.Separator)) {
				return nil
			}
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		task.ArchiveFiles = append(task.ArchiveFiles, fstask.ArchiveFile{
			RelativePath: relativePath,
			Content:      content,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	task.OriginOlympiad = "LIO"

	return task, nil
}
