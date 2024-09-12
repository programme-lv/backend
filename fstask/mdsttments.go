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

func (t *Task) GetMarkdownStatements() []MarkdownStatement {
	markdownStatements := make([]MarkdownStatement, 0, len(t.mdStatements))
	for _, statement := range t.mdStatements {
		markdownStatements = append(markdownStatements, MarkdownStatement(statement))
	}

	return markdownStatements
}

func (t *Task) SetMarkdownStatements(statements []MarkdownStatement) {
	t.mdStatements = make([]mDStatement, 0, len(statements))
	for _, statement := range statements {
		t.mdStatements = append(t.mdStatements, mDStatement(statement))
	}
}

func (task *Task) readMdSttmentsFromTaskDir(dir string) error {
	mdDirPath := filepath.Join(dir, "statements", "md")
	res := make([]mDStatement, 0)
	if _, err := os.Stat(mdDirPath); os.IsNotExist(err) {
		return nil
	}

	// statements -> md -> [language] -> {story.md,input.md,output.md}
	langs, err := os.ReadDir(mdDirPath)
	if err != nil {
		return fmt.Errorf("error reading md directory: %w", err)
	}

	for _, lang := range langs {
		if !lang.IsDir() {
			continue
		}

		files, err := os.ReadDir(filepath.Join(mdDirPath, lang.Name()))
		if err != nil {
			return fmt.Errorf("error reading md directory: %w", err)
		}

		res2 := mDStatement{
			Language: "lv",
			Story:    "",
			Input:    "",
			Output:   "",
			Notes:    nil, // string pointer
			Scoring:  nil, // string pointer
		}
		langStr := lang.Name()
		res2.Language = langStr
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".md") {
				continue
			}

			content, err := os.ReadFile(filepath.Join(mdDirPath, lang.Name(), f.Name()))
			if err != nil {
				return fmt.Errorf("error reading md file: %w", err)
			}

			switch f.Name() {
			case "story.md":
				res2.Story = string(content)
			case "input.md":
				res2.Input = string(content)
			case "output.md":
				res2.Output = string(content)
			case "notes.md":
				res2.Notes = &([]string{string(content)}[0])
			case "scoring.md":
				res2.Scoring = &([]string{string(content)}[0])
			}
		}

		if res2.Story == "" || res2.Input == "" || res2.Output == "" {
			return fmt.Errorf("invalid MD statement: %+v", res2)
		}

		res = append(res, res2)
	}

	task.mdStatements = res

	return nil
}
