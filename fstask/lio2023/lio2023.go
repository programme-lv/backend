package lio2023

import (
	"fmt"
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

	return task, nil
}
