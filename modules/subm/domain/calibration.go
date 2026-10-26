package domain

import (
	"math"
	"slices"
	"sort"

	"github.com/google/uuid"
	tasksrvc "github.com/programme-lv/backend/modules/task/srvc"
)

// Calibration classifies a solution by the subtasks it is expected to solve.
// A declared subtask must be fully solved; an undeclared one must not be fully
// solved. Partial points from groups or interactor percentages in an undeclared
// subtask are allowed, because the subtask is not maxed.

type BoundKind string

const (
	BoundFinite  BoundKind = "finite"
	BoundOpen    BoundKind = "open"    // no upper bound
	BoundUnknown BoundKind = "unknown" // outside the profiled range
	BoundNone    BoundKind = "none"    // not classified
)

type ViableBounds struct {
	Lower     *int      `json:"lower"`
	LowerKind BoundKind `json:"lower_kind"`
	Upper     *int      `json:"upper"`
	UpperKind BoundKind `json:"upper_kind"`
}

type ProfileScoreDistribution struct {
	Min    int `json:"min"`
	Median int `json:"median"`
	Max    int `json:"max"`
}

type SubtaskOutcome struct {
	SubtaskID          int  `json:"subtask_id"`
	Points             int  `json:"points"`
	Declared           bool `json:"declared"`
	FullySolvedAllRuns bool `json:"fully_solved_all_runs"`
	FullySolvedAnyRun  bool `json:"fully_solved_any_run"`
	EarnedMinTenths    int  `json:"earned_min_tenths"`
	EarnedMedianTenths int  `json:"earned_median_tenths"`
	EarnedMaxTenths    int  `json:"earned_max_tenths"`
}

type Verdict struct {
	Applicable     bool                     `json:"applicable"`
	Matches        bool                     `json:"matches"`
	Score          ProfileScoreDistribution `json:"score"`
	ExpectedPoints int                      `json:"expected_points"`
	PossiblePoints int                      `json:"possible_points"`
	Delta          int                      `json:"delta"`
	Subtasks       []SubtaskOutcome         `json:"subtasks"`
}

type SolutionCalibration struct {
	Fname            string       `json:"fname"`
	ExpectedSubtasks []int        `json:"expected_subtasks"`
	ExpectedPoints   int          `json:"expected_points"`
	PossiblePoints   int          `json:"possible_points"`
	ProfileID        *uuid.UUID   `json:"profile_id,omitempty"`
	Stale            bool         `json:"stale"`
	CPU              ViableBounds `json:"cpu"`
	Mem              ViableBounds `json:"mem"`
	Verdict          Verdict      `json:"verdict"`
}

type AxisCalibration struct {
	Lower           *int      `json:"lower"`
	LowerKind       BoundKind `json:"lower_kind"`
	Upper           *int      `json:"upper"`
	UpperKind       BoundKind `json:"upper_kind"`
	PaddedLower     *int      `json:"padded_lower"`
	PaddedUpper     *int      `json:"padded_upper"`
	RelativePadding float64   `json:"relative_padding"`
	AbsolutePadding int       `json:"absolute_padding"`
	Current         int       `json:"current"`
	CurrentStatus   string    `json:"current_status"` // ok | too_low | too_high | unknown | empty | none
	LowerMargin     *int      `json:"lower_margin"`
	UpperMargin     *int      `json:"upper_margin"`
}

type Calibration struct {
	CPU               AxisCalibration       `json:"cpu"`
	Mem               AxisCalibration       `json:"mem"`
	Solutions         []SolutionCalibration `json:"solutions"`
	CurrentMatches    int                   `json:"current_matches"`
	CurrentMismatches int                   `json:"current_mismatches"`
	Unclassified      []string              `json:"unclassified"`
	MissingProfiles   []string              `json:"missing_profiles"`
	StaleProfiles     []string              `json:"stale_profiles"`
}

const (
	CPURelativePadding    = 0.20
	CPUAbsolutePaddingMs  = 100
	MemRelativePadding    = 0.15
	MemAbsolutePaddingKiB = 15_625 // 16 decimal MB
)

