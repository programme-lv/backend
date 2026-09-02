package srvc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetIllustrationAssetURLs(t *testing.T) {
	ts := NewTaskSrvc(nil, nil, nil, WithPublicAPIBaseURL("https://api.programme.lv"))

	list, view, full, err := ts.GetIllustrationAssetURLs(context.Background(), "deadbeef.png")
	require.NoError(t, err)
	require.Equal(t, "https://api.programme.lv/assets/illustrations/deadbeef.png.list.webp", list)
	require.Equal(t, "https://api.programme.lv/assets/illustrations/deadbeef.png.view.webp", view)
	require.Equal(t, "https://api.programme.lv/assets/illustrations/deadbeef.png.full.webp", full)

	list, view, full, err = ts.GetIllustrationAssetURLs(context.Background(), "deadbeef.webp")
	require.NoError(t, err)
	require.Equal(t, "https://api.programme.lv/assets/illustrations/deadbeef.webp.list.webp", list)
	require.Equal(t, "https://api.programme.lv/assets/illustrations/deadbeef.webp.view.webp", view)
	require.Equal(t, "https://api.programme.lv/assets/illustrations/deadbeef.webp", full)
}
