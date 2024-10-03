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
	Notes    string
	Scoring  string
}

func ReadMarkdownStatementsFromTaskDir(dir TaskDir) ([]MarkdownStatement, error) {
	requiredSpec := Version{major: 2}
	if dir.Specification.LessThan(requiredSpec) {
		format := "specification version %s is not supported, required at least %s"
		return nil, fmt.Errorf(format, dir.Specification.String(), requiredSpec.String())
	}

	var markdownStatements []MarkdownStatement

	statementsDir := filepath.Join(dir.AbsPath, "statements")
	if _, err := os.Stat(statementsDir); !os.IsNotExist(err) {
		// find all files that end with .md
		markdownFiles, err := filepath.Glob(filepath.Join(statementsDir, "*.md"))
		if err != nil {
			return nil, fmt.Errorf("error finding markdown files: %w", err)
		}

		for _, file := range markdownFiles {
			content, err := os.ReadFile(file)
			if err != nil {
				return nil, fmt.Errorf("error reading markdown file '%s': %w", file, err)
			}

			sections := strings.Split(string(content), "\n\n---\n\n")
			if len(sections) < 3 {
				return nil, fmt.Errorf("invalid markdown file '%s': not enough sections", file)
			}

			statement := MarkdownStatement{
				Language: strings.TrimSuffix(filepath.Base(file), ".md"),
				Story:    sections[0],
				Input:    sections[1],
				Output:   sections[2],
			}

			markdownStatements = append(markdownStatements, statement)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error accessing statements directory: %w", err)
	}

	mdDirPath := filepath.Join(dir.AbsPath, "statements", "md")
	if _, err := os.Stat(mdDirPath); os.IsNotExist(err) {
		return markdownStatements, nil
	}

	langs, err := os.ReadDir(mdDirPath)
	if err != nil {
		return nil, fmt.Errorf("error reading markdown directory: %w", err)
	}

	for _, lang := range langs {
		if !lang.IsDir() {
			// check if has suffix .md
			// if it has, read it, divide into sections by ---\n
			if !strings.HasSuffix(lang.Name(), ".md") {
				continue
			}

			content, err := os.ReadFile(filepath.Join(mdDirPath, lang.Name()))
			if err != nil {
				return nil, fmt.Errorf("error reading markdown file '%s': %w", lang.Name(), err)
			}

			sections := strings.Split(string(content), "\n\n---\n\n")
			statement := MarkdownStatement{
				Language: strings.TrimSuffix(lang.Name(), ".md"),
				Story:    sections[0],
				Input:    sections[1],
				Output:   sections[2],
			}

			markdownStatements = append(markdownStatements, statement)
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
				statement.Notes = string(content)
			case "scoring.md":
				statement.Scoring = string(content)
			}
		}

		// Validate mandatory fields
		if statement.Story == "" || statement.Input == "" || statement.Output == "" {
			return nil, fmt.Errorf("invalid markdown statement for language '%s': missing mandatory fields", statement.Language)
		}

		markdownStatements = append(markdownStatements, statement)
	}

	// check if duplicate language
	seen := make(map[string]bool)
	for _, statement := range markdownStatements {
		if seen[statement.Language] {
			return nil, fmt.Errorf("duplicate language '%s' in markdown statements", statement.Language)
		}
		seen[statement.Language] = true
	}

	return markdownStatements, nil
}

func (task *Task) LoadMarkdownStatements(dir TaskDir) error {
	markdownStatements, err := ReadMarkdownStatementsFromTaskDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read markdown statements: %w", err)
	}
	task.MarkdownStatements = markdownStatements
	return nil
}
