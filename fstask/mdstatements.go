package fstask

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MarkdownStatement struct {
	Language string
	Story    string
	Input    string
	Output   string
	Notes    *string
	Scoring  *string
}

func ReadMarkdownStatementsFromTaskDir(dir TaskDir) ([]MarkdownStatement, error) {
	requiredSpec := SemVer{major: 2}
	if dir.Spec.LessThan(requiredSpec) {
		format := "specification version %s is not supported, required at least %s"
		return nil, fmt.Errorf(format, dir.Spec.String(), requiredSpec.String())
	}

	mdDirPath := filepath.Join(dir.Path, "statements", "md")
	if _, err := os.Stat(mdDirPath); os.IsNotExist(err) {
		// No markdown statements to read
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("error accessing markdown directory: %w", err)
	}

	langs, err := os.ReadDir(mdDirPath)
	if err != nil {
		return nil, fmt.Errorf("error reading markdown directory: %w", err)
	}

	var markdownStatements []MarkdownStatement

	for _, lang := range langs {
		if !lang.IsDir() {
			continue
		}

		langPath := filepath.Join(mdDirPath, lang.Name())
		files, err := os.ReadDir(langPath)
		if err != nil {
			return nil, fmt.Errorf("error reading language directory '%s': %w", lang.Name(), err)
		}

		statement := MarkdownStatement{
			Language: lang.Name(),
		}

		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".md") {
				continue
			}

			filePath := filepath.Join(langPath, file.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("error reading markdown file '%s': %w", file.Name(), err)
			}

			switch file.Name() {
			case "story.md":
				statement.Story = string(content)
			case "input.md":
				statement.Input = string(content)
			case "output.md":
				statement.Output = string(content)
			case "notes.md":
				statement.Notes = ptr(string(content))
			case "scoring.md":
				statement.Scoring = ptr(string(content))
			}
		}

		// Validate mandatory fields
		if statement.Story == "" || statement.Input == "" || statement.Output == "" {
			return nil, fmt.Errorf("invalid markdown statement for language '%s': missing mandatory fields", statement.Language)
		}

		markdownStatements = append(markdownStatements, statement)
	}

	return markdownStatements, nil
}

// ptr is a helper function to obtain a pointer to a string
func ptr(s string) *string {
	return &s
}

func (task *Task) LoadMarkdownStatementsFromDir(dir TaskDir) error {
	markdownStatements, err := ReadMarkdownStatementsFromTaskDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read markdown statements: %w", err)
	}
	task.MarkdownStatements = markdownStatements
	return nil
}
