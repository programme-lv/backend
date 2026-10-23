package domain

import (
	"slices"
	"sort"
	"time"

	"github.com/google/uuid"
	tasksrvc "github.com/programme-lv/backend/modules/task/srvc"
)

// Profiling runs a solution once at the loosest limits the tester accepts and
// derives score-vs-limit curves from the measured per-test CPU and memory use.
const (
	ProfileCpuMs  = 10_000    // 10 s
	ProfileMemKiB = 1_048_576 // 1 GiB
)

type ProfileStatus string

const (
	ProfileRunning  ProfileStatus = "running"
	ProfileFinished ProfileStatus = "finished"
	ProfileFailed   ProfileStatus = "failed"
)

// ProfileTest is one test measurement from a profiling run.
type ProfileTest struct {
	TestID  int `json:"test_id"`
	CpuMs   int `json:"cpu_ms"`
	MemKiB  int `json:"mem_kib"`
	Percent int `json:"percent"` // score earned at the profiling limits
}

// ProfileRun is one repetition of the profiling execution.
type ProfileRun struct {
	ExecUUID uuid.UUID     `json:"exec_uuid"`
	Tests    []ProfileTest `json:"tests"`
}

func (r ProfileRun) Test(testID int) (ProfileTest, bool) {
	for _, test := range r.Tests {
		if test.TestID == testID {
			return test, true
		}
	}
	return ProfileTest{}, false
}

// ProfileProgress is the in-memory progress of a running profile. It is not
// persisted: after a restart the profile is interrupted anyway.
type ProfileProgress struct {
	Run        int `json:"run"` // 1-based
	Runs       int `json:"runs"`
	TestsDone  int `json:"tests_done"`
	TestsTotal int `json:"tests_total"`
}

// SolutionProfile is a persisted calibration run for one task solution.
type SolutionProfile struct {
	ID             uuid.UUID
	TaskShortID    string
	Fname          string
	LangID         string
	Status         ProfileStatus
	Error          *string
	ExpectedPoints int
	PossiblePoints int
	RequestedRuns  int
	TestsVersion   int
	SolutionSha256 string
	Runs           []ProfileRun
	CreatedAt      time.Time
	FinishedAt     *time.Time

	// Progress is populated for running profiles only.
	Progress *ProfileProgress
}

// Resource selects the axis of a profiling chart.
type Resource int

const (
	ResourceCPU Resource = iota
	ResourceMemory
)

// LimitPoint is one point of a step chart. Min and Max are the worst and best
// scores across repetitions; Median is the middle repetition.
type LimitPoint struct {
	Limit  int `json:"limit"` // milliseconds or kibibytes
	Min    int `json:"min"`
	Median int `json:"median"`
	Max    int `json:"max"`
}

// LimitChart is a score-vs-limit step function.
type LimitChart struct {
	Unit       string       `json:"unit"` // "ms" or "kib"
	Max        int          `json:"max"`
	Points     []LimitPoint `json:"points"`
	LowerBound *int         `json:"lower_bound"`
	UpperBound *int         `json:"upper_bound"`
}

// ProfilePossiblePoints returns the maximum score of the task.
func ProfilePossiblePoints(task tasksrvc.Task) int {
	eval := NewEval(uuid.Nil, uuid.Nil, task)
	return eval.CalculateScore().PossibleScore
}

// ProfileExpectedPoints returns the score a solution should receive when it
// passes exactly the tests of its declared subtasks. Scoring goes through the
// real scorer so test-group tasks are handled correctly.
func ProfileExpectedPoints(task tasksrvc.Task, sol tasksrvc.Solution) int {
	wanted := make(map[int]struct{})
	for _, subtaskID := range sol.Subtasks {
		if subtaskID < 1 || subtaskID > len(task.Subtasks) {
			continue
		}
		for _, testID := range task.Subtasks[subtaskID-1].TestIDs {
			wanted[testID] = struct{}{}
		}
	}

	eval := NewEval(uuid.Nil, uuid.Nil, task)
	for i := range eval.Tests {
		if _, ok := wanted[i+1]; ok {
			eval.Tests[i].Ac = true
			eval.Tests[i].Reached = true
			eval.Tests[i].Finished = true
		}
	}
	return eval.CalculateScore().ReceivedScore
}

// ProfileRunScore returns the score of a run at the profiling limits.
func ProfileRunScore(task tasksrvc.Task, run ProfileRun) int {
	return profileScoreAt(task, run, ProfileCpuMs, ProfileMemKiB)
}

// ProfileScoreAt returns the predicted score of a measured run at the given
// limits.
func ProfileScoreAt(task tasksrvc.Task, run ProfileRun, cpuMs, memKiB int) int {
	return profileScoreAt(task, run, cpuMs, memKiB)
}

// BuildProfileChart derives a score-vs-limit step chart. The other resource
// stays at its profiling maximum so the chart isolates one dimension.
func BuildProfileChart(task tasksrvc.Task, runs []ProfileRun, resource Resource, expected int) LimitChart {
	unit := "ms"
	maxLimit := ProfileCpuMs
	if resource == ResourceMemory {
		unit = "kib"
		maxLimit = ProfileMemKiB
	}

	chart := LimitChart{Unit: unit, Max: maxLimit, Points: []LimitPoint{}}
	if len(runs) == 0 {
		return chart
	}

	limits := []int{0, maxLimit}
	for _, run := range runs {
		for _, test := range run.Tests {
			if test.Percent == 0 {
				continue
			}
			value := test.CpuMs
			if resource == ResourceMemory {
				value = test.MemKiB
			}
			if value < maxLimit {
				// The judge compares with >=, so a test passes from value+1.
				limits = append(limits, value, value+1)
			}
		}
	}
	sort.Ints(limits)
	limits = slices.Compact(limits)

	for _, limit := range limits {
		cpuMs, memKiB := ProfileCpuMs, ProfileMemKiB
		if resource == ResourceCPU {
			cpuMs = limit
		} else {
			memKiB = limit
		}

		scores := make([]int, len(runs))
		for i, run := range runs {
			scores[i] = profileScoreAt(task, run, cpuMs, memKiB)
		}
		sort.Ints(scores)
		chart.Points = append(chart.Points, LimitPoint{
			Limit:  limit,
			Min:    scores[0],
			Median: scores[len(scores)/2],
			Max:    scores[len(scores)-1],
		})
	}

	// A solution without declared subtasks has no expected score; leave both
	// bounds unset instead of treating zero as a target.
	if expected > 0 {
		for _, point := range chart.Points {
			if point.Min >= expected {
				lower := point.Limit
				chart.LowerBound = &lower
				break
			}
		}
		for _, point := range chart.Points {
			if point.Max > expected {
				upper := point.Limit
				chart.UpperBound = &upper
				break
			}
		}
	}

	return chart
}

// profileScoreAt scores a measured run as if the tester had used cpuMs and
// memKiB. Tests that scored zero at the profiling limits cannot recover at a
// tighter limit; the rest pass while their measured use stays below the limit.
func profileScoreAt(task tasksrvc.Task, run ProfileRun, cpuMs, memKiB int) int {
	eval := evalFromRun(task, run, cpuMs, memKiB)
	return eval.CalculateScore().ReceivedScore
}
