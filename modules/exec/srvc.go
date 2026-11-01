package exec

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/nats-io/nats.go"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	testerapi "github.com/programme-lv/tester/api"
)

type CodeExecutionService interface {
	Enqueue(ctx context.Context, uuid uuid.UUID, srcCode string, prLangId string, tests []TestFile, params TestingParams) srvcerror.E
	Listen(ctx context.Context, uuid uuid.UUID) (<-chan Event, srvcerror.E)
	Get(ctx context.Context, execUuid uuid.UUID) (Execution, srvcerror.E)
}

var _ CodeExecutionService = &execSrvc{}

// ExecRepo interface for execution storage
type ExecRepo interface {
	Save(ctx context.Context, exec *Execution) error
	Get(ctx context.Context, id uuid.UUID) (*Execution, error)
}

const NatsSubject = "tester.jobs"

// noRespondersStatus is the NATS status header value sent to the job's reply
// subject when no tester is subscribed to NatsSubject. Core NATS does not keep
// the job, so it must be published again.
const noRespondersStatus = "503"

// User-facing reasons for ending an execution without a tester result.
const (
	testerUnavailableMsg = "testētājs nav pieejams"
	testerTimeoutMsg     = "testēšanas laiks pārsniegts"
)

// execSrvc handles communication with testers
// for code execution and result streaming
type execSrvc struct {
	logger *slog.Logger

	// either in-mem or s3
	execRepo ExecRepo

	natsConn      *nats.Conn
	natsInbox     string
	fileSubject   string
	testfileStore TestFileStore
	publishJob    func(*nats.Msg) error

	mu sync.Mutex
	// maps exec IDs to client result channels
	notifiers map[uuid.UUID]chan Event
	// tracks completion status of executions
	execWg sync.Map // notifies get listener when execution is finished

	isPolling atomic.Bool

	organizers map[uuid.UUID]*ExecResStreamOrganizer
	executions map[uuid.UUID]*Execution
	fileHashes map[uuid.UUID]map[string]struct{}

	// jobMsgs keeps the published job so it can be published again when no
	// tester was subscribed at publish time.
	jobMsgs map[uuid.UUID]*nats.Msg
	// lastEventAt is the last publish attempt or received event. It drives the
	// watchdog that detects jobs lost after a successful publish.
	lastEventAt map[uuid.UUID]time.Time
	// publishTries counts the publish attempts for the current execution.
	publishTries map[uuid.UUID]int
	// eventsSeen tells whether the tester has started sending results.
	eventsSeen map[uuid.UUID]bool

	// Timeouts and retry policy, configurable in tests.
	firstEventTimeout  time.Duration
	inactivityTimeout  time.Duration
	publishRetryDelay  time.Duration
	maxPublishAttempts int
	watchdogInterval   time.Duration
}

func (e *execSrvc) StartPollingResultQueue(ctx context.Context) error {
	notAlreadyPolling := e.isPolling.CompareAndSwap(false, true)
	if !notAlreadyPolling {
		return fmt.Errorf("already polling result queue")
	}

	fileSub, err := e.natsConn.Subscribe(e.fileSubject, func(msg *nats.Msg) {
		go e.handleFileRequest(msg)
	})
	if err != nil {
		e.isPolling.Store(false)
		return fmt.Errorf("subscribe to test file subject: %w", err)
	}

	// Every execution has its own reply subject below this prefix. That lets
	// the no-responders status message be attributed to the execution whose
	// job no tester received.
	resultSub, err := e.natsConn.Subscribe(e.natsInbox+".>", func(msg *nats.Msg) {
		e.handleResultMessage(ctx, msg)
	})
	if err != nil {
		_ = fileSub.Unsubscribe()
		e.isPolling.Store(false)
		return fmt.Errorf("subscribe to nats inbox: %w", err)
	}
	go func() {
		<-ctx.Done()
		_ = resultSub.Unsubscribe()
		_ = fileSub.Unsubscribe()
		e.isPolling.Store(false)
	}()
	go e.watchExecutions(ctx)

	return nil
}

