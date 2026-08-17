package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateUsernameRejectsReservedNames(t *testing.T) {
	for _, username := range []string{"admin", "Admin", "ADMIN", "test", "System"} {
		t.Run(username, func(t *testing.T) {
			err := validateUsername(username)

			require.Error(t, err)
			assert.Equal(t, ErrUsernameReserved.ErrorCode(), err.ErrorCode())
		})
	}
}

func TestValidateUsernameAllowsReservedWordsWithinUsername(t *testing.T) {
	for _, username := range []string{"administrator", "contestant", "ecosystem"} {
		t.Run(username, func(t *testing.T) {
			assert.NoError(t, validateUsername(username))
		})
	}
}
