package srvc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskIllustrationObjectKey(t *testing.T) {
	t.Parallel()

	require.Equal(t, "", taskIllustrationObjectKey(""))
	require.Equal(t, "illustrations/image.png", taskIllustrationObjectKey("image.png"))
	require.Equal(t, "illustrations/image.png", taskIllustrationObjectKey("illustrations/image.png"))
	require.Equal(t, "task-illustrations/image.png", taskIllustrationObjectKey("task-illustrations/image.png"))
}

func TestTaskIllustrationStoredKey(t *testing.T) {
	t.Parallel()

	require.Equal(t, "", taskIllustrationStoredKey(""))
	require.Equal(t, "image.png", taskIllustrationStoredKey("image.png"))
	require.Equal(t, "image.png", taskIllustrationStoredKey("illustrations/image.png"))
	require.Equal(t, "task-illustrations/image.png", taskIllustrationStoredKey("task-illustrations/image.png"))
}
