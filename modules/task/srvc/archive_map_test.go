package srvc

import (
	"testing"

	taskzipv1 "github.com/programme-lv/backend/modules/task/taskzip"
	"github.com/stretchr/testify/require"
)

func TestMapTaskZipMetadataDropsInvalidSlugs(t *testing.T) {
	difficulty := uint8(3)
	archive := taskzipv1.Task{Metadata: &taskzipv1.Metadata{
		Topics:         []string{"graphs", "invalid"},
		Techniques:     []string{"bfs", "invalid"},
		DataStructures: []string{"queue", "invalid"},
		Difficulty:     &difficulty,
	}}
	var task Task

	mapTaskZipOrigin(archive, &task)

	require.Equal(t, []string{"graphs", "bfs", "queue"}, task.ProblemTags)
	require.Equal(t, 3, task.DifficultyRating)
}