// BuildViableBounds computes the range of limits where the solution fully
// solves every declared subtask and does not fully solve any undeclared one.
func BuildViableBounds(task tasksrvc.Task, runs []ProfileRun, resource Resource, declared []int) ViableBounds {
	if len(runs) == 0 {
		return ViableBounds{LowerKind: BoundNone, UpperKind: BoundNone}
	}
	if len(task.Subtasks) == 0 {
		return buildBinaryBounds(task, runs, resource)
	}
	if len(declared) == 0 {
		return ViableBounds{LowerKind: BoundNone, UpperKind: BoundNone}
	}

	declaredSet := make(map[int]bool, len(declared))
	for _, id := range declared {
		if id >= 1 && id <= len(task.Subtasks) {
			declaredSet[id] = true
		}
	}
	undeclared := make([]int, 0, len(task.Subtasks))
	for id := 1; id <= len(task.Subtasks); id++ {
		if !declaredSet[id] {
			undeclared = append(undeclared, id)
		}
	}

	maxLimit := ProfileCpuMs
	if resource == ResourceMemory {
		maxLimit = ProfileMemKiB
	}

	limits := []int{0, maxLimit}
	for _, run := range runs {
		for _, test := range run.Tests {
			if test.Percent != 100 {
				continue
			}
			value := test.CpuMs
			if resource == ResourceMemory {
				value = test.MemKiB
			}
			if value < maxLimit {
				limits = append(limits, value, value+1)
			}
		}
	}
	sort.Ints(limits)
	limits = slices.Compact(limits)

	var lower, upper *int
	for _, limit := range limits {
		cpuMs, memKiB := ProfileCpuMs, ProfileMemKiB
		if resource == ResourceCPU {
			cpuMs = limit
		} else {
			memKiB = limit
		}

		allDeclaredSolved := true
		anyUndeclaredSolved := false
		for _, run := range runs {
			eval := evalFromRun(task, run, cpuMs, memKiB)
			for id := 1; id <= len(task.Subtasks); id++ {
				full := subtaskFullySolved(eval.Tests, task.Subtasks[id-1].TestIDs)
				if declaredSet[id] && !full {
					allDeclaredSolved = false
				}
				if !declaredSet[id] && full {
					anyUndeclaredSolved = true
				}
			}
		}

		if lower == nil && allDeclaredSolved {
			value := limit
			lower = &value
		}
		if upper == nil && len(undeclared) > 0 && anyUndeclaredSolved {
			value := limit
			upper = &value
		}
	}

	bounds := ViableBounds{LowerKind: BoundUnknown, UpperKind: BoundUnknown}
	if lower != nil {
		bounds.Lower = lower
		bounds.LowerKind = BoundFinite
	}
	if len(undeclared) == 0 {
		bounds.UpperKind = BoundOpen
	} else if upper != nil {
		bounds.Upper = upper
		bounds.UpperKind = BoundFinite
	}
	return bounds
}

// buildBinaryBounds handles a task without subtasks. Its only scoring unit is
// "all tests pass", so the viable lower bound is the first limit where every
// run gets the full score. There is no undeclared unit, so the upper bound is
// open.
func buildBinaryBounds(task tasksrvc.Task, runs []ProfileRun, resource Resource) ViableBounds {
	maxLimit := ProfileCpuMs
	if resource == ResourceMemory {
		maxLimit = ProfileMemKiB
	}

	limits := []int{0, maxLimit}
	for _, run := range runs {
		for _, test := range run.Tests {
			if test.Percent != 100 {
				continue
			}
			value := test.CpuMs
			if resource == ResourceMemory {
				value = test.MemKiB
			}
			if value < maxLimit {
				limits = append(limits, value, value+1)
			}
		}
	}
	sort.Ints(limits)
	limits = slices.Compact(limits)

	expected := ProfilePossiblePoints(task)
	var lower *int
	for _, limit := range limits {
		cpuMs, memKiB := ProfileCpuMs, ProfileMemKiB
		if resource == ResourceCPU {
			cpuMs = limit
		} else {
			memKiB = limit
		}

		allFull := true
		for _, run := range runs {
			if profileScoreAt(task, run, cpuMs, memKiB) != expected {
				allFull = false
				break
			}
		}
		if allFull {
			value := limit
			lower = &value
			break
		}
	}

	bounds := ViableBounds{LowerKind: BoundUnknown, UpperKind: BoundOpen}
	if lower != nil {
		bounds.Lower = lower
		bounds.LowerKind = BoundFinite
	}
	return bounds
}

