package domain

import (
	"testing"

	tasksrvc "github.com/programme-lv/backend/modules/task/srvc"
)

func profileTask() tasksrvc.Task {
	return tasksrvc.Task{
		ShortId: "profile-test",
		Tests:   []tasksrvc.Test{{}, {}, {}, {}},
		Subtasks: []tasksrvc.Subtask{
			{Score: 10, TestIDs: []int{1, 2}},
			{Score: 10, TestIDs: []int{3, 4}},
		},
	}
}

func TestProfileExpectedPoints(t *testing.T) {
	task := profileTask()

	got := ProfileExpectedPoints(task, tasksrvc.Solution{Subtasks: []int{1}})
	if got != 10 {
		t.Fatalf("expected 10 points, got %d", got)
	}

	got = ProfileExpectedPoints(task, tasksrvc.Solution{Subtasks: []int{1, 2}})
	if got != 20 {
		t.Fatalf("expected 20 points, got %d", got)
	}
}

func TestProfileExpectedPointsWithoutSubtasks(t *testing.T) {
	task := tasksrvc.Task{Tests: []tasksrvc.Test{{}, {}, {}, {}}}

	got := ProfileExpectedPoints(task, tasksrvc.Solution{})
	if got != 4 {
		t.Fatalf("expected full score 4, got %d", got)
	}
}

func TestProfileExpectedPointsUsesTestGroups(t *testing.T) {
	task := profileTask()
	task.TestGroups = []tasksrvc.TestGroup{
		{Points: 10, TestIDs: []int{1, 2}},
		{Points: 10, TestIDs: []int{3, 4}},
	}

	got := ProfileExpectedPoints(task, tasksrvc.Solution{Subtasks: []int{1}})
	if got != 10 {
		t.Fatalf("expected group score 10, got %d", got)
	}
}

func TestBuildProfileChartCPU(t *testing.T) {
	task := profileTask()
	run := ProfileRun{Tests: []ProfileTest{
		{TestID: 1, CpuMs: 100, MemKiB: 1_000, Percent: 100},
		{TestID: 2, CpuMs: 200, MemKiB: 2_000, Percent: 100},
		{TestID: 3, CpuMs: 5_000, MemKiB: 3_000, Percent: 100},
		{TestID: 4, CpuMs: 6_000, MemKiB: 4_000, Percent: 100},
	}}

	chart := BuildProfileChart(task, []ProfileRun{run}, ResourceCPU, 20)

	if chart.Unit != "ms" || chart.Max != ProfileCpuMs {
		t.Fatalf("unexpected chart header: %+v", chart)
	}
	if got := chart.LowerBound; got == nil || *got != 6_001 {
		t.Fatalf("expected lower bound 6001, got %v", got)
	}
	if chart.UpperBound != nil {
		t.Fatalf("expected no upper bound, got %d", *chart.UpperBound)
	}
	if got := pointAt(chart, 100).Median; got != 0 {
		t.Fatalf("at 100ms expected 0, got %d", got)
	}
	if got := pointAt(chart, 101).Median; got != 0 {
		t.Fatalf("at 101ms expected 0, got %d", got)
	}
	if got := pointAt(chart, 201).Median; got != 10 {
		t.Fatalf("at 201ms expected 10, got %d", got)
	}
	if got := pointAt(chart, 6_001).Median; got != 20 {
		t.Fatalf("at 6001ms expected 20, got %d", got)
	}
}

func TestBuildProfileChartMemory(t *testing.T) {
	task := profileTask()
	run := ProfileRun{Tests: []ProfileTest{
		{TestID: 1, CpuMs: 100, MemKiB: 1_000, Percent: 100},
		{TestID: 2, CpuMs: 200, MemKiB: 2_000, Percent: 100},
		{TestID: 3, CpuMs: 300, MemKiB: 3_000, Percent: 100},
		{TestID: 4, CpuMs: 400, MemKiB: 4_000, Percent: 100},
	}}

	chart := BuildProfileChart(task, []ProfileRun{run}, ResourceMemory, 20)

	if chart.Unit != "kib" || chart.Max != ProfileMemKiB {
		t.Fatalf("unexpected chart header: %+v", chart)
	}
	if got := chart.LowerBound; got == nil || *got != 4_001 {
		t.Fatalf("expected lower bound 4001, got %v", got)
	}
	if got := pointAt(chart, 4_000).Median; got != 10 {
		t.Fatalf("at 4000 KiB expected 10, got %d", got)
	}
	if got := pointAt(chart, 4_001).Median; got != 20 {
		t.Fatalf("at 4001 KiB expected 20, got %d", got)
	}
}

