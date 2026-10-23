package domain

import (
	"testing"

	"github.com/google/uuid"
	tasksrvc "github.com/programme-lv/backend/modules/task/srvc"
)

func fullRun(cpu []int, mem []int) ProfileRun {
	run := ProfileRun{}
	for i := range cpu {
		run.Tests = append(run.Tests, ProfileTest{
			TestID: i + 1, CpuMs: cpu[i], MemKiB: mem[i], Percent: 100,
		})
	}
	return run
}

func TestViableBoundsPartialSolution(t *testing.T) {
	task := profileTask()
	run := fullRun([]int{100, 200, 300, 400}, []int{1000, 2000, 3000, 4000})

	bounds := BuildViableBounds(task, []ProfileRun{run}, ResourceCPU, []int{1})
	if bounds.Lower == nil || *bounds.Lower != 201 {
		t.Fatalf("expected lower 201, got %+v", bounds)
	}
	if bounds.Upper == nil || *bounds.Upper != 401 {
		t.Fatalf("expected upper 401, got %+v", bounds)
	}
	if bounds.LowerKind != BoundFinite || bounds.UpperKind != BoundFinite {
		t.Fatalf("unexpected bound kinds: %+v", bounds)
	}
}

func TestViableBoundsFullSolutionHasOpenUpper(t *testing.T) {
	task := profileTask()
	run := fullRun([]int{100, 200, 300, 400}, []int{1000, 2000, 3000, 4000})

	bounds := BuildViableBounds(task, []ProfileRun{run}, ResourceCPU, []int{1, 2})
	if bounds.UpperKind != BoundOpen {
		t.Fatalf("expected open upper bound, got %+v", bounds)
	}
}

func TestViableBoundsUnknownUpper(t *testing.T) {
	task := profileTask()
	run := fullRun([]int{100, 200, 300, 400}, []int{1000, 2000, 3000, 4000})
	run.Tests[3].Percent = 0 // undeclared subtask can never be fully solved

	bounds := BuildViableBounds(task, []ProfileRun{run}, ResourceCPU, []int{1})
	if bounds.UpperKind != BoundUnknown {
		t.Fatalf("expected unknown upper bound, got %+v", bounds)
	}
}

func TestVerdictAtCurrentLimits(t *testing.T) {
	task := profileTask()
	run := fullRun([]int{100, 200, 300, 400}, []int{1000, 2000, 3000, 4000})

	ok := BuildVerdict(task, []ProfileRun{run}, []int{1}, 250, 100_000)
	if !ok.Applicable || !ok.Matches {
		t.Fatalf("expected matching verdict at 250ms, got %+v", ok)
	}
	if ok.Score.Median != 10 || ok.ExpectedPoints != 10 {
		t.Fatalf("unexpected score: %+v", ok)
	}

	bad := BuildVerdict(task, []ProfileRun{run}, []int{1}, 500, 100_000)
	if bad.Matches {
		t.Fatalf("expected non-matching verdict at 500ms, got %+v", bad)
	}
	if bad.Score.Median != 20 || bad.Delta != 10 {
		t.Fatalf("unexpected score at 500ms: %+v", bad)
	}
}

func groupProfileTask() tasksrvc.Task {
	return tasksrvc.Task{
		Tests: []tasksrvc.Test{{}, {}, {}, {}, {}},
		Subtasks: []tasksrvc.Subtask{
			{TestIDs: []int{1}},
			{TestIDs: []int{2, 3}},
			{TestIDs: []int{4, 5}},
		},
		TestGroups: []tasksrvc.TestGroup{
			{Points: 20, TestIDs: []int{1}},
			{Points: 20, TestIDs: []int{2, 3}},
			{Points: 10, TestIDs: []int{4}},
			{Points: 10, TestIDs: []int{5}},
		},
	}
}

func TestVerdictAllowsPartialGroupOverage(t *testing.T) {
	task := groupProfileTask()
	run := fullRun([]int{100, 200, 250, 300, 400}, []int{1000, 2000, 2500, 3000, 4000})

	verdict := BuildVerdict(task, []ProfileRun{run}, []int{1, 2}, 350, 100_000)
	if !verdict.Matches {
		t.Fatalf("expected partial group points to be allowed, got %+v", verdict)
	}
	if verdict.Score.Median != 50 || verdict.ExpectedPoints != 40 || verdict.Delta != 10 {
		t.Fatalf("expected 50 vs 40 (+10), got %+v", verdict)
	}
	if len(verdict.Subtasks) != 3 {
		t.Fatalf("expected 3 subtasks, got %d", len(verdict.Subtasks))
	}
	third := verdict.Subtasks[2]
	if third.FullySolvedAnyRun {
		t.Fatalf("subtask 3 must not be fully solved: %+v", third)
	}
	if third.EarnedMedianTenths != 100 {
		t.Fatalf("expected 10 partial points from subtask 3, got %d tenths", third.EarnedMedianTenths)
	}
}

func TestViableBoundsGroupScoring(t *testing.T) {
	task := groupProfileTask()
	run := fullRun([]int{100, 200, 250, 300, 400}, []int{1000, 2000, 2500, 3000, 4000})

	bounds := BuildViableBounds(task, []ProfileRun{run}, ResourceCPU, []int{1, 2})
	if bounds.Lower == nil || *bounds.Lower != 251 {
		t.Fatalf("expected lower 251, got %+v", bounds)
	}
	if bounds.Upper == nil || *bounds.Upper != 401 {
		t.Fatalf("expected upper 401, got %+v", bounds)
	}
}

