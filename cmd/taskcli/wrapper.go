package main

import "github.com/programme-lv/backend/fstask"

type taskWrapper struct {
	task            *fstask.Task
	pdfSttmntLangs  []string
	mdSttmntLangs   []string
	totalScore      *int
	testCount       *int
	testGroupPoints []int
	testTotalSize   *int
}

func newTaskWrapper(t *fstask.Task) *taskWrapper {
	return &taskWrapper{
		task: t,
	}
}

func (t *taskWrapper) GetTestTotalSize() int {
	if t.testTotalSize == nil {
		totalSize := 0
		for _, test := range t.task.GetTestsSortedByID() {
			totalSize += len(test.Input)
			totalSize += len(test.Answer)
		}
		t.testTotalSize = &totalSize
		return totalSize
	} else {
		return *t.testTotalSize
	}
}

func (t *taskWrapper) GetTestGroupPoints() []int {
	if t.testGroupPoints == nil {
		groups := t.task.GetTestGroupIDs()
		t.testGroupPoints = make([]int, len(groups))
		for i, groupID := range groups {
			t.testGroupPoints[i] = t.task.GetInfoOnTestGroup(groupID).Points
		}
		return t.testGroupPoints
	} else {
		return t.testGroupPoints
	}
}

func (t *taskWrapper) GetPdfStatementLangs() []string {
	if t.pdfSttmntLangs == nil {
		pdfSttments := t.task.GetAllPDFStatements()
		t.pdfSttmntLangs = make([]string, len(pdfSttments))
		for i, pdfSttmnt := range pdfSttments {
			t.pdfSttmntLangs[i] = pdfSttmnt.Language
		}
		return t.pdfSttmntLangs
	} else {
		return t.pdfSttmntLangs
	}
}

func (t *taskWrapper) GetMdStatementLangs() []string {
	if t.mdSttmntLangs == nil {
		mdSttments := t.task.GetMarkdownStatements()
		t.mdSttmntLangs = make([]string, len(mdSttments))
		for i, mdSttmnt := range mdSttments {
			t.mdSttmntLangs[i] = mdSttmnt.Language
		}
		return t.mdSttmntLangs
	} else {
		return t.mdSttmntLangs
	}
}

func (t *taskWrapper) GetTestTotalCount() int {
	if t.testCount != nil {
		return *t.testCount
	}
	tests := t.task.GetTestsSortedByID()
	res := len(tests)
	t.testCount = &res
	return res
}

func (t *taskWrapper) GetTotalScore() int {
	if t.totalScore != nil {
		return *t.totalScore
	}

	tests := t.task.GetTestsSortedByID()
	testTotalCount := 0
	testTotalSize := 0
	for _, test := range tests {
		testTotalCount++
		testTotalSize += len(test.Answer)
		testTotalSize += len(test.Input)
	}

	groups := t.task.GetTestGroupIDs()
	testGroupPoints := make([]int, len(groups))
	for _, groupID := range groups {
		info := t.task.GetInfoOnTestGroup(groupID)
		testGroupPoints[groupID-1] = info.Points
	}

	totalScore := 0
	if len(groups) == 0 {
		totalScore = len(tests)
	} else {
		totalScore = 0
		for _, groupID := range groups {
			totalScore += testGroupPoints[groupID-1]
		}
	}

	t.totalScore = &totalScore

	return totalScore
}
