package evalsrvc_test

import (
	"context"
	"testing"
	"time"

	"github.com/programme-lv/backend/evalsrvc"
	"github.com/stretchr/testify/require"
)

// TestEvalServiceCmpListenNoCompile tests the end-to-end evaluation flow:
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
func TestEvalServiceCmpListenNoCompile(t *testing.T) {
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
	require.Len(t, events, 7)
	require.Equal(t, events[0].Type(), evalsrvc.ReceivedSubmissionType)
	require.Equal(t, events[1].Type(), evalsrvc.StartedTestingType)
	require.Equal(t, events[2].Type(), evalsrvc.ReachedTestType)
	require.Equal(t, events[3].Type(), evalsrvc.FinishedTestType)
	require.Equal(t, events[4].Type(), evalsrvc.ReachedTestType)
	require.Equal(t, events[5].Type(), evalsrvc.FinishedTestType)
	require.Equal(t, events[6].Type(), evalsrvc.FinishedTestingType)
}

func TestEvalServiceCmpListenWithCompile(t *testing.T) {
	// 1. enqueue a submission
	srvc := evalsrvc.NewEvalSrvc()
	evalId, err := srvc.Enqueue(evalsrvc.CodeWithLang{
		SrcCode: "#include <iostream>\nint main() {int a,b;std::cin>>a>>b;std::cout<<a+b<<std::endl;}",
		LangId:  "cpp17",
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
	require.Len(t, events, 9)
	expectedEvents := []string{
		evalsrvc.ReceivedSubmissionType,
		evalsrvc.StartedCompilationType,
		evalsrvc.FinishedCompilationType,
		evalsrvc.StartedTestingType,
		evalsrvc.ReachedTestType,
		evalsrvc.FinishedTestType,
		evalsrvc.ReachedTestType,
		evalsrvc.FinishedTestType,
		evalsrvc.FinishedTestingType,
	}
	for i, ev := range events {
		require.Equal(t, expectedEvents[i], ev.Type())
	}
}

func TestEvalServiceCmpGet(t *testing.T) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	eval, err := srvc.Get(ctx, evalId)
	require.NoError(t, err)
	require.Equal(t, eval.Stage, evalsrvc.EvalStageFinished)
}

func strPtr(s string) *string {
	return &s
}
