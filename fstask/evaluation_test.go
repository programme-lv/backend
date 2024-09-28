package fstask_test

import (
	"strings"
	"testing"

	"github.com/programme-lv/backend/fstask"
	"github.com/stretchr/testify/require"
)

func TestReadingWritingCheckerAndInteractor(t *testing.T) {
	kvadrputekl, err := fstask.Read(kvadrputeklPath)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	tornis, err := fstask.Read(tornisPath)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	ensureEvalCorrespondsToKvadrputeklTestdata(t, kvadrputekl)
	ensureEvalCorrespondsToTornisTestdata(t, tornis)

	writtenKvadrputeklTask := writeAndReReadTask(t, kvadrputekl)
	writtenTornisTask := writeAndReReadTask(t, tornis)

	ensureEvalCorrespondsToKvadrputeklTestdata(t, writtenKvadrputeklTask)
	ensureEvalCorrespondsToTornisTestdata(t, writtenTornisTask)
}

func ensureEvalCorrespondsToKvadrputeklTestdata(t *testing.T, task *fstask.Task) {
	require.Nilf(t, task.TestlibChecker, "task.TestlibChecker is not nil")
	require.Nilf(t, task.TestlibInteractor, "task.TestlibInteractor is not nil")
}

func ensureEvalCorrespondsToTornisTestdata(t *testing.T, task *fstask.Task) {
	require.NotNilf(t, task.TestlibChecker, "task.TestlibChecker is nil")
	require.Nilf(t, task.TestlibInteractor, "task.TestlibInteractor is not nil")

	startsWith := strings.HasPrefix(*task.TestlibChecker, "#include \"testlib.h\"")
	require.Truef(t, startsWith, "task.TestlibChecker does not start with #include \"testlib.h\"")
}
