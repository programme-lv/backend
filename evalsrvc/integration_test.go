package evalsrvc

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestEnqueueAndReceiveResults verifies that submission is correctly enqueued
// and that ALL corresponding evaluation results are received:
// - started evaluation / received submission
// - started compilation iff the lang needs compilation
// - finished compilation iff the lang needs compilation
// - started testing
// - reached & finished test for every single test
// - finished testing
// - finished evaluation
// This does NOT verify that the messages are processed correctly,
// only that all of them are received.
func TestEnqueueAndReceiveResults(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	submSqsUrl, responseSqsUrl, sqsClient :=
		getSubmSqsUrlFromEnv(), getResponseSqsUrlFromEnv(),
		getSqsClientFromEnv()

	t.Logf("submSqsUrl: %s", submSqsUrl)
	t.Logf("responseSqsUrl: %s", responseSqsUrl)

	tests := []TestFile{
		{InContent: strPtr("1 2"), AnsContent: strPtr("3")},
		{InContent: strPtr("3 4"), AnsContent: strPtr("6")},
	}

	lock := sync.Mutex{}
	msgs := []Msg{}
	unreachedTestIDs := []int64{1, 2}
	unfinishedTestIDs := []int64{1, 2}
	receivedStartedEvaluation := false
	receivedStartedCompilation := false
	receivedFinishedCompilation := false
	receivedStartedTesting := false
	receivedReachedTest := false
	receivedFinishedTesting := false
	receivedFinishedEvaluation := false
	receivedAll := make(chan uuid.UUID, 1)
	inSlice := func(id int64, slice []int64) bool {
		for _, v := range slice {
			if v == id {
				return true
			}
		}
		return false
	}
	removeFromSlice := func(slice []int64, id int64) []int64 {
		for i, v := range slice {
			if v == id {
				return append(slice[:i], slice[i+1:]...)
			}
		}
		return slice
	}
	var evalId uuid.UUID
	preEnqueue := func(e Evaluation) error {
		evalId = e.UUID
		return nil
	}
	everythingExceptTests := false
	allTestsReceived := false
	handle := func(msg Msg) error {
		lock.Lock()
		defer lock.Unlock()
		if everythingExceptTests && allTestsReceived {
			return fmt.Errorf("received message after all tests received: %s", msg.Data.Type())
		}
		if msg.EvalId != evalId {
			log.Printf("received message for wrong eval id: %s, expected: %s", msg.EvalId, evalId)
			return nil
		}
		t.Logf("received message: %s", msg.Data.Type())
		msgs = append(msgs, msg)
		switch msg.Data.Type() {
		case MsgTypeStartedEvaluation:
			receivedStartedEvaluation = true
		case MsgTypeStartedCompilation:
			receivedStartedCompilation = true
		case MsgTypeFinishedCompilation:
			receivedFinishedCompilation = true
		case MsgTypeStartedTesting:
			receivedStartedTesting = true
		case MsgTypeReachedTest:
			reachedTest := msg.Data.(ReachedTest)
			receivedReachedTest = true
			in := inSlice(reachedTest.TestId, unreachedTestIDs)
			require.True(t, in)
			unreachedTestIDs = removeFromSlice(unreachedTestIDs, reachedTest.TestId)
		case MsgTypeFinishedTest:
			finishedTest := msg.Data.(FinishedTest)
			in := inSlice(finishedTest.TestID, unfinishedTestIDs)
			require.True(t, in)
			unfinishedTestIDs = removeFromSlice(unfinishedTestIDs, finishedTest.TestID)
			if len(unfinishedTestIDs) == 0 {
				allTestsReceived = true
			}
		case MsgTypeFinishedTesting:
			receivedFinishedTesting = true
		case MsgTypeFinishedEvaluation:
			receivedFinishedEvaluation = true
		case MsgTypeIgnoredTest:
			ignoredTest := msg.Data.(IgnoredTest)
			in := inSlice(ignoredTest.TestId, unfinishedTestIDs)
			require.True(t, in)
			unfinishedTestIDs = removeFromSlice(unfinishedTestIDs, ignoredTest.TestId)
			if len(unfinishedTestIDs) == 0 {
				allTestsReceived = true
			}
		}
		if receivedStartedEvaluation &&
			receivedStartedTesting && receivedFinishedTesting && receivedFinishedEvaluation {
			everythingExceptTests = true
		}
		if everythingExceptTests && allTestsReceived {
			cancel()
			receivedAll <- evalId
		}
		return nil
	}

	go func() {
		err := receiveResultsFromSqs(ctx,
			responseSqsUrl, sqsClient,
			handle)
		require.NoError(t, err)
	}()

	evalId, err := evaluate(CodeWithLang{
		SrcCode: "a=int(input());b=int(input());print(a+b)",
		LangId:  "python3.11",
	}, tests, TesterParams{
		CpuMs:      100,
		MemKiB:     1024 * 100,
		Checker:    strPtr(checker),
		Interactor: nil,
	}, sqsClient, submSqsUrl, responseSqsUrl, preEnqueue)
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
	require.True(t, receivedFinishedEvaluation)
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
