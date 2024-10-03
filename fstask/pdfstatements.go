package fstask

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PdfStatement struct {
	Language string
	Content  []byte
}

// ReadPDFStatementsFromTaskDir reads PDF statements from the specified task directory.
// It returns a slice of PDFStatement and an error, if any.
func ReadPDFStatementsFromTaskDir(dir TaskDir) ([]PdfStatement, error) {
	requiredSpec := Version{major: 2}
	if dir.Specification.LessThan(requiredSpec) {
		format := "specification version %s is not supported, required at least %s"
		return nil, fmt.Errorf(format, dir.Specification.String(), requiredSpec.String())
	}
	statementsDir := filepath.Join(dir.AbsPath, "statements")
	var pdfStatements []PdfStatement
	if _, err := os.Stat(statementsDir); !os.IsNotExist(err) {
		pdfFiles, err := filepath.Glob(filepath.Join(statementsDir, "*.pdf"))
		if err != nil {
			return nil, fmt.Errorf("error finding pdf files: %w", err)
		}

		for _, file := range pdfFiles {
			content, err := os.ReadFile(file)
			if err != nil {
				return nil, fmt.Errorf("error reading pdf file '%s': %w", file, err)
			}

			statement := PdfStatement{
				Language: strings.TrimSuffix(filepath.Base(file), ".pdf"),
				Content:  content,
			}

			pdfStatements = append(pdfStatements, statement)
		}
	}

	pdfDirPath := filepath.Join(statementsDir, "pdf")
	if _, err := os.Stat(pdfDirPath); os.IsNotExist(err) {
		// PDF directory does not exist; no PDF statements to read.
		return pdfStatements, nil
	} else if err != nil {
		return nil, fmt.Errorf("error accessing PDF directory '%s': %w", pdfDirPath, err)
	}

	files, err := os.ReadDir(pdfDirPath)
	if err != nil {
		return nil, fmt.Errorf("error reading PDF directory '%s': %w", pdfDirPath, err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".pdf") {
			// Unsupported file type found; return an error.
			return nil, fmt.Errorf("unsupported file in PDF directory: %s", file.Name())
		}

		filePath := filepath.Join(pdfDirPath, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("error reading PDF file '%s': %w", file.Name(), err)
		}

		// Extract language identifier from the filename (e.g., "lv" from "lv.pdf").
		lang := strings.TrimSuffix(file.Name(), ".pdf")
		if lang == "" {
			return nil, fmt.Errorf("invalid PDF filename (empty language identifier): %s", file.Name())
		}

		pdfStatements = append(pdfStatements, PdfStatement{
			Language: lang,
			Content:  content,
		})
	}

	return pdfStatements, nil
}

// LoadPDFStatements loads PDF statements into the task from the specified directory.
func (task *Task) LoadPDFStatements(dir TaskDir) error {
	pdfStatements, err := ReadPDFStatementsFromTaskDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read PDF statements: %w", err)
	}
	task.PdfStatements = pdfStatements
	return nil
}
