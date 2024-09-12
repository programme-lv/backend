package fstask

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func Read(taskRootDirPath string) (*Task, error) {
	t := Task{
		problemTomlContent:   []byte{},
		ProblemTags:          []string{},
		ProblemAuthors:       []string{},
		TaskFullName:         "",
		OriginOlympiad:       "",
		DifficultyOneToFive:  0,
		MemoryLimInMegabytes: 0,
		CpuTimeLimInSeconds:  0,
		examples:             []example{},
		visibleInputSubtasks: []int{},
		mdStatements:         []mDStatement{},
		pdfStatements:        map[string][]byte{},
		testFnamesSorted:     []string{},
		testFilenameToID:     map[string]int{},
		testIDOverwrite:      map[string]int{},
		testIDToFilename:     map[int]string{},
		tests:                []test{},
		testGroupIDs:         []int{},
		isTGroupPublic:       map[int]bool{},
		tGroupPoints:         map[int]int{},
		tGroupToStMap:        map[int]int{},
		tGroupTestIDs:        map[int][]int{},
		tGroupFnames:         map[int][]string{},
		illstrImgFname:       "",
		assets:               []asset{},
		OriginNotes:          map[string]string{},
	}

	problemTomlPath := filepath.Join(taskRootDirPath, "problem.toml")
	log.Printf("Reading problem.toml from: %s\n", problemTomlPath)
	problemTomlContent, err := os.ReadFile(problemTomlPath)
	if err != nil {
		log.Printf("Error reading problem.toml: %v\n", err)
		return nil, fmt.Errorf("error reading problem.toml: %w", err)
	}

	t.problemTomlContent = problemTomlContent
	log.Println("problem.toml content read successfully")

	var specVersStruct struct {
		Specification string `toml:"specification"`
	}

	err = toml.Unmarshal(problemTomlContent, &specVersStruct)
	if err != nil {
		log.Printf("Failed to unmarshal the specification: %v\n", err)
		return nil, fmt.Errorf("failed to unmarshal the specification: %w", err)
	}

	specVers := specVersStruct.Specification
	if len(specVers) == 0 {
		log.Println("Empty specification found")
		return nil, fmt.Errorf("empty specification")
	}
	if specVers[0] == 'v' {
		specVers = specVers[1:]
	}

	log.Printf("Specification version: %s\n", specVers)

	semVersCmpRes, err := getCmpSemVersionsResult(specVers, proglvFSTaskFormatSpecVersOfScript)
	if err != nil {
		log.Printf("Error comparing sem versions: %v\n", err)
		return nil, fmt.Errorf("error comparing sem versions: %w", err)
	}

	if semVersCmpRes > 0 {
		log.Printf("Unsupported specification version (too new): %s\n", specVers)
		return nil, fmt.Errorf("unsupported specification version (too new): %s", specVers)
	}

	if semVersCmpRes < 0 {
		log.Printf("Warning: outdated specification version (too old): %s\n", specVers)
	}

	t.TaskFullName, err = readTaskName(specVers, string(problemTomlContent))
	if err != nil {
		log.Printf("Error reading task name: %v\n", err)
		return nil, fmt.Errorf("error reading task name: %w", err)
	}

	err = t.readConstraintsFromToml(problemTomlContent)
	if err != nil {
		return nil, fmt.Errorf("error reading constraints: %w", err)
	}

	err = t.readMetadataFromToml(problemTomlContent)
	if err != nil {
		return nil, fmt.Errorf("error reading metadata: %w", err)
	}

	log.Println("Reading test filenames from the tests directory")
	t.testFnamesSorted, err = readTestFNamesSorted(filepath.Join(taskRootDirPath, "tests"))
	if err != nil {
		log.Printf("Error reading test filenames: %v\n", err)
		return nil, fmt.Errorf("error reading test filenames: %w", err)
	}

	for i, fname := range t.testFnamesSorted {
		t.testFilenameToID[fname] = i + 1
		t.testIDToFilename[i+1] = fname
	}

	log.Println("Reading test ID overwrite")
	t.testIDOverwrite, err = readTestIDOverwrite(specVers, problemTomlContent)
	if err != nil {
		log.Printf("Error reading test ID overwrite: %v\n", err)
		return nil, fmt.Errorf("error reading test id overwrite: %w", err)
	}

	for k, v := range t.testIDOverwrite {
		t.testIDToFilename[v] = k
		t.testFilenameToID[k] = v
	}

	spottedFnames := make(map[int]bool)
	for _, fname := range t.testFnamesSorted {
		if _, ok := spottedFnames[t.testFilenameToID[fname]]; ok {
			log.Printf("Duplicate filename for ID: %s\n", fname)
			return nil, fmt.Errorf("duplicate filename for ID: %s", fname)
		}
		spottedFnames[t.testFilenameToID[fname]] = true
	}

	spottedIDs := make(map[string]bool)
	for _, id := range t.testIDToFilename {
		if _, ok := spottedIDs[id]; ok {
			log.Printf("Duplicate ID for filename: %s\n", id)
			return nil, fmt.Errorf("duplicate ID for filename: %s", id)
		}
		spottedIDs[id] = true
	}

	log.Println("Reading tests directory")
	t.tests, err = readTestsDir(taskRootDirPath, t.testFilenameToID)
	if err != nil {
		log.Printf("Error reading tests directory: %v\n", err)
		return nil, fmt.Errorf("error reading tests directory: %w", err)
	}

	log.Println("Reading examples directory")
	t.examples, err = readExamplesDir(taskRootDirPath)
	if err != nil {
		log.Printf("Error reading examples directory: %v\n", err)
		return nil, fmt.Errorf("error reading examples directory: %w", err)
	}

	log.Println("Reading test group IDs")
	t.testGroupIDs, err = readTestGroupIDs(specVers, problemTomlContent)
	if err != nil {
		log.Printf("Error reading test group IDs: %v\n", err)
		return nil, fmt.Errorf("error reading test group IDs: %w", err)
	}

	log.Println("Reading is test group public")
	t.isTGroupPublic, err = readIsTGroupPublic(specVers, problemTomlContent, t.testGroupIDs)
	if err != nil {
		log.Printf("Error reading is test group public: %v\n", err)
		return nil, fmt.Errorf("error reading is test group public: %w", err)
	}

	log.Println("Reading test group points")
	t.tGroupPoints, err = readTGroupPoints(specVers, problemTomlContent, t.testGroupIDs)
	if err != nil {
		log.Printf("Error reading test group points: %v\n", err)
		return nil, fmt.Errorf("error reading test group points: %w", err)
	}

	log.Println("Reading test group to subtask map")
	t.tGroupToStMap, err = readTGroupToStMap(specVers, problemTomlContent)
	if err != nil {
		log.Printf("Error reading test group to subtask map: %v\n", err)
		return nil, fmt.Errorf("error reading test group to subtask map: %w", err)
	}

	log.Println("Reading test group test IDs")
	t.tGroupTestIDs, err = readTGroupTestIDs(specVers, problemTomlContent, t.testGroupIDs)
	if err != nil {
		log.Printf("Error reading test group test IDs: %v\n", err)
		return nil, fmt.Errorf("error reading test group test IDs: %w", err)
	}

	log.Println("Reading test group filenames")
	t.tGroupFnames, err = readTGroupFnames(specVers, problemTomlContent, t.testGroupIDs)
	if err != nil {
		log.Printf("Error reading test group filenames: %v\n", err)
		return nil, fmt.Errorf("error reading test group filenames: %w", err)
	}

	for k, v := range t.tGroupFnames {
		for _, fname := range v {
			t.tGroupTestIDs[k] = append(t.tGroupTestIDs[k], t.testFilenameToID[fname])
		}
	}

	idsSpotted := make(map[int]bool)
	for _, v := range t.testGroupIDs {
		for _, id := range t.tGroupTestIDs[v] {
			if _, ok := idsSpotted[id]; ok {
				log.Printf("Duplicate test ID in test group: %d\n", id)
				return nil, fmt.Errorf("duplicate test ID in test group: %d", id)
			}
			idsSpotted[id] = true
		}
	}

	log.Println("Reading PDF statements")
	t.pdfStatements, err = readPDFStatements(specVers, taskRootDirPath)
	if err != nil {
		log.Printf("Error reading PDF statements: %v\n", err)
	}

	err = t.readMdSttmentsFromTaskDir(taskRootDirPath)
	if err != nil {
		log.Printf("Error reading MD statements from root dir: %v\n", err)
	}

	// read task illustration
	log.Println("Reading task illustration filename")
	t.illstrImgFname, err = readIllstrImgFnameFromPToml(problemTomlContent)
	if err != nil {
		log.Printf("Error reading task illustration filename: %v\n", err)
	}

	log.Println("Reading all assets")
	t.assets, err = readAssets(taskRootDirPath)
	if err != nil {
		log.Printf("Error reading all assets: %v\n", err)
	}

	log.Println("Reading origin notes")
	t.OriginNotes, err = readOriginNotes(problemTomlContent)
	if err != nil {
		log.Printf("Error reading origin notes: %v\n", err)
	}
	log.Println("Reading task name")

	log.Println("Reading visible input subtasks")
	t.visibleInputSubtasks, err = readVisibleInputSubtasks(specVers, problemTomlContent)
	if err != nil {
		log.Printf("Error reading visible input subtasks: %v\n", err)
	}
	log.Println("Successfully read visible input subtasks")

	log.Println("Successfully read and parsed task")
	return &t, nil
}

