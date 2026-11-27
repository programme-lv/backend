package user

import (
	"strings"
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

func TestValidateUsernameLengthBounds(t *testing.T) {
	assert.NoError(t, validateUsername(strings.Repeat("a", 24)))

	err := validateUsername(strings.Repeat("a", 25))

	require.Error(t, err)
	assert.Equal(t, ErrUsernameTooLong.ErrorCode(), err.ErrorCode())
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

func TestValidateUsernameAddressabilityAndByteLength(t *testing.T) {
	for _, name := range []string{"..", "ab\x00cd"} {
		err := validateUsername(name)
		require.Error(t, err)
		assert.Equal(t, ErrUsernameInvalid.ErrorCode(), err.ErrorCode())
	}
	for _, name := range []string{"a..b", "..a", " a ", strings.Repeat("ā", 12)} {
		assert.NoError(t, validateUsername(name))
	}
	err := validateUsername(strings.Repeat("ā", 13))
	require.Error(t, err)
	assert.Equal(t, ErrUsernameTooLong.ErrorCode(), err.ErrorCode())
}
