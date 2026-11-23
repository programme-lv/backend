package highlight

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidation(t *testing.T) {
	for _, id := range []string{"", "ABC", "a--b", "-a", "a-", "a_b", strings.Repeat("a", 49), "ā"} {
		require.False(t, validID(id), id)
	}
	for _, id := range []string{"a", "two-sum", strings.Repeat("a", 48)} {
		require.True(t, validID(id), id)
	}
	for _, s := range []string{"", " \t\n", strings.Repeat("ā", 61)} {
		_, ok := cleanLabel(s)
		require.False(t, ok)
	}
	s, ok := cleanLabel("  Ābols \t")
	require.True(t, ok)
	require.Equal(t, "Ābols", s)
}