func TestBuildProfileChartFluctuation(t *testing.T) {
	task := tasksrvc.Task{
		Tests: []tasksrvc.Test{{}, {}},
		Subtasks: []tasksrvc.Subtask{
			{Score: 10, TestIDs: []int{1}},
			{Score: 10, TestIDs: []int{2}},
		},
	}
	fast := ProfileRun{Tests: []ProfileTest{
		{TestID: 1, CpuMs: 800, Percent: 100},
		{TestID: 2, CpuMs: 1_500, Percent: 100},
	}}
	slow := ProfileRun{Tests: []ProfileTest{
		{TestID: 1, CpuMs: 900, Percent: 100},
		{TestID: 2, CpuMs: 1_600, Percent: 100},
	}}

	chart := BuildProfileChart(task, []ProfileRun{fast, slow}, ResourceCPU, 10)

	if got := chart.LowerBound; got == nil || *got != 901 {
		t.Fatalf("expected lower bound 901, got %v", got)
	}
	if got := chart.UpperBound; got == nil || *got != 1_501 {
		t.Fatalf("expected upper bound 1501, got %v", got)
	}
	if got := pointAt(chart, 1_200).Min; got != 10 {
		t.Fatalf("worst run at 1200ms expected 10, got %d", got)
	}
	if got := pointAt(chart, 1_200).Max; got != 10 {
		t.Fatalf("best run at 1200ms expected 10, got %d", got)
	}
}

func TestBuildProfileChartNeverReachesExpected(t *testing.T) {
	task := profileTask()
	run := ProfileRun{Tests: []ProfileTest{
		{TestID: 1, CpuMs: 100, Percent: 100},
		{TestID: 2, CpuMs: 200, Percent: 100},
		{TestID: 3, CpuMs: 300, Percent: 0},
		{TestID: 4, CpuMs: 400, Percent: 0},
	}}

	chart := BuildProfileChart(task, []ProfileRun{run}, ResourceCPU, 20)

	if chart.LowerBound != nil {
		t.Fatalf("expected no lower bound, got %d", *chart.LowerBound)
	}
	if chart.UpperBound != nil {
		t.Fatalf("expected no upper bound, got %d", *chart.UpperBound)
	}
}

func TestBuildProfileChartWithoutExpectedScore(t *testing.T) {
	task := profileTask()
	run := ProfileRun{Tests: []ProfileTest{
		{TestID: 1, CpuMs: 100, Percent: 100},
		{TestID: 2, CpuMs: 200, Percent: 100},
		{TestID: 3, CpuMs: 300, Percent: 100},
		{TestID: 4, CpuMs: 400, Percent: 100},
	}}

	chart := BuildProfileChart(task, []ProfileRun{run}, ResourceCPU, 0)

	if chart.LowerBound != nil || chart.UpperBound != nil {
		t.Fatalf("expected no bounds when no expected score is declared: %+v", chart)
	}
}

func TestBuildProfileChartEmpty(t *testing.T) {
	chart := BuildProfileChart(profileTask(), nil, ResourceCPU, 10)
	if len(chart.Points) != 0 || chart.LowerBound != nil {
		t.Fatalf("unexpected chart for empty runs: %+v", chart)
	}
}

func TestProfileRunScore(t *testing.T) {
	task := profileTask()
	run := ProfileRun{Tests: []ProfileTest{
		{TestID: 1, CpuMs: 100, Percent: 100},
		{TestID: 2, CpuMs: 200, Percent: 100},
		{TestID: 3, CpuMs: 300, Percent: 0},
		{TestID: 4, CpuMs: 400, Percent: 0},
	}}

	if got := ProfileRunScore(task, run); got != 10 {
		t.Fatalf("expected 10, got %d", got)
	}
}

func pointAt(chart LimitChart, limit int) LimitPoint {
	result := LimitPoint{}
	for _, point := range chart.Points {
		if point.Limit > limit {
			break
		}
		result = point
	}
	return result
}
