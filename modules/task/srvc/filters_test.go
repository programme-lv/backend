package srvc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildFilterTreeNormalizesAndNests(t *testing.T) {
	tree := BuildFilterTree([]OriginCount{
		{Olympiad: "LIO", Year: "2024/2025", Stage: "national", Divisions: []string{"junior"}, Count: 2},
		{Olympiad: "LIO", Year: "2025", Stage: "national", Divisions: []string{"senior"}, Count: 1},
		{Olympiad: "LIO", Year: "2025", Stage: "national", Divisions: []string{"junior", "senior"}, Count: 1},
		{Olympiad: "LIO", Year: "2025", Stage: "school", Divisions: []string{"junior"}, Count: 3},
		{Olympiad: "", Year: "", Stage: "", Divisions: nil, Count: 4},
		{Olympiad: "BOI", Year: "2024", Stage: "", Divisions: nil, Count: 2},
	})

	require.Len(t, tree.Olympiads, 3)
	require.Equal(t, "LIO", tree.Olympiads[0].ID)
	require.Equal(t, "BOI", tree.Olympiads[1].ID)
	require.Equal(t, "other", tree.Olympiads[2].ID)

	lio := tree.Olympiads[0]
	require.Equal(t, 7, lio.Count)
	require.Len(t, lio.Years, 1)
	require.Equal(t, "2025", lio.Years[0].ID)
	require.Equal(t, 7, lio.Years[0].Count)
	require.Equal(t, []string{"school", "national"}, stageIDs(lio.Years[0].Stages))

	national := lio.Years[0].Stages[1]
	require.Equal(t, "national", national.ID)
	require.Equal(t, 4, national.Count)
	require.Equal(t, []FilterDivision{
		{ID: "junior", Count: 2},
		{ID: "senior", Count: 1},
		{ID: "both", Count: 1},
	}, national.Divisions)

	boi := tree.Olympiads[1]
	require.Equal(t, 2, boi.Count)
	require.Equal(t, "2024", boi.Years[0].ID)
	require.Empty(t, boi.Years[0].Stages)

	other := tree.Olympiads[2]
	require.Equal(t, 4, other.Count)
	require.Empty(t, other.Years)
}

func TestNormalizeYear(t *testing.T) {
	require.Equal(t, "2025", NormalizeYear("2024/2025"))
	require.Equal(t, "2025", NormalizeYear("2025"))
	require.Equal(t, "", NormalizeYear("  "))
	require.Equal(t, "soon", NormalizeYear("soon"))
}

func stageIDs(stages []FilterStage) []string {
	ids := make([]string, len(stages))
	for i, stage := range stages {
		ids[i] = stage.ID
	}
	return ids
}
