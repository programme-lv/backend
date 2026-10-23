package http

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/modules/plang"
	"github.com/programme-lv/backend/modules/subm/domain"
	submsrvc "github.com/programme-lv/backend/modules/subm/srvc"
	tasksrvc "github.com/programme-lv/backend/modules/task/srvc"
)

// SolutionView is an admin-only view of a stored reference solution.
type SolutionView struct {
	Fname          string                `json:"fname"`
	LangID         string                `json:"lang_id"`
	LangDisplay    string                `json:"lang_display"`
	Content        string                `json:"content"`
	Subtasks       []int                 `json:"subtasks"`
	ExpectedPoints int                   `json:"expected_points"`
	LatestProfile  *SolutionProfileBrief `json:"latest_profile,omitempty"`
}

type SolutionProfileBrief struct {
	ID             uuid.UUID `json:"id"`
	Status         string    `json:"status"`
	ExpectedPoints int       `json:"expected_points"`
	CreatedAt      string    `json:"created_at"`
	FinishedAt     *string   `json:"finished_at,omitempty"`
}

type StartSolutionProfileReq struct {
	Runs   int    `json:"runs"`
	LangID string `json:"lang_id"`
}

type StartAllSolutionProfilesReq struct {
	Runs int    `json:"runs"`
	Only string `json:"only"` // "", "stale", "mismatched"
}

type StartSolutionProfileResp struct {
	ProfileID uuid.UUID `json:"profile_id"`
}

type ProfileRunView struct {
	ExecUUID  uuid.UUID            `json:"exec_uuid"`
	Score     int                  `json:"score"`
	MaxCpuMs  int                  `json:"max_cpu_ms"`
	MaxMemKiB int                  `json:"max_mem_kib"`
	Tests     []domain.ProfileTest `json:"tests"`
}

type SolutionProfileView struct {
	ID              uuid.UUID                       `json:"id"`
	TaskID          string                          `json:"task_id"`
	Fname           string                          `json:"fname"`
	LangID          string                          `json:"lang_id"`
	LangDisplay     string                          `json:"lang_display"`
	Status          string                          `json:"status"`
	Error           *string                         `json:"error,omitempty"`
	ExpectedPoints  int                             `json:"expected_points"`
	PossiblePoints  int                             `json:"possible_points"`
	RequestedRuns   int                             `json:"requested_runs"`
	Progress        *domain.ProfileProgress         `json:"progress,omitempty"`
	TestsVersion    int                             `json:"tests_version"`
	SolutionStale   bool                            `json:"solution_stale"`
	CurrentCpuMs    int                             `json:"current_cpu_ms"`
	CurrentMemKiB   int                             `json:"current_mem_kib"`
	CurrentScore    domain.ProfileScoreDistribution `json:"current_score"`
	CPUViable       domain.ViableBounds             `json:"cpu_viable"`
	MemViable       domain.ViableBounds             `json:"mem_viable"`
	Verdict         domain.Verdict                  `json:"verdict"`
	ProfilingCpuMs  int                             `json:"profiling_cpu_ms"`
	ProfilingMemKiB int                             `json:"profiling_mem_kib"`
	CreatedAt       string                          `json:"created_at"`
	FinishedAt      *string                         `json:"finished_at,omitempty"`
	Runs            []ProfileRunView                `json:"runs"`
	Cpu             domain.LimitChart               `json:"cpu"`
	Mem             domain.LimitChart               `json:"mem"`
}

func (h *SubmHttpHandler) ListTaskSolutions(ctx context.Context) ([]SolutionView, jsonresp.HttpStatusCoder) {
	taskID := chi.URLParamFromCtx(ctx, "taskId")

	solutions, err := h.profileSrvc.ListTaskSolutions(ctx, taskID)
	if err != nil {
		return nil, err
	}

	views := make([]SolutionView, 0, len(solutions))
	for _, solution := range solutions {
		view := SolutionView{
			Fname:          solution.Fname,
			LangID:         solution.LangID,
			LangDisplay:    solution.LangDisplay,
			Content:        solution.Content,
			Subtasks:       solution.Subtasks,
			ExpectedPoints: solution.ExpectedPoints,
		}
		if solution.LatestProfile != nil {
			view.LatestProfile = mapProfileBrief(*solution.LatestProfile)
		}
		views = append(views, view)
	}
	return views, nil
}

func (h *SubmHttpHandler) StartSolutionProfile(ctx context.Context, req StartSolutionProfileReq) (StartSolutionProfileResp, jsonresp.HttpStatusCoder) {
	taskID := chi.URLParamFromCtx(ctx, "taskId")
	fname := chi.URLParamFromCtx(ctx, "fname")

	profileID, err := h.profileSrvc.StartSolutionProfile(ctx, submsrvc.StartProfileParams{
		TaskShortID: taskID,
		Fname:       fname,
		LangID:      req.LangID,
		Runs:        req.Runs,
	})
	if err != nil {
		return StartSolutionProfileResp{}, err
	}
	return StartSolutionProfileResp{ProfileID: profileID}, nil
}

func (h *SubmHttpHandler) StartAllSolutionProfiles(ctx context.Context, req StartAllSolutionProfilesReq) (submsrvc.StartAllProfilesResult, jsonresp.HttpStatusCoder) {
	taskID := chi.URLParamFromCtx(ctx, "taskId")
	result, err := h.profileSrvc.StartAllSolutionProfiles(ctx, taskID, req.Runs, req.Only)
	if err != nil {
		return submsrvc.StartAllProfilesResult{}, err
	}
	return result, nil
}