func (e *execSrvc) handleResultMessage(ctx context.Context, msg *nats.Msg) {
	if msg.Header.Get("Status") == noRespondersStatus {
		e.handleNoResponders(ctx, msg.Subject)
		return
	}

	var header testerapi.Header
	if err := json.Unmarshal(msg.Data, &header); err != nil {
		e.logger.Error("unsmarshall NATS msg header", "error", err)
		return
	}
	execUUID, err := uuid.Parse(header.EvalUuid)
	if err != nil {
		e.logger.Error("parse eval uuid", "error", err)
		return
	}
	event, err := mapTesterMsgJsonToEvent(msg.Data, header.MsgType)
	if err != nil {
		e.logger.Error("map tester msg json to event", "error", err)
		return
	}

	e.dispatchEvent(ctx, execUUID, event)
}

// dispatchEvent applies an event to an execution and streams the ordered
// result to the submission or profile listener.
func (e *execSrvc) dispatchEvent(ctx context.Context, execUUID uuid.UUID, event Event) {
	e.mu.Lock()
	org, exists := e.organizers[execUUID]
	if !exists || org == nil || org.HasFinished() {
		e.mu.Unlock()
		e.logger.Error("stream organizer unavailable", "exec_uuid", execUUID)
		return
	}
	events, err := org.Add(event)
	if err != nil {
		e.mu.Unlock()
		e.logger.Error("add event to stream organizer", "error", err)
		return
	}
	e.lastEventAt[execUUID] = time.Now()
	e.eventsSeen[execUUID] = true
	execution := e.executions[execUUID]
	notifier := e.notifiers[execUUID]
	if execution == nil || notifier == nil {
		e.mu.Unlock()
		e.logger.Error("execution state unavailable", "exec_uuid", execUUID)
		return
	}
	for _, event := range events {
		if err := applyEventToExec(execution, event); err != nil {
			e.logger.Error("apply ev to exec", "error", err)
		}
	}
	finished := org.HasFinished()
	if finished {
		delete(e.notifiers, execUUID)
		delete(e.organizers, execUUID)
		delete(e.executions, execUUID)
		delete(e.fileHashes, execUUID)
		delete(e.jobMsgs, execUUID)
		delete(e.lastEventAt, execUUID)
		delete(e.publishTries, execUUID)
		delete(e.eventsSeen, execUUID)
	}
	e.mu.Unlock()

	for _, event := range events {
		notifier <- event
	}
	if !finished {
		return
	}
	close(notifier)
	if err := e.execRepo.Save(ctx, execution); err != nil {
		e.logger.Error("save exec", "error", err)
		return
	}
	wgVal, exists := e.execWg.Load(execUUID)
	if !exists {
		e.logger.Error("wait group not found", "exec_uuid", execUUID)
		return
	}
	wgVal.(*sync.WaitGroup).Done()
}

// handleNoResponders handles the NATS status message sent when a job was
// published while no tester was subscribed. The job is gone; publish it again
// or end the execution with an error.
func (e *execSrvc) handleNoResponders(ctx context.Context, subject string) {
	execUUID, ok := evalUUIDFromSubject(subject)
	if !ok {
		e.logger.Error("no responders for unknown reply subject", "subject", subject)
		return
	}
	e.logger.Warn("no tester received execution job", "exec_uuid", execUUID)
	e.republishOrFail(ctx, execUUID, testerUnavailableMsg)
}

// republishOrFail publishes the job again after a backoff, or ends the
// execution with an internal error once the attempts are exhausted.
func (e *execSrvc) republishOrFail(ctx context.Context, execUUID uuid.UUID, reason string) {
	e.mu.Lock()
	msg, active := e.jobMsgs[execUUID]
	if !active {
		e.mu.Unlock()
		return
	}
	tries := e.publishTries[execUUID]
	if tries < 1 {
		tries = 1
	}
	if tries >= e.maxPublishAttempts {
		e.mu.Unlock()
		e.failExecutionUnavailable(ctx, execUUID, reason)
		return
	}
	e.publishTries[execUUID] = tries + 1
	e.lastEventAt[execUUID] = time.Now()
	e.mu.Unlock()

	delay := e.publishRetryDelay * time.Duration(1<<(tries-1))
	e.logger.Warn("republish execution job",
		"exec_uuid", execUUID,
		"attempt", tries+1,
		"delay", delay,
		"reason", reason,
	)
	time.AfterFunc(delay, func() {
		e.mu.Lock()
		_, active := e.jobMsgs[execUUID]
		e.mu.Unlock()
		if !active {
			return
		}
		if err := e.publishJob(msg); err != nil {
			e.logger.Error("republish execution job", "exec_uuid", execUUID, "error", err)
		}
	})
}

