package srvc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLioOriginNotes(t *testing.T) {
	tests := []struct {
		name      string
		year      string
		stage     string
		divisions []string
		long      string
		short     string
	}{
		{
			name: "senior", year: "2025", stage: "national", divisions: []string{"senior"},
			long:  "Uzdevums no Latvijas 38. (2024./2025. m.g.) informātikas olimpiādes (LIO) valsts kārtas; vecākajai (11.-12. klašu) grupai.",
			short: "Uzdevums no 2025.\u00a0g. LIO valsts kārtas; 11.-12. klasei.",
		},
		{
			name: "both", year: "2025", stage: "national",
			divisions: []string{"junior", "senior"},
			long:      "Uzdevums no Latvijas 38. (2024./2025. m.g.) informātikas olimpiādes (LIO) valsts kārtas; abām grupām.",
			short:     "Uzdevums no 2025.\u00a0g. LIO valsts kārtas; abām grupām.",
		},
		{
			name: "no division", year: "2023", stage: "national",
			long:  "Uzdevums no Latvijas 36. (2022./2023. m.g.) informātikas olimpiādes (LIO) valsts kārtas.",
			short: "Uzdevums no 2023.\u00a0g. LIO valsts kārtas.",
		},
		{
			name: "junior academic year", year: "1996/1997", stage: "municipal",
			divisions: []string{"junior"},
			long:      "Uzdevums no Latvijas 10. (1996./1997. m.g.) informātikas olimpiādes (LIO) novada kārtas; jaunākajai (8.-10. klašu) grupai.",
			short:     "Uzdevums no 1997.\u00a0g. LIO novada kārtas; 8.-10. klasei.",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			long, short, ok := lioOriginNotes("LIO", test.year, test.stage, test.divisions)
			require.True(t, ok)
			require.Equal(t, test.long, long)
			require.Equal(t, test.short, short)
		})
	}
}

func TestApplyOriginNotesReplacesLatvianNote(t *testing.T) {
	task := Task{
		OriginOlympiad: "LIO", OriginYear: "2025", OlympStage: "national",
		OriginNotes: []OriginNote{{Lang: "lv", Info: "old"}, {Lang: "en", Info: "keep"}},
	}

	applyOriginNotes(&task)

	require.Len(t, task.OriginNotes, 2)
	require.Equal(t, OriginNote{Lang: "en", Info: "keep"}, task.OriginNotes[0])
	require.Contains(t, task.OriginNotes[1].Info, "Latvijas 38.")
}
