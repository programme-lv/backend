package img

import (
	"bytes"
	"fmt"
	"image"

	"github.com/KarpelesLab/gowebp"
	xdraw "golang.org/x/image/draw"

	_ "golang.org/x/image/webp"
	_ "image/jpeg"
	_ "image/png"
)

const (
	webpQuality = 80
	webpMethod  = 4
)

func decodeRaster(data []byte) (image.Image, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	return src, nil
}

func rasterWebP(src image.Image, maxEdge int) ([]byte, error) {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	nw, nh := fitMaxEdge(w, h, maxEdge)
	out := src
	if nw != w || nh != h {
		dst := image.NewNRGBA(image.Rect(0, 0, nw, nh))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Src, nil)
		out = dst
	}
	var buf bytes.Buffer
	err := gowebp.Encode(&buf, out, &gowebp.Options{
		Lossy:   true,
		Quality: webpQuality,
		Method:  webpMethod,
	})
	if err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}
	return buf.Bytes(), nil
}

func fitMaxEdge(w, h, maxEdge int) (int, int) {
	if maxEdge <= 0 || w <= maxEdge && h <= maxEdge {
		return w, h
	}
	if w <= 0 || h <= 0 {
		return w, h
	}
	if w >= h {
		nh := h * maxEdge / w
		if nh < 1 {
			nh = 1
		}
		return maxEdge, nh
	}
	nw := w * maxEdge / h
	if nw < 1 {
		nw = 1
	}
	return nw, maxEdge
}