// failExecutionUnavailable ends an execution that no tester accepted. The
// submission keeps a distinct tester_unavailable error so it can be told
// apart from a tester that ran the job and failed.
func (e *execSrvc) failExecutionUnavailable(ctx context.Context, execUUID uuid.UUID, reason string) {
	e.logger.Error("end execution without a tester", "exec_uuid", execUUID, "reason", reason)
	e.dispatchEvent(ctx, execUUID, TesterUnavailable{ErrorMsg: &reason})
}

// failExecutionInternal ends an execution that a tester started but did not
// finish, for example after it stopped streaming events.
func (e *execSrvc) failExecutionInternal(ctx context.Context, execUUID uuid.UUID, reason string) {
	e.logger.Error("end stalled execution", "exec_uuid", execUUID, "reason", reason)
	e.dispatchEvent(ctx, execUUID, InternalServerError{ErrorMsg: &reason})
}

// watchExecutions detects jobs that were published but never produced an
// event, and executions that stopped streaming events mid-run.
func (e *execSrvc) watchExecutions(ctx context.Context) {
	ticker := time.NewTicker(e.watchdogInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			e.sweepExecutions(ctx, now)
		}
	}
}

func (e *execSrvc) sweepExecutions(ctx context.Context, now time.Time) {
	var retry []uuid.UUID
	type stalledExecution struct {
		uuid    uuid.UUID
		started bool
	}
	var fail []stalledExecution

	e.mu.Lock()
	for execUUID, lastEventAt := range e.lastEventAt {
		if _, active := e.jobMsgs[execUUID]; !active {
			continue
		}
		started := e.eventsSeen[execUUID]
		timeout := e.firstEventTimeout
		if started {
			timeout = e.inactivityTimeout
		}
		if now.Sub(lastEventAt) < timeout {
			continue
		}
		if !started && e.publishTries[execUUID] < e.maxPublishAttempts {
			retry = append(retry, execUUID)
			continue
		}
		fail = append(fail, stalledExecution{uuid: execUUID, started: started})
	}
	e.mu.Unlock()

	for _, execUUID := range retry {
		e.republishOrFail(ctx, execUUID, testerUnavailableMsg)
	}
	for _, execution := range fail {
		if execution.started {
			e.failExecutionInternal(ctx, execution.uuid, testerTimeoutMsg)
			continue
		}
		e.failExecutionUnavailable(ctx, execution.uuid, testerUnavailableMsg)
	}
}

func (e *execSrvc) resultSubject(execUUID uuid.UUID) string {
	return e.natsInbox + "." + execUUID.String()
}

func evalUUIDFromSubject(subject string) (uuid.UUID, bool) {
	idx := strings.LastIndexByte(subject, '.')
	if idx < 0 || idx == len(subject)-1 {
		return uuid.Nil, false
	}
	execUUID, err := uuid.Parse(subject[idx+1:])
	if err != nil {
		return uuid.Nil, false
	}
	return execUUID, true
}

