package execsrvc

import (
	"fmt"
	"sync"
)

// ExecResStreamOrganizer processes and orders a stream
// of testing events to ensure:
//   - Sequential ordering: Events are emitted only after
//     their dependencies are satisfied
//   - Deduplication: Events with identical keys are
//     processed only once
//   - Completeness: All required events must be received
//     before testing completion
//
// The organizer supports concurrent test execution
// while maintaining sequential event emission.
// For example, if test #2 finishes before test #1,
// the organizer will buffer test #2's results until
// test #1's results are processed.
type ExecResStreamOrganizer struct {
	// indicates if compilation is required
	hasCompilation bool

	// tracks received event keys for deduplication
	rcvEvKeys map[string]bool

	// buffers events by type until ready for emission
	evsOfType map[string][]Event

	// tracks emitted event keys to enforce ordering
	retKeys map[string]bool

	// total number of tests to execute
	expNumOfTests int

	// count of completed or ignored tests
	numFinTests int

	// indicates if internal server error occurred
	returnedISE bool

	// synchronizes access to internal state
	mu sync.Mutex
}

// NewExecResStreamOrganizer initializes a stream
// organizer for testing events.
// Parameters:
//   - hasCompilation: requires compilation
//   - numTests: number of test cases
//
// Returns error if numTests is invalid.
func NewExecResStreamOrganizer(
	hasCompilation bool,
	numTests int,
) (*ExecResStreamOrganizer, error) {
	if numTests < 0 {
		return nil, fmt.Errorf(
			"numTests must be non-negative",
		)
	}
	const maxTests = 1000 // Safe upper limit
	if numTests > maxTests {
		return nil, fmt.Errorf(
			"numTests must be less than %d",
			maxTests,
		)
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

// Add processes an incoming event and returns any
// events that are now ready for emission.
// Events are considered ready when all their
// dependencies have been satisfied and emitted.
// For example:
//   - Test results require their "test reached" event
//   - Tests require compilation success if compiled
//
// Handles deduplication and sequential ordering.
// Returns error if event type is unknown.
func (o *ExecResStreamOrganizer) Add(
	event Event,
) ([]Event, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Internal server error encountered
	if o.returnedISE {
		return nil, nil
	}

	key := eventKey(event)

	// Skip duplicate events
	if o.rcvEvKeys[key] {
		return nil, nil
	}
	o.rcvEvKeys[key] = true

	// Add to events of same type
	o.evsOfType[event.Type()] = append(
		o.evsOfType[event.Type()],
		event,
	)

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
		return nil, fmt.Errorf(
			"unknown event type: %s",
			event.Type(),
		)
	}
}

// HasFinished indicates whether evaluation has
// completed and no more events will be processed.
// This occurs when:
//   - Internal server error encountered
//   - All tests complete successfully
//   - Compilation fails
func (o *ExecResStreamOrganizer) HasFinished() bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.returnedISE ||
		o.retKeys[FinishedTestingType] ||
		o.retKeys[CompilationErrorType]
}

