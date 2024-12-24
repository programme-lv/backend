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

	mu sync.Mutex
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

	key := eventKey(event)

	if o.received[key] {
		return []Event{}, nil
	}
	o.received[key] = true

	o.events[key] = append(o.events[key], event)

	switch event.Type() {
	case StartedEvaluationType:
		return o.startEval()
	case StartedCompilationType:
		return o.startCompile()
	case FinishedCompilationType:
		return o.finishCompile()
	}

	return []Event{},
		fmt.Errorf("unknown event type: %s", event.Type())
}

func (o *EvalResOrganizer) startEval() ([]Event, error) {
	// skip if evaluation has not been reached yet
	if !o.received[StartedEvaluationType] {
		return []Event{}, nil
	}

	// break if evaluation has already been returned
	if o.returned[StartedEvaluationType] {
		return []Event{}, nil
	}

	// get the started evaluation event
	e, err := o.getSingleEvent(StartedEvaluationType)
	if err != nil {
		return []Event{}, err
	}

	// return this value and check whether next is available
	o.returned[StartedEvaluationType] = true
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
	if !o.returned[StartedEvaluationType] {
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
	nxt, err := o.startTesting()
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
		if !o.received[FinishedCompilationType] {
			return []Event{}, nil
		}
	} else {
		if !o.received[StartedEvaluationType] {
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
	}

	// return this value and check whether next is available
	o.returned[key] = true
	res := []Event{e}
	var nxt []Event
	if id < o.NumTests {
		nxt, err = o.reachTest(id + 1)
		res = append(res, nxt...)
		if err != nil {
			return res, err
		}
	}
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

	// return this value and check whether next is available
	o.returned[key] = true
	res := []Event{}
	nxt, err := o.finishTesting()
	res = append(res, nxt...)
	if err != nil {
		return res, err
	}
	return res, nil
}

func (o *EvalResOrganizer) finishTesting() ([]Event, error) {
	return []Event{}, nil
}

func (o *EvalResOrganizer) getReachedTestEvent(id int) (Event, error) {
	events, ok := o.events[ReachedTestType]
	if !ok {
		return nil, fmt.Errorf("no events for key: %s", ReachedTestType)
	}
	for _, event := range events {
		if event.(ReachedTest).TestId == id {
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
		if event.(IgnoredTest).TestId == id {
			return event, nil
		}
	}
	return nil, fmt.Errorf("no event for id: %d", id)
}

func (o *EvalResOrganizer) getSingleEvent(key string) (Event, error) {
	events, ok := o.events[key]
	if !ok {
		return nil, fmt.Errorf("no events for key: %s", key)
	}
	if len(events) > 1 {
		return nil, fmt.Errorf("multiple events for key: %s", key)
	}
	return events[0], nil
}

func eventKey(event Event) string {
	switch event.Type() {
	case ReachedTestType:
		reachedTest := event.(ReachedTest)
		return fmt.Sprintf("%s-%d", ReachedTestType, reachedTest.TestId)
	case IgnoredTestType:
		ignoredTest := event.(IgnoredTest)
		return fmt.Sprintf("%s-%d", IgnoredTestType, ignoredTest.TestId)
	case FinishedTestType:
		finishedTest := event.(FinishedTest)
		return fmt.Sprintf("%s-%d", FinishedTestType, finishedTest.TestID)
	}
	return event.Type()
}
