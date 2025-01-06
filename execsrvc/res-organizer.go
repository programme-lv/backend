package execsrvc

import (
	"fmt"
	"sync"
)

// ExecResStreamOrganizer processes and orders a stream of evaluation events to ensure:
// - Sequential ordering: Events are emitted only after their dependencies are satisfied
// - Deduplication: Events with identical keys are processed only once
// - Completeness: All required events must be received before evaluation completion
//
// The organizer supports concurrent test execution while maintaining sequential event emission.
// For example, if test #2 finishes before test #1, the organizer will buffer test #2's results
// until test #1's results are processed.
type ExecResStreamOrganizer struct {
	hasCompilation bool // indicates if the language requires a compilation step

	rcvEvKeys map[string]bool    // tracks received event keys for deduplication
	evsOfType map[string][]Event // buffers events by type until ready for emission
	retKeys   map[string]bool    // tracks emitted event keys to enforce ordering

	expNumOfTests int // total number of tests to execute
	numFinTests   int // count of completed or ignored tests

	returnedISE bool // indicates if an internal server error occurred

	mu sync.Mutex // synchronizes access to internal state
}

// NewExecResStreamOrganizer initializes a stream organizer for evaluation events.
// Parameters:
//   - hasCompilation: whether the language requires compilation
//   - numTests: number of test cases to execute
//
// Returns error if numTests is invalid (negative or exceeds maximum limit).
func NewExecResStreamOrganizer(hasCompilation bool, numTests int) (*ExecResStreamOrganizer, error) {
	if numTests < 0 {
		return nil, fmt.Errorf("numTests must be non-negative")
	}
	const maxTests = 1000 // Safe upper limit, most tasks have <200 tests
	if numTests > maxTests {
		return nil, fmt.Errorf("numTests must be less than %d", maxTests)
	}

	return &ExecResStreamOrganizer{
		hasCompilation: hasCompilation,
		expNumOfTests:  numTests,
		rcvEvKeys:      make(map[string]bool),
		evsOfType:      make(map[string][]Event),
		retKeys:        make(map[string]bool),
		numFinTests:    0,
	}, nil
}

// Add processes an incoming event and returns any events that are now ready for emission.
// Events are considered ready when all their dependencies have been satisfied and emitted.
// For example:
//   - Test results require their corresponding "test reached" event
//   - Test execution requires compilation success (for compiled languages)
//
// The function handles event deduplication and maintains sequential ordering.
// Returns error if the event type is unknown or event processing fails.
func (o *ExecResStreamOrganizer) Add(event Event) ([]Event, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Internal server error has been encountered
	if o.returnedISE {
		return nil, nil
	}

	key := eventKey(event)

	// Skip duplicate events
	if o.rcvEvKeys[key] {
		return nil, nil
	}
	o.rcvEvKeys[key] = true

	// Add event to the list of events of the same type
	o.evsOfType[event.Type()] = append(o.evsOfType[event.Type()], event)

	switch e := event.(type) {
	case ReceivedSubmission:
		return o.receiveSubm()
	case StartedCompiling:
		return o.startCompile()
	case CompilationError:
		return o.compileError()
	case FinishedCompiling:
		return o.finishCompile()
	case StartedTesting:
		return o.startTesting()
	case ReachedTest:
		return o.reachTest(e.TestId)
	case IgnoredTest:
		return o.ignoreTest(e.TestId)
	case FinishedTest:
		return o.finishTest(e.TestID)
	case FinishedTesting:
		return o.finishTesting()
	case InternalServerError:
		o.returnedISE = true
		return []Event{event}, nil
	default:
		return nil, fmt.Errorf("unknown event type: %s", event.Type())
	}
}

// HasFinished indicates whether the evaluation has completed and no more events
// will be processed. This occurs when:
//   - An internal server error is encountered
//   - All tests complete successfully (FinishedTesting)
//   - Compilation fails (CompilationError)
func (o *ExecResStreamOrganizer) HasFinished() bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.returnedISE || o.retKeys[FinishedTestingType] || o.retKeys[CompilationErrorType]
}

