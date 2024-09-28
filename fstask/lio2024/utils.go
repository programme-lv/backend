package lio2024

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// Unzip extracts a zip archive to a specified destination.
func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

// TaskYAML represents the structure of the task.yaml file.
type TaskYAML struct {
	Name          string  `yaml:"name"`
	Title         string  `yaml:"title"`
	TimeLimit     float64 `yaml:"time_limit"`
	MemoryLimit   int     `yaml:"memory_limit"`
	SubtaskPoints []int   `yaml:"subtask_points"`
	TestsGroups   []struct {
		Groups  interface{} `yaml:"groups"`
		Points  int         `yaml:"points"`
		Public  interface{} `yaml:"public"`
		Subtask int         `yaml:"subtask"`
		Comment string      `yaml:"comment,omitempty"`
	} `yaml:"tests_groups"`
}

// ReadTaskYAML reads and parses the task.yaml file.
func ReadTaskYAML(filePath string) (*TaskYAML, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var task TaskYAML
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&task)
	if err != nil {
		return nil, err
	}

	return &task, nil
}

// CopyPDF moves PDF files from the source to the destination.
func CopyPDF(srcDir, destPath string) error {
	err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm)
	if err != nil {
		return err
	}

	pdfFiles, err := filepath.Glob(filepath.Join(srcDir, "*.pdf"))
	if err != nil {
		return err
	}

	for _, pdfFile := range pdfFiles {
		content, err := os.ReadFile(pdfFile)
		if err != nil {
			return err
		}

		err = os.WriteFile(destPath, content, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}
