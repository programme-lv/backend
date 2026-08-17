package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSubmListLimit(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{name: "default", raw: "", want: defaultSubmListLimit},
		{name: "valid", raw: "50", want: 50},
		{name: "capped", raw: "1000000", want: maxSubmListLimit},
		{name: "zero", raw: "0", want: defaultSubmListLimit},
		{name: "negative", raw: "-1", want: defaultSubmListLimit},
		{name: "invalid", raw: "nope", want: defaultSubmListLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseSubmListLimit(tt.raw))
		})
	}
}