// receiveSubm handles the initial submission event and triggers the appropriate next phase.
// For compiled languages, this initiates compilation. For interpreted languages,
// testing begins immediately.
func (o *ExecResStreamOrganizer) receiveSubm() ([]Event, error) {
	// Return if we haven't received the "received submission" event yet
	if !o.rcvEvKeys[ReceivedSubmissionType] {
		return nil, nil
	}

	// Return if we've already processed "received submission" event
	if o.retKeys[ReceivedSubmissionType] {
		return nil, nil
	}

	e, err := o.getSingleEvent(ReceivedSubmissionType)
	if err != nil {
		return nil, err
	}

	// Mark "received submission" event as processed
	o.retKeys[ReceivedSubmissionType] = true
	res := []Event{e}

	// For compiled languages, expect compilation to start
	// For interpreted languages, expect testing to start
	var nxt []Event
	if o.hasCompilation {
		nxt, err = o.startCompile()
	} else {
		nxt, err = o.startTesting()
	}
	if err != nil {
		return append(res, nxt...), err
	}
	return append(res, nxt...), nil
}

// startCompile processes the compilation start event for compiled languages.
// This represents the beginning of the compilation phase and is only valid
// after submission has been received.
func (o *ExecResStreamOrganizer) startCompile() ([]Event, error) {
	// Verify this is a compiled language
	if !o.hasCompilation {
		return nil, fmt.Errorf("unexpected compile for non-compiled language")
	}

	// Return if we haven't received StartedCompilation event yet
	// or if we've already processed it
	if !o.rcvEvKeys[StartedCompilationType] || o.retKeys[StartedCompilationType] {
		return nil, nil
	}

	// Return if we haven't processed ReceivedSubmission event yet
	// (compilation can't start before submission is received)
	if !o.retKeys[ReceivedSubmissionType] {
		return nil, nil
	}

	// Get the StartedCompilation event
	e, err := o.getSingleEvent(StartedCompilationType)
	if err != nil {
		return nil, err
	}

	// Mark StartedCompilation event as processed
	o.retKeys[StartedCompilationType] = true
	res := []Event{e}

	// Try to process FinishedCompilation event if it's ready
	nxt, err := o.finishCompile()
	if err != nil {
		return append(res, nxt...), err
	}
	return append(res, nxt...), nil
}

// finishCompile handles compilation completion and determines the next phase.
// On success, testing begins. On failure, a compilation error is emitted.
func (o *ExecResStreamOrganizer) finishCompile() ([]Event, error) {
	// Verify this is a compiled language
	if !o.hasCompilation {
		return nil, fmt.Errorf("unexpected compile for non-compiled language")
	}

	// Return if we haven't received FinishedCompilation event yet
	// or if we've already processed it
	if !o.rcvEvKeys[FinishedCompilationType] || o.retKeys[FinishedCompilationType] {
		return nil, nil
	}

	// Return if we haven't processed StartedCompilation event yet
	// (compilation can't finish before it starts)
	if !o.retKeys[StartedCompilationType] {
		return nil, nil
	}

	// Get the FinishedCompilation event
	e, err := o.getSingleEvent(FinishedCompilationType)
	if err != nil {
		return nil, err
	}

	// Mark FinishedCompilation event as processed
	o.retKeys[FinishedCompilationType] = true
	res := []Event{e}

	// After compilation finishes, we need to check two possible paths:
	// 1. If compilation failed, we'll get a compilation error event
	// 2. If compilation succeeded, we can start testing
	nxt, err := o.compileError()
	if err != nil {
		return append(res, nxt...), err
	}
	res = append(res, nxt...)

	nxt, err = o.startTesting()
	if err != nil {
		return append(res, nxt...), err
	}
	return append(res, nxt...), nil
}

