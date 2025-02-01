package execsrvc_test

import (
	"context"
	"log"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/stretchr/testify/require"
)

func init() {
	// Load .env file from the execsrvc directory
	err := godotenv.Load("../.env")
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}
}

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
	srvc := execsrvc.NewExecSrvc()
	execUuid := uuid.New()
	err := srvc.Enqueue(context.Background(), execUuid, "a=int(input());b=int(input());print(a+b)", "python3.11", []execsrvc.TestFile{
		{InContent: strPtr("1 2"), AnsContent: strPtr("3")},
		{InContent: strPtr("3 4"), AnsContent: strPtr("6")},
	}, execsrvc.TestingParams{
		CpuMs:  1000,
		MemKiB: 1024,
	})
	require.NoError(t, err)

	// 2. start listening to eval uuid
	ch, err := srvc.Listen(context.Background(), execUuid)
	require.NoError(t, err)

	timeout := time.After(30 * time.Second)
	var events []execsrvc.Event

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
	require.Equal(t, events[0].Type(), execsrvc.ReceivedSubmissionType)
	require.Equal(t, events[1].Type(), execsrvc.StartedTestingType)
	require.Equal(t, events[2].Type(), execsrvc.ReachedTestType)
	require.Equal(t, events[3].Type(), execsrvc.FinishedTestType)
	require.Equal(t, events[4].Type(), execsrvc.ReachedTestType)
	require.Equal(t, events[5].Type(), execsrvc.FinishedTestType)
	require.Equal(t, events[6].Type(), execsrvc.FinishedTestingType)

	srvc.Close()
}

func TestEvalServiceCmpListenWithCompile(t *testing.T) {
	// 1. enqueue a submission
	srvc := execsrvc.NewExecSrvc()
	execUuid := uuid.New()
	err := srvc.Enqueue(context.Background(), execUuid, "#include <iostream>\nint main() {int a,b;std::cin>>a>>b;std::cout<<a+b<<std::endl;}", "cpp17", []execsrvc.TestFile{
		{InContent: strPtr("1 2"), AnsContent: strPtr("3")},
		{InContent: strPtr("3 4"), AnsContent: strPtr("6")},
	}, execsrvc.TestingParams{
		CpuMs:  1000,
		MemKiB: 1024,
	})
	require.NoError(t, err)

	// 2. start listening to eval uuid
	ch, err := srvc.Listen(context.Background(), execUuid)
	require.NoError(t, err)

	timeout := time.After(30 * time.Second)
	var events []execsrvc.Event

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
		execsrvc.ReceivedSubmissionType,
		execsrvc.StartedCompilationType,
		execsrvc.FinishedCompilationType,
		execsrvc.StartedTestingType,
		execsrvc.ReachedTestType,
		execsrvc.FinishedTestType,
		execsrvc.ReachedTestType,
		execsrvc.FinishedTestType,
		execsrvc.FinishedTestingType,
	}
	for i, ev := range events {
		require.Equal(t, expectedEvents[i], ev.Type())
	}
	srvc.Close()
}

