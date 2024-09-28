package lio2023

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/programme-lv/backend/fstask"
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
		task.TestlibChecker = new(string)
		*task.TestlibChecker = string(content)
	}

	interactorPath := filepath.Join(dirPath, "riki", "interactor.cpp")
	if _, err := os.Stat(interactorPath); !errors.Is(err, fs.ErrNotExist) {
		content, err := os.ReadFile(interactorPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read interactor: %w", err)
		}
		task.TestlibInteractor = new(string)
		*task.TestlibInteractor = string(content)
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

	return task, nil
}