func mapTesterMsgJsonToEvent(msgJson []byte, msgType testerapi.MsgType) (Event, error) {
	switch msgType {
	case testerapi.StartJobMsg:
		var msg testerapi.StartJob
		if err := json.Unmarshal(msgJson, &msg); err != nil {
			return nil, fmt.Errorf("unmarshal StartJob: %w", err)
		}
		startedAt, err := time.Parse(time.RFC3339, msg.StartedTime)
		if err != nil {
			return nil, fmt.Errorf("parse started_time: %w", err)
		}
		return ReceivedSubmission{
			SysInfo:   msg.SystemInfo,
			StartedAt: startedAt,
		}, nil

	case testerapi.StartCompileMsg:
		return StartedCompiling{}, nil

	case testerapi.FinishCompileMsg:
		var msg testerapi.FinishCompile
		if err := json.Unmarshal(msgJson, &msg); err != nil {
			return nil, fmt.Errorf("unmarshal FinishCompile: %w", err)
		}
		return FinishedCompiling{
			RuntimeData: mapTesterRuntimeData(msg.RuntimeData),
		}, nil

	case testerapi.ReachTestMsg:
		var msg testerapi.ReachTest
		if err := json.Unmarshal(msgJson, &msg); err != nil {
			return nil, fmt.Errorf("unmarshal ReachTest: %w", err)
		}
		return ReachedTest{
			TestId: int(msg.TestId),
			In:     msg.Input,
			Ans:    msg.Answer,
		}, nil

	case testerapi.IgnoreTestMsg:
		var msg testerapi.IgnoreTest
		if err := json.Unmarshal(msgJson, &msg); err != nil {
			return nil, fmt.Errorf("unmarshal IgnoreTest: %w", err)
		}
		return IgnoredTest{
			TestId: int(msg.TestId),
		}, nil

	case testerapi.FinishTestMsg:
		var msg testerapi.FinishTest
		if err := json.Unmarshal(msgJson, &msg); err != nil {
			return nil, fmt.Errorf("unmarshal FinishTest: %w", err)
		}
		return FinishedTest{
			TestID:  int(msg.TestId),
			Subm:    mapTesterRuntimeData(msg.Submission),
			Checker: mapTesterRuntimeData(msg.Checker),
		}, nil

	case testerapi.FinishJobMsg:
		var msg testerapi.FinishJob
		if err := json.Unmarshal(msgJson, &msg); err != nil {
			return nil, fmt.Errorf("unmarshal FinishJob: %w", err)
		}
		if msg.CompileError {
			return CompilationError{
				ErrorMsg: msg.ErrorMessage,
			}, nil
		}
		if msg.InternalError {
			return InternalServerError{
				ErrorMsg: msg.ErrorMessage,
			}, nil
		}
		return FinishedTesting{}, nil

	default:
		return nil, fmt.Errorf("unknown msg type: %s", msgType)
	}
}

func mapTesterRuntimeData(rd *testerapi.RuntimeData) *RunData {
	if rd == nil {
		return nil
	}
	return &RunData{
		StdIn:       rd.Stdin,
		StdOut:      rd.Stdout,
		StdErr:      rd.Stderr,
		ExitCode:    rd.ExitCode,
		CpuMs:       rd.CpuMillis,
		WallMs:      rd.WallMillis,
		MemKiB:      rd.RamKiBytes,
		CtxSwV:      rd.CtxSwV,
		CtxSwF:      rd.CtxSwF,
		Signal:      rd.ExitSignal,
		IsOomKilled: rd.CgOomKilled,
		IsolStatus:  rd.IsolateStatus,
		IsolMsg:     rd.IsolateMsg,
	}
}

func NewExecSrvc(ctx context.Context, repo ExecRepo, natsConn *nats.Conn, testfileStore TestFileStore) *execSrvc {
	esrvc := &execSrvc{
		logger:        ctxlog.FromContext(ctx),
		natsConn:      natsConn,
		natsInbox:     nats.NewInbox(),
		fileSubject:   nats.NewInbox(),
		testfileStore: testfileStore,
		publishJob:    natsConn.PublishMsg,
		execRepo:      repo,
		notifiers:     make(map[uuid.UUID]chan Event),
		organizers:    make(map[uuid.UUID]*ExecResStreamOrganizer),
		executions:    make(map[uuid.UUID]*Execution),
		fileHashes:    make(map[uuid.UUID]map[string]struct{}),
		jobMsgs:       make(map[uuid.UUID]*nats.Msg),
		lastEventAt:   make(map[uuid.UUID]time.Time),
		publishTries:  make(map[uuid.UUID]int),
		eventsSeen:    make(map[uuid.UUID]bool),

		// A tester processes one job at a time, so a job may legitimately wait
		// in its client queue for several minutes before the first event.
		firstEventTimeout: 10 * time.Minute,
		// Once an execution has started it should keep streaming events.
		inactivityTimeout: 5 * time.Minute,
		// No-responders retries cover a short tester restart: 1+2+4+8+16+32s.
		publishRetryDelay:  time.Second,
		maxPublishAttempts: 7,
		watchdogInterval:   time.Second,
	}

	return esrvc
}

