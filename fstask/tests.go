package fstask

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func readTestsDir(srcDirPath string, fnameToID map[string]int) ([]Test, error) {
	dir := filepath.Join(srcDirPath, "tests")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error reading tests directory: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	tests := make([]Test, 0, len(entries)/2)

	for i := 0; i < len(entries); i += 2 {
		inPath := filepath.Join(dir, entries[i].Name())
		ansPath := filepath.Join(dir, entries[i+1].Name())

		inFilename := entries[i].Name()
		ansFilename := entries[i+1].Name()

		inFilenameBase := strings.TrimSuffix(inFilename, filepath.Ext(inFilename))
		ansFilenameBase := strings.TrimSuffix(ansFilename, filepath.Ext(ansFilename))

		if inFilenameBase != ansFilenameBase {
			return nil, fmt.Errorf("input and answer file base names do not match: %s, %s", inFilenameBase, ansFilenameBase)
		}

		// sometimes the test answer is stored as .out, sometimes as .ans
		if strings.Contains(inFilename, ".ans") || strings.Contains(ansFilename, ".in") {
			// swap the file paths
			inPath, ansPath = ansPath, inPath
		}

		input, err := os.ReadFile(inPath)
		if err != nil {
			return nil, fmt.Errorf("error reading input file: %w", err)
		}

		answer, err := os.ReadFile(ansPath)
		if err != nil {
			return nil, fmt.Errorf("error reading answer file: %w", err)
		}

		// check if mapping to id exists
		if _, ok := fnameToID[inFilenameBase]; !ok {
			return nil, fmt.Errorf("mapping from filename to id does not exist: %s", inFilenameBase)
		}

		tests = append(tests, Test{
			ID:     fnameToID[inFilenameBase],
			Input:  input,
			Answer: answer,
		})
	}

	return tests, nil
}

func readExamplesDir(srcDirPath string) ([]example, error) {
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

	examples := make([]example, len(groupedByBase))
	i := 0
	for _, key := range keys {
		baseName := key
		files := groupedByBase[key]
		e := example{
			Input:  []byte{},
			Output: []byte{},
			MdNote: []byte{},
			Name:   &baseName,
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

func readTestIDOverwrite(specVers string, tomlContent []byte) (map[string]int, error) {
	semVerCmpRes, err := getCmpSemVersionsResult(specVers, "v2.3.0")
	if err != nil {
		return nil, fmt.Errorf("error comparing sem versions: %w", err)
	}

	if semVerCmpRes < 0 {
		return make(map[string]int), nil
	}

	tomlStruct := struct {
		TestIDOverwrite map[string]int `toml:"test_id_overwrite"`
	}{}

	err = toml.Unmarshal(tomlContent, &tomlStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal the test ID overwrite: %w", err)
	}

	return tomlStruct.TestIDOverwrite, nil
}

func readTestFNamesSorted(dirPath string) ([]string, error) {
	fnames, err := os.ReadDir(dirPath)
	if err != nil {
		log.Printf("Error reading test filenames: %v\n", err)
		return nil, fmt.Errorf("error reading test filenames: %w", err)
	}

	sort.Slice(fnames, func(i, j int) bool {
		return fnames[i].Name() < fnames[j].Name()
	})

	if len(fnames)%2 != 0 {
		return nil, fmt.Errorf("odd number of test filenames")
	}

	res := make([]string, 0, len(fnames)/2)
	for i := 0; i < len(fnames); i += 2 {
		a_name := fnames[i].Name()
		// remove extension
		a_name = a_name[:len(a_name)-len(filepath.Ext(a_name))]

		b_name := fnames[i+1].Name()
		// remove extension
		b_name = b_name[:len(b_name)-len(filepath.Ext(b_name))]

		if a_name != b_name {
			return nil, fmt.Errorf("input and answer file base names do not match: %s, %s", a_name, b_name)
		}

		res = append(res, a_name)
	}

	return res, nil
}
