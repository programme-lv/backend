package main

import (
	"fmt"

	"github.com/programme-lv/backend/fstask"
	"github.com/programme-lv/backend/tasksrvc"
)

// TaskWrapper provides convenient methods to access task-related data.
type TaskWrapper struct {
	Task            *fstask.Task
	PdfSttmntLangs  []string
	MdSttmntLangs   []string
	TotalScore      *int
	TestCount       *int
	TestGroupPoints []int
	TestTotalSize   *int
}

// NewTaskWrapper initializes a new TaskWrapper.
func NewTaskWrapper(task *fstask.Task) *TaskWrapper {
	return &TaskWrapper{
		Task: task,
	}
}

// GetTestTotalSize calculates the total size of all tests.
func (tw *TaskWrapper) GetTestTotalSize() int {
	if tw.TestTotalSize != nil {
		return *tw.TestTotalSize
	}

	totalSize := 0
	for _, test := range tw.Task.GetTestsSortedByID() {
		totalSize += len(test.Input) + len(test.Answer)
	}
	tw.TestTotalSize = &totalSize
	return totalSize
}

// GetTestGroupPoints retrieves the points for each test group.
func (tw *TaskWrapper) GetTestGroupPoints() []int {
	if tw.TestGroupPoints != nil {
		return tw.TestGroupPoints
	}

	groups := tw.Task.GetTestGroupIDs()
	tw.TestGroupPoints = make([]int, len(groups))
	for i, groupID := range groups {
		tw.TestGroupPoints[i] = tw.Task.GetInfoOnTestGroup(groupID).Points
	}
	return tw.TestGroupPoints
}

// GetTestGroupPointsRLE returns the run-length encoded points of test groups.
func (tw *TaskWrapper) GetTestGroupPointsRLE() []string {
	points := tw.GetTestGroupPoints()
	if len(points) == 0 {
		return []string{}
	}

	type rleElement struct {
		count int
		ele   int
	}

	var rle []rleElement
	rle = append(rle, rleElement{count: 1, ele: points[0]})

	for i := 1; i < len(points); i++ {
		if points[i] == points[i-1] {
			rle[len(rle)-1].count++
		} else {
			rle = append(rle, rleElement{count: 1, ele: points[i]})
		}
	}

	res := make([]string, len(rle))
	for i, elem := range rle {
		res[i] = fmt.Sprintf("%d*%d", elem.count, elem.ele)
	}
	return res
}

// GetPdfStatementLangs retrieves all PDF statement languages.
func (tw *TaskWrapper) GetPdfStatementLangs() []string {
	if tw.PdfSttmntLangs != nil {
		return tw.PdfSttmntLangs
	}

	pdfStmts := tw.Task.GetPdfStatements()
	tw.PdfSttmntLangs = make([]string, len(pdfStmts))
	for i, stmt := range pdfStmts {
		tw.PdfSttmntLangs[i] = stmt.Language
	}
	return tw.PdfSttmntLangs
}

// GetMdStatementLangs retrieves all Markdown statement languages.
func (tw *TaskWrapper) GetMdStatementLangs() []string {
	if tw.MdSttmntLangs != nil {
		return tw.MdSttmntLangs
	}

	mdStmts := tw.Task.GetMarkdownStatements()
	tw.MdSttmntLangs = make([]string, len(mdStmts))
	for i, stmt := range mdStmts {
		tw.MdSttmntLangs[i] = stmt.Language
	}
	return tw.MdSttmntLangs
}

// GetTestTotalCount returns the total number of tests.
func (tw *TaskWrapper) GetTestTotalCount() int {
	if tw.TestCount != nil {
		return *tw.TestCount
	}

	count := len(tw.Task.GetTestsSortedByID())
	tw.TestCount = &count
	return count
}

// GetTotalScore calculates the total score for the task.
func (tw *TaskWrapper) GetTotalScore() int {
	if tw.TotalScore != nil {
		return *tw.TotalScore
	}

	tests := tw.Task.GetTestsSortedByID()
	groups := tw.Task.GetTestGroupIDs()

	if len(groups) == 0 {
		score := len(tests)
		tw.TotalScore = &score
		return score
	}

	groupPoints := tw.GetTestGroupPoints()
	score := 0
	for _, points := range groupPoints {
		score += points
	}

	tw.TotalScore = &score
	return score
}

// GetVisibleInputSubtasks retrieves visible input subtasks.
func (tw *TaskWrapper) GetVisibleInputSubtasks() []tasksrvc.VisInpSt {
	tests := tw.Task.GetTestsSortedByID()
	visibleSubtasks := tw.Task.GetVisibleInputSubtaskIds()
	visInpSts := make([]tasksrvc.VisInpSt, len(visibleSubtasks))

	for i, stID := range visibleSubtasks {
		visInpSts[i].Subtask = stID
		visInpSts[i].Inputs = []tasksrvc.TestWithOnlyInput{}

		for _, tGroup := range tw.Task.GetTestGroups() {
			if tGroup.Subtask != stID {
				continue
			}
			for _, testID := range tGroup.TestIDs {
				for _, test := range tests {
					if test.ID == testID {
						visInpSts[i].Inputs = append(visInpSts[i].Inputs, tasksrvc.TestWithOnlyInput{
							TestID: testID,
							Input:  string(test.Input),
						})
						break
					}
				}
			}
		}
	}

	return visInpSts
}
