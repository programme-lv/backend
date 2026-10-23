package srvc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/modules/exec"
	"github.com/programme-lv/backend/modules/plang"
	"github.com/programme-lv/backend/modules/subm/domain"
	tasksrvc "github.com/programme-lv/backend/modules/task/srvc"
)

const (
	MinProfileRuns     = 1
	MaxProfileRuns     = 5
	DefaultProfileRuns = 3

	profileRunBaseTimeout = 2 * time.Minute
	profileTestTimeout    = 12 * time.Second
	profileRunMaxTimeout  = 30 * time.Minute
)

type StartProfileParams struct {
	TaskShortID string
	Fname       string
	LangID      string // optional; inferred from the filename when empty
	Runs        int
}

type SolutionSummary struct {
	Fname          string
	LangID         string
	LangDisplay    string
	Content        string
	Subtasks       []int
	ExpectedPoints int
	LatestProfile  *domain.SolutionProfile
}

type ProfileFailure struct {
	Fname string `json:"fname"`
	Error string `json:"error"`
}

type StartAllProfilesResult struct {
	Started int              `json:"started"`
	Skipped []string         `json:"skipped"`
	Failed  []ProfileFailure `json:"failed"`
}

type ProfileRepo interface {
	SaveProfile(ctx context.Context, profile domain.SolutionProfile) error
	GetProfile(ctx context.Context, id uuid.UUID) (domain.SolutionProfile, error)
	ListProfiles(ctx context.Context, taskID, fname string, limit int) ([]domain.SolutionProfile, error)
	LatestFinishedProfile(ctx context.Context, taskID, fname string) (domain.SolutionProfile, error)
}

type ProfileService interface {
	StartSolutionProfile(ctx context.Context, p StartProfileParams) (uuid.UUID, srvcerror.E)
	GetSolutionProfile(ctx context.Context, id uuid.UUID) (domain.SolutionProfile, srvcerror.E)
	ListSolutionProfiles(ctx context.Context, taskID, fname string) ([]domain.SolutionProfile, srvcerror.E)
	ListTaskSolutions(ctx context.Context, taskID string) ([]SolutionSummary, srvcerror.E)
	GetTaskCalibration(ctx context.Context, taskID string) (domain.Calibration, srvcerror.E)
	StartAllSolutionProfiles(ctx context.Context, taskID string, runs int) (StartAllProfilesResult, srvcerror.E)
}

var _ ProfileService = &profileSrvc{}

// ErrProfileNotFoundInRepo is returned by ProfileRepo implementations when the
// profile row does not exist.
var ErrProfileNotFoundInRepo = errors.New("solution profile not found")

type profileSrvc struct {
	taskSrvc tasksrvc.TaskService
	execSrvc ExecSrvcFacade
	repo     ProfileRepo

	// Profiles compile and execute solutions for up to 10 s per test. Run one
	// at a time so calibration cannot flood the tester queue.
	sem chan struct{}

	progressMu sync.Mutex
	progress   map[uuid.UUID]domain.ProfileProgress
}

func NewProfileSrvc(taskSrvc tasksrvc.TaskService, execSrvc ExecSrvcFacade, repo ProfileRepo) *profileSrvc {
	return &profileSrvc{
		taskSrvc: taskSrvc,
		execSrvc: execSrvc,
		repo:     repo,
		sem:      make(chan struct{}, 1),
		progress: make(map[uuid.UUID]domain.ProfileProgress),
	}
}

func (s *profileSrvc) StartSolutionProfile(ctx context.Context, p StartProfileParams) (uuid.UUID, srvcerror.E) {
	log := ctxlog.FromContext(ctx).With("module", "subm", "layer", "srvc", "cmd", "start solution profile")

	task, err := s.taskSrvc.GetTask(ctx, p.TaskShortID)
	if err != nil {
		log.Error("get task", "task_id", p.TaskShortID, "error", err)
		return uuid.Nil, err
	}

	solution, ok := findSolution(task, p.Fname)
	if !ok {
		return uuid.Nil, ErrSolutionNotFound
	}

	return s.startProfile(ctx, task, solution, p.LangID, p.Runs)
}