// startTesting initiates the test execution phase.
// For compiled languages, this requires successful compilation.
// For interpreted languages, this follows immediately after submission.
func (o *ExecResStreamOrganizer) startTesting() ([]Event, error) {
	if !o.rcvEvKeys[StartedTestingType] || o.retKeys[StartedTestingType] {
		return nil, nil
	}

	// Check dependencies
	if o.hasCompilation {
		if !o.retKeys[FinishedCompilationType] {
			return nil, nil
		}
	} else if !o.retKeys[ReceivedSubmissionType] {
		return nil, nil
	}

	e, err := o.getSingleEvent(StartedTestingType)
	if err != nil {
		return nil, err
	}

	o.retKeys[StartedTestingType] = true
	res := []Event{e}

	// Try both reach test and ignore test paths for first test
	nxt, err := o.reachTest(1)
	if err != nil {
		return append(res, nxt...), err
	}
	res = append(res, nxt...)

	nxt, err = o.ignoreTest(1)
	if err != nil {
		return append(res, nxt...), err
	}
	return append(res, nxt...), nil
}

// reachTest processes a test case initiation event.
// Ensures tests are processed sequentially by buffering out-of-order results.
func (o *ExecResStreamOrganizer) reachTest(id int) ([]Event, error) {
	key := fmt.Sprintf("%s-%d", ReachedTestType, id)

	if !o.rcvEvKeys[key] || o.retKeys[key] {
		return nil, nil
	}

	if !o.retKeys[StartedTestingType] {
		return nil, nil
	}

	// For tests after first, check previous test completion
	if id > 1 {
		prevReachedKey := fmt.Sprintf("%s-%d", ReachedTestType, id-1)
		prevFinishedKey := fmt.Sprintf("%s-%d", FinishedTestType, id-1)
		if !o.retKeys[prevReachedKey] || !o.retKeys[prevFinishedKey] {
			return nil, nil
		}
	}

	e, err := o.getReachedTestEvent(id)
	if err != nil {
		return nil, err
	}

	o.retKeys[key] = true
	res := []Event{e}

	nxt, err := o.finishTest(id)
	if err != nil {
		return append(res, nxt...), err
	}
	return append(res, nxt...), nil
}

// ignoreTest handles skipped test cases while maintaining sequential order.
// A test may be ignored if previous tests failed or resource limits were exceeded.
func (o *ExecResStreamOrganizer) ignoreTest(id int) ([]Event, error) {
	key := fmt.Sprintf("%s-%d", IgnoredTestType, id)

	if !o.rcvEvKeys[key] || o.retKeys[key] {
		return nil, nil
	}

	if !o.retKeys[StartedTestingType] {
		return nil, nil
	}

	if id > 1 {
		prevReachedKey := fmt.Sprintf("%s-%d", ReachedTestType, id-1)
		if !o.retKeys[prevReachedKey] {
			return nil, nil
		}
	}

	e, err := o.getIgnoredTestEvent(id)
	if err != nil {
		return nil, err
	}

	o.retKeys[key] = true
	o.numFinTests++
	res := []Event{e}

	if id < o.expNumOfTests {
		// Try both ignore and reach paths for next test
		nxt, err := o.ignoreTest(id + 1)
		if err != nil {
			return append(res, nxt...), err
		}
		res = append(res, nxt...)

		nxt, err = o.reachTest(id + 1)
		if err != nil {
			return append(res, nxt...), err
		}
		res = append(res, nxt...)
	}

	nxt, err := o.finishTesting()
	if err != nil {
		return append(res, nxt...), err
	}
	return append(res, nxt...), nil
}

// finishTest processes a test completion event.
// Ensures the test was properly initiated and maintains sequential ordering.
func (o *ExecResStreamOrganizer) finishTest(id int) ([]Event, error) {
	if id < 1 || id > o.expNumOfTests {
		return nil, fmt.Errorf("invalid test id: %d", id)
	}

	key := fmt.Sprintf("%s-%d", FinishedTestType, id)
	if !o.rcvEvKeys[key] || o.retKeys[key] {
		return nil, nil
	}

	reachedKey := fmt.Sprintf("%s-%d", ReachedTestType, id)
	if !o.retKeys[reachedKey] {
		return nil, nil
	}

	e, err := o.getFinishedTestEvent(id)
	if err != nil {
		return nil, err
	}

	o.retKeys[key] = true
	o.numFinTests++
	res := []Event{e}

	if id < o.expNumOfTests {
		nxt, err := o.reachTest(id + 1)
		if err != nil {
			return append(res, nxt...), err
		}
		res = append(res, nxt...)
	}

	nxt, err := o.finishTesting()
	if err != nil {
		return append(res, nxt...), err
	}
	return append(res, nxt...), nil
}