// Enqueue processes a code execution request by:
//  1. Validating the programming language and
//     constraints
//  2. Setting up result handlers and notification
//     channels
//  3. Sending the execution request to the
//     processing queue
//
// Returns the execution UUID for tracking
func (e *execSrvc) Enqueue(
	ctx context.Context,
	execUuid uuid.UUID,
	srcCode string,
	langId string,
	tests []TestFile,
	params TestingParams,
) srvcerror.E {
	l := ctxlog.FromContext(ctx).With("cmd", "enqueue execution")

	// 1. construct execution request
	execReq := ExecRequest{
		UUID:       execUuid,
		Code:       srcCode,
		Lang:       PrLang{},
		Tests:      tests,
		CpuMs:      params.CpuMs,
		MemKiB:     params.MemKiB,
		Checker:    params.Checker,
		Interactor: params.Interactor,
		Groups:     params.Groups,
	}
	lang, err := getPrLangById(langId)
	if err != nil {
		l.Error("get programming language", "lang_id", langId, "error", err)
		return srvcerror.InternalServerError()
	}
	execReq.Lang = lang

	// 2. validate sanity of request
	validationErr := execReq.IsValid()
	if validationErr != nil {
		// IsValid returns srvcerror.E for validation errors
		if se, ok := validationErr.(srvcerror.E); ok {
			return se
		}
		l.Error("validate exec request", "error", validationErr)
		return srvcerror.InternalServerError()
	}

	// 3. setup stream organizer
	hasCompile := lang.CompCmd != nil
	noOfTests := len(tests)
	org, orgErr := newResultStreamOrganizer(hasCompile, noOfTests)
	if orgErr != nil {
		l.Error("setup result stream organizer", "error", orgErr)
		return srvcerror.InternalServerError()
	}
	// 4. initialize empty execution
	exec := Execution{
		UUID:      execUuid,
		Stage:     StageWaiting,
		TestRes:   []TestRes{},
		PrLang:    lang,
		Params:    params,
		ErrorMsg:  nil,
		SysInfo:   nil,
		CreatedAt: time.Now(),
		SubmComp:  nil,
	}
	for i := 0; i < noOfTests; i++ {
		exec.TestRes = append(exec.TestRes, TestRes{ID: i + 1})
	}

	// 5. encode the execution request
	testerReq := execReq.MapToTesterApiType()
	reqJson, marshalErr := json.Marshal(testerReq)
	if marshalErr != nil {
		l.Error("marshal eval request in tester format", "error", marshalErr)
		return srvcerror.InternalServerError()
	}
	zstdEncoder, zstdErr := zstd.NewWriter(nil)
	if zstdErr != nil {
		l.Error("create zstd encoder", "error", zstdErr)
		return srvcerror.InternalServerError()
	}
	defer zstdEncoder.Close()
	compressed := zstdEncoder.EncodeAll(reqJson, make([]byte, 0, len(reqJson)))
	encoded := base64.StdEncoding.EncodeToString(compressed)
	encodedBytes := []byte(encoded)

	// 6. setup execution state
	wg := &sync.WaitGroup{}
	wg.Add(1)
	e.mu.Lock()
	e.execWg.Store(execUuid, wg)
	e.notifiers[execUuid] = make(chan Event, 1000)
	e.organizers[execUuid] = org
	e.executions[execUuid] = &exec
	e.fileHashes[execUuid] = allowedFileHashes(tests)

	// 7. send encoded message to job queue
	msg := nats.NewMsg(NatsSubject)
	msg.Reply = e.resultSubject(execUuid)
	msg.Header.Set(fileSubjectHeader, e.fileSubject)
	msg.Data = encodedBytes
	e.jobMsgs[execUuid] = msg
	e.lastEventAt[execUuid] = time.Now()
	e.publishTries[execUuid] = 1
	e.eventsSeen[execUuid] = false
	pubErr := e.publishJob(msg)
	if pubErr != nil {
		delete(e.notifiers, execUuid)
		delete(e.organizers, execUuid)
		delete(e.executions, execUuid)
		delete(e.fileHashes, execUuid)
		delete(e.jobMsgs, execUuid)
		delete(e.lastEventAt, execUuid)
		delete(e.publishTries, execUuid)
		delete(e.eventsSeen, execUuid)
		e.execWg.Delete(execUuid)
		e.mu.Unlock()
		l.Error("publish eval req to nats", "error", pubErr)
		return srvcerror.InternalServerError()
	}
	e.mu.Unlock()

	return nil
}