func readVisibleInputSubtasks(_ string, pToml []byte) ([]int, error) {
	metadata := struct {
		VisInpSTs []int `toml:"visible_input_subtasks"`
	}{}

	err := toml.Unmarshal(pToml, &metadata)
	if err != nil {
		log.Printf("Failed to unmarshal the visible input subtasks: %v\n", err)
		return nil, fmt.Errorf("failed to unmarshal the visible input subtasks: %w", err)
	}

	return metadata.VisInpSTs, nil
}

func readOriginNotes(pToml []byte) (map[string]string, error) {
	type Metadata struct {
		OriginNotes map[string]string `toml:"origin_notes,omitempty"`
	}
	metadata := struct {
		Metadata Metadata `toml:"metadata"`
	}{}

	err := toml.Unmarshal(pToml, &metadata)
	if err != nil {
		log.Printf("Failed to unmarshal the origin notes: %v\n", err)
		return nil, fmt.Errorf("failed to unmarshal the origin notes: %w", err)
	}

	return metadata.Metadata.OriginNotes, nil
}

func readAssets(rootDirPath string) ([]asset, error) {
	res := make([]asset, 0)
	dirPath := filepath.Join(rootDirPath, "assets")
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return res, nil
	}

	files, err := os.ReadDir(dirPath)
	if err != nil {
		return res, fmt.Errorf("error reading assets directory: %w", err)
	}

	for _, f := range files {
		if f.IsDir() {
			return nil, fmt.Errorf("directories are currently not supported")
		}
		bytes, err := os.ReadFile(filepath.Join(dirPath, f.Name()))
		if err != nil {
			return nil, fmt.Errorf("error reading asset: %w", err)
		}
		res = append(res, asset{
			RelativePath: f.Name(),
			Content:      bytes,
		})
	}

	return res, nil
}

