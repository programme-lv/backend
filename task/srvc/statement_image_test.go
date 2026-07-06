package srvc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskStatementImageObjectKey(t *testing.T) {
	t.Parallel()

	require.Equal(t, "", taskStatementImageObjectKey(""))
	require.Equal(t, "md-images/paklajs/image.png", taskStatementImageObjectKey("paklajs/image.png"))
	require.Equal(t, "md-images/paklajs/image.png", taskStatementImageObjectKey("md-images/paklajs/image.png"))
	require.Equal(t, "task/paklajs/md-images/image.png", taskStatementImageObjectKey("task/paklajs/md-images/image.png"))
	require.Equal(t, "task-md-images/image.png", taskStatementImageObjectKey("task-md-images/image.png"))
}

func TestTaskStatementImageStoredKey(t *testing.T) {
	t.Parallel()

	require.Equal(t, "", taskStatementImageStoredKey(""))
	require.Equal(t, "paklajs/image.png", taskStatementImageStoredKey("paklajs/image.png"))
	require.Equal(t, "paklajs/image.png", taskStatementImageStoredKey("md-images/paklajs/image.png"))
	require.Equal(t, "task/paklajs/md-images/image.png", taskStatementImageStoredKey("task/paklajs/md-images/image.png"))
}
