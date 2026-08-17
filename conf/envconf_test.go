package conf

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveLogLevel(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		dotenvLoaded bool
		want         slog.Level
		wantErr      bool
	}{
		{name: "prod default", raw: "", dotenvLoaded: false, want: slog.LevelWarn},
		{name: "dev default", raw: "", dotenvLoaded: true, want: slog.LevelInfo},
		{name: "debug", raw: "debug", dotenvLoaded: false, want: slog.LevelDebug},
		{name: "dbg", raw: "DBG", dotenvLoaded: false, want: slog.LevelDebug},
		{name: "info override", raw: "info", dotenvLoaded: false, want: slog.LevelInfo},
		{name: "warn override", raw: "WARN", dotenvLoaded: true, want: slog.LevelWarn},
		{name: "warning", raw: "warning", dotenvLoaded: true, want: slog.LevelWarn},
		{name: "error", raw: "error", dotenvLoaded: true, want: slog.LevelError},
		{name: "invalid", raw: "trace", dotenvLoaded: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveLogLevel(tt.raw, tt.dotenvLoaded)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
