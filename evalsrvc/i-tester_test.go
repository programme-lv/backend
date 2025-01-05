package evalsrvc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestEnqueueAndReceiveResults verifies the complete evaluation lifecycle:
// 1. Submission enqueuing
// 2. Event emission and ordering:
//   - Evaluation start
//   - Compilation phase (language-dependent)
//   - Test execution phase
//   - Individual test results
//   - Evaluation completion
//
// The test focuses on event completeness rather than correctness of event processing.
// It ensures all expected events are received for both compiled and interpreted languages.
func TestEnqueueAndReceiveResults(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	submSqsUrl, responseSqsUrl, sqsClient :=
		getSubmSqsUrlFromEnv(), getResponseSqsUrlFromEnv(),
		getSqsClientFromEnvNoLogging()

	t.Logf("subm queue: %s", strings.Split(submSqsUrl, "/")[4])
	t.Logf("resp queue: %s", strings.Split(responseSqsUrl, "/")[4])

	tests := []TestFile{
		{InContent: strPtr("1 2"), AnsContent: strPtr("3")},
		{InContent: strPtr("3 4"), AnsContent: strPtr("6")},
	}

	lock := sync.Mutex{}
	msgs := []SqsResponseMsg{}
	unreachedTestIDs := []int{1, 2}
	unfinishedTestIDs := []int{1, 2}
	receivedStartedEvaluation := false
	receivedStartedCompilation := false
	receivedFinishedCompilation := false
	receivedStartedTesting := false
	receivedReachedTest := false
	receivedFinishedTesting := false
	receivedAll := make(chan uuid.UUID, 1)
	inSlice := func(id int, slice []int) bool {
		for _, v := range slice {
			if v == id {
				return true
			}
		}
		return false
	}
	removeFromSlice := func(slice []int, id int) []int {
		for i, v := range slice {
			if v == id {
				return append(slice[:i], slice[i+1:]...)
			}
		}
		return slice
	}
	evalId, err := uuid.NewV7()
	require.NoError(t, err)

	everythingExceptTests := false
	allTestsReceived := false
	handle := func(msg SqsResponseMsg) error {
		lock.Lock()
		defer lock.Unlock()
		if everythingExceptTests && allTestsReceived {
			return fmt.Errorf("received message after all tests received: %s", msg.Data.Type())
		}
		if msg.EvalId != evalId {
			t.Logf("received msg for wrong eval: %s", msg.EvalId)
			return nil
		}
		t.Logf("received msg: %s", msg.Data.Type())
		msgs = append(msgs, msg)
		switch msg.Data.Type() {
		case ReceivedSubmissionType:
			receivedStartedEvaluation = true
		case StartedCompilationType:
			receivedStartedCompilation = true
		case FinishedCompilationType:
			receivedFinishedCompilation = true
		case StartedTestingType:
			receivedStartedTesting = true
		case ReachedTestType:
			reachedTest := msg.Data.(ReachedTest)
			receivedReachedTest = true
			in := inSlice(reachedTest.TestId, unreachedTestIDs)
			require.True(t, in)
			unreachedTestIDs = removeFromSlice(unreachedTestIDs, reachedTest.TestId)
		case FinishedTestType:
			finishedTest := msg.Data.(FinishedTest)
			in := inSlice(int(finishedTest.TestID), unfinishedTestIDs)
			require.True(t, in)
			unfinishedTestIDs = removeFromSlice(unfinishedTestIDs, int(finishedTest.TestID))
			if len(unfinishedTestIDs) == 0 {
				allTestsReceived = true
			}
		case FinishedTestingType:
			receivedFinishedTesting = true
		case IgnoredTestType:
			ignoredTest := msg.Data.(IgnoredTest)
			in := inSlice(int(ignoredTest.TestId), unfinishedTestIDs)
			require.True(t, in)
			unfinishedTestIDs = removeFromSlice(unfinishedTestIDs, int(ignoredTest.TestId))
			if len(unfinishedTestIDs) == 0 {
				allTestsReceived = true
			}
		}
		if receivedStartedEvaluation &&
			receivedStartedTesting && receivedFinishedTesting {
			everythingExceptTests = true
		}
		if everythingExceptTests && allTestsReceived {
			receivedAll <- evalId
			cancel()
		}
		return nil
	}

	go func() {
		err := receiveResultsFromSqs(ctx,
			responseSqsUrl, sqsClient,
			handle,
			slog.Default(),
		)
		if err != nil && err != context.Canceled {
			require.NoError(t, err)
		}
	}()

	lang, err := getPrLangById("python3.11")
	require.NoError(t, err)

	err = enqueue(evalId,
		"a=int(input());b=int(input());print(a+b)",
		lang,
		tests,
		TesterParams{
			CpuMs:      100,
			MemKiB:     1024 * 100,
			Checker:    strPtr(checker),
			Interactor: nil,
		},
		sqsClient,
		submSqsUrl,
		responseSqsUrl,
	)
	require.NoError(t, err)

	timeout := time.After(30 * time.Second)
	select {
	case <-timeout:
		t.Fatal("timed out")
	case e := <-receivedAll:
		require.Equal(t, evalId, e)
	}

	require.True(t, receivedStartedEvaluation)
	require.False(t, receivedStartedCompilation)
	require.False(t, receivedFinishedCompilation)
	require.True(t, receivedStartedTesting)
	require.True(t, receivedReachedTest)
	require.True(t, receivedFinishedTesting)
	require.Empty(t, unreachedTestIDs)
	require.Empty(t, unfinishedTestIDs)
}

func strPtr(s string) *string {
	return &s
}

const checker = `
#include "testlib.h"

using namespace std;

int main(int argc, char *argv[]) {
    setName("compare sequences of tokens");
    registerTestlibCmd(argc, argv);

    int n = 0;
    string j, p;

    while (!ans.seekEof() && !ouf.seekEof()) {
        n++;

        ans.readWordTo(j);
        ouf.readWordTo(p);

        if (j != p)
            quitf(_wa, "%d%s words differ - expected: '%s', found: '%s'", n, englishEnding(n).c_str(),
                  compress(j).c_str(), compress(p).c_str());
    }

    if (ans.seekEof() && ouf.seekEof()) {
        if (n == 1)
            quitf(_ok, "\"%s\"", compress(j).c_str());
        else
            quitf(_ok, "%d tokens", n);
    } else {
        if (ans.seekEof())
            quitf(_wa, "Participant output contains extra tokens");
        else
            quitf(_wa, "Unexpected EOF in the participants output");
    }
}
`