func (h *SubmHttpHandler) ListSolutionProfiles(ctx context.Context) ([]SolutionProfileBrief, jsonresp.HttpStatusCoder) {
	taskID := chi.URLParamFromCtx(ctx, "taskId")
	fname := chi.URLParamFromCtx(ctx, "fname")

	profiles, err := h.profileSrvc.ListSolutionProfiles(ctx, taskID, fname)
	if err != nil {
		return nil, err
	}

	views := make([]SolutionProfileBrief, 0, len(profiles))
	for _, profile := range profiles {
		views = append(views, *mapProfileBrief(profile))
	}
	return views, nil
}

func (h *SubmHttpHandler) GetTaskCalibration(ctx context.Context) (domain.Calibration, jsonresp.HttpStatusCoder) {
	taskID := chi.URLParamFromCtx(ctx, "taskId")
	calibration, err := h.profileSrvc.GetTaskCalibration(ctx, taskID)
	if err != nil {
		return domain.Calibration{}, err
	}
	return calibration, nil
}

func (h *SubmHttpHandler) GetSolutionProfile(ctx context.Context) (SolutionProfileView, jsonresp.HttpStatusCoder) {
	profileID, parseErr := uuid.Parse(chi.URLParamFromCtx(ctx, "profileId"))
	if parseErr != nil {
		return SolutionProfileView{}, jsonresp.ErrHttpBadRequest.WithMsg("nederīgs profila ID")
	}

	profile, err := h.profileSrvc.GetSolutionProfile(ctx, profileID)
	if err != nil {
		return SolutionProfileView{}, err
	}

	task, taskErr := h.taskSrvc.GetTask(ctx, profile.TaskShortID)
	if taskErr != nil {
		return SolutionProfileView{}, taskErr
	}

	view := SolutionProfileView{
		ID:              profile.ID,
		TaskID:          profile.TaskShortID,
		Fname:           profile.Fname,
		LangID:          profile.LangID,
		Status:          string(profile.Status),
		Error:           profile.Error,
		ExpectedPoints:  profile.ExpectedPoints,
		PossiblePoints:  profile.PossiblePoints,
		RequestedRuns:   profile.RequestedRuns,
		Progress:        profile.Progress,
		TestsVersion:    profile.TestsVersion,
		CurrentCpuMs:    task.CpuMillis(),
		CurrentMemKiB:   task.MemoryKiB(),
		ProfilingCpuMs:  domain.ProfileCpuMs,
		ProfilingMemKiB: domain.ProfileMemKiB,
		CreatedAt:       profile.CreatedAt.Format(time.RFC3339),
		Runs:            make([]ProfileRunView, 0, len(profile.Runs)),
	}

	if lang, langErr := plang.GetProgrLangById(profile.LangID); langErr == nil {
		view.LangDisplay = lang.FullName
	}

	if solution, ok := findSolution(task, profile.Fname); ok {
		sum := sha256.Sum256([]byte(solution.Content))
		view.SolutionStale = hex.EncodeToString(sum[:]) != profile.SolutionSha256
		view.CPUViable = domain.BuildViableBounds(task, profile.Runs, domain.ResourceCPU, solution.Subtasks)
		view.MemViable = domain.BuildViableBounds(task, profile.Runs, domain.ResourceMemory, solution.Subtasks)
		view.Verdict = domain.BuildVerdict(
			task, profile.Runs, solution.Subtasks, task.CpuMillis(), task.MemoryKiB(),
		)
	}

	if profile.FinishedAt != nil {
		finishedAt := profile.FinishedAt.Format(time.RFC3339)
		view.FinishedAt = &finishedAt
	}

	for _, run := range profile.Runs {
		runView := ProfileRunView{
			ExecUUID: run.ExecUUID,
			Score:    domain.ProfileRunScore(task, run),
			Tests:    run.Tests,
		}
		for _, test := range run.Tests {
			if test.CpuMs > runView.MaxCpuMs {
				runView.MaxCpuMs = test.CpuMs
			}
			if test.MemKiB > runView.MaxMemKiB {
				runView.MaxMemKiB = test.MemKiB
			}
		}
		view.Runs = append(view.Runs, runView)
	}

	view.Cpu = domain.BuildProfileChart(task, profile.Runs, domain.ResourceCPU, profile.ExpectedPoints)
	view.Mem = domain.BuildProfileChart(task, profile.Runs, domain.ResourceMemory, profile.ExpectedPoints)
	view.CurrentScore = scoreRangeAt(task, profile.Runs, task.CpuMillis(), task.MemoryKiB())

	return view, nil
}

func scoreRangeAt(task tasksrvc.Task, runs []domain.ProfileRun, cpuMs, memKiB int) domain.ProfileScoreDistribution {
	if len(runs) == 0 {
		return domain.ProfileScoreDistribution{}
	}
	scores := make([]int, len(runs))
	for i, run := range runs {
		scores[i] = domain.ProfileScoreAt(task, run, cpuMs, memKiB)
	}
	sort.Ints(scores)
	return domain.ProfileScoreDistribution{
		Min:    scores[0],
		Median: scores[len(scores)/2],
		Max:    scores[len(scores)-1],
	}
}

func findSolution(task tasksrvc.Task, fname string) (tasksrvc.Solution, bool) {
	for _, solution := range task.Solutions {
		if solution.Fname == fname {
			return solution, true
		}
	}
	return tasksrvc.Solution{}, false
}

func mapProfileBrief(profile domain.SolutionProfile) *SolutionProfileBrief {
	brief := &SolutionProfileBrief{
		ID:             profile.ID,
		Status:         string(profile.Status),
		ExpectedPoints: profile.ExpectedPoints,
		CreatedAt:      profile.CreatedAt.Format(time.RFC3339),
	}
	if profile.FinishedAt != nil {
		finishedAt := profile.FinishedAt.Format(time.RFC3339)
		brief.FinishedAt = &finishedAt
	}
	return brief
}
