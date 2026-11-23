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

func TestAliasValidation(t *testing.T) {
	for _, tc := range []struct {
		value string
		limit int
		valid bool
		want  string
	}{
		{"", 16, true, ""}, {"  ĀB  ", 16, true, "ĀB"},
		{strings.Repeat("ā", 16), 16, true, strings.Repeat("ā", 16)},
		{strings.Repeat("ā", 17), 16, false, ""},
		{strings.Repeat("x", 80), 80, true, strings.Repeat("x", 80)},
		{strings.Repeat("x", 81), 80, false, ""},
		{"a\nb", 80, false, ""}, {"\tDP", 16, false, ""}, {string([]byte{0xff}), 16, false, ""},
	} {
		value, ok := cleanAlias(&tc.value, tc.limit)
		require.Equal(t, tc.valid, ok, "%q", tc.value)
		if ok {
			require.Equal(t, tc.want, *value)
		}
	}
}
