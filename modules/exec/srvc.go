package exec

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
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

	mu sync.Mutex
	// maps exec IDs to client result channels
	notifiers map[uuid.UUID]chan Event
	// tracks completion status of executions
	execWg sync.Map // notifies get listener when execution is finished

	isPolling atomic.Bool

	organizers map[uuid.UUID]*ExecResStreamOrganizer
	executions map[uuid.UUID]*Execution
	fileHashes map[uuid.UUID]map[string]struct{}
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

	resultSub, err := e.natsConn.Subscribe(e.natsInbox, func(msg *nats.Msg) {
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

	return nil
}

func (e *execSrvc) handleResultMessage(ctx context.Context, msg *nats.Msg) {
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
		execRepo:      repo,
		notifiers:     make(map[uuid.UUID]chan Event),
		organizers:    make(map[uuid.UUID]*ExecResStreamOrganizer),
		executions:    make(map[uuid.UUID]*Execution),
		fileHashes:    make(map[uuid.UUID]map[string]struct{}),
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
	e.execWg.Store(execUuid, wg)
	e.mu.Lock()
	e.notifiers[execUuid] = make(chan Event, 1000)
	e.organizers[execUuid] = org
	e.executions[execUuid] = &exec
	e.fileHashes[execUuid] = allowedFileHashes(tests)
	e.mu.Unlock()

	// 7. send encoded message to job queue
	msg := nats.NewMsg(NatsSubject)
	msg.Reply = e.natsInbox
	msg.Header.Set(fileSubjectHeader, e.fileSubject)
	msg.Data = encodedBytes
	pubErr := e.natsConn.PublishMsg(msg)
	if pubErr != nil {
		e.mu.Lock()
		delete(e.notifiers, execUuid)
		delete(e.organizers, execUuid)
		delete(e.executions, execUuid)
		delete(e.fileHashes, execUuid)
		e.mu.Unlock()
		e.execWg.Delete(execUuid)
		l.Error("publish eval req to nats", "error", pubErr)
		return srvcerror.InternalServerError()
	}

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

	// Get the WaitGroup for this execution
	wgVal, exists := e.execWg.Load(execId)
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
	case CompilationError:
		exec.Stage = StageCompileError
		exec.ErrorMsg = e.ErrorMsg
	}
	return nil
}
