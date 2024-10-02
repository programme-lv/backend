package fstask_test

import (
	"strings"
	"testing"

	"github.com/programme-lv/backend/fstask"
	"github.com/stretchr/testify/require"
)

func TestReadingWritingCheckerAndInteractor(t *testing.T) {
	kvadrputekl, err := fstask.Read(kvadrputeklV2Dot5Path)
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
	require.Emptyf(t, task.TestlibChecker, "task.TestlibChecker is not empty")
	require.Emptyf(t, task.TestlibInteractor, "task.TestlibInteractor is not empty")
}

func ensureEvalCorrespondsToTornisTestdata(t *testing.T, task *fstask.Task) {
	require.NotEmptyf(t, task.TestlibChecker, "task.TestlibChecker is empty")
	require.Emptyf(t, task.TestlibInteractor, "task.TestlibInteractor is not empty")

	startsWith := strings.HasPrefix(task.TestlibChecker, "#include \"testlib.h\"")
	require.Truef(t, startsWith, "task.TestlibChecker does not start with #include \"testlib.h\"")
}