// Listen returns a channel that streams execution
// events to clients. The channel is automatically
// closed once the execution is complete
func (e *execSrvc) Listen(
	ctx context.Context,
	execId uuid.UUID,
) (<-chan Event, srvcerror.E) {
	l := ctxlog.FromContext(ctx).With("query", "listen to execution")

	e.mu.Lock()
	defer e.mu.Unlock()

	ch, ok := e.notifiers[execId]
	if !ok {
		l.Error("no listener for exec", "exec_id", execId)
		return nil, srvcerror.InternalServerError()
	}
	return ch, nil
}

// Get retrieves the execution results for a given
// execution ID. It waits for completion if the
// execution is still in progress
func (e *execSrvc) Get(
	ctx context.Context,
	execId uuid.UUID,
) (Execution, srvcerror.E) {
	l := ctxlog.FromContext(ctx).With("query", "get execution")

	e.mu.Lock()
	wgVal, exists := e.execWg.Load(execId)
	e.mu.Unlock()
	if !exists {
		exec, err := e.execRepo.Get(ctx, execId)
		if err != nil {
			l.Error("get execution from repo", "exec_id", execId, "error", err)
			return Execution{}, ErrEvalNotFound
		}
		return *exec, nil
	}

	wg := wgVal.(*sync.WaitGroup)

	// Wait for completion with context
	// cancellation support
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Clean up the WaitGroup
		e.execWg.Delete(execId)
		exec, err := e.execRepo.Get(ctx, execId)
		if err != nil {
			l.Error("get execution from repo after wait", "exec_id", execId, "error", err)
			return Execution{}, srvcerror.InternalServerError()
		}
		return *exec, nil
	case <-ctx.Done():
		l.Error("context cancelled while waiting for execution", "exec_id", execId)
		return Execution{}, srvcerror.InternalServerError()
	}
}

// based on incoming events. Handles various event
// types including compilation, testing, and error
// states
func applyEventToExec(
	exec *Execution,
	event Event,
) error {
	switch e := event.(type) {
	case ReceivedSubmission:
		exec.SysInfo = &e.SysInfo
	case StartedCompiling:
		exec.Stage = StageCompiling
	case FinishedCompiling:
		exec.Stage = StageFinished
		exec.SubmComp = e.RuntimeData
	case ReachedTest:
		exec.TestRes[e.TestId-1].Input = e.In
		exec.TestRes[e.TestId-1].Answer = e.Ans
		exec.TestRes[e.TestId-1].Reached = true
	case FinishedTest:
		exec.TestRes[e.TestID-1].Subm = e.Subm
		exec.TestRes[e.TestID-1].Checker = e.Checker
		exec.TestRes[e.TestID-1].Finished = true
	case IgnoredTest:
		exec.TestRes[e.TestId-1].Ignored = true
	case FinishedTesting:
		exec.Stage = StageFinished
	case InternalServerError:
		exec.Stage = StageInternalError
		exec.ErrorMsg = e.ErrorMsg
	case TesterUnavailable:
		exec.Stage = StageInternalError
		exec.ErrorMsg = e.ErrorMsg
	case CompilationError:
		exec.Stage = StageCompileError
		exec.ErrorMsg = e.ErrorMsg
	}
	return nil
}
