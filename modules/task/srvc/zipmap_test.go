package srvc

import (
	"testing"

	taskzipv1 "github.com/programme-lv/backend/modules/task/taskzip"
	"github.com/stretchr/testify/require"
)

func TestMapTaskZipMetadataDropsInvalidSlugs(t *testing.T) {
	difficulty := uint8(3)
	archive := taskzipv1.Task{
		Origin: &taskzipv1.Origin{Divisions: []string{"junior", "senior"}},
		Metadata: &taskzipv1.Metadata{
			Topics:         []string{"graphs", "invalid"},
			Techniques:     []string{"bfs", "invalid"},
			DataStructures: []string{"queue", "invalid"},
			Difficulty:     &difficulty,
		},
	}
	var task Task

	mapTaskZipOrigin(archive, &task)

	require.Equal(t, []string{"graphs", "bfs", "queue"}, task.ProblemTags)
	require.Equal(t, []string{"junior", "senior"}, task.OriginDivisions)
	require.Equal(t, 3, task.DifficultyRating)
}

func TestMapTaskZipInteractiveExamplesKeepTraces(t *testing.T) {
	archive := taskzipv1.Task{
		Interactor: []byte("int main() {}\n"),
		Examples: []taskzipv1.Example{{
			Trace: []byte("1\n---\n2\n"),
			Notes: map[string][]byte{"lv": []byte("Piemērs.\n")},
		}},
	}
	var task Task

	mapTaskZipContent(archive, &task)

	require.Equal(t, "int main() {}\n", task.Interactor)
	require.Equal(t, "1\n---\n2\n", task.Examples[0].Trace)
	require.Equal(t, "Piemērs.\n", task.Examples[0].MdNote["lv"])

	var back taskzipv1.Task

	mapServiceContent(task, &back)

	require.Equal(t, []byte("1\n---\n2\n"), back.Examples[0].Trace)
	require.Equal(t, []byte("Piemērs.\n"), back.Examples[0].Notes["lv"])
}
