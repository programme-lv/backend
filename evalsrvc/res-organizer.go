package evalsrvc

import (
	"fmt"
	"sync"
)

// takes a stream of events and reorders them into the correct order
// for a single evaluation. It also removes duplicate events.
type EvalResOrganizer struct {
	HasCompilation bool // if the subm pr lang has a compilation step
	NumTests       int  // number of tests the task has

	received map[string]bool    // whether KEY was already rcv
	events   map[string][]Event // mapping from TYPE to events
	returned map[string]bool    // KEY returned from Add method
	finTests int                // finished or ignored tests

	returnedISE bool // whether InternalServerError has been returned

	mu sync.Mutex
}

func NewEvalResOrganizer(hasCompilation bool, numTests int) (*EvalResOrganizer, error) {
	if numTests < 0 {
		return nil, fmt.Errorf("numTests must be non-negative")
	}
	if numTests > 1000 {
		return nil, fmt.Errorf("numTests must be less than 1000")
	}
	return &EvalResOrganizer{
		HasCompilation: hasCompilation,
		NumTests:       numTests,
		received:       make(map[string]bool),
		events:         make(map[string][]Event),
		returned:       make(map[string]bool),
		finTests:       0,
	}, nil
}

/*
1. StartedEvaluation
2. StartedCompilation
3. FinishedCompilation
4. StartedTesting
5. ReachedTest
6. FinishedTest
7. IgnoredTest
8. FinishedTesting
9. FinishedEvaluation
*/

func (o *EvalResOrganizer) Add(event Event) ([]Event, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.returnedISE {
		return []Event{}, nil
	}

	key := eventKey(event)

	if o.received[key] {
		return []Event{}, nil
	}
	o.received[key] = true

	o.events[event.Type()] = append(o.events[event.Type()], event)

	switch event.Type() {
	case ReceivedSubmissionType:
		return o.receiveSubm()
	case StartedCompilationType:
		return o.startCompile()
	case FinishedCompilationType:
		return o.finishCompile()
	case StartedTestingType:
		return o.startTesting()
	case ReachedTestType:
		reachedTest, ok := event.(ReachedTest)
		if !ok {
			return []Event{}, fmt.Errorf("event is not a ReachedTest")
		}
		return o.reachTest(reachedTest.TestId)
	case IgnoredTestType:
		ignoredTest, ok := event.(IgnoredTest)
		if !ok {
			return []Event{}, fmt.Errorf("event is not an IgnoredTest")
		}
		return o.ignoreTest(ignoredTest.TestId)
	case FinishedTestType:
		finishedTest, ok := event.(FinishedTest)
		if !ok {
			return []Event{}, fmt.Errorf("event is not a FinishedTest")
		}
		return o.finishTest(finishedTest.TestID)
	case FinishedTestingType:
		return o.finishTesting()
	case CompilationErrorType:
		return o.compileError()
	case InternalServerErrorType:
		o.returnedISE = true
		return []Event{event}, nil
	}

	return []Event{},
		fmt.Errorf("unknown event type: %s", event.Type())
}

func (o *EvalResOrganizer) receiveSubm() ([]Event, error) {
	// skip if evaluation has not been reached yet
	if !o.received[ReceivedSubmissionType] {
		return []Event{}, nil
	}

	// break if evaluation has already been returned
	if o.returned[ReceivedSubmissionType] {
		return []Event{}, nil
	}

	// get the started evaluation event
	e, err := o.getSingleEvent(ReceivedSubmissionType)
	if err != nil {
		return []Event{}, err
	}

	// return this value and check whether next is available
	o.returned[ReceivedSubmissionType] = true
	res := []Event{e}
	var nxt []Event
	if o.HasCompilation {
		nxt, err = o.startCompile()
	} else {
		nxt, err = o.startTesting()
	}
	res = append(res, nxt...)
	if err != nil {
		return res, err
	}
	return res, nil
}

func (o *EvalResOrganizer) startCompile() ([]Event, error) {
	// pr lang is not to be compiled
	if !o.HasCompilation {
		return []Event{}, fmt.Errorf("unexpected compile")
	}

	// compilation has not been received yet
	if !o.received[StartedCompilationType] {
		return []Event{}, nil
	}

	// break if compilation has already been returned
	if o.returned[StartedCompilationType] {
		return []Event{}, nil
	}

	// get the started compilation event
	e, err := o.getSingleEvent(StartedCompilationType)
	if err != nil {
		return []Event{}, err
	}

	// check whether the dependencies are satisfied
	if !o.returned[ReceivedSubmissionType] {
		return []Event{}, nil
	}

	// return this value and check whether next is available
	o.returned[StartedCompilationType] = true
	res := []Event{e}
	nxt, err := o.finishCompile()
	res = append(res, nxt...)
	if err != nil {
		return res, err
	}
	return res, nil
}

