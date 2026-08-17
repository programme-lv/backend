package srvcerror_test

import (
	"testing"

	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/modules/user"
	"github.com/stretchr/testify/require"
)

func TestIs(t *testing.T) {
	err1 := srvcerror.New("test_error", "test error msg to user")
	err2 := srvcerror.New("test_error", "another test error msg to user")
	err1Derived := err1
	err2Derived := err2
	require.ErrorIs(t, err1Derived, err1)
	require.ErrorIs(t, err2Derived, err2)
	require.ErrorIs(t, err1Derived, err2)
	require.ErrorIs(t, err2Derived, err1)
	require.True(t, err1.Is(err1Derived))
	require.True(t, err2.Is(err2Derived))
	require.True(t, err1.Is(err2))
	require.True(t, err2.Is(err1))

	withMsg := err1.WithMsg("changed")
	require.Equal(t, "changed", withMsg.Error())
	require.Equal(t, err1.ErrorCode(), withMsg.ErrorCode())
	require.ErrorIs(t, withMsg, err1)
	require.Equal(t, "test error msg to user", err1.Error())

	err3 := srvcerror.New("another_error", "another error msg to user")
	err3Derived := err3
	require.NotErrorIs(t, err3Derived, err1)
	require.NotErrorIs(t, err3Derived, err2)
	require.ErrorIs(t, err3Derived, err3)
	require.False(t, err3.Is(err1))
	require.False(t, err3.Is(err2))

	err4 := user.ErrUserNotFound
	require.ErrorIs(t, err4, user.ErrUserNotFound)

	err6 := user.ErrEmailAlreadyExists
	require.NotErrorIs(t, err6, err4)

	// Verify that nil srvcerror.E interface can be returned safely
	// and compared to nil error interface
	var emptyError srvcerror.E = nil
	var asStdError error = emptyError
	require.Nil(t, asStdError)
	require.Nil(t, emptyError)
}
