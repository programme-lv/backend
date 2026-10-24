package srvc

import (
	"context"
	"strings"
	"testing"

	"github.com/programme-lv/backend/common/img"
	"github.com/programme-lv/backend/modules/task/storage"
	"github.com/stretchr/testify/require"
)

func assetURL(key string) string {
	return "https://api.programme.lv/assets/" + strings.TrimPrefix(key, storage.AssetsRoot+"/")
}

func TestGetIllustrationAssetURLs(t *testing.T) {
	ts := NewTaskSrvc(nil, nil, nil, WithPublicAPIBaseURL("https://api.programme.lv"))

	orig := storage.Illustration("example", "deadbeef", ".png")
	list, view, full, err := ts.GetIllustrationAssetURLs(context.Background(), orig)
	require.NoError(t, err)
	require.Equal(t, assetURL(img.ServeKey(orig, img.VariantList)), list)
	require.Equal(t, assetURL(img.ServeKey(orig, img.VariantView)), view)
	require.Equal(t, assetURL(img.ServeKey(orig, img.VariantFull)), full)

	orig = storage.Illustration("example", "deadbeef", ".webp")
	list, view, full, err = ts.GetIllustrationAssetURLs(context.Background(), orig)
	require.NoError(t, err)
	require.Equal(t, assetURL(img.ServeKey(orig, img.VariantList)), list)
	require.Equal(t, assetURL(img.ServeKey(orig, img.VariantView)), view)
	require.Equal(t, "https://api.programme.lv/assets/tasks/example/illustration/deadbeef.webp", full)
}

func TestGetHttpUrlForStatementImage(t *testing.T) {
	ts := NewTaskSrvc(nil, nil, nil, WithPublicAPIBaseURL("https://api.programme.lv"))

	orig := storage.Image("aplusb", "deadbeef", "abc.png")
	url, err := ts.GetHttpUrlForStatementImage(context.Background(), orig)
	require.NoError(t, err)
	require.Equal(t, assetURL(img.ServeKey(orig, img.VariantFull)), url)

	orig = storage.Image("aplusb", "deadbeef", "abc.webp")
	url, err = ts.GetHttpUrlForStatementImage(context.Background(), orig)
	require.NoError(t, err)
	require.Equal(t, "https://api.programme.lv/assets/tasks/aplusb/images/deadbeef/abc.webp", url)

	orig = storage.Image("aplusb", "deadbeef", "abc.svg")
	url, err = ts.GetHttpUrlForStatementImage(context.Background(), orig)
	require.NoError(t, err)
	require.Equal(t, "https://api.programme.lv/assets/tasks/aplusb/images/deadbeef/abc.svg", url)
}
