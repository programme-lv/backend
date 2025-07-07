package test

import (
	"context"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/programme-lv/backend/task/srvc"
	"github.com/stretchr/testify/require"
)

func TestGetTaskPreview(t *testing.T) {
	ts := newTaskSrvc(t)

	// Create a test task
	task := srvc.Task{
		ShortId:          "test-preview",
		FullName:         "Test Task for Preview",
		OriginOlympiad:   "TEST",
		DifficultyRating: 5,
		OriginNotes: []srvc.OriginNote{
			{
				Lang: "en",
				Info: "Test origin note",
			},
		},
		MdStatements: []srvc.MarkdownStatement{
			{
				LangIso639: "en",
				Story:      "This is a test story for preview",
				Input:      "Test input",
				Output:     "Test output",
			},
		},
	}

	err := ts.CreateTask(context.Background(), task)
	require.NoError(t, err, "Failed to create test task")

	// Test getting task preview
	taskPreview, err := ts.GetTaskPreview(context.Background(), "test-preview")
	require.NoError(t, err, "Failed to get task preview")

	// Verify task preview fields
	require.Equal(t, "test-preview", taskPreview.ShortId)
	require.Equal(t, "Test Task for Preview", taskPreview.FullName)
	require.Equal(t, "TEST", taskPreview.OriginOlympiad)
	require.Equal(t, 5, taskPreview.DifficultyRating)

	// Verify origin notes
	require.Len(t, taskPreview.OriginNotes, 1)
	require.Equal(t, "en", taskPreview.OriginNotes[0].Lang)
	require.Equal(t, "Test origin note", taskPreview.OriginNotes[0].Info)

	// Verify markdown statement story
	require.Contains(t, taskPreview.MdStatementStory, "This is a test story for preview")

	// Test getting preview for non-existent task
	_, err = ts.GetTaskPreview(context.Background(), "non-existent-task")
	require.Error(t, err, "Getting preview for non-existent task should fail")
	require.Contains(t, err.Error(), "uzdevums 'non-existent-task' netika atrasts", "Error message should indicate task not found")
}