// receiveSubm handles initial submission event and
// triggers next phase. For compiled languages, starts
// compilation. For interpreted, starts testing.
func (o *ExecResStreamOrganizer) receiveSubm() (
	[]Event,
	error,
) {
	// Return if submission not received yet
	if !o.rcvEvKeys[ReceivedSubmissionType] {
		return nil, nil
	}

	// Return if already processed
	if o.retKeys[ReceivedSubmissionType] {
		return nil, nil
	}

	e, err := o.getSingleEvent(ReceivedSubmissionType)
	if err != nil {
		return nil, err
	}

	// Mark as processed
	o.retKeys[ReceivedSubmissionType] = true
	res := []Event{e}

	// Start compilation or testing
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

// startCompile processes compilation start event.
// Only valid after submission received.
func (o *ExecResStreamOrganizer) startCompile() (
	[]Event,
	error,
) {
	// Verify compiled language
	if !o.hasCompilation {
		return nil, fmt.Errorf(
			"unexpected compile for non-compiled lang",
		)
	}

	// Return if not received or already processed
	if !o.rcvEvKeys[StartedCompilationType] ||
		o.retKeys[StartedCompilationType] {
		return nil, nil
	}

	// Return if submission not processed
	if !o.retKeys[ReceivedSubmissionType] {
		return nil, nil
	}

	e, err := o.getSingleEvent(StartedCompilationType)
	if err != nil {
		return nil, err
	}

	// Mark as processed
	o.retKeys[StartedCompilationType] = true
	res := []Event{e}

	// Try finish compilation
	nxt, err := o.finishCompile()
	if err != nil {
		return append(res, nxt...), err
	}
	return append(res, nxt...), nil
}

// finishCompile handles compilation completion.
// On success starts testing, on failure emits error.
func (o *ExecResStreamOrganizer) finishCompile() (
	[]Event,
	error,
) {
	// Verify compiled language
	if !o.hasCompilation {
		return nil, fmt.Errorf(
			"unexpected compile for non-compiled lang",
		)
	}

	// Return if not received or already processed
	if !o.rcvEvKeys[FinishedCompilationType] ||
		o.retKeys[FinishedCompilationType] {
		return nil, nil
	}

	// Return if compilation not started
	if !o.retKeys[StartedCompilationType] {
		return nil, nil
	}

	e, err := o.getSingleEvent(FinishedCompilationType)
	if err != nil {
		return nil, err
	}

	// Mark as processed
	o.retKeys[FinishedCompilationType] = true
	res := []Event{e}

	// Check compilation error or start testing
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

// startTesting initiates test execution phase.
// Requires compilation success if compiled.
func (o *ExecResStreamOrganizer) startTesting() (
	[]Event,
	error,
) {
	if !o.rcvEvKeys[StartedTestingType] ||
		o.retKeys[StartedTestingType] {
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

	// Try reach and ignore paths for first test
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

// reachTest processes test case initiation.
// Ensures sequential processing.
func (o *ExecResStreamOrganizer) reachTest(
	id int,
) ([]Event, error) {
	key := fmt.Sprintf("%s-%d", ReachedTestType, id)

	if !o.rcvEvKeys[key] || o.retKeys[key] {
		return nil, nil
	}

	if !o.retKeys[StartedTestingType] {
		return nil, nil
	}

	// Check previous test completion
	if id > 1 {
		prevReachedKey := fmt.Sprintf(
			"%s-%d",
			ReachedTestType,
			id-1,
		)
		prevFinishedKey := fmt.Sprintf(
			"%s-%d",
			FinishedTestType,
			id-1,
		)
		if !o.retKeys[prevReachedKey] ||
			!o.retKeys[prevFinishedKey] {
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

// ignoreTest handles skipped test cases.
// Maintains sequential order.
func (o *ExecResStreamOrganizer) ignoreTest(
	id int,
) ([]Event, error) {
	key := fmt.Sprintf("%s-%d", IgnoredTestType, id)

	if !o.rcvEvKeys[key] || o.retKeys[key] {
		return nil, nil
	}

	if !o.retKeys[StartedTestingType] {
		return nil, nil
	}

	if id > 1 {
		prevReachedKey := fmt.Sprintf(
			"%s-%d",
			ReachedTestType,
			id-1,
		)
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
		// Try ignore and reach for next test
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

// finishTest processes test completion event.
// Ensures proper test initiation and ordering.
func (o *ExecResStreamOrganizer) finishTest(
	id int,
) ([]Event, error) {
	if id < 1 || id > o.expNumOfTests {
		return nil, fmt.Errorf("invalid test id: %d", id)
	}

	key := fmt.Sprintf("%s-%d", FinishedTestType, id)
	if !o.rcvEvKeys[key] || o.retKeys[key] {
		return nil, nil
	}

	reachedKey := fmt.Sprintf(
		"%s-%d",
		ReachedTestType,
		id,
	)
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

// finishTesting handles completion of all test cases.
// Verifies all tests completed or ignored.
func (o *ExecResStreamOrganizer) finishTesting() (
	[]Event,
	error,
) {
	if !o.rcvEvKeys[FinishedTestingType] ||
		o.retKeys[FinishedTestingType] {
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
// Valid after compilation finish, before testing.
func (o *ExecResStreamOrganizer) compileError() (
	[]Event,
	error,
) {
	if !o.rcvEvKeys[CompilationErrorType] ||
		o.retKeys[CompilationErrorType] {
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

// getReachedTestEvent retrieves test start event.
// Returns error if missing or duplicate.
func (o *ExecResStreamOrganizer) getReachedTestEvent(
	id int,
) (Event, error) {
	events, ok := o.evsOfType[ReachedTestType]
	if !ok {
		return nil, fmt.Errorf(
			"no events for type: %s",
			ReachedTestType,
		)
	}

	for _, event := range events {
		if reachedTest, ok := event.(ReachedTest); ok &&
			reachedTest.TestId == id {
			return event, nil
		}
	}
	return nil, fmt.Errorf(
		"no ReachedTest event for id: %d",
		id,
	)
}

// getIgnoredTestEvent retrieves test skip event.
// Returns error if missing or duplicate.
func (o *ExecResStreamOrganizer) getIgnoredTestEvent(
	id int,
) (Event, error) {
	events, ok := o.evsOfType[IgnoredTestType]
	if !ok {
		return nil, fmt.Errorf(
			"no events for type: %s",
			IgnoredTestType,
		)
	}

	for _, event := range events {
		if ignoredTest, ok := event.(IgnoredTest); ok &&
			ignoredTest.TestId == id {
			return event, nil
		}
	}
	return nil, fmt.Errorf(
		"no IgnoredTest event for id: %d",
		id,
	)
}

// getFinishedTestEvent retrieves test completion event.
// Returns error if missing or duplicate.
func (o *ExecResStreamOrganizer) getFinishedTestEvent(
	id int,
) (Event, error) {
	events, ok := o.evsOfType[FinishedTestType]
	if !ok {
		return nil, fmt.Errorf(
			"no events for type: %s",
			FinishedTestType,
		)
	}

	for _, event := range events {
		if finishedTest, ok := event.(FinishedTest); ok &&
			finishedTest.TestID == id {
			return event, nil
		}
	}
	return nil, fmt.Errorf(
		"no FinishedTest event for id: %d",
		id,
	)
}

// getSingleEvent retrieves unique event by type.
// Returns error if missing or duplicate.
func (o *ExecResStreamOrganizer) getSingleEvent(
	eventType string,
) (Event, error) {
	events, ok := o.evsOfType[eventType]
	if !ok {
		return nil, fmt.Errorf(
			"no events for type: %s",
			eventType,
		)
	}
	if len(events) > 1 {
		return nil, fmt.Errorf(
			"multiple events for type: %s",
			eventType,
		)
	}
	return events[0], nil
}

// eventKey generates unique identifier.
// Test events include test ID in key.
func eventKey(event Event) string {
	switch e := event.(type) {
	case ReachedTest:
		return fmt.Sprintf(
			"%s-%d",
			ReachedTestType,
			e.TestId,
		)
	case IgnoredTest:
		return fmt.Sprintf(
			"%s-%d",
			IgnoredTestType,
			e.TestId,
		)
	case FinishedTest:
		return fmt.Sprintf(
			"%s-%d",
			FinishedTestType,
			e.TestID,
		)
	default:
		return event.Type()
	}
}
