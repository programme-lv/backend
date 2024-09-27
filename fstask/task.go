package fstask

type Task struct {
	problemTomlContent []byte

	// specificationVersion string
	// srcDirPath           string
	ProblemTags []string
	TaskAuthors []string

	FullName             string
	OriginOlympiad       string
	DifficultyOneToFive  int
	MemoryLimInMegabytes int
	CpuTimeLimInSeconds  float64
	examples             []example
	// exampleFilenameToID  map[string]int
	visibleInputSubtasks []int

	MarkdownStatements []MarkdownStatement
	PdfStatements      []PdfStatement

	/*
		=== TESTS ===
		1) read test filenames, sort them lexiographically
		2) initialize a map from test filename to its ID
		3) initialize a map from test ID to its filename
		4) override the map with test filename-id dictionary in problem.toml
		5) override the map with test id-filename dictionary in problem.toml
		6) read tests into memory
	*/

	testFnamesSorted []string
	testFilenameToID map[string]int
	testIDOverwrite  map[string]int // used only during reading directory
	testIDToFilename map[int]string
	tests            []test

	/*
		=== TEST GROUPS ===
		1) read all group IDs from problem.toml
		2) read wchich groups are public
		3) reach how many points each group has
		4) read to which subtask each group belongs
		5) read which test ids belong to each group
		6) read which filenames belong to each group
		7) append to groups test id(filename) for names
	*/

	testGroupIDs   []int
	isTGroupPublic map[int]bool
	tGroupPoints   map[int]int
	tGroupToStMap  map[int]int
	tGroupTestIDs  map[int][]int
	tGroupFnames   map[int][]string // used only during reading directory

	illstrImgFname string

	assets []Asset

	OriginNotes       map[string]string
	OriginInstitution string
}

type mDStatement struct {
	Language string
	Story    string
	Input    string
	Output   string
	Notes    *string
	Scoring  *string
}

// tests are executed in order of ID
type test struct {
	// ID is the order in which the file comes in lexicographical order
	// OR overriden by the filename-testID dictionary in problem.toml
	ID     int
	Input  []byte
	Answer []byte
}

type example struct {
	// ID is the order in which the file comes in lexicographical order
	// OR overriden by the filename-exampleID dictionary in problem.toml
	// TODO: create the filename-exampleID dictionary
	// ID     int
	Input  []byte
	Output []byte
	MdNote []byte
	Name   *string
}

func NewTask(taskName string) (*Task, error) {
	t := Task{
		problemTomlContent:   []byte{},
		ProblemTags:          []string{},
		TaskAuthors:          []string{},
		FullName:             taskName,
		OriginOlympiad:       "",
		DifficultyOneToFive:  0,
		MemoryLimInMegabytes: 256,
		CpuTimeLimInSeconds:  1.0,
		examples:             []example{},
		visibleInputSubtasks: []int{},
		MarkdownStatements:   []MarkdownStatement{},
		PdfStatements:        []PdfStatement{},
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
		assets:               []Asset{},
		OriginNotes:          map[string]string{},
		OriginInstitution:    "",
	}

	return &t, nil
}
