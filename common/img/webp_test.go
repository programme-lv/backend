package img

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/image/webp"
)

func TestRasterWebPJPEGAndPNG(t *testing.T) {
	for _, tc := range []struct {
		name string
		enc  func(image.Image) []byte
	}{
		{"png", encodePNG},
		{"jpeg", encodeJPEG},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := solid(80, 40, color.RGBA{R: 200, G: 10, B: 10, A: 255})
			webpBytes, err := rasterWebP(mustDecode(t, tc.enc(src)), 0)
			require.NoError(t, err)
			cfg, err := webp.DecodeConfig(bytes.NewReader(webpBytes))
			require.NoError(t, err)
			require.Equal(t, 80, cfg.Width)
			require.Equal(t, 40, cfg.Height)
		})
	}
}

func TestFitMaxEdgeDoesNotUpscale(t *testing.T) {
	w, h := fitMaxEdge(40, 30, 160)
	require.Equal(t, 40, w)
	require.Equal(t, 30, h)
}

func TestFitMaxEdgeDownscales(t *testing.T) {
	w, h := fitMaxEdge(320, 160, 160)
	require.Equal(t, 160, w)
	require.Equal(t, 80, h)
}

func TestRasterWebPDoesNotUpscale(t *testing.T) {
	src := solid(40, 30, color.RGBA{R: 10, G: 200, B: 10, A: 255})
	webpBytes, err := rasterWebP(src, VariantList.MaxEdge())
	require.NoError(t, err)
	cfg, err := webp.DecodeConfig(bytes.NewReader(webpBytes))
	require.NoError(t, err)
	require.Equal(t, 40, cfg.Width)
	require.Equal(t, 30, cfg.Height)
}

func solid(w, h int, c color.RGBA) *image.RGBA {
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			m.SetRGBA(x, y, c)
		}
	}
	return m
}

func encodePNG(src image.Image) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func encodeJPEG(src image.Image) []byte {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: 90}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func mustDecode(t *testing.T, data []byte) image.Image {
	t.Helper()
	src, err := decodeRaster(data)
	require.NoError(t, err)
	return src
}
