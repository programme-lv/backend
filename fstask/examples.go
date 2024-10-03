package fstask

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func readExamplesDir(srcDirPath string) ([]Example, error) {
	dir := filepath.Join(srcDirPath, "examples")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error reading examples directory: %w", err)
	}
	// tests are to be read exactly like examples

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	groupedByBase := make(map[string][]os.DirEntry)
	for _, entry := range entries {
		base := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		groupedByBase[base] = append(groupedByBase[base], entry)
	}

	keys := make([]string, 0, len(groupedByBase))
	for k := range groupedByBase {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	examples := make([]Example, len(groupedByBase))
	i := 0
	for _, key := range keys {
		baseName := key
		files := groupedByBase[key]
		e := Example{
			Input:  []byte{},
			Output: []byte{},
			MdNote: []byte{},
		}

		// check if .in exists, if not throw error
		foundIn := false
		for _, entry := range files {
			if strings.Contains(entry.Name(), ".in") {
				e.Input, err = os.ReadFile(filepath.Join(dir, entry.Name()))
				if err != nil {
					return nil, fmt.Errorf("error reading input file: %w", err)
				}
				foundIn = true
				break
			}
		}
		if !foundIn {
			return nil, fmt.Errorf("input file does not exist for example: %s", baseName)
		}

		// check if .out or .ans exists, if not throw error
		foundOut := false
		for _, entry := range files {
			if strings.Contains(entry.Name(), ".out") || strings.Contains(entry.Name(), ".ans") {
				e.Output, err = os.ReadFile(filepath.Join(dir, entry.Name()))
				if err != nil {
					return nil, fmt.Errorf("error reading output file: %w", err)
				}
				foundOut = true
				break
			}
		}
		if !foundOut {
			return nil, fmt.Errorf("output file does not exist for example: %s", baseName)
		}

		// check if .md exists, it is optional
		for _, entry := range files {
			if strings.Contains(entry.Name(), ".md") {
				e.MdNote, err = os.ReadFile(filepath.Join(dir, entry.Name()))
				if err != nil {
					log.Printf("Error reading md file: %v\n", err)
					return nil, fmt.Errorf("error reading md file: %w", err)
				}
				break
			}
		}

		examples[i] = e
		i += 1
	}

	return examples, nil
}
