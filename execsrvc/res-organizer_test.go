package execsrvc

import (
	"math/rand/v2"
	"strconv"
	"testing"
	"time"
)

// Tests that event processing produces consistent
// results regardless of event arrival order.
// Tests submission with compilation step.
func TestProcessResultsAnyOrderWithCompilation(
	t *testing.T,
) {
	events := []Event{
		ReceivedSubmission{
			SysInfo:   "some sys info",
			StartedAt: time.Now(),
		},
		StartedCompiling{},
		FinishedCompiling{
			RuntimeData: getExampleRunData(),
		},
		StartedTesting{},
		ReachedTest{
			TestId: 1,
			In:     getExampleStrPtr(),
			Ans:    getExampleStrPtr(),
		},
		FinishedTest{
			TestID:  1,
			Subm:    getExampleRunData(),
			Checker: getExampleRunData(),
		},
		ReachedTest{
			TestId: 2,
			In:     getExampleStrPtr(),
			Ans:    getExampleStrPtr(),
		},
		FinishedTest{
			TestID:  2,
			Subm:    getExampleRunData(),
			Checker: getExampleRunData(),
		},
		FinishedTesting{},
	}

	shuffleAndCmp(t, events, true, 2)
}

// Tests that event processing produces consistent
// results regardless of event arrival order.
// Tests submission without compilation step.
func TestProcessResultsAnyOrderNoCompilation(
	t *testing.T,
) {
	events := []Event{
		ReceivedSubmission{
			SysInfo:   "some sys info",
			StartedAt: time.Now(),
		},
		StartedTesting{},
		ReachedTest{
			TestId: 1,
			In:     getExampleStrPtr(),
			Ans:    getExampleStrPtr(),
		},
		FinishedTest{
			TestID:  1,
			Subm:    getExampleRunData(),
			Checker: getExampleRunData(),
		},
		ReachedTest{
			TestId: 2,
			In:     getExampleStrPtr(),
			Ans:    getExampleStrPtr(),
		},
		FinishedTest{
			TestID:  2,
			Subm:    getExampleRunData(),
			Checker: getExampleRunData(),
		},
		FinishedTesting{},
	}

	shuffleAndCmp(t, events, false, 2)
}

// Tests that compilation errors are handled
// correctly regardless of event arrival order.
func TestProcessResultsAnyOrderCompilationError(
	t *testing.T,
) {
	events := []Event{
		ReceivedSubmission{
			SysInfo:   "some sys info",
			StartedAt: time.Now(),
		},
		StartedCompiling{},
		FinishedCompiling{
			RuntimeData: getExampleRunData(),
		},
		CompilationError{
			ErrorMsg: getExampleStrPtr(),
		},
	}

	shuffleAndCmp(t, events, true, 2)
}

// Tests that internal server errors are handled
// correctly regardless of event arrival order.
func TestProcessResultsInternalServerError(
	t *testing.T,
) {
	events := []Event{
		ReceivedSubmission{
			SysInfo:   "some sys info",
			StartedAt: time.Now(),
		},
		StartedCompiling{},
		FinishedCompiling{
			RuntimeData: getExampleRunData(),
		},
		InternalServerError{
			ErrorMsg: getExampleStrPtr(),
		},
	}
	shuffleAndCmp(t, events, true, 2)
}

// Helper function that tests event processing by:
// 1. Creating multiple random permutations of events
// 2. Processing each permutation
// 3. Verifying results match original event order
func shuffleAndCmp(
	t *testing.T,
	events []Event,
	hasCompilation bool,
	numTests int,
) {
	for i := 0; i < 100; i++ {
		shuffled := make([]Event, len(events))
		copy(shuffled, events)

		rand.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j],
				shuffled[i]
		})

		organizer, err := NewExecResStreamOrganizer(
			hasCompilation,
			numTests,
		)
		if err != nil {
			t.Fatalf(
				"error creating organizer: %v",
				err,
			)
		}

		received := []Event{}
		for _, event := range shuffled {
			res, err := organizer.Add(event)
			if err != nil {
				t.Fatalf(
					"error adding event: %v, "+
						"event: %v",
					err,
					event,
				)
			}
			received = append(received, res...)
		}

		if received[len(received)-1].Type() ==
			InternalServerErrorType {
			continue
		}

		if len(received) != len(events) {
			t.Fatalf(
				"received %d events, expected %d",
				len(received),
				len(events),
			)
		}

		for i := range received {
			if received[i] != events[i] {
				t.Fatalf(
					"received event %v (%T), "+
						"expected %v (%T)",
					received[i],
					received[i],
					events[i],
					events[i],
				)
			}
		}
	}
}

// Helper that generates random run data for tests
func getExampleRunData() *RunData {
	return &RunData{
		StdIn:    "some std in",
		StdOut:   "some std out",
		StdErr:   "some std err",
		CpuMs:    1 + int64(rand.IntN(100)),
		WallMs:   2 + int64(rand.IntN(100)),
		MemKiB:   3 + int64(rand.IntN(100)),
		ExitCode: 4 + int64(rand.IntN(100)),
		CtxSwV:   rand.Int64(),
		CtxSwF:   rand.Int64(),
		Signal:   nil,
	}
}

// Helper that generates random string pointer
func getExampleStrPtr() *string {
	s := strconv.Itoa(rand.IntN(100))
	return &s
}
