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

func TestValidateUsernameRejectsAtSign(t *testing.T) {
	for _, username := range []string{"user@example.com", "@user", "user@", "a@b", "@"} {
		t.Run(username, func(t *testing.T) {
			err := validateUsername(username)

			require.Error(t, err)
			assert.Equal(t, ErrUsernameContainsAt.ErrorCode(), err.ErrorCode())
		})
	}
}

func TestValidateUsernameAllowsExistingSyntax(t *testing.T) {
	for _, username := range []string{
		"ab",
		"user-name",
		"user_name",
		"user.name",
		"user123",
		"User.Name-123_ok",
	} {
		t.Run(username, func(t *testing.T) {
			assert.NoError(t, validateUsername(username))
		})
	}
}
