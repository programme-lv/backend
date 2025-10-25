package srvcerror_test

import (
	"testing"

	"github.com/programme-lv/backend/common/srvcerror"
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

	err3 := srvcerror.New("another_error", "another error msg to user")
	err3Derived := err3
	require.NotErrorIs(t, err3Derived, err1)
	require.NotErrorIs(t, err3Derived, err2)
	require.ErrorIs(t, err3Derived, err3)

}
