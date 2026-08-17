package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomShortID_CharsetAndLength(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 64; i++ {
		id, err := RandomShortID()
		require.NoError(t, err)
		assert.True(t, ValidShortID(id), "id %q", id)
		assert.NotEqual(t, reservedShortID, id)
		seen[id] = struct{}{}
	}
	assert.Greater(t, len(seen), 1)
}

func TestValidShortID(t *testing.T) {
	assert.True(t, ValidShortID("a1B2c3"))
	assert.True(t, ValidShortID("scores"))
	assert.False(t, ValidShortID(""))
	assert.False(t, ValidShortID("a1B2c"))
	assert.False(t, ValidShortID("a1B2c3d"))
	assert.False(t, ValidShortID("a1B2c!"))
	assert.False(t, ValidShortID("a1B2-3"))
}
