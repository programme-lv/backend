package fstask_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/programme-lv/backend/fstask"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadingWritingEvaluationConstraints(t *testing.T) {
	parsedTask, err := fstask.Read(kvadrputeklPath)
	require.NoErrorf(t, err, "failed to read task: %v", err)
	assert.Equal(t, 0.5, parsedTask.CpuTimeLimInSeconds)
	assert.Equal(t, 256, parsedTask.MemoryLimInMegabytes)

	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")
	t.Logf("Created directory for output: %s", outputDirectory)

	err = parsedTask.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)
	assert.Equal(t, 0.5, storedTask.CpuTimeLimInSeconds)
	assert.Equal(t, 256, storedTask.MemoryLimInMegabytes)

	createdTask, err := fstask.NewTask(storedTask.GetTaskName())
	require.NoErrorf(t, err, "failed to create task: %v", err)

	createdTask.CpuTimeLimInSeconds = 0.5
	createdTask.MemoryLimInMegabytes = 256

	assert.Equal(t, parsedTask.CpuTimeLimInSeconds, createdTask.CpuTimeLimInSeconds)
	assert.Equal(t, parsedTask.MemoryLimInMegabytes, createdTask.MemoryLimInMegabytes)
}
