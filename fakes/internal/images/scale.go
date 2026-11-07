package images

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"strconv"

	"golang.org/x/image/draw"
)

// Scale is a recorded image at a TMDB size: wN is N wide and hN is N
// high, keeping the aspect ratio, scaled up when the stored copy is
// smaller, as image.tmdb.org does. An SVG, the original, or an image
// already at size is the recorded bytes.
func Scale(f Format, stored []byte, size string) ([]byte, error) {
	if f == SVG || size == "original" {
		return stored, nil
	}
	src, _, err := image.Decode(bytes.NewReader(stored))
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	n, _ := strconv.Atoi(size[1:])
	b := src.Bounds()
	width, height := n, b.Dy()*n/b.Dx()
	if size[0] == 'h' {
		width, height = b.Dx()*n/b.Dy(), n
	}
	if width == b.Dx() && height == b.Dy() {
		return stored, nil
	}
	dst := image.NewNRGBA(image.Rect(0, 0, max(width, 1), max(height, 1)))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
	var out bytes.Buffer
	if f == PNG {
		err = png.Encode(&out, dst)
	} else {
		err = jpeg.Encode(&out, dst, &jpeg.Options{Quality: 85})
	}
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	return out.Bytes(), nil
}
