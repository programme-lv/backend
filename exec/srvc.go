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
	testerapi "github.com/programme-lv/tester/api"
)

// ExecRepo interface for execution storage
type ExecRepo interface {
	Save(ctx context.Context, exec *Execution) error
	Get(ctx context.Context, id uuid.UUID) (*Execution, error)
}

type ExecSrvcClient interface {
	Enqueue(ctx context.Context, uuid uuid.UUID, srcCode string, prLangId string, tests []TestFile, params TestingParams) error
	Listen(ctx context.Context, uuid uuid.UUID) (<-chan Event, error)
	Get(ctx context.Context, execUuid uuid.UUID) (Execution, error)
}

type ExecSrvcFacade interface {
	ExecSrvcClient
	StartPollingResultQueue(ctx context.Context) error
}

const NatsSubject = "tester.jobs"

// ExecSrvcImpl handles communication with testers
// for code execution and result streaming
type ExecSrvcImpl struct {
	logger *slog.Logger

	// either in-mem or s3
	execRepo ExecRepo

	natsConn  *nats.Conn
	natsInbox string

	mu sync.Mutex
	// maps exec IDs to client result channels
	notifiers map[uuid.UUID]chan Event
	// tracks completion status of executions
	execWg sync.Map // notifies get listener when execution is finished

	isPolling atomic.Bool

	organizers map[uuid.UUID]*ExecResStreamOrganizer
	executions map[uuid.UUID]*Execution
}

var _ ExecSrvcFacade = &ExecSrvcImpl{}

