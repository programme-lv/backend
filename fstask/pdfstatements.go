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
	requiredSpec := SemVer{major: 2}
	if dir.Spec.LessThan(requiredSpec) {
		format := "specification version %s is not supported, required at least %s"
		return nil, fmt.Errorf(format, dir.Spec.String(), requiredSpec.String())
	}

	pdfDirPath := filepath.Join(dir.Path, "statements", "pdf")
	if _, err := os.Stat(pdfDirPath); os.IsNotExist(err) {
		// PDF directory does not exist; no PDF statements to read.
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("error accessing PDF directory '%s': %w", pdfDirPath, err)
	}

	files, err := os.ReadDir(pdfDirPath)
	if err != nil {
		return nil, fmt.Errorf("error reading PDF directory '%s': %w", pdfDirPath, err)
	}

	var pdfStatements []PdfStatement

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

// LoadPDFStatementsFromDir loads PDF statements into the task from the specified directory.
func (task *Task) LoadPDFStatementsFromDir(dir TaskDir) error {
	pdfStatements, err := ReadPDFStatementsFromTaskDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read PDF statements: %w", err)
	}
	task.PdfStatements = pdfStatements
	return nil
}
