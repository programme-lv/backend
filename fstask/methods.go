package fstask

import (
	"fmt"
	"log"
	"sort"
)

func (t *Task) SwapTestOrder(id1 int, id2 int) {
	id1Filename, id1HasFilename := t.testIDToFilename[id1]
	id2Filename, id2HasFilename := t.testIDToFilename[id2]

	for i := 0; i < len(t.tests); i++ {
		if t.tests[i].ID == id1 {
			t.tests[i].ID = id2
		} else if t.tests[i].ID == id2 {
			t.tests[i].ID = id1
		}
	}

	delete(t.testFilenameToID, id1Filename)
	delete(t.testIDToFilename, id1)
	delete(t.testFilenameToID, id2Filename)
	delete(t.testIDToFilename, id2)

	if id1HasFilename {
		t.testFilenameToID[id1Filename] = id2
		t.testIDToFilename[id2] = id1Filename
	}
	if id2HasFilename {
		t.testFilenameToID[id2Filename] = id1
		t.testIDToFilename[id1] = id2Filename
	}

	for k, v := range t.tGroupTestIDs {
		for i := 0; i < len(v); i++ {
			if v[i] == id1 {
				v[i] = id2
			} else if v[i] == id2 {
				v[i] = id1
			}
		}
		t.tGroupTestIDs[k] = v
	}
}

func (t *Task) GetTestsSortedByID() []test {
	sort.Slice(t.tests, func(i, j int) bool {
		return t.tests[i].ID < t.tests[j].ID
	})

	return t.tests
}

// creates a new test and returns its ID
func (t *Task) AddTest(input []byte, answer []byte) int {
	// find the minimum positive excluded id from tests

	// we assign this test the number but it may not correspond to lex order
	// that is the responsibility of the persistence layer

	mex := 1
	found := true
	for found {
		found = false
		for i := 0; i < len(t.tests); i++ {
			if t.tests[i].ID == mex {
				found = true
				mex++
			}
		}
	}

	t.tests = append(t.tests, test{
		ID:     mex,
		Input:  input,
		Answer: answer,
	})

	return mex
}

func (t *Task) AssignFilenameToTest(filename string, testID int) {
	_, ok1 := t.testIDToFilename[testID]
	_, ok2 := t.testFilenameToID[filename]
	if ok1 || ok2 {
		log.Fatalf("test with ID %d or filename %s already exists", testID, filename)
	}

	t.testIDToFilename[testID] = filename
	t.testFilenameToID[filename] = testID
}

type Example struct {
	Input  []byte
	Output []byte
	MdNote []byte
	FName  *string // original file base name for example
}

func (t *Task) GetExamples() []Example {
	res := make([]Example, 0, len(t.examples))
	for _, e := range t.examples {
		res = append(res, Example{
			Input:  e.Input,
			Output: e.Output,
			MdNote: e.MdNote,
			FName:  e.Name,
		})
	}
	return res
}

func (t *Task) AddExample(input []byte, output []byte, mdNote []byte) {
	t.examples = append(t.examples, example{
		Input:  input,
		Output: output,
		MdNote: mdNote,
		Name:   nil,
	})
}

func (t *Task) GetTaskName() string {
	return t.FullName
}

func (t *Task) SetTaskName(name string) {
	t.FullName = name
}

type TestGroupInfo struct {
	GroupID int
	Points  int
	Public  bool
	TestIDs []int
	Subtask int
}

func (t *Task) GetInfoOnTestGroup(id int) TestGroupInfo {
	return TestGroupInfo{
		GroupID: id,
		Points:  t.tGroupPoints[id],
		Public:  t.isTGroupPublic[id],
		TestIDs: t.tGroupTestIDs[id],
		Subtask: t.tGroupToStMap[id],
	}
}

func (t *Task) GetTestGroupIDs() []int {
	return t.testGroupIDs
}

func (t *Task) GetTestFilenameFromID(testID int) string {
	filename, ok := t.testIDToFilename[testID]
	if !ok {
		return ""
	}
	return filename
}

func (t *Task) testGroupWithIDExists(id int) bool {
	for i := 0; i < len(t.testGroupIDs); i++ {
		if t.testGroupIDs[i] == id {
			return true
		}
	}
	return false
}

func (t *Task) AddTestGroupWithID(groupID int, points int, public bool, testIDs []int, subtask int) error {
	if t.testGroupWithIDExists(groupID) {
		return fmt.Errorf("test group with ID %d already exists", groupID)
	}
	t.testGroupIDs = append(t.testGroupIDs, groupID)
	t.isTGroupPublic[groupID] = public
	t.tGroupPoints[groupID] = points
	t.tGroupTestIDs[groupID] = testIDs
	t.tGroupToStMap[groupID] = subtask

	return nil
}

func (t *Task) testGroupMexPositiveID() int {
	mex := 1
	found := true
	for found {
		found = false
		for i := 0; i < len(t.testGroupIDs); i++ {
			if t.testGroupIDs[i] == mex {
				found = true
				mex++
			}
		}
	}
	return mex
}

func (t *Task) AddTestGroup(points int, public bool, testIDs []int, subtask int) {
	err := t.AddTestGroupWithID(t.testGroupMexPositiveID(), points, public, testIDs, subtask)
	if err != nil {
		log.Fatalf("error adding test group: %v", err)
	}
}

func (t *Task) GetPDFStatement(lang string) ([]byte, error) {
	statement, ok := t.pdfStatements[lang]
	if !ok {
		return nil, fmt.Errorf("pdf statement for language %s not found", lang)
	}

	return statement, nil
}

type PDFStatement struct {
	Language  string
	Statement []byte
}

func (t *Task) GetAllPDFStatements() []PDFStatement {
	pdfStatements := make([]PDFStatement, 0, len(t.pdfStatements))
	for lang, statement := range t.pdfStatements {
		pdfStatements = append(pdfStatements, PDFStatement{
			Language:  lang,
			Statement: statement,
		})
	}

	return pdfStatements
}

func (t *Task) AddPDFStatement(lang string, statement []byte) error {
	_, ok := t.pdfStatements[lang]
	if ok {
		return fmt.Errorf("pdf statement for language %s already exists", lang)
	}

	t.pdfStatements[lang] = statement
	return nil
}

func (t *Task) AddVisibleInputSubtask(subtask int) error {
	alreadyAdded := false

	for i := 0; i < len(t.visibleInputSubtasks); i++ {
		if t.visibleInputSubtasks[i] == subtask {
			alreadyAdded = true
		}
	}

	if alreadyAdded {
		return fmt.Errorf("subtask %d already added", subtask)
	}

	t.visibleInputSubtasks = append(t.visibleInputSubtasks, subtask)
	// sort by id
	sort.Ints(t.visibleInputSubtasks)
	return nil
}

func (t *Task) GetVisibleInputSubtasks() []int {
	return t.visibleInputSubtasks
}

type Asset struct {
	RelativePath string // relative path from assets directory
	Content      []byte
}

func (t *Task) GetTaskIllustrationImage() *Asset {
	for _, asset := range t.assets {
		if asset.RelativePath == t.illstrImgFname {
			return &Asset{
				RelativePath: asset.RelativePath,
				Content:      asset.Content,
			}
		}
	}

	return nil
}

func (t *Task) GetAssets() []asset {
	return t.assets
}

func (t *Task) GetOriginNotes() map[string]string {
	return t.OriginNotes
}