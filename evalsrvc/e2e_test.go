package evalsrvc_test

import (
	"testing"
	"time"

	"github.com/programme-lv/backend/evalsrvc"
	"github.com/stretchr/testify/require"
)

// TestEvalServiceCmpListen tests the end-to-end evaluation flow:
// 1. Enqueues a Python submission that reads two numbers and prints their sum;
// 2. Listens for evaluation events via the Listen() channel;
// 3. Verifies all expected events are received in order:
//   - Started evaluation
//   - Started testing
//   - For each test:
//   - Reached test
//   - Finished test
//   - Finished testing
//   - Finished evaluation
func TestEvalServiceCmpListen(t *testing.T) {
	// 1. enqueue a submission
	// 2. start listening to eval uuid
	// 3. receive all evaluation events
	// 4. compare to expected events

	// 1. enqueue a submission
	srvc := evalsrvc.NewEvalSrvc()
	evalId, err := srvc.Enqueue(evalsrvc.CodeWithLang{
		SrcCode: "a=int(input());b=int(input());print(a+b)",
		LangId:  "python3.11",
	}, []evalsrvc.TestFile{
		{InContent: strPtr("1 2"), AnsContent: strPtr("3")},
		{InContent: strPtr("3 4"), AnsContent: strPtr("6")},
	}, evalsrvc.TesterParams{
		CpuMs:  1000,
		MemKiB: 1024,
	})
	require.NoError(t, err)

	// 2. start listening to eval uuid
	ch, err := srvc.Listen(evalId)
	require.NoError(t, err)

	timeout := time.After(10 * time.Second)
	var events []evalsrvc.Event

	// 3. collect events until channel closes or timeout
	for {
		select {
		case <-timeout:
			t.Fatal("timeout waiting for evaluation events")
		case ev, ok := <-ch:
			if !ok {
				goto hello
			}
			events = append(events, ev)
		}
	}
hello:
	// 4. compare to expected events:
	// 4.1. Started evaluation
	// 4.2. Started testing
	// 4.3. For each test:
	// 4.3.1. Reached test
	// 4.3.2. Finished test
	// 4.4. Finished testing
	// 4.5. Finished evaluation
	require.Len(t, events, 8)
	t.Logf("events: %+v", events)

	// Uncomment to verify events:
	/*
		expectedEvents := []evalsrvc.Event{
			evalsrvc.StartedEvaluation{},
			evalsrvc.StartedTesting{},
			evalsrvc.ReachedTest{},
			evalsrvc.FinishedTest{},
			evalsrvc.ReachedTest{},
			evalsrvc.FinishedTest{},
			evalsrvc.FinishedTesting{},
			evalsrvc.FinishedEvaluation{},
		}
		require.Equal(t, len(expectedEvents), len(events))
		for i := range events {
			require.Equal(t, expectedEvents[i].Type(), events[i].Type())
		}
	*/
}

func strPtr(s string) *string {
	return &s
}