// test the asynchronocity of the Get() method and persistence after closing the srvc
func TestEvalServiceCmpGet(t *testing.T) {
	srvc := execsrvc.NewExecSrvc()
	execUuid := uuid.New()
	err := srvc.Enqueue(context.Background(), execUuid, "a=int(input());b=int(input());print(a+b)", "python3.10", []execsrvc.TestFile{
		{InContent: strPtr("1\n2\n"), AnsContent: strPtr("3\n")},
		{InContent: strPtr("3\n4\n"), AnsContent: strPtr("6\n")},
	}, execsrvc.TestingParams{
		CpuMs:  1000,
		MemKiB: 20024,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	eval, err := srvc.Get(ctx, execUuid)
	require.NoError(t, err)
	srvc.Close()
	require.Equal(t, eval.Stage, execsrvc.StageFinished)
	require.Nil(t, eval.ErrorMsg)
	require.Len(t, eval.TestRes, 2)
	require.Equal(t, strPtr("1\n2\n"), eval.TestRes[0].Input)
	require.Equal(t, strPtr("3\n"), eval.TestRes[0].Answer)
	require.Equal(t, true, eval.TestRes[0].Reached)
	require.Equal(t, true, eval.TestRes[0].Finished)
	require.Equal(t, false, eval.TestRes[0].Ignored)
	require.Equal(t, int64(0), eval.TestRes[0].Checker.ExitCode)
	require.Equal(t, int64(0), eval.TestRes[0].Subm.ExitCode)
	srvc2 := execsrvc.NewExecSrvc()
	eval2, err := srvc2.Get(ctx, execUuid)
	require.NoError(t, err)
	require.Equal(t, eval, eval2)

	srvc2.Close()
}

func BenchmarkEvalServiceMemoryConsumption(b *testing.B) {
	var m runtime.MemStats
	pageSize := os.Getpagesize()

	// Initial memory state
	runtime.GC()
	runtime.ReadMemStats(&m)
	b.Logf("Initial: HeapSys = %.3f MiB, HeapAlloc = %.3f MiB, %.3f pages",
		float64(m.HeapSys)/1024.0/1024.0,
		float64(m.HeapAlloc)/1024.0/1024.0,
		float64(m.HeapSys)/float64(pageSize),
	)

	testInSha256 := "a8fd6142d7451cf76e8f96c284b4cbab5a97e3780d2904649a13d1ec31922bb7"
	testAnsSha256 := "447beb0ea695c8da673c5b1c3412b1f5ebdf796488a1c1484f0bdd0c55e54a64"
	testFile := execsrvc.TestFile{
		InSha256:    &testInSha256,
		AnsSha256:   &testAnsSha256,
		InDownlUrl:  strPtr("https://proglv-tests.s3.eu-central-1.amazonaws.com/a8fd6142d7451cf76e8f96c284b4cbab5a97e3780d2904649a13d1ec31922bb7.zst?response-content-disposition=inline&X-Amz-Content-Sha256=UNSIGNED-PAYLOAD&X-Amz-Security-Token=IQoJb3JpZ2luX2VjELr%2F%2F%2F%2F%2F%2F%2F%2F%2F%2FwEaDGV1LWNlbnRyYWwtMSJHMEUCIQDIPzYm%2FDaynNZuMd%2BXmqnabb%2FQ6aqJuOxr%2FlpHw874zgIgc%2B2mH9PHHoqPExSUtdxtnf5TDoVKC1CptA%2FKiEfyNlwq1AMIo%2F%2F%2F%2F%2F%2F%2F%2F%2F%2F%2FARAAGgw5NzUwNDk4ODYxMTUiDC0fMJcGEhLHA%2BQWsSqoA18XvGIAFy4Bdus0BKmEnLdir%2F9PMcAYV9PtysMeXjSuggv2KGJbB3P34G%2BTcL%2FCGtIfAC4aQzLEmZDuVXXtf7dbH9ewbD9buTP%2Fnn562hzcNMtigPrLNNExsbHB%2BnlOZm%2BohNe9jowVGZtCENtKmV%2FNvM9%2BLwjahaOJ49MPFtItyqbb5VU0u15dmK%2FbAnlnkuuzTc90K4U7ON1GDdnTGZjqyQLIATtv0IBoap0MjD6hx8TRp3kgEZsT%2BnmfvG%2FeSDxoDmMBOHAmpLtuDk3gjZyt5VhVS%2BV7NFFkZEv5IGR11fSi4TgV77gjdYbXNtjz%2FOajnauTJudlysfgfxxwI2San1NPufQg4Y0EsfAjHLPcHrQnmsIKlfQs0V%2B8%2FrxY2YEBCAGcbVE9ISkcv5gGrPRFtDJft059jqGpCKpH9sHn6abxd%2BgvXLNR%2Fye9GnxnFt8vABcFoc3NAKCc276f%2BbR%2FNpHf7tOg7TbxPPvPk8fpMs8XfWNJxJm5F0kyiinADydikfYaEw2IHC9nKL3SU6tRKZnGRMc11gCL9GXI1xwyJe%2BdG9%2F%2B32Qw%2BNSDvAY65AJxdd%2BEJcaYz36wtHrORwgQRd%2F3GZvyWTunSzUvkN10h7g5mYGTccctUWdr2mF%2F8aEgjVBE4lxJxFeGPH1odXKNub67rbZRf4Ag0u11eXntwdz%2FI19IKXOvXIjSqxZfvuOEONdJ%2Bhw%2FXD2sbJBSaSjOaUuKAKxExTup69ak0Mz0ZsR8YVdkD%2Bk9IFWgk%2Bf6pIZAPGLB97yiEoXQElFz5gkkng5FVprpDd9Ig0dLW%2Bujbz3InBftDbTdmkG%2BhQVnNwyIfK8oj0qX%2FRtcmQKDbmhY7O9XaNNcmKMnPJ%2FnLMlEn32t%2B5BZW%2FxDTSYKcst6b9yQG3hlbsX4KAvV9iutJ%2Fh3KoYAyw4XsqYVJTC6r%2BFkAubtYK%2BJuj4H5THqCpBMR5D96ZjXCVF66Kk24pKjX%2B90RDRsTucJA3%2BjCBrBruaVMZMBbrmofVo4zHvTxBkgwUTq4CM%2BRxzSRKuJaP4iDwDFq5jPZw%3D%3D&X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=ASIA6GBMARGRXH5KCWST%2F20250110%2Feu-central-1%2Fs3%2Faws4_request&X-Amz-Date=20250110T094042Z&X-Amz-Expires=43200&X-Amz-SignedHeaders=host&X-Amz-Signature=50de94c7aba89606a6a55dd0fe560b1c3b5058681755dd08f3522146ba8fb4fb"),
		AnsDownlUrl: strPtr("https://proglv-tests.s3.eu-central-1.amazonaws.com/447beb0ea695c8da673c5b1c3412b1f5ebdf796488a1c1484f0bdd0c55e54a64.zst?response-content-disposition=inline&X-Amz-Content-Sha256=UNSIGNED-PAYLOAD&X-Amz-Security-Token=IQoJb3JpZ2luX2VjELr%2F%2F%2F%2F%2F%2F%2F%2F%2F%2FwEaDGV1LWNlbnRyYWwtMSJHMEUCIQDIPzYm%2FDaynNZuMd%2BXmqnabb%2FQ6aqJuOxr%2FlpHw874zgIgc%2B2mH9PHHoqPExSUtdxtnf5TDoVKC1CptA%2FKiEfyNlwq1AMIo%2F%2F%2F%2F%2F%2F%2F%2F%2F%2F%2FARAAGgw5NzUwNDk4ODYxMTUiDC0fMJcGEhLHA%2BQWsSqoA18XvGIAFy4Bdus0BKmEnLdir%2F9PMcAYV9PtysMeXjSuggv2KGJbB3P34G%2BTcL%2FCGtIfAC4aQzLEmZDuVXXtf7dbH9ewbD9buTP%2Fnn562hzcNMtigPrLNNExsbHB%2BnlOZm%2BohNe9jowVGZtCENtKmV%2FNvM9%2BLwjahaOJ49MPFtItyqbb5VU0u15dmK%2FbAnlnkuuzTc90K4U7ON1GDdnTGZjqyQLIATtv0IBoap0MjD6hx8TRp3kgEZsT%2BnmfvG%2FeSDxoDmMBOHAmpLtuDk3gjZyt5VhVS%2BV7NFFkZEv5IGR11fSi4TgV77gjdYbXNtjz%2FOajnauTJudlysfgfxxwI2San1NPufQg4Y0EsfAjHLPcHrQnmsIKlfQs0V%2B8%2FrxY2YEBCAGcbVE9ISkcv5gGrPRFtDJft059jqGpCKpH9sHn6abxd%2BgvXLNR%2Fye9GnxnFt8vABcFoc3NAKCc276f%2BbR%2FNpHf7tOg7TbxPPvPk8fpMs8XfWNJxJm5F0kyiinADydikfYaEw2IHC9nKL3SU6tRKZnGRMc11gCL9GXI1xwyJe%2BdG9%2F%2B32Qw%2BNSDvAY65AJxdd%2BEJcaYz36wtHrORwgQRd%2F3GZvyWTunSzUvkN10h7g5mYGTccctUWdr2mF%2F8aEgjVBE4lxJxFeGPH1odXKNub67rbZRf4Ag0u11eXntwdz%2FI19IKXOvXIjSqxZfvuOEONdJ%2Bhw%2FXD2sbJBSaSjOaUuKAKxExTup69ak0Mz0ZsR8YVdkD%2Bk9IFWgk%2Bf6pIZAPGLB97yiEoXQElFz5gkkng5FVprpDd9Ig0dLW%2Bujbz3InBftDbTdmkG%2BhQVnNwyIfK8oj0qX%2FRtcmQKDbmhY7O9XaNNcmKMnPJ%2FnLMlEn32t%2B5BZW%2FxDTSYKcst6b9yQG3hlbsX4KAvV9iutJ%2Fh3KoYAyw4XsqYVJTC6r%2BFkAubtYK%2BJuj4H5THqCpBMR5D96ZjXCVF66Kk24pKjX%2B90RDRsTucJA3%2BjCBrBruaVMZMBbrmofVo4zHvTxBkgwUTq4CM%2BRxzSRKuJaP4iDwDFq5jPZw%3D%3D&X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=ASIA6GBMARGRXH5KCWST%2F20250110%2Feu-central-1%2Fs3%2Faws4_request&X-Amz-Date=20250110T094122Z&X-Amz-Expires=43200&X-Amz-SignedHeaders=host&X-Amz-Signature=cd2d2d5a018e90e4a7020d0ff269372a5b7a7562811ebc3cecbe3291d3caf6fd"),
	}

	hundredTestFiles := make([]execsrvc.TestFile, 100)
	for i := 0; i < 100; i++ {
		hundredTestFiles[i] = testFile
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		srvc := execsrvc.NewExecSrvc()
		execUuid := uuid.New()
		err := srvc.Enqueue(context.Background(), execUuid, "#include <iostream>\nint main() {int a,b;std::cin>>a>>b;std::cout<<a+b<<std::endl;}", "cpp17", hundredTestFiles, execsrvc.TestingParams{
			CpuMs:  1000,
			MemKiB: 20024,
		})
		if err != nil {
			b.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		eval, err := srvc.Get(ctx, execUuid)
		if err != nil {
			b.Fatal(err)
		}
		cancel()
		srvc.Close()

		// Memory measurement after each iteration
		runtime.GC()
		runtime.ReadMemStats(&m)
		b.Logf("Iteration %d: HeapSys = %.3f MiB, HeapAlloc = %.3f MiB, %.3f pages",
			i+1,
			float64(m.HeapSys)/1024.0/1024.0,
			float64(m.HeapAlloc)/1024.0/1024.0,
			float64(m.HeapSys)/float64(pageSize),
		)

		// Basic validation
		if eval.Stage != execsrvc.StageFinished {
			b.Fatal("evaluation did not finish")
		}
		if len(eval.TestRes) != 100 {
			b.Fatal("unexpected number of test results")
		}
	}

	// Final memory state
	runtime.GC()
	runtime.ReadMemStats(&m)
	b.Logf("Final: HeapSys = %.3f MiB, HeapAlloc = %.3f MiB, %.3f pages",
		float64(m.HeapSys)/1024.0/1024.0,
		float64(m.HeapAlloc)/1024.0/1024.0,
		float64(m.HeapSys)/float64(pageSize),
	)
}

func strPtr(s string) *string {
	return &s
}
