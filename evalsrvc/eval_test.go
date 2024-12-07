package evalsrvc

import (
	"sync"
	"testing"
)

type EvalEventBus struct {
}

type MockProcessor struct {
	lock sync.Mutex
	msgs []Msg
}

func (m *MockProcessor) Handle(msg Msg) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.msgs = append(m.msgs, msg)
	return nil
}

func NewMockProcessor() *MockProcessor {
	return &MockProcessor{}
}

func TestEnqueueAndReceiveResults(t *testing.T) {
	esrvc := NewEvalSrvc()
	processor := NewMockProcessor()
	esrvc.ReceiveResultsFromSqs(esrvc.resSqsUrl, processor)
	esrvc.NewEvaluation(CodeWithLang{
		SrcCode: "a=int(input());b=int(input());print(a+b)",
		LangId:  "python3.11",
	}, []TestFile{
		{InContent: strPtr("1 2"), AnsContent: strPtr("3")},
		{InContent: strPtr("3 4"), AnsContent: strPtr("6")},
	}, TesterParams{
		CpuMs:      100,
		MemKiB:     1024 * 100,
		Checker:    strPtr(checker),
		Interactor: nil,
	})

	// we want the testing to be finished withing 10 seconds

	// we should listen to some kind of notifier,
	// await a finished evaluation event

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