func TestBuildCalibrationPaddedIntersection(t *testing.T) {
	task := profileTask()
	acID := uuid.New()
	bruteID := uuid.New()
	acLower := 100
	bruteLower := 200
	bruteUpper := 1_000
	memACLower := 10_000
	memBruteLower := 20_000
	memBruteUpper := 100_000

	calibration := BuildCalibration(task, 500, 50_000, []SolutionCalibration{
		{
			Fname: "ac", ProfileID: &acID, ExpectedSubtasks: []int{1, 2},
			CPU: ViableBounds{Lower: &acLower, LowerKind: BoundFinite, UpperKind: BoundOpen},
			Mem: ViableBounds{Lower: &memACLower, LowerKind: BoundFinite, UpperKind: BoundOpen},
		},
		{
			Fname: "brute", ProfileID: &bruteID, ExpectedSubtasks: []int{1},
			CPU: ViableBounds{Lower: &bruteLower, LowerKind: BoundFinite, Upper: &bruteUpper, UpperKind: BoundFinite},
			Mem: ViableBounds{Lower: &memBruteLower, LowerKind: BoundFinite, Upper: &memBruteUpper, UpperKind: BoundFinite},
		},
	}, nil, nil, nil)

	if calibration.CPU.Lower == nil || *calibration.CPU.Lower != 200 {
		t.Fatalf("expected cpu lower 200, got %+v", calibration.CPU)
	}
	if calibration.CPU.PaddedLower == nil || *calibration.CPU.PaddedLower != 300 {
		t.Fatalf("expected padded cpu lower 300, got %+v", calibration.CPU)
	}
	if calibration.CPU.PaddedUpper == nil || *calibration.CPU.PaddedUpper != 800 {
		t.Fatalf("expected padded cpu upper 800, got %+v", calibration.CPU)
	}
	if calibration.CPU.CurrentStatus != "ok" {
		t.Fatalf("expected ok cpu status, got %s", calibration.CPU.CurrentStatus)
	}
	if calibration.CPU.LowerMargin == nil || *calibration.CPU.LowerMargin != 200 {
		t.Fatalf("unexpected cpu lower margin: %+v", calibration.CPU)
	}
	if calibration.CPU.UpperMargin == nil || *calibration.CPU.UpperMargin != 300 {
		t.Fatalf("unexpected cpu upper margin: %+v", calibration.CPU)
	}

	if calibration.Mem.PaddedLower == nil || *calibration.Mem.PaddedLower != 35_625 {
		t.Fatalf("expected padded mem lower 35625, got %+v", calibration.Mem)
	}
	if calibration.Mem.PaddedUpper == nil || *calibration.Mem.PaddedUpper != 84_375 {
		t.Fatalf("expected padded mem upper 84375, got %+v", calibration.Mem)
	}
}

func TestBuildCalibrationStatuses(t *testing.T) {
	task := profileTask()
	id := uuid.New()
	lower, upper := 200, 1_000
	solutions := []SolutionCalibration{{
		Fname: "brute", ProfileID: &id, ExpectedSubtasks: []int{1},
		CPU: ViableBounds{Lower: &lower, LowerKind: BoundFinite, Upper: &upper, UpperKind: BoundFinite},
		Mem: ViableBounds{LowerKind: BoundOpen, UpperKind: BoundOpen},
	}}

	tooLow := BuildCalibration(task, 250, 1, solutions, nil, nil, nil)
	if tooLow.CPU.CurrentStatus != "too_low" {
		t.Fatalf("expected too_low, got %s", tooLow.CPU.CurrentStatus)
	}
	tooHigh := BuildCalibration(task, 900, 1, solutions, nil, nil, nil)
	if tooHigh.CPU.CurrentStatus != "too_high" {
		t.Fatalf("expected too_high, got %s", tooHigh.CPU.CurrentStatus)
	}
}

func TestBuildCalibrationUnknownAndEmpty(t *testing.T) {
	task := profileTask()
	id := uuid.New()
	lower := 200
	unknown := []SolutionCalibration{{
		Fname: "brute", ProfileID: &id, ExpectedSubtasks: []int{1},
		CPU: ViableBounds{Lower: &lower, LowerKind: BoundFinite, UpperKind: BoundUnknown},
		Mem: ViableBounds{LowerKind: BoundNone, UpperKind: BoundNone},
	}}
	calibration := BuildCalibration(task, 500, 1, unknown, nil, nil, nil)
	if calibration.CPU.CurrentStatus != "unknown" {
		t.Fatalf("expected unknown status, got %s", calibration.CPU.CurrentStatus)
	}

	sameLower, sameUpper := 401, 401
	emptySolutions := []SolutionCalibration{{
		Fname: "brute", ProfileID: &id, ExpectedSubtasks: []int{1},
		CPU: ViableBounds{Lower: &sameLower, LowerKind: BoundFinite, Upper: &sameUpper, UpperKind: BoundFinite},
		Mem: ViableBounds{LowerKind: BoundNone, UpperKind: BoundNone},
	}}
	empty := BuildCalibration(task, 401, 1, emptySolutions, nil, nil, nil)
	if empty.CPU.CurrentStatus != "empty" {
		t.Fatalf("expected empty status, got %s", empty.CPU.CurrentStatus)
	}
}
