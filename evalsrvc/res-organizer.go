package evalsrvc

import (
	"fmt"
	"sync"
)

// ExecResStreamOrganizer transforms a stream of events to gurantee:
// 1. correct order;
// 2. no duplicates;
// 3. no missing events.
// Organizer supports parallel execution of tests by the tester.
type ExecResStreamOrganizer struct {
	hasCompilation bool // if the submission programming language has a compile step

	rcvEvKeys map[string]bool    // tracks whether an event KEY was received
	evsOfType map[string][]Event // unflushed received events by TYPE
	retKeys   map[string]bool    // tracks which event KEYs have been returned from Add method

	expNumOfTests int // expected number of tests
	numFinTests   int // number of received finished or ignored tests

	returnedISE bool // whether internal server error has been returned

	mu sync.Mutex // ensures thread-safe access
}

// NewExecResStreamOrganizer creates a new stream organizer.
// Returns an error if the number of tests is invalid.
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

// Add adds an event to stream organizer to be returned when appropriate.
// Returns events for which this event was a prerequisite and this event if
// the dependencies are met in the correct order. Does not return the same
// event twice if it is determined to have the same key.
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

// HasFinished returns true if the evaluation has finished, i.e.
// 1. an internal server error has been returned;
// 2. the finished testing event has been returned;
// 3. the compilation error event has been returned.
func (o *ExecResStreamOrganizer) HasFinished() bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.returnedISE || o.retKeys[FinishedTestingType] || o.retKeys[CompilationErrorType]
}

func (o *ExecResStreamOrganizer) receiveSubm() ([]Event, error) {
	if !o.rcvEvKeys[ReceivedSubmissionType] {
		return nil, nil
	}

	if o.retKeys[ReceivedSubmissionType] {
		return nil, nil
	}

	e, err := o.getSingleEvent(ReceivedSubmissionType)
	if err != nil {
		return nil, err
	}

	o.retKeys[ReceivedSubmissionType] = true
	res := []Event{e}

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

func (o *ExecResStreamOrganizer) startCompile() ([]Event, error) {
	if !o.hasCompilation {
		return nil, fmt.Errorf("unexpected compile for non-compiled language")
	}

	if !o.rcvEvKeys[StartedCompilationType] || o.retKeys[StartedCompilationType] {
		return nil, nil
	}

	if !o.retKeys[ReceivedSubmissionType] {
		return nil, nil
	}

	e, err := o.getSingleEvent(StartedCompilationType)
	if err != nil {
		return nil, err
	}

	o.retKeys[StartedCompilationType] = true
	res := []Event{e}

	nxt, err := o.finishCompile()
	if err != nil {
		return append(res, nxt...), err
	}
	return append(res, nxt...), nil
}

func (o *ExecResStreamOrganizer) finishCompile() ([]Event, error) {
	if !o.hasCompilation {
		return nil, fmt.Errorf("unexpected compile for non-compiled language")
	}

	if !o.rcvEvKeys[FinishedCompilationType] || o.retKeys[FinishedCompilationType] {
		return nil, nil
	}

	if !o.retKeys[StartedCompilationType] {
		return nil, nil
	}

	e, err := o.getSingleEvent(FinishedCompilationType)
	if err != nil {
		return nil, err
	}

	o.retKeys[FinishedCompilationType] = true
	res := []Event{e}

	// Try both compilation error and start testing paths
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
