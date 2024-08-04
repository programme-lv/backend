package task

/*
package domain

import "fmt"

type Task struct {
	id string

	taskFullName      string
	memoryLimitMBytes int
	cpuTimeLimitSecs  float64
	difficulty        int // [1;5]
	originOlympiad    string
	problemTags       []string
	pdfStatements     []pdfSha256Ref
	mdStatements      map[string]*MarkdownStatement // map[language]statement
	ImgUuidToObjKey   map[string]string
	examples          []Example

	illustrationImgObjKey string

	originNotes map[string]string

	visInpSubtasks []int
	visInpStInputs map[int][]string

	tests      []TestSha256Ref
	testGroups []TestGroup
	subtasks   []Subtask
}

type TestSha256Ref struct {
	TestId       int64
	InputSha256  string
	AnswerSha256 string
}

type TestGroup struct {
	GroupId    int
	TestIds    []int
	SubtaskIds []int
}

type Subtask struct {
	SubtaskId int
	TestIds   []int
}

type Example struct {
	Input  string
	Output string
	MdNote *string
}

type MarkdownStatement struct {
	Story   string
	Input   string
	Output  string
	Notes   *string
	Scoring *string
}

type VisInpStInputs struct {
	Subtask int
	Inputs  []string
}

func (t *Task) GetVisInpStInputs() []VisInpStInputs {
	res := make([]VisInpStInputs, 0, len(t.visInpStInputs))
	for _, st := range t.visInpSubtasks {
		res = append(res, VisInpStInputs{
			Subtask: st,
			Inputs:  t.visInpStInputs[st],
		})
	}
	return res
}

func (t *Task) AddVisibleInputSubtask(subtask int, inputs []string) {
	t.visInpSubtasks = append(t.visInpSubtasks, subtask)
	if t.visInpStInputs == nil {
		t.visInpStInputs = make(map[int][]string)
	}
	t.visInpStInputs[subtask] = inputs
}

func (t *Task) GetImgUuidToObjKey() map[string]string {
	return t.ImgUuidToObjKey
}

func (t *Task) SetImgUuidToObjKey(imgUuidToObjKey map[string]string) {
	t.ImgUuidToObjKey = imgUuidToObjKey
}

func (t *Task) GetExamples() []Example {
	return t.examples
}

func (t *Task) AddExample(example Example) {
	t.examples = append(t.examples, example)
}

func (t *Task) GetOriginNotes() map[string]string {
	return t.originNotes
}

func (t *Task) SetOriginNotes(notes map[string]string) {
	t.originNotes = notes
}

func (t *Task) GetDefaultMarkdownStatement() *MarkdownStatement {
	for _, lang := range []string{"lv", "en", ""} {
		if _, ok := t.mdStatements[lang]; ok {
			return t.mdStatements[lang]
		}
	}
	return nil
}

func (t *Task) AddMarkdownStatement(language string, statement MarkdownStatement) {
	t.mdStatements[language] = &statement
}

func (t *Task) GetIllustrationImgObjKey() string {
	return t.illustrationImgObjKey
}

func (t *Task) SetIllustrationImgObjKey(key string) {
	t.illustrationImgObjKey = key
}

func (t *Task) GetId() string {
	return t.id
}

func (t *Task) GetTaskFullName() string {
	return t.taskFullName
}

func (t *Task) GetMemoryLimitMBytes() int {
	return t.memoryLimitMBytes
}

func (t *Task) GetCpuTimeLimitSecs() float64 {
	return t.cpuTimeLimitSecs
}

func (t *Task) GetDifficulty() int {
	return t.difficulty
}

func (t *Task) GetOriginOlympiad() string {
	return t.originOlympiad
}

func (t *Task) GetProblemTags() []string {
	return t.problemTags
}

func (t *Task) GetLvOrOtherPdfSha256() string {
	if len(t.pdfStatements) == 0 {
		return ""
	}
	return t.pdfStatements[0].Sha256
}

type pdfSha256Ref struct {
	Language string
	Sha256   string
}

func NewTask(id string, fullName string) (*Task, error) {
	task := &Task{
		id:                    id,
		taskFullName:          "",
		memoryLimitMBytes:     256,
		cpuTimeLimitSecs:      1.0,
		difficulty:            1,
		originOlympiad:        "",
		problemTags:           []string{},
		pdfStatements:         []pdfSha256Ref{},
		mdStatements:          map[string]*MarkdownStatement{},
		examples:              []Example{},
		illustrationImgObjKey: "",
		originNotes:           map[string]string{},
	}

	err := task.SetTaskFullName(fullName)
	if err != nil {
		return nil, fmt.Errorf("failed to set task full name: %w", err)
	}

	return task, nil
}

func (t *Task) SetTaskFullName(fullName string) error {
	if fullName == "" {
		return errorTaskFullNameIsRequired()
	}
	t.taskFullName = fullName
	return nil
}

func (t *Task) SetMemoryLimitMBytes(memoryLimit int) {
	t.memoryLimitMBytes = memoryLimit
}

func (t *Task) SetCpuTimeLimitSecs(cpuTimeLimit float64) {
	t.cpuTimeLimitSecs = cpuTimeLimit
}

func (t *Task) SetOriginOlympiad(origin string) {
	t.originOlympiad = origin
}

func (t *Task) AddPdfStatementSha256(language string, sha256 string) {
	t.pdfStatements = append(t.pdfStatements, pdfSha256Ref{Language: language, Sha256: sha256})
}

func (t *Task) SetProblemTags(tags []string) {
	t.problemTags = tags
}

func (t *Task) SetDifficulty(difficulty int) error {
	if difficulty < 1 || difficulty > 5 {
		return errorDifficultyMustBeBetweenOneAndFive()
	}
	t.difficulty = difficulty
	return nil
}

func (t *Task) SetTests(tests []TestSha256Ref) error {
	for _, test := range tests {
		if test.InputSha256 == "" || test.AnswerSha256 == "" {
			return errorEmptyTestSha256()
		}
		if test.TestId <= 0 {
			return errorTestIdMustBePositive()
		}
	}
	t.tests = tests
	return nil
}

func (t *Task) GetTests() []TestSha256Ref {
	return t.tests
}

func (t *Task) SetTestGroups(groups []TestGroup) {
	t.testGroups = groups
}

func (t *Task) GetTestGroups() []TestGroup {
	return t.testGroups
}

func (t *Task) SetSubtasks(subtasks []Subtask) {
	t.subtasks = subtasks
}

func (t *Task) GetSubtasks() []Subtask {
	return t.subtasks
}
*/

/*

 */