// StartAllSolutionProfiles enqueues one profile per solution. Solutions that
// already have a running profile are skipped; the service semaphore makes the
// queued profiles execute one at a time.
func (s *profileSrvc) StartAllSolutionProfiles(ctx context.Context, taskID string, runs int) (StartAllProfilesResult, srvcerror.E) {
	log := ctxlog.FromContext(ctx).With("module", "subm", "layer", "srvc", "cmd", "start all solution profiles")

	task, err := s.taskSrvc.GetTask(ctx, taskID)
	if err != nil {
		log.Error("get task", "task_id", taskID, "error", err)
		return StartAllProfilesResult{}, err
	}

	result := StartAllProfilesResult{
		Skipped: []string{},
		Failed:  []ProfileFailure{},
	}
	for _, solution := range task.Solutions {
		profiles, listErr := s.repo.ListProfiles(ctx, taskID, solution.Fname, 1)
		if listErr != nil {
			log.Error("list profiles", "fname", solution.Fname, "error", listErr)
			return StartAllProfilesResult{}, srvcerror.InternalServerError()
		}
		if len(profiles) > 0 && profiles[0].Status == domain.ProfileRunning {
			result.Skipped = append(result.Skipped, solution.Fname)
			continue
		}

		if _, startErr := s.startProfile(ctx, task, solution, "", runs); startErr != nil {
			result.Failed = append(result.Failed, ProfileFailure{
				Fname: solution.Fname,
				Error: startErr.Error(),
			})
			continue
		}
		result.Started++
	}
	return result, nil
}

func (s *profileSrvc) startProfile(
	ctx context.Context,
	task tasksrvc.Task,
	solution tasksrvc.Solution,
	langID string,
	runs int,
) (uuid.UUID, srvcerror.E) {
	log := ctxlog.FromContext(ctx).With("module", "subm", "layer", "srvc", "cmd", "start solution profile")

	if langID == "" {
		inferred, inferErr := plang.ProgrLangIDFromFilename(solution.Fname)
		if inferErr != nil {
			log.Warn("infer programming language", "fname", solution.Fname, "error", inferErr)
			return uuid.Nil, ErrInvalidProgLang
		}
		langID = inferred
	}
	if _, langErr := plang.GetProgrLangById(langID); langErr != nil {
		return uuid.Nil, langErr
	}

	runs = clampRuns(runs)
	sum := sha256.Sum256([]byte(solution.Content))

	profile := domain.SolutionProfile{
		ID:             uuid.New(),
		TaskShortID:    task.ShortId,
		Fname:          solution.Fname,
		LangID:         langID,
		Status:         domain.ProfileRunning,
		ExpectedPoints: domain.ProfileExpectedPoints(task, solution),
		PossiblePoints: domain.ProfilePossiblePoints(task),
		RequestedRuns:  runs,
		TestsVersion:   task.TestsVersion,
		SolutionSha256: hex.EncodeToString(sum[:]),
		Runs:           []domain.ProfileRun{},
		CreatedAt:      time.Now(),
	}

	if err := s.repo.SaveProfile(ctx, profile); err != nil {
		log.Error("save profile", "profile_id", profile.ID, "error", err)
		return uuid.Nil, srvcerror.InternalServerError()
	}

	go s.runProfile(profile, task, solution.Content, runs)
	return profile.ID, nil
}

