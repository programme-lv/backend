package evalsrvc

import (
	"math/rand/v2"
	"testing"
	"time"
)

// test that we achieve the same result no matter the order or received events
func TestProcessResultsAnyOrderOkCpp(t *testing.T) {
	events := []Event{
		StartedEvaluation{
			SysInfo:   "",
			StartedAt: time.Time{},
		},
		StartedCompiling{},
		FinishedCompiling{
			RuntimeData: &RunData{
				StdIn:    "",
				StdOut:   "",
				StdErr:   "",
				CpuMs:    0,
				WallMs:   0,
				MemKiB:   0,
				ExitCode: 0,
				CtxSwV:   new(int64),
				CtxSwF:   new(int64),
				Signal:   new(int64),
			},
		},
		StartedTesting{},
		ReachedTest{
			TestId: 0,
			In:     new(string),
			Ans:    new(string),
		},
		FinishedTest{
			TestID:  0,
			Subm:    &RunData{},
			Checker: &RunData{},
		},
		FinishedTesting{}, // useless event
		FinishedEvaluation{
			CompileError:  false,
			InternalError: false,
			ErrorMsg:      new(string),
		},
	}

	for i := 0; i < 100; i++ {
		// random shuffle events
		rand.Shuffle(len(events), func(i, j int) {
			events[i], events[j] = events[j], events[i]
		})
	}

}
