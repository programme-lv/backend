package srvc

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/programme-lv/backend/common/filestore"
	"github.com/programme-lv/backend/common/img"
	"github.com/stretchr/testify/require"
)

func TestUploadIllustrationImgWarmsVariants(t *testing.T) {
	store, err := filestore.NewStore(t.TempDir())
	require.NoError(t, err)
	ts := NewTaskSrvc(nil, store, store)

	body := solidPNG(t, 80, 40)
	key, serr := ts.UploadIllustrationImg(context.Background(), "image/png", body)
	require.Nil(t, serr)

	orig := "illustrations/" + key
	exists, err := store.Exists(orig)
	require.NoError(t, err)
	require.True(t, exists)

	for _, v := range img.Variants() {
		serve := img.ServeKey(orig, v)
		exists, err := store.Exists(serve)
		require.NoError(t, err)
		require.True(t, exists, string(v))
	}
}

func solidPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			m.SetRGBA(x, y, color.RGBA{R: 12, G: 34, B: 56, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, m))
	return buf.Bytes()
}