func buildBinaryVerdict(
	task tasksrvc.Task, runs []ProfileRun, verdict Verdict, cpuMs, memKiB int,
) Verdict {
	verdict.Applicable = true
	scores := make([]int, len(runs))
	for i, run := range runs {
		scores[i] = profileScoreAt(task, run, cpuMs, memKiB)
	}
	sort.Ints(scores)
	verdict.Score = ProfileScoreDistribution{
		Min:    scores[0],
		Median: scores[len(scores)/2],
		Max:    scores[len(scores)-1],
	}
	verdict.Delta = verdict.Score.Median - verdict.ExpectedPoints
	verdict.Matches = true
	for _, score := range scores {
		if score != verdict.ExpectedPoints {
			verdict.Matches = false
			break
		}
	}
	return verdict
}

// BuildVerdict evaluates the solution at the given limits. Declared subtasks
// must be solved in every repetition; an undeclared subtask is a risk if it is
// solved in any repetition.
func BuildVerdict(task tasksrvc.Task, runs []ProfileRun, declared []int, cpuMs, memKiB int) Verdict {
	verdict := Verdict{
		ExpectedPoints: ProfileExpectedPoints(task, solutionWithSubtasks(declared)),
		PossiblePoints: ProfilePossiblePoints(task),
		Subtasks:       []SubtaskOutcome{},
	}
	if len(runs) == 0 {
		return verdict
	}
	if len(task.Subtasks) == 0 {
		return buildBinaryVerdict(task, runs, verdict, cpuMs, memKiB)
	}
	if len(declared) == 0 {
		return verdict
	}
	verdict.Applicable = true

	declaredSet := make(map[int]bool, len(declared))
	for _, id := range declared {
		declaredSet[id] = true
	}
	points := SubtaskPoints(task)
	n := len(task.Subtasks)
	allFull := make([]bool, n)
	anyFull := make([]bool, n)
	earned := make([][]int, n)
	for i := range allFull {
		allFull[i] = true
		earned[i] = make([]int, 0, len(runs))
	}

	scores := make([]int, len(runs))
	for ri, run := range runs {
		eval := evalFromRun(task, run, cpuMs, memKiB)
		scores[ri] = eval.CalculateScore().ReceivedScore
		for i, subtask := range task.Subtasks {
			full := subtaskFullySolved(eval.Tests, subtask.TestIDs)
			allFull[i] = allFull[i] && full
			anyFull[i] = anyFull[i] || full
			earned[i] = append(earned[i], subtaskEarnedTenths(task, eval, i+1))
		}
	}
	sort.Ints(scores)
	verdict.Score = ProfileScoreDistribution{
		Min:    scores[0],
		Median: scores[len(scores)/2],
		Max:    scores[len(scores)-1],
	}
	verdict.Delta = verdict.Score.Median - verdict.ExpectedPoints
	verdict.Matches = true

	for i := range task.Subtasks {
		id := i + 1
		declaredSubtask := declaredSet[id]
		if declaredSubtask && !allFull[i] {
			verdict.Matches = false
		}
		if !declaredSubtask && anyFull[i] {
			verdict.Matches = false
		}
		sort.Ints(earned[i])
		verdict.Subtasks = append(verdict.Subtasks, SubtaskOutcome{
			SubtaskID:          id,
			Points:             points[id],
			Declared:           declaredSubtask,
			FullySolvedAllRuns: allFull[i],
			FullySolvedAnyRun:  anyFull[i],
			EarnedMinTenths:    earned[i][0],
			EarnedMedianTenths: earned[i][len(earned[i])/2],
			EarnedMaxTenths:    earned[i][len(earned[i])-1],
		})
	}
	return verdict
}

