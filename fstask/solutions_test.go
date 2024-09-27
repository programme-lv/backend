package fstask_test

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/programme-lv/backend/fstask"
	"github.com/stretchr/testify/require"
)

// TestReadingWritingSolutions tests the reading and writing of solutions.
func TestReadingWritingSolutions(t *testing.T) {
	task, err := fstask.Read(testTaskPath)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	expFilenames := []string{
		"kp_kp_ok.cpp",
		"kp_kp_tle.cpp",
		"kp_nv.cpp",
	}

	expChsums := []string{
		"80039b1550dc700b7cb02652aaa0c28a604b30f0c9b1260450818eaa7fa9fa90",
		"b502b098b9a7b9cb92f781c539eba26bdbd88d66717fb52aec046704fec646ba",
		"a25f4e5951c60805942d471fd12504c3bb66c7f4749ec24bd7ad8fc8de88b640",
	}

	solutions := task.Solutions
	require.Len(t, solutions, 3)

	for i, solution := range task.Solutions {
		require.Equal(t, expFilenames[i], solution.Filename)
		chsum := getSha256Sum(solution.Content)
		require.Equal(t, expChsums[i], chsum)
	}

	require.Equal(t, 100, *solutions[0].ScoreEq)
	require.Nil(t, solutions[1].ScoreEq)
	require.Equal(t, 100, *solutions[2].ScoreEq)

	require.Equal(t, 50, *solutions[1].ScoreLte)

	require.Equal(t, "Krišjānis Petručeņa", solutions[0].Author)
	require.Equal(t, "Krišjānis Petručeņa", solutions[1].Author)
	require.Equal(t, "Normunds Vilciņš", solutions[2].Author)

}

func getSha256Sum(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:])
}