func (s *profileSrvc) GetSolutionProfile(ctx context.Context, id uuid.UUID) (domain.SolutionProfile, srvcerror.E) {
	profile, err := s.repo.GetProfile(ctx, id)
	if err != nil {
		if errors.Is(err, ErrProfileNotFoundInRepo) {
			return domain.SolutionProfile{}, ErrProfileNotFound
		}
		ctxlog.FromContext(ctx).Error("get solution profile", "profile_id", id, "error", err)
		return domain.SolutionProfile{}, srvcerror.InternalServerError()
	}

	s.progressMu.Lock()
	if progress, ok := s.progress[id]; ok {
		profile.Progress = &progress
	}
	s.progressMu.Unlock()

	return profile, nil
}

func (s *profileSrvc) ListSolutionProfiles(ctx context.Context, taskID, fname string) ([]domain.SolutionProfile, srvcerror.E) {
	profiles, err := s.repo.ListProfiles(ctx, taskID, fname, 20)
	if err != nil {
		ctxlog.FromContext(ctx).Error("list solution profiles", "task_id", taskID, "fname", fname, "error", err)
		return nil, srvcerror.InternalServerError()
	}
	return profiles, nil
}

func (s *profileSrvc) ListTaskSolutions(ctx context.Context, taskID string) ([]SolutionSummary, srvcerror.E) {
	task, err := s.taskSrvc.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	summaries := make([]SolutionSummary, 0, len(task.Solutions))
	for _, solution := range task.Solutions {
		summary := SolutionSummary{
			Fname:          solution.Fname,
			Content:        solution.Content,
			Subtasks:       solution.Subtasks,
			ExpectedPoints: domain.ProfileExpectedPoints(task, solution),
		}
		if langID, langErr := plang.ProgrLangIDFromFilename(solution.Fname); langErr == nil {
			summary.LangID = langID
			if lang, getErr := plang.GetProgrLangById(langID); getErr == nil {
				summary.LangDisplay = lang.FullName
			}
		}
		profiles, listErr := s.repo.ListProfiles(ctx, taskID, solution.Fname, 1)
		if listErr != nil {
			return nil, srvcerror.InternalServerError()
		}
		if len(profiles) > 0 {
			summary.LatestProfile = &profiles[0]
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func (s *profileSrvc) GetTaskCalibration(ctx context.Context, taskID string) (domain.Calibration, srvcerror.E) {
	task, err := s.taskSrvc.GetTask(ctx, taskID)
	if err != nil {
		return domain.Calibration{}, err
	}

	solutions := make([]domain.SolutionCalibration, 0, len(task.Solutions))
	var unclassified, missingProfiles, staleProfiles []string

	for _, solution := range task.Solutions {
		entry := domain.SolutionCalibration{
			Fname:            solution.Fname,
			ExpectedSubtasks: solution.Subtasks,
			ExpectedPoints:   domain.ProfileExpectedPoints(task, solution),
			PossiblePoints:   domain.ProfilePossiblePoints(task),
			CPU:              domain.ViableBounds{LowerKind: domain.BoundNone, UpperKind: domain.BoundNone},
			Mem:              domain.ViableBounds{LowerKind: domain.BoundNone, UpperKind: domain.BoundNone},
			Verdict:          domain.Verdict{Subtasks: []domain.SubtaskOutcome{}},
		}
		if len(solution.Subtasks) == 0 {
			unclassified = append(unclassified, solution.Fname)
			solutions = append(solutions, entry)
			continue
		}

		profile, profileErr := s.repo.LatestFinishedProfile(ctx, taskID, solution.Fname)
		if profileErr != nil {
			if errors.Is(profileErr, ErrProfileNotFoundInRepo) {
				missingProfiles = append(missingProfiles, solution.Fname)
				solutions = append(solutions, entry)
				continue
			}
			ctxlog.FromContext(ctx).Error("get latest finished profile",
				"task_id", taskID, "fname", solution.Fname, "error", profileErr)
			return domain.Calibration{}, srvcerror.InternalServerError()
		}

		sum := sha256.Sum256([]byte(solution.Content))
		entry.ProfileID = &profile.ID
		entry.Stale = profile.TestsVersion != task.TestsVersion ||
			profile.SolutionSha256 != hex.EncodeToString(sum[:])
		if entry.Stale {
			staleProfiles = append(staleProfiles, solution.Fname)
			solutions = append(solutions, entry)
			continue
		}

		entry.CPU = domain.BuildViableBounds(task, profile.Runs, domain.ResourceCPU, solution.Subtasks)
		entry.Mem = domain.BuildViableBounds(task, profile.Runs, domain.ResourceMemory, solution.Subtasks)
		entry.Verdict = domain.BuildVerdict(task, profile.Runs, solution.Subtasks, task.CpuMillis(), task.MemoryKiB())
		solutions = append(solutions, entry)
	}

	return domain.BuildCalibration(
		task,
		task.CpuMillis(),
		task.MemoryKiB(),
		solutions,
		unclassified,
		missingProfiles,
		staleProfiles,
	), nil
}

// runProfile executes the solution at the profiling limits, one repetition at
// a time, and persists the measurements after each one.
func (s *profileSrvc) runProfile(profile domain.SolutionProfile, task tasksrvc.Task, content string, runs int) {
	s.sem <- struct{}{}
	defer func() { <-s.sem }()
	defer s.clearProgress(profile.ID)

	ctx := context.Background()
	log := slog.Default().With("module", "subm", "layer", "srvc", "cmd", "run solution profile")

	eval := domain.NewEval(uuid.New(), uuid.Nil, task)
	eval.CpuLimMs = domain.ProfileCpuMs
	eval.MemLimKiB = domain.ProfileMemKiB
	tests := constructExecEnqueueTests(ctx, eval, s.taskSrvc.GetTestDownlUrl)

	for i := 0; i < runs; i++ {
		s.setProgress(profile.ID, domain.ProfileProgress{
			Run:        i + 1,
			Runs:       runs,
			TestsTotal: len(task.Tests),
		})

		execUUID := uuid.New()
		enqueueErr := s.execSrvc.Enqueue(ctx, execUUID, content, profile.LangID, tests, exec.TestingParams{
			CpuMs:      domain.ProfileCpuMs,
			MemKiB:     domain.ProfileMemKiB,
			Checker:    eval.Checker,
			Interactor: eval.Interactor,
			// No groups: the tester must run every test, even after failures.
		})
		if enqueueErr != nil {
			s.fail(ctx, profile, fmt.Errorf("enqueue execution: %w", enqueueErr))
			return
		}

		// The listener is best-effort: if the execution finished before we
		// subscribed, Get below still returns the stored result.
		if progressCh, listenErr := s.execSrvc.Listen(ctx, execUUID); listenErr == nil {
			go s.trackProgress(profile.ID, i+1, progressCh)
		} else {
			log.Debug("listen for profile progress", "profile_id", profile.ID, "error", listenErr)
		}

		// Bound each repetition. Without this a missing tester leaves the
		// profile running forever because NATS silently drops the job.
		runCtx, cancel := context.WithTimeout(ctx, profileTimeout(len(task.Tests)))
		execution, getErr := s.execSrvc.Get(runCtx, execUUID)
		cancel()
		if getErr != nil {
			s.fail(ctx, profile, fmt.Errorf("get execution: %w", getErr))
			return
		}

		run, runErr := profileRunFromExec(task, execution)
		if runErr != nil {
			s.fail(ctx, profile, runErr)
			return
		}

		profile.Runs = append(profile.Runs, run)
		if saveErr := s.repo.SaveProfile(ctx, profile); saveErr != nil {
			log.Error("save profile progress", "profile_id", profile.ID, "error", saveErr)
			return
		}
	}

	finishedAt := time.Now()
	profile.Status = domain.ProfileFinished
	profile.FinishedAt = &finishedAt
	if saveErr := s.repo.SaveProfile(ctx, profile); saveErr != nil {
		log.Error("finish profile", "profile_id", profile.ID, "error", saveErr)
	}
}

func (s *profileSrvc) setProgress(id uuid.UUID, progress domain.ProfileProgress) {
	s.progressMu.Lock()
	s.progress[id] = progress
	s.progressMu.Unlock()
}

func (s *profileSrvc) clearProgress(id uuid.UUID) {
	s.progressMu.Lock()
	delete(s.progress, id)
	s.progressMu.Unlock()
}

func (s *profileSrvc) trackProgress(id uuid.UUID, run int, events <-chan exec.Event) {
	for event := range events {
		switch event.(type) {
		case exec.FinishedTest, exec.IgnoredTest:
			s.progressMu.Lock()
			progress := s.progress[id]
			// Ignore stragglers from the previous repetition.
			if progress.Run == run {
				progress.TestsDone++
				s.progress[id] = progress
			}
			s.progressMu.Unlock()
		}
	}
}

func (s *profileSrvc) fail(ctx context.Context, profile domain.SolutionProfile, cause error) {
	log := ctxlog.FromContext(ctx).With("module", "subm", "layer", "srvc", "cmd", "run solution profile")
	log.Error("solution profile failed", "profile_id", profile.ID, "error", cause)

	message := cause.Error()
	finishedAt := time.Now()
	profile.Status = domain.ProfileFailed
	profile.Error = &message
	profile.FinishedAt = &finishedAt
	if err := s.repo.SaveProfile(ctx, profile); err != nil {
		log.Error("save failed profile", "profile_id", profile.ID, "error", err)
	}
}

// profileRunFromExec converts tester measurements into a profile run using the
// same verdict mapping as a real submission.
func profileRunFromExec(task tasksrvc.Task, execution exec.Execution) (domain.ProfileRun, error) {
	if execution.Stage == exec.StageCompileError || execution.Stage == exec.StageInternalError {
		message := "execution failed"
		if execution.ErrorMsg != nil {
			message = *execution.ErrorMsg
		}
		return domain.ProfileRun{}, fmt.Errorf("%s", message)
	}
	if execution.Stage != exec.StageFinished {
		return domain.ProfileRun{}, fmt.Errorf("execution stopped in stage %q", execution.Stage)
	}

	eval := domain.NewEval(uuid.Nil, uuid.Nil, task)
	eval.CpuLimMs = domain.ProfileCpuMs
	eval.MemLimKiB = domain.ProfileMemKiB

	for _, result := range execution.TestRes {
		if result.Subm == nil || result.Ignored {
			continue
		}
		index := result.ID - 1
		if index < 0 || index >= len(eval.Tests) {
			continue
		}
		applyTestResult(&eval.Tests[index], eval, result.Subm, result.Checker)
	}

	run := domain.ProfileRun{ExecUUID: execution.UUID}
	for i, test := range eval.Tests {
		run.Tests = append(run.Tests, domain.ProfileTest{
			TestID:  i + 1,
			CpuMs:   intOrZero(test.CpuMs),
			MemKiB:  intOrZero(test.MemKiB),
			Percent: test.Percent(),
		})
	}
	return run, nil
}

func findSolution(task tasksrvc.Task, fname string) (tasksrvc.Solution, bool) {
	for _, solution := range task.Solutions {
		if solution.Fname == fname {
			return solution, true
		}
	}
	return tasksrvc.Solution{}, false
}

func profileTimeout(testCount int) time.Duration {
	timeout := profileRunBaseTimeout + time.Duration(testCount)*profileTestTimeout
	if timeout > profileRunMaxTimeout {
		return profileRunMaxTimeout
	}
	return timeout
}

func clampRuns(runs int) int {
	if runs < MinProfileRuns {
		return DefaultProfileRuns
	}
	if runs > MaxProfileRuns {
		return MaxProfileRuns
	}
	return runs
}

func intOrZero(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
