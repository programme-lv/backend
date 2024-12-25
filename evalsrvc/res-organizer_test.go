package evalsrvc

import (
	"math/rand/v2"
	"strconv"
	"testing"
	"time"
)

// test that we achieve the same result no matter the order or received events
func TestProcessResultsAnyOrderWithCompilation(t *testing.T) {
	events := []Event{
		StartedEvaluation{SysInfo: "some sys info", StartedAt: time.Now()},
		StartedCompiling{},
		FinishedCompiling{RuntimeData: getExRunDataEv()},
		StartedTesting{},
		ReachedTest{TestId: 1, In: getExStrPtr(), Ans: getExStrPtr()},
		FinishedTest{TestID: 1, Subm: getExRunDataEv(), Checker: getExRunDataEv()},
		ReachedTest{TestId: 2, In: getExStrPtr(), Ans: getExStrPtr()},
		FinishedTest{TestID: 2, Subm: getExRunDataEv(), Checker: getExRunDataEv()},
		FinishedTesting{},
	}

	shuffleAndCmp(t, events)
}

func TestProcessResultsAnyOrderNoCompilation(t *testing.T) {
	events := []Event{
		StartedEvaluation{SysInfo: "some sys info", StartedAt: time.Now()},
		StartedTesting{},
		ReachedTest{TestId: 1, In: getExStrPtr(), Ans: getExStrPtr()},
		FinishedTest{TestID: 1, Subm: getExRunDataEv(), Checker: getExRunDataEv()},
		ReachedTest{TestId: 2, In: getExStrPtr(), Ans: getExStrPtr()},
		FinishedTest{TestID: 2, Subm: getExRunDataEv(), Checker: getExRunDataEv()},
		FinishedTesting{},
	}

	shuffleAndCmp(t, events)
}

func TestProcessResultsAnyOrderCompilationError(t *testing.T) {
	events := []Event{
		StartedEvaluation{SysInfo: "some sys info", StartedAt: time.Now()},
		StartedCompiling{},
		FinishedCompiling{RuntimeData: getExRunDataEv()},
		CompilationError{ErrorMsg: getExStrPtr()},
	}

	shuffleAndCmp(t, events)
}

func TestProcessResultsAnyOrderInternalServerError(t *testing.T) {
	events := []Event{
		StartedEvaluation{SysInfo: "some sys info", StartedAt: time.Now()},
		StartedCompiling{},
		FinishedCompiling{RuntimeData: getExRunDataEv()},
		InternalServerError{ErrorMsg: getExStrPtr()},
	}
	shuffleAndCmp(t, events)
}

func shuffleAndCmp(t *testing.T, events []Event) {
	hasCompilation := false
	for _, event := range events {
		if event.Type() == StartedCompilationType {
			hasCompilation = true
			break
		}
	}

	numTests := 0
	for _, event := range events {
		if event.Type() == ReachedTestType {
			numTests++
		}
	}

	for i := 0; i < 100; i++ {
		shuffled := make([]Event, len(events))
		copy(shuffled, events)
		// random shuffle events
		rand.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		organizer, err := NewEvalResOrganizer(hasCompilation, numTests)
		if err != nil {
			t.Fatalf("error creating organizer: %v", err)
		}

		received := []Event{}
		for _, event := range shuffled {
			res, err := organizer.Add(event)
			if err != nil {
				t.Fatalf("error adding event: %v, event: %v", err, event)
			}
			received = append(received, res...)
		}

		if len(received) != len(events) {
			t.Fatalf("received %d events, expected %d", len(received), len(events))
		}

		for i := range received {
			if received[i] != events[i] {
				t.Fatalf("received event %v (%T), expected %v (%T)", received[i], received[i], events[i], events[i])
			}
		}
	}
}

func getExRunDataEv() *RunData {
	return &RunData{
		StdIn:    "some std in",
		StdOut:   "some std out",
		StdErr:   "some std err",
		CpuMs:    1 + int64(rand.IntN(100)),
		WallMs:   2 + int64(rand.IntN(100)),
		MemKiB:   3 + int64(rand.IntN(100)),
		ExitCode: 4 + int64(rand.IntN(100)),
		CtxSwV:   nil,
		CtxSwF:   nil,
		Signal:   nil,
	}
}

func getExStrPtr() *string {
	s := "some string" + strconv.Itoa(rand.IntN(100))
	return &s
}