func (o *EvalResOrganizer) finishCompile() ([]Event, error) {
	// pr lang is not to be compiled
	if !o.HasCompilation {
		return []Event{}, fmt.Errorf("unexpected compile")
	}

	// if compilation has not been reached yet, return nothing
	if !o.received[FinishedCompilationType] {
		return []Event{}, nil
	}

	// break if compilation has already been returned
	if o.returned[FinishedCompilationType] {
		return []Event{}, nil
	}

	// check whether the dependencies are satisfied
	if !o.returned[StartedCompilationType] {
		return []Event{}, nil
	}

	// get the finished compilation event
	e, err := o.getSingleEvent(FinishedCompilationType)
	if err != nil {
		return []Event{}, err
	}

	// return this value and check whether next is available
	o.returned[FinishedCompilationType] = true
	res := []Event{e}
	var nxt []Event
	nxt, err = o.compileError()
	res = append(res, nxt...)
	if err != nil {
		return res, err
	}
	nxt, err = o.startTesting()
	res = append(res, nxt...)
	if err != nil {
		return res, err
	}
	return res, nil
}

func (o *EvalResOrganizer) startTesting() ([]Event, error) {
	// start testing has not been reached yet
	if !o.received[StartedTestingType] {
		return []Event{}, nil
	}

	// break if testing has already been returned
	if o.returned[StartedTestingType] {
		return []Event{}, nil
	}

	// get the started testing event
	e, err := o.getSingleEvent(StartedTestingType)
	if err != nil {
		return []Event{}, err
	}

	// check whether the dependencies are satisfied
	if o.HasCompilation {
		if !o.returned[FinishedCompilationType] {
			return []Event{}, nil
		}
	} else {
		if !o.returned[ReceivedSubmissionType] {
			return []Event{}, nil
		}
	}

	// return this value and check whether next is available
	o.returned[StartedTestingType] = true
	res := []Event{e}
	nxt, err := o.reachTest(1)
	res = append(res, nxt...)
	if err != nil {
		return res, err
	}
	nxt, err = o.ignoreTest(1)
	res = append(res, nxt...)
	if err != nil {
		return res, err
	}
	return res, nil
}

func (o *EvalResOrganizer) reachTest(id int) ([]Event, error) {
	key := fmt.Sprintf("%s-%d", ReachedTestType, id)

	// if the test has not been reached yet, return nothing
	if !o.received[key] {
		return []Event{}, nil
	}

	// if the test has already been returned, return nothing
	if o.returned[key] {
		return []Event{}, nil
	}

	// get the reached test event
	e, err := o.getReachedTestEvent(id)
	if err != nil {
		return []Event{}, err
	}

	// check whether the dependencies are satisfied
	if !o.returned[StartedTestingType] {
		return []Event{}, nil
	}
	if id > 1 {
		reachedKey := fmt.Sprintf("%s-%d", ReachedTestType, id-1)
		if !o.returned[reachedKey] {
			return []Event{}, nil
		}
		finishedKey := fmt.Sprintf("%s-%d", FinishedTestType, id-1)
		if !o.returned[finishedKey] {
			return []Event{}, nil
		}
	}

	// return this value and check whether next is available
	o.returned[key] = true
	res := []Event{e}
	var nxt []Event
	nxt, err = o.finishTest(id)
	res = append(res, nxt...)
	if err != nil {
		return res, err
	}
	return res, nil
}

func (o *EvalResOrganizer) ignoreTest(id int) ([]Event, error) {
	key := fmt.Sprintf("%s-%d", IgnoredTestType, id)

	// if the test has not been reached yet, return nothing
	if !o.received[key] {
		return []Event{}, nil
	}

	// if the test has already been returned, return nothing
	if o.returned[key] {
		return []Event{}, nil
	}

	// check whether the dependencies are satisfied
	if !o.returned[StartedTestingType] {
		return []Event{}, nil
	}
	if id > 1 {
		reachedKey := fmt.Sprintf("%s-%d", ReachedTestType, id-1)
		if !o.returned[reachedKey] {
			return []Event{}, nil
		}
	}

	e, err := o.getIgnoredTestEvent(id)
	if err != nil {
		return []Event{}, err
	}

	// return this value and check whether next is available
	o.returned[key] = true
	o.finTests++
	res := []Event{e}
	var nxt []Event
	if id < o.NumTests {
		nxt, err = o.ignoreTest(id + 1)
		res = append(res, nxt...)
		if err != nil {
			return res, err
		}
		nxt, err = o.reachTest(id + 1)
		res = append(res, nxt...)
		if err != nil {
			return res, err
		}
	}
	nxt, err = o.finishTesting()
	res = append(res, nxt...)
	if err != nil {
		return res, err
	}
	return res, nil
}

func (o *EvalResOrganizer) finishTest(id int) ([]Event, error) {
	if id < 1 || id > o.NumTests {
		return []Event{}, fmt.Errorf("invalid test id: %d", id)
	}
	key := fmt.Sprintf("%s-%d", FinishedTestType, id)

	// if the event has not been received yet, return nothing
	if !o.received[key] {
		return []Event{}, nil
	}

	// if the test has already been returned, return nothing
	if o.returned[key] {
		return []Event{}, nil
	}

	// check whether the dependencies are satisfied
	reachedKey := fmt.Sprintf("%s-%d", ReachedTestType, id)
	if !o.returned[reachedKey] {
		return []Event{}, nil
	}

	e, err := o.getFinishedTestEvent(id)
	if err != nil {
		return []Event{}, err
	}

	// return this value and check whether next is available
	o.returned[key] = true
	o.finTests++
	res := []Event{e}
	var nxt []Event
	if id < o.NumTests {
		nxt, err = o.reachTest(id + 1)
		res = append(res, nxt...)
		if err != nil {
			return res, err
		}
	}
	nxt, err = o.finishTesting()
	res = append(res, nxt...)
	if err != nil {
		return res, err
	}
	return res, nil
}

