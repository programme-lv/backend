package srvc

import (
	"testing"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/modules/exec"
	"github.com/programme-lv/backend/modules/subm/domain"
	tasksrvc "github.com/programme-lv/backend/modules/task/srvc"
)

func profileTestTask() tasksrvc.Task {
	return tasksrvc.Task{
		ShortId: "profile-test",
		Tests:   []tasksrvc.Test{{}, {}, {}, {}},
		Subtasks: []tasksrvc.Subtask{
			{Score: 10, TestIDs: []int{1, 2}},
			{Score: 10, TestIDs: []int{3, 4}},
		},
	}
}

func TestProfileRunFromExec(t *testing.T) {
	execution := exec.Execution{
		UUID:  uuid.New(),
		Stage: exec.StageFinished,
		TestRes: []exec.TestRes{
			{ID: 1, Subm: &exec.RunData{CpuMs: 100, MemKiB: 1_000}},
			{ID: 2, Subm: &exec.RunData{CpuMs: 200, MemKiB: 2_000}},
			{ID: 3, Subm: &exec.RunData{CpuMs: 300, MemKiB: 3_000}},
			{ID: 4, Subm: &exec.RunData{CpuMs: 400, MemKiB: 4_000}},
		},
	}

	run, err := profileRunFromExec(profileTestTask(), execution)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(run.Tests) != 4 {
		t.Fatalf("expected 4 tests, got %d", len(run.Tests))
	}
	for _, test := range run.Tests {
		if test.Percent != 100 {
			t.Fatalf("test %d: expected 100%%, got %d", test.TestID, test.Percent)
		}
	}
	if score := domain.ProfileRunScore(profileTestTask(), run); score != 20 {
		t.Fatalf("expected score 20, got %d", score)
	}
}

func TestProfileRunFromExecMarksTimeoutAsZero(t *testing.T) {
	execution := exec.Execution{
		UUID:  uuid.New(),
		Stage: exec.StageFinished,
		TestRes: []exec.TestRes{
			{ID: 1, Subm: &exec.RunData{CpuMs: domain.ProfileCpuMs, MemKiB: 1_000}},
			{ID: 2, Subm: &exec.RunData{CpuMs: 200, MemKiB: 2_000}},
			{ID: 3, Subm: &exec.RunData{CpuMs: 300, MemKiB: 3_000}},
			{ID: 4, Subm: &exec.RunData{CpuMs: 400, MemKiB: 4_000}},
		},
	}

	run, err := profileRunFromExec(profileTestTask(), execution)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.Tests[0].Percent != 0 {
		t.Fatalf("expected test 1 to score 0, got %d", run.Tests[0].Percent)
	}
	if score := domain.ProfileRunScore(profileTestTask(), run); score != 10 {
		t.Fatalf("expected subtask 2 score 10, got %d", score)
	}
}

func TestProfileRunFromExecCompileError(t *testing.T) {
	message := "compilation failed"
	execution := exec.Execution{
		Stage:    exec.StageCompileError,
		ErrorMsg: &message,
	}

	if _, err := profileRunFromExec(profileTestTask(), execution); err == nil {
		t.Fatal("expected an error")
	}
}
