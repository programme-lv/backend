package fstask

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func readPDFStatements(_ string, rootDirPath string) (map[string][]byte, error) {
	pdfDirPath := filepath.Join(rootDirPath, "statements", "pdf")

	res := make(map[string][]byte)
	if _, err := os.Stat(pdfDirPath); os.IsNotExist(err) {
		log.Println("PDF directory does not exist")
		return res, nil
		// return res, fmt.Errorf("pdf directory does not exist: %s", pdfDirPath)
	}

	files, err := os.ReadDir(pdfDirPath)
	if err != nil {
		return res, fmt.Errorf("error reading pdf directory: %w", err)
	}
	/* lv.pdf */
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".pdf") {
			log.Fatalf("Unsupported PDF file: %s\n", f.Name())
		}

		content, err := os.ReadFile(filepath.Join(pdfDirPath, f.Name()))
		if err != nil {
			return nil, fmt.Errorf("error reading pdf file: %w", err)
		}

		res[f.Name()[0:len(f.Name())-4]] = content
	}

	return res, nil
}