func (o *EvalResOrganizer) finishTesting() ([]Event, error) {
	key := FinishedTestingType

	// if the event has not been received yet, return nothing
	if !o.received[key] {
		return []Event{}, nil
	}

	// if the event has already been returned, return nothing
	if o.returned[key] {
		return []Event{}, nil
	}

	// if not all tests have been finished, return nothing
	if o.finTests < o.NumTests {
		return []Event{}, nil
	}

	e, err := o.getSingleEvent(key)
	if err != nil {
		return []Event{}, err
	}

	// return this value and check whether next is available
	o.returned[key] = true
	res := []Event{e}
	return res, nil
}

func (o *EvalResOrganizer) compileError() ([]Event, error) {
	key := CompilationErrorType

	// if the event has not been received yet, return nothing
	if !o.received[key] {
		return []Event{}, nil
	}

	// if the event has already been returned, return nothing
	if o.returned[key] {
		return []Event{}, nil
	}

	// check whether the dependencies are satisfied
	if !o.returned[FinishedCompilationType] {
		return []Event{}, nil
	}

	// get the compilation error event
	e, err := o.getSingleEvent(key)
	if err != nil {
		return []Event{}, err
	}

	// return this value and check whether next is available
	o.returned[key] = true
	res := []Event{e}
	return res, nil
}

func (o *EvalResOrganizer) internalServerError() ([]Event, error) {
	key := InternalServerErrorType

	// if the event has not been received yet, return nothing
	if !o.received[key] {
		return []Event{}, nil
	}

	// if the event has already been returned, return nothing
	if o.returned[key] {
		return []Event{}, nil
	}

	// get the internal server error event
	e, err := o.getSingleEvent(key)
	if err != nil {
		return []Event{}, err
	}

	// return this value and check whether next is available
	o.returned[key] = true
	res := []Event{e}
	return res, nil
}

func (o *EvalResOrganizer) getReachedTestEvent(id int) (Event, error) {
	events, ok := o.events[ReachedTestType]
	if !ok {
		return nil, fmt.Errorf("no events for key: %s, id: %d", ReachedTestType, id)
	}
	for _, event := range events {
		reachedTest, ok := event.(ReachedTest)
		if !ok {
			return nil, fmt.Errorf("event is not a ReachedTest")
		}
		if reachedTest.TestId == id {
			return event, nil
		}
	}
	return nil, fmt.Errorf("no event for id: %d", id)
}

func (o *EvalResOrganizer) getIgnoredTestEvent(id int) (Event, error) {
	events, ok := o.events[IgnoredTestType]
	if !ok {
		return nil, fmt.Errorf("no events for key: %s", IgnoredTestType)
	}
	for _, event := range events {
		ignoredTest, ok := event.(IgnoredTest)
		if !ok {
			return nil, fmt.Errorf("event is not an IgnoredTest")
		}
		if ignoredTest.TestId == id {
			return event, nil
		}
	}
	return nil, fmt.Errorf("no event for id: %d", id)
}

func (o *EvalResOrganizer) getFinishedTestEvent(id int) (Event, error) {
	events, ok := o.events[FinishedTestType]
	if !ok {
		return nil, fmt.Errorf("no events for key: %s", FinishedTestType)
	}
	for _, event := range events {
		finishedTest, ok := event.(FinishedTest)
		if !ok {
			return nil, fmt.Errorf("event is not a FinishedTest")
		}
		if finishedTest.TestID == id {
			return event, nil
		}
	}
	return nil, fmt.Errorf("no event for id: %d", id)
}

func (o *EvalResOrganizer) getSingleEvent(eventType string) (Event, error) {
	events, ok := o.events[eventType]
	if !ok {
		return nil, fmt.Errorf("no events for key: %s", eventType)
	}
	if len(events) > 1 {
		return nil, fmt.Errorf("multiple events for key: %s", eventType)
	}
	return events[0], nil
}

func eventKey(event Event) string {
	switch event.Type() {
	case ReachedTestType:
		reachedTest, ok := event.(ReachedTest)
		if !ok {
			return ""
		}
		return fmt.Sprintf("%s-%d", ReachedTestType, reachedTest.TestId)
	case IgnoredTestType:
		ignoredTest, ok := event.(IgnoredTest)
		if !ok {
			return ""
		}
		return fmt.Sprintf("%s-%d", IgnoredTestType, ignoredTest.TestId)
	case FinishedTestType:
		finishedTest, ok := event.(FinishedTest)
		if !ok {
			return ""
		}
		return fmt.Sprintf("%s-%d", FinishedTestType, finishedTest.TestID)
	default:
		return event.Type()
	}
}