func (e *ExecSrvcImpl) StartPollingResultQueue(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	notAlreadyPolling := e.isPolling.CompareAndSwap(false, true)
	if !notAlreadyPolling {
		return fmt.Errorf("already polling result queue")
	}

	sub, err := e.natsConn.Subscribe(e.natsInbox, func(msg *nats.Msg) {
		e.mu.Lock()
		defer e.mu.Unlock()

		var header testerapi.Header
		err := json.Unmarshal(msg.Data, &header)
		if err != nil {
			e.logger.Error("unsmarshall NATS msg header", "error", err)
			return
		}

		execUuid, err := uuid.Parse(header.EvalUuid)
		if err != nil {
			e.logger.Error("parse eval uuid", "error", err)
			return
		}

		org, exists := e.organizers[execUuid]
		if !exists {
			e.logger.Error("stream organizer not found", "exec_uuid", execUuid)
			return
		}
		if org == nil {
			panic("stream organizer is nil")
		}

		if org.HasFinished() {
			e.logger.Error("stream organizer has finished", "exec_uuid", execUuid)
			return
		}

		ev, err := mapTesterMsgJsonToEvent(msg.Data, header.MsgType)
		if err != nil {
			e.logger.Error("map tester msg json to event", "error", err)
			return
		}

		events, err := org.Add(ev)
		if err != nil {
			e.logger.Error("add event to stream organizer", "error", err)
			return
		}

		for _, event := range events {
			e.notifiers[execUuid] <- event
		}

		exec, ok := e.executions[execUuid]
		if !ok {
			e.logger.Error("execution not found", "exec_uuid", execUuid)
			return
		}
		if exec == nil {
			panic("execution is nil")
		}
		for _, event := range events {
			err := applyEventToExec(exec, event)
			if err != nil {
				e.logger.Error("apply ev to exec", "error", err)
			}
		}
		if org.HasFinished() {
			close(e.notifiers[execUuid])
			delete(e.notifiers, execUuid) // deleting closes the channel
			delete(e.organizers, execUuid)
			delete(e.executions, execUuid)

			err = e.execRepo.Save(ctx, exec)
			if err != nil {
				e.logger.Error("save exec", "error", err)
				return
			}

			wgVal, exists := e.execWg.Load(execUuid)
			if !exists {
				e.logger.Error("wait group not found", "exec_uuid", execUuid)
				return
			}
			wg := wgVal.(*sync.WaitGroup)
			wg.Done()
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe to nats inbox: %w", err)
	}
	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
	}()

	return nil
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

func NewExecSrvc(ctx context.Context, repo ExecRepo, natsConn *nats.Conn) ExecSrvcFacade {
	esrvc := &ExecSrvcImpl{
		logger:     ctxlog.FromContext(ctx),
		natsConn:   natsConn,
		natsInbox:  nats.NewInbox(),
		execRepo:   repo,
		notifiers:  make(map[uuid.UUID]chan Event),
		organizers: make(map[uuid.UUID]*ExecResStreamOrganizer),
		executions: make(map[uuid.UUID]*Execution),
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
func (e *ExecSrvcImpl) Enqueue(
	ctx context.Context,
	execUuid uuid.UUID,
	srcCode string,
	langId string,
	tests []TestFile,
	params TestingParams,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

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
		return err
	}
	execReq.Lang = lang

	// 2. validate sanity of request
	err = execReq.IsValid()
	if err != nil {
		return fmt.Errorf("validate exec request: %w", err)
	}

	// 3. create WaitGroup for awaiting results
	wg := &sync.WaitGroup{}
	wg.Add(1)
	e.execWg.Store(execUuid, wg)

	// 4. setup notification channel
	e.notifiers[execUuid] = make(chan Event, 1000)

	// 5. setup stream organizer
	hasCompile := lang.CompCmd != nil
	noOfTests := len(tests)
	org, err := newResultStreamOrganizer(hasCompile, noOfTests)
	if err != nil {
		return fmt.Errorf("setup result stream organizer: %w", err)
	}
	e.organizers[execUuid] = org

	// 6. initialize empty execution
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
	e.executions[execUuid] = &exec

	// 7. encode the execution request
	testerReq := execReq.MapToTesterApiType()
	reqJson, err := json.Marshal(testerReq)
	if err != nil {
		return fmt.Errorf("marshal eval request in tester format: %w", err)
	}
	zstdEncoder, err := zstd.NewWriter(nil)
	if err != nil {
		return fmt.Errorf("failed to create zstd encoder: %w", err)
	}
	defer zstdEncoder.Close()
	compressed := zstdEncoder.EncodeAll(reqJson, make([]byte, 0, len(reqJson)))
	encoded := base64.StdEncoding.EncodeToString(compressed)
	encodedBytes := []byte(encoded)

	// 8. send encoded message to job queue
	err = e.natsConn.PublishRequest(NatsSubject, e.natsInbox, encodedBytes)
	if err != nil {
		return fmt.Errorf("publish eval req to nats: %w", err)
	}

	return nil
}

// Listen returns a channel that streams execution
// events to clients. The channel is automatically
// closed once the execution is complete
func (e *ExecSrvcImpl) Listen(
	ctx context.Context,
	execId uuid.UUID,
) (<-chan Event, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	ch, ok := e.notifiers[execId]
	if !ok {
		format := "no listener for exec %s"
		errMsg := fmt.Errorf(format, execId)
		return nil, errMsg
	}
	return ch, nil
}

// Get retrieves the execution results for a given
// execution ID. It waits for completion if the
// execution is still in progress
func (e *ExecSrvcImpl) Get(
	ctx context.Context,
	execId uuid.UUID,
) (Execution, error) {
	// Get the WaitGroup for this execution
	wgVal, exists := e.execWg.Load(execId)
	if !exists {
		exec, err := e.execRepo.Get(
			ctx,
			execId,
		)
		if err != nil {
			return Execution{}, fmt.Errorf(
				"no execution found for id %s",
				execId,
			)
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
		exec, err := e.execRepo.Get(
			ctx,
			execId,
		)
		if err != nil {
			return Execution{}, err
		}
		return *exec, nil
	case <-ctx.Done():
		return Execution{}, ctx.Err()
	}
}

// handleSqsMsg processes incoming SQS messages
// and routes them to appropriate handlers
func (e *ExecSrvcImpl) handleSqsMsg(
	msg SqsResponseMsg,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	org, exists := e.organizers[msg.ExecId]
	if !exists {
		e.logger.Debug(
			"no organizer found for execution",
			"exec_id",
			msg.ExecId,
		)
		return fmt.Errorf("no organizer found for execution %s", msg.ExecId)
	}

	if org == nil {
		e.logger.Error(
			"organizer is nil for execution",
			"exec_id",
			msg.ExecId,
		)
		return fmt.Errorf("organizer is nil for execution %s", msg.ExecId)
	}

	if org.HasFinished() {
		return nil
	}

	events, err := org.Add(msg.Data)
	if err != nil {
		return fmt.Errorf(
			"failed to process msg: %w",
			err,
		)
	}
	exec := e.executions[msg.ExecId]
	if exec == nil {
		e.logger.Error(
			"execution not found",
			"exec_id",
			msg.ExecId,
		)
		return fmt.Errorf("execution not found for %s", msg.ExecId)
	}

	for _, event := range events {
		err := applyEventToExec(exec, event)
		if err != nil {
			return fmt.Errorf(
				"failed to apply event: %w",
				err,
			)
		}
		e.notifiers[msg.ExecId] <- event
	}
	if !org.HasFinished() {
		return nil
	}
	close(e.notifiers[msg.ExecId])
	delete(e.notifiers, msg.ExecId)  // deleting closes the channel
	delete(e.organizers, msg.ExecId) // cleanup the organizer
	delete(e.executions, msg.ExecId) // cleanup the execution
	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()
	err = e.execRepo.Save(ctxWithTimeout, exec)
	if err != nil {
		e.logger.Error(
			"failed to save execution",
			"error",
			err,
		)
		return fmt.Errorf(
			"failed to save execution: %w",
			err,
		)
	}
	wgVal, exists := e.execWg.Load(msg.ExecId)
	if exists {
		wg := wgVal.(*sync.WaitGroup)
		wg.Done()
	}
	return nil
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