// BuildCalibration intersects the per-solution intervals and checks the current
// task limits against the padded intersection.
func BuildCalibration(
	task tasksrvc.Task,
	currentCPU, currentMem int,
	solutions []SolutionCalibration,
	unclassified, missingProfiles, staleProfiles []string,
) Calibration {
	if unclassified == nil {
		unclassified = []string{}
	}
	if missingProfiles == nil {
		missingProfiles = []string{}
	}
	if staleProfiles == nil {
		staleProfiles = []string{}
	}
	calibration := Calibration{
		Solutions:       solutions,
		Unclassified:    unclassified,
		MissingProfiles: missingProfiles,
		StaleProfiles:   staleProfiles,
	}
	for _, solution := range solutions {
		if solution.ProfileID == nil || solution.Stale || !solution.Verdict.Applicable {
			continue
		}
		if solution.Verdict.Matches {
			calibration.CurrentMatches++
		} else {
			calibration.CurrentMismatches++
		}
	}
	calibration.CPU = buildAxis(currentCPU, solutions, ResourceCPU)
	calibration.Mem = buildAxis(currentMem, solutions, ResourceMemory)
	return calibration
}

func buildAxis(current int, solutions []SolutionCalibration, resource Resource) AxisCalibration {
	axis := AxisCalibration{
		LowerKind:       BoundNone,
		UpperKind:       BoundNone,
		Current:         current,
		CurrentStatus:   "none",
		RelativePadding: CPURelativePadding,
		AbsolutePadding: CPUAbsolutePaddingMs,
	}
	if resource == ResourceMemory {
		axis.RelativePadding = MemRelativePadding
		axis.AbsolutePadding = MemAbsolutePaddingKiB
	}

	var finiteLowers, finiteUppers []int
	unknownLower, unknownUpper, anyOpen := false, false, false
	for _, solution := range solutions {
		if solution.ProfileID == nil || solution.Stale {
			continue
		}
		if len(solution.ExpectedSubtasks) == 0 && solution.ExpectedPoints == 0 {
			continue
		}
		bounds := solution.CPU
		if resource == ResourceMemory {
			bounds = solution.Mem
		}
		switch bounds.LowerKind {
		case BoundFinite:
			if bounds.Lower != nil {
				finiteLowers = append(finiteLowers, *bounds.Lower)
			}
		case BoundUnknown:
			unknownLower = true
		}
		switch bounds.UpperKind {
		case BoundFinite:
			if bounds.Upper != nil {
				finiteUppers = append(finiteUppers, *bounds.Upper)
			}
		case BoundUnknown:
			unknownUpper = true
		case BoundOpen:
			anyOpen = true
		}
	}

	if len(finiteLowers) == 0 && !unknownLower {
		return axis
	}
	if len(finiteLowers) > 0 {
		lower := slices.Max(finiteLowers)
		axis.Lower = &lower
	}
	if unknownLower {
		axis.LowerKind = BoundUnknown
	} else {
		axis.LowerKind = BoundFinite
	}

	if len(finiteUppers) > 0 {
		upper := slices.Min(finiteUppers)
		axis.Upper = &upper
		axis.UpperKind = BoundFinite
	} else if unknownUpper {
		axis.UpperKind = BoundUnknown
	} else if anyOpen {
		axis.UpperKind = BoundOpen
	}

	if axis.LowerKind == BoundFinite && axis.Lower != nil {
		padded := *axis.Lower + paddingMargin(*axis.Lower, axis.RelativePadding, axis.AbsolutePadding)
		axis.PaddedLower = &padded
	}
	if axis.UpperKind == BoundFinite && axis.Upper != nil {
		padded := *axis.Upper - paddingMargin(*axis.Upper, axis.RelativePadding, axis.AbsolutePadding)
		axis.PaddedUpper = &padded
	}

	if axis.PaddedLower != nil && axis.PaddedUpper != nil && *axis.PaddedLower > *axis.PaddedUpper {
		axis.CurrentStatus = "empty"
		return axis
	}

	switch {
	case axis.PaddedLower != nil && current < *axis.PaddedLower:
		axis.CurrentStatus = "too_low"
	case axis.LowerKind == BoundUnknown && axis.Lower != nil && current < *axis.Lower:
		axis.CurrentStatus = "too_low"
	case axis.PaddedUpper != nil && current > *axis.PaddedUpper:
		axis.CurrentStatus = "too_high"
	case axis.LowerKind == BoundUnknown || axis.UpperKind == BoundUnknown:
		axis.CurrentStatus = "unknown"
	default:
		axis.CurrentStatus = "ok"
	}

	if axis.PaddedLower != nil {
		margin := current - *axis.PaddedLower
		axis.LowerMargin = &margin
	}
	if axis.PaddedUpper != nil {
		margin := *axis.PaddedUpper - current
		axis.UpperMargin = &margin
	}
	return axis
}

