package fstask_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/programme-lv/backend/fstask"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"
)

func TestReadingWritingTests(t *testing.T) {
	parsedTask, err := fstask.Read(kvadrputeklPath)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	parsedTests := parsedTask.GetTestsSortedByID()
	require.Equal(t, 6, len(parsedTests))

	parsedTestNames := []string{}
	for i := 0; i < 6; i++ {
		filename := parsedTask.GetTestFilename(parsedTests[i].ID)
		parsedTestNames = append(parsedTestNames, filename)
	}
	expectedTestNames := []string{"kp01a", "kp01b", "kp01c", "kp02a", "kp02b", "kp02c"}
	assert.Equal(t, expectedTestNames, parsedTestNames)

	parsedIDs := []int{}
	for i := 0; i < 6; i++ {
		parsedIDs = append(parsedIDs, parsedTests[i].ID)
	}
	expectedIDs := []int{1, 2, 3, 4, 5, 6}
	assert.Equal(t, expectedIDs, parsedIDs)

	parsedInputs := []string{}
	for i := 0; i < 6; i++ {
		parsedInputs = append(parsedInputs, string(parsedTests[i].Input))
	}

	testPath := filepath.Join(kvadrputeklPath, "tests")
	expectedInputs := []string{}
	for i := 0; i < 6; i++ {
		filename := parsedTask.GetTestFilename(parsedTests[i].ID)
		inPath := filepath.Join(testPath, fmt.Sprintf("%s.in", filename))

		in, err := os.ReadFile(inPath)
		require.NoErrorf(t, err, "failed to read input file: %v", err)

		expectedInputs = append(expectedInputs, string(in))
	}
	assert.Equal(t, expectedInputs, parsedInputs)

	parsedAnswers := []string{}
	for i := 0; i < 6; i++ {
		parsedAnswers = append(parsedAnswers, string(parsedTests[i].Answer))
	}
	expectedAnsers := []string{}
	for i := 0; i < 6; i++ {
		filename := parsedTask.GetTestFilename(parsedTests[i].ID)
		ansPath := filepath.Join(testPath, fmt.Sprintf("%s.out", filename))

		ans, err := os.ReadFile(ansPath)
		require.NoErrorf(t, err, "failed to read answer file: %v", err)

		expectedAnsers = append(expectedAnsers, string(ans))
	}

	assert.Equal(t, expectedAnsers, parsedAnswers)

	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")

	t.Logf("Created directory for output: %s", outputDirectory)

	err = parsedTask.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	storedTestNames := []string{}
	tests := storedTask.GetTestsSortedByID()
	for i := 0; i < 6; i++ {
		filename := storedTask.GetTestFilename(tests[i].ID)
		storedTestNames = append(storedTestNames, filename)
	}
	assert.Equal(t, expectedTestNames, storedTestNames)

	storedIDs := []int{}
	for i := 0; i < 6; i++ {
		storedIDs = append(storedIDs, storedTask.GetTestsSortedByID()[i].ID)
	}
	assert.Equal(t, expectedIDs, storedIDs)

	storedInputs := []string{}
	for i := 0; i < 6; i++ {
		storedInputs = append(storedInputs, string(storedTask.GetTestsSortedByID()[i].Input))
	}
	assert.Equal(t, expectedInputs, storedInputs)

	storedAnswers := []string{}
	for i := 0; i < 6; i++ {
		storedAnswers = append(storedAnswers, string(storedTask.GetTestsSortedByID()[i].Answer))
	}
	assert.Equal(t, expectedAnsers, storedAnswers)

	createdTask, err := fstask.NewTask(storedTask.GetTaskName())
	require.NoErrorf(t, err, "failed to create task: %v", err)

	// set tests
	for i := 0; i < 6; i++ {
		createdTask.AddTest(parsedTests[i].Input, parsedTests[i].Answer)
		if filename := createdTask.GetTestFilename(parsedTests[i].ID); filename != "" {
			createdTask.AssignFilenameToTest(filename, parsedTests[i].ID)
		}
	}

	// compare tests
	assert.Equal(t, parsedTask.GetTestsSortedByID(), createdTask.GetTestsSortedByID())

	// shuffle test order via assigning new ids or swapping pairwise
	for i := 0; i < 10; i++ {
		a := rand.Intn(6) + 1
		b := rand.Intn(6) + 1
		createdTask.SwapTestOrder(a, b)
	}

	// store it again
	anotherOutputDir := filepath.Join(tmpDirectory, "kvadrputekl2")

	err = createdTask.Store(anotherOutputDir)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask2, err := fstask.Read(anotherOutputDir)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	// compare the tests
	assert.Equal(t, createdTask.GetTestsSortedByID(), storedTask2.GetTestsSortedByID())
}
