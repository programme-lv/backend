package lio

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type LioTest struct {
	TaskName string

	TestGroup         int
	NoInTestGroup     int
	NoInLexFnameOrder int

	Input  []byte
	Answer []byte
}

func ReadLioTestsFromZip(testZipPath string) ([]LioTest, error) {
	// create a tmp directory where to unzip the test zip
	tmpDirPath, err := os.MkdirTemp("", "lio-tests")
	if err != nil {
		return nil, fmt.Errorf("failed to create tmp directory: %v", err)
	}
	defer os.RemoveAll(tmpDirPath)

	err = Unzip(testZipPath, tmpDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to unzip %s: %v", testZipPath, err)
	}

	return ReadLioTestsFromDir(tmpDirPath)
}

func ReadLioTestsFromDir(testDir string) ([]LioTest, error) {
	res := []LioTest{}

	listDir, err := os.ReadDir(testDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %v", testDir, err)
	}

	// sort by filename in lexicographical order
	sort.Slice(listDir, func(i, j int) bool {
		return listDir[i].Name() < listDir[j].Name()
	})

	if len(listDir)%2 != 0 {
		return nil, fmt.Errorf("unexpected number of files in the directory: %d", len(listDir))
	}

	inputEntries := listDir[:len(listDir)/2]
	answerEntries := listDir[len(listDir)/2:]

	for i := 0; i < len(inputEntries); i++ {
		inputPath := filepath.Join(testDir, inputEntries[i].Name())
		answerPath := filepath.Join(testDir, answerEntries[i].Name())

		inFname := filepath.Base(inputPath)
		ansFname := filepath.Base(answerPath)

		inFnameSplit, err := lioTestName(inFname)
		if err != nil {
			return nil, fmt.Errorf("failed to parse input filename: %v", err)
		}
		ansFnameSplit, err := lioTestName(ansFname)
		if err != nil {
			return nil, fmt.Errorf("failed to parse answer filename: %v", err)
		}

		inTaskName := inFnameSplit[0]
		ansTaskName := ansFnameSplit[0]

		if inTaskName != ansTaskName {
			return nil, fmt.Errorf("input and answer task names do not match: %s, %s", inTaskName, ansTaskName)
		}

		if inFnameSplit[1] != "i" || ansFnameSplit[1] != "o" {
			return nil, fmt.Errorf("unexpected filename format: %s, %s", inFname, ansFname)
		}

		inGroup, err := strconv.Atoi(inFnameSplit[2])
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s to int: %v", inFnameSplit[2], err)
		}
		ansGroup, err := strconv.Atoi(ansFnameSplit[2])
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s to int: %v", ansFnameSplit[2], err)
		}

		if inGroup != ansGroup {
			return nil, fmt.Errorf("input and answer groups do not match: %d, %d", inGroup, ansGroup)
		}

		inGroupNo := 1
		if len(inFnameSplit) == 4 {
			if len(inFnameSplit[3]) != 1 {
				return nil, fmt.Errorf("unexpected filename format: %s", inFname)
			}
			inGroupNo = int(inFnameSplit[3][0]) - int('a') + 1
		}

		ansGroupNo := 1
		if len(ansFnameSplit) == 4 {
			if len(ansFnameSplit[3]) != 1 {
				return nil, fmt.Errorf("unexpected filename format: %s", ansFname)
			}
			ansGroupNo = int(ansFnameSplit[3][0]) - int('a') + 1
		}

		if inGroupNo != ansGroupNo {
			return nil, fmt.Errorf("input and answer groups do not match: %d, %d", inGroupNo, ansGroupNo)
		}

		inBytes, err := os.ReadFile(inputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read input file: %v", err)
		}
		ansBytes, err := os.ReadFile(answerPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read answer file: %v", err)
		}

		res = append(res, LioTest{
			TaskName:          inTaskName,
			TestGroup:         inGroup,
			NoInTestGroup:     inGroupNo,
			NoInLexFnameOrder: i,
			Input:             inBytes,
			Answer:            ansBytes,
		})
	}

	return res, nil
}

/*
kp.i00 -> ["kp", "i", "00"]
kp.i01a -> ["kp", "i", "01", "a"]
kp.i01b
kp.o00
kp.o01a
kp.o01b
*/
func lioTestName(fname string) ([]string, error) {
	res := []string{}

	splitByDot := strings.Split(fname, ".")
	if len(splitByDot) != 2 {
		return nil, fmt.Errorf("unexpected filename: %s", fname)
	}
	res = append(res, splitByDot[0])

	ext := splitByDot[1]
	if ext[0] != 'i' && ext[0] != 'o' {
		return nil, fmt.Errorf("unexpected second part: %s", ext)
	}

	res = append(res, ext[:1])

	hasLetter := false
	for i := 1; i < len(ext); i++ {
		if !(ext[i] >= '0' && ext[i] <= '9') {
			res = append(res, ext[1:i])
			res = append(res, ext[i:])
			hasLetter = true
			break
		}
	}
	if !hasLetter {
		res = append(res, ext[1:])
	}

	if len(res) != 3 && len(res) != 4 {
		return nil, fmt.Errorf("unexpected number of parts: %d", len(res))
	}

	return res, nil
}