// finishTesting handles the completion of all test cases.
// Verifies that all expected tests have either completed or been ignored.
func (o *ExecResStreamOrganizer) finishTesting() ([]Event, error) {
	if !o.rcvEvKeys[FinishedTestingType] || o.retKeys[FinishedTestingType] {
		return nil, nil
	}

	if o.numFinTests < o.expNumOfTests {
		return nil, nil
	}

	e, err := o.getSingleEvent(FinishedTestingType)
	if err != nil {
		return nil, err
	}

	o.retKeys[FinishedTestingType] = true
	return []Event{e}, nil
}

// compileError processes compilation failures.
// Only valid after compilation has finished and before testing begins.
func (o *ExecResStreamOrganizer) compileError() ([]Event, error) {
	if !o.rcvEvKeys[CompilationErrorType] || o.retKeys[CompilationErrorType] {
		return nil, nil
	}

	if !o.retKeys[FinishedCompilationType] {
		return nil, nil
	}

	e, err := o.getSingleEvent(CompilationErrorType)
	if err != nil {
		return nil, err
	}

	o.retKeys[CompilationErrorType] = true
	return []Event{e}, nil
}

// getReachedTestEvent retrieves the event indicating a test case has started.
// Returns error if the event doesn't exist or if multiple events exist for the same test.
func (o *ExecResStreamOrganizer) getReachedTestEvent(id int) (Event, error) {
	events, ok := o.evsOfType[ReachedTestType]
	if !ok {
		return nil, fmt.Errorf("no events for type: %s", ReachedTestType)
	}

	for _, event := range events {
		if reachedTest, ok := event.(ReachedTest); ok && reachedTest.TestId == id {
			return event, nil
		}
	}
	return nil, fmt.Errorf("no ReachedTest event for id: %d", id)
}

// getIgnoredTestEvent retrieves the event indicating a test case was skipped.
// Returns error if the event doesn't exist or if multiple events exist for the same test.
func (o *ExecResStreamOrganizer) getIgnoredTestEvent(id int) (Event, error) {
	events, ok := o.evsOfType[IgnoredTestType]
	if !ok {
		return nil, fmt.Errorf("no events for type: %s", IgnoredTestType)
	}

	for _, event := range events {
		if ignoredTest, ok := event.(IgnoredTest); ok && ignoredTest.TestId == id {
			return event, nil
		}
	}
	return nil, fmt.Errorf("no IgnoredTest event for id: %d", id)
}

// getFinishedTestEvent retrieves the event indicating a test case has completed.
// Returns error if the event doesn't exist or if multiple events exist for the same test.
func (o *ExecResStreamOrganizer) getFinishedTestEvent(id int) (Event, error) {
	events, ok := o.evsOfType[FinishedTestType]
	if !ok {
		return nil, fmt.Errorf("no events for type: %s", FinishedTestType)
	}

	for _, event := range events {
		if finishedTest, ok := event.(FinishedTest); ok && finishedTest.TestID == id {
			return event, nil
		}
	}
	return nil, fmt.Errorf("no FinishedTest event for id: %d", id)
}

// getSingleEvent retrieves a unique event of the specified type.
// Returns error if no events exist or if multiple events are found.
func (o *ExecResStreamOrganizer) getSingleEvent(eventType string) (Event, error) {
	events, ok := o.evsOfType[eventType]
	if !ok {
		return nil, fmt.Errorf("no events for type: %s", eventType)
	}
	if len(events) > 1 {
		return nil, fmt.Errorf("multiple events for type: %s", eventType)
	}
	return events[0], nil
}

// eventKey generates a unique identifier for deduplication.
// Test-related events include the test ID in their key.
func eventKey(event Event) string {
	switch e := event.(type) {
	case ReachedTest:
		return fmt.Sprintf("%s-%d", ReachedTestType, e.TestId)
	case IgnoredTest:
		return fmt.Sprintf("%s-%d", IgnoredTestType, e.TestId)
	case FinishedTest:
		return fmt.Sprintf("%s-%d", FinishedTestType, e.TestID)
	default:
		return event.Type()
	}
}
