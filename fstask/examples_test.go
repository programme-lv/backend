package fstask_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/programme-lv/backend/fstask"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadingWritingExamples(t *testing.T) {
	parsedTask, err := fstask.Read(kvadrputeklPath)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	parsedExamples := parsedTask.GetExamples()
	require.Equal(t, 2, len(parsedExamples))

	parsedExampleNames := []string{}
	for i := 0; i < len(parsedExamples); i++ {
		parsedExampleNames = append(parsedExampleNames, *parsedExamples[i].FName)
	}
	expectedExampleNames := []string{"kp00", "kp01"}
	assert.Equal(t, expectedExampleNames, parsedExampleNames)

	parsedInputs := []string{}
	for i := 0; i < len(parsedExamples); i++ {
		parsedInputs = append(parsedInputs, string(parsedExamples[i].Input))
	}

	examplePath := filepath.Join(kvadrputeklPath, "examples")
	expectedInputs := []string{}
	for i := 0; i < len(parsedExamples); i++ {
		inPath := filepath.Join(examplePath, fmt.Sprintf("%s.in", *parsedExamples[i].FName))

		in, err := os.ReadFile(inPath)
		require.NoErrorf(t, err, "failed to read input file: %v", err)

		expectedInputs = append(expectedInputs, string(in))
	}
	assert.Equal(t, expectedInputs, parsedInputs)

	parsedOutputs := []string{}
	for i := 0; i < len(parsedExamples); i++ {
		parsedOutputs = append(parsedOutputs, string(parsedExamples[i].Output))
	}
	expectedOutputs := []string{}
	for i := 0; i < len(parsedExamples); i++ {
		outPath := filepath.Join(examplePath, fmt.Sprintf("%s.out", *parsedExamples[i].FName))

		out, err := os.ReadFile(outPath)
		require.NoErrorf(t, err, "failed to read output file: %v", err)

		outStr := string(out)
		expectedOutputs = append(expectedOutputs, outStr)
	}
	assert.Equal(t, expectedOutputs, parsedOutputs)

	parsedNotes := []string{}
	for i := 0; i < len(parsedExamples); i++ {
		mdNoteStr := string(parsedExamples[i].MdNote)
		parsedNotes = append(parsedNotes, mdNoteStr)
	}
	expectedNotes := []string{}
	for i := 0; i < len(parsedExamples); i++ {
		if len(parsedExamples[i].MdNote) == 0 {
			expectedNotes = append(expectedNotes, "")
			continue
		}
		notePath := filepath.Join(examplePath, fmt.Sprintf("%s.md", *parsedExamples[i].FName))

		note, err := os.ReadFile(notePath)
		require.NoErrorf(t, err, "failed to read note file: %v", err)

		outStr := string(note)
		expectedNotes = append(expectedNotes, outStr)
	}
	require.Equal(t, expectedNotes, parsedNotes)

	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")

	t.Logf("Created directory for output: %s", outputDirectory)

	err = parsedTask.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	storedExampleNames := []string{}
	for i := 0; i < 2; i++ {
		storedExampleNames = append(storedExampleNames, *storedTask.GetExamples()[i].FName)
	}
	assert.Equal(t, expectedExampleNames, storedExampleNames)

	storedInputs := []string{}
	for i := 0; i < 2; i++ {
		storedInputs = append(storedInputs, string(storedTask.GetExamples()[i].Input))
	}
	assert.Equal(t, expectedInputs, storedInputs)

	storedOutputs := []string{}
	for i := 0; i < 2; i++ {
		storedOutputs = append(storedOutputs, string(storedTask.GetExamples()[i].Output))
	}
	assert.Equal(t, expectedOutputs, storedOutputs)

	require.Equal(t, 2, len(storedTask.GetExamples()))
	require.Equal(t, expectedNotes[0], string(storedTask.GetExamples()[0].MdNote))
	require.Equal(t, expectedNotes[1], string(storedTask.GetExamples()[1].MdNote))

	createdTask, err := fstask.NewTask(storedTask.GetTaskName())
	if err != nil {
		t.Errorf("failed to create task: %v", err)
	}

	createdTask.AddExample([]byte(storedInputs[0]), []byte(storedOutputs[0]), []byte(expectedNotes[0]))

	// store created task
	outputDirectory2 := filepath.Join(tmpDirectory, "kvadrputekl2")
	err = createdTask.Store(outputDirectory2)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask2, err := fstask.Read(outputDirectory2)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	assert.Equal(t, storedTask2.GetExamples()[0].Input, parsedTask.GetExamples()[0].Input)
	assert.Equal(t, storedTask2.GetExamples()[0].Output, parsedTask.GetExamples()[0].Output)
}