func paddingMargin(value int, relative float64, absolute int) int {
	relativeMargin := int(math.Ceil(float64(value) * relative))
	if relativeMargin < absolute {
		return absolute
	}
	return relativeMargin
}

// SubtaskPoints returns the maximum points of each 1-based subtask. Group
// scoring stores points on groups, not on subtasks.
func SubtaskPoints(task tasksrvc.Task) map[int]int {
	points := make(map[int]int, len(task.Subtasks))
	if len(task.TestGroups) > 0 {
		for i := range task.Subtasks {
			points[i+1] = 0
		}
		for groupIndex, group := range task.TestGroups {
			subtasks := task.FindTestgroupSubtasks(groupIndex + 1)
			if len(subtasks) == 0 {
				continue
			}
			// Valid TaskZip assigns every group to exactly one subtask.
			points[subtasks[0]] += group.Points
		}
		return points
	}
	for i, subtask := range task.Subtasks {
		points[i+1] = subtask.Score
	}
	return points
}

func evalFromRun(task tasksrvc.Task, run ProfileRun, cpuMs, memKiB int) Eval {
	eval := NewEval(uuid.Nil, uuid.Nil, task)
	eval.CpuLimMs = cpuMs
	eval.MemLimKiB = memKiB

	for i := range eval.Tests {
		test := &eval.Tests[i]
		test.Reached = true
		test.Finished = true

		measured, ok := run.Test(i + 1)
		if !ok || measured.Percent == 0 {
			test.Wa = true
			continue
		}
		if measured.CpuMs >= cpuMs {
			test.Tle = true
			continue
		}
		if measured.MemKiB >= memKiB {
			test.Mle = true
			continue
		}
		test.Score = measured.Percent
		test.Ac = measured.Percent == 100
	}
	return eval
}

func subtaskFullySolved(tests []Test, testIDs []int) bool {
	if len(testIDs) == 0 {
		return false
	}
	for _, id := range testIDs {
		if id < 1 || id > len(tests) || !tests[id-1].Ac {
			return false
		}
	}
	return true
}

func subtaskEarnedTenths(task tasksrvc.Task, eval Eval, subtaskID int) int {
	partial := eval.Interactor != nil
	if eval.ScoreUnit == ScoreUnitTestGroup {
		centi := 0
		for _, group := range eval.Groups {
			if !slices.Contains(group.Subtasks, subtaskID) {
				continue
			}
			centi += unitCenti(group.Points, group.TgTests, eval.Tests, partial)
		}
		return (centi + 5) / 10
	}
	if subtaskID < 1 || subtaskID > len(eval.Subtasks) {
		return 0
	}
	subtask := eval.Subtasks[subtaskID-1]
	centi := unitCenti(subtask.Points, subtask.StTests, eval.Tests, partial)
	return (centi + 5) / 10
}

func solutionWithSubtasks(subtasks []int) tasksrvc.Solution {
	return tasksrvc.Solution{Subtasks: subtasks}
}
