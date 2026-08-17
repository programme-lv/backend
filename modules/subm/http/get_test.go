package http

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSubmPathID(t *testing.T) {
	u := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	parsed, ok := parseSubmPathID(u.String())
	require.True(t, ok)
	assert.Equal(t, u, parsed.UUID)
	assert.Empty(t, parsed.ShortID)

	parsed, ok = parseSubmPathID("a1B2c3")
	require.True(t, ok)
	assert.Equal(t, uuid.Nil, parsed.UUID)
	assert.Equal(t, "a1B2c3", parsed.ShortID)

	_, ok = parseSubmPathID("nope")
	assert.False(t, ok)
	_, ok = parseSubmPathID("")
	assert.False(t, ok)
}