func readIllstrImgFnameFromPToml(pToml []byte) (string, error) {
	illustrationPath := ""
	tomlStruct := struct {
		IllstrImgFname string `toml:"illustration_image"`
	}{}

	err := toml.Unmarshal(pToml, &tomlStruct)
	if err != nil {
		log.Printf("Failed to unmarshal the task name: %v\n", err)
		return "", fmt.Errorf("failed to unmarshal the task name: %w", err)
	}

	illustrationPath = tomlStruct.IllstrImgFname
	return illustrationPath, nil
}

func readTaskName(specVers string, tomlContent string) (string, error) {
	log.Printf("Reading task name for specification version: %s\n", specVers)
	cmpres, err := largerOrEqualSemVersionThan(specVers, "2.2")
	if err != nil {
		log.Printf("Error comparing semversions: %v\n", err)
		return "", fmt.Errorf("error comparing semversions: %w", err)
	}
	if !cmpres {
		log.Printf("Unsupported specification version: %s\n", specVers)
		return "", fmt.Errorf("unsupported specification version: %s", specVers)
	}

	tomlStruct := struct {
		TaskName string `toml:"task_name"`
	}{}

	err = toml.Unmarshal([]byte(tomlContent), &tomlStruct)
	if err != nil {
		log.Printf("Failed to unmarshal the task name: %v\n", err)
		return "", fmt.Errorf("failed to unmarshal the task name: %w", err)
	}

	log.Printf("Successfully read task name: %s\n", tomlStruct.TaskName)
	return tomlStruct.TaskName, nil
}
