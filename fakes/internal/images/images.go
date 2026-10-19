// Package images generates the images the image fakes serve: one field of
// color per name and size, so a test that reads an image back can tell
// which name and size it got by its dominant color alone.
package images

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"

	// Registers WebP with image.Decode for Dominant.
	_ "golang.org/x/image/webp"
)

type Format int

const (
	PNG Format = iota + 1
	JPEG
	WebP
	SVG
)

var formats = map[string]Format{".png": PNG, ".jpg": JPEG, ".jpeg": JPEG, ".webp": WebP, ".svg": SVG}

var contentTypes = map[Format]string{PNG: "image/png", JPEG: "image/jpeg", WebP: "image/webp", SVG: "image/svg+xml"}

// FormatOf is the format a file name's extension names.
func FormatOf(name string) (Format, bool) {
	f, ok := formats[strings.ToLower(path.Ext(name))]
	return f, ok
}

func (f Format) ContentType() string { return contentTypes[f] }

// Color is the dominant color of the image at name and size.
func Color(name, size string) color.RGBA {
	sum := sha256.Sum256([]byte(name + "\x00" + size))
	return color.RGBA{R: sum[0], G: sum[1], B: sum[2], A: 0xff}
}

// Dominant is the color most of an image's pixels have, read from body as
// any format Serve writes. It counts a grid of samples pixels a side.
func Dominant(body []byte) (color.RGBA, error) {
	if m := svgFill.FindSubmatch(body); m != nil {
		c := color.RGBA{A: 0xff}
		if _, err := fmt.Sscanf(string(m[1]), "%02x%02x%02x", &c.R, &c.G, &c.B); err != nil {
			return color.RGBA{}, fmt.Errorf("read svg fill: %w", err)
		}
		return c, nil
	}
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return color.RGBA{}, fmt.Errorf("decode image: %w", err)
	}
	counts := map[color.RGBA]int{}
	var most color.RGBA
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y += max(b.Dy()/samples, 1) {
		for x := b.Min.X; x < b.Max.X; x += max(b.Dx()/samples, 1) {
			c, _ := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			counts[c]++
			if counts[c] > counts[most] {
				most = c
			}
		}
	}
	return most, nil
}

const samples = 64

var svgFill = regexp.MustCompile(`^<svg [^>]*><rect [^>]*fill="#([0-9a-f]{6})"`)

// Serve answers the image at name and size, width by height pixels of
// Color(name, size), encoded as f.
func Serve(w http.ResponseWriter, f Format, name, size string, width, height int) {
	body := render(f, Color(name, size), width, height)
	w.Header().Set("Content-Type", f.ContentType())
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	_, _ = w.Write(body)
}

func render(f Format, c color.RGBA, width, height int) []byte {
	var b bytes.Buffer
	switch f {
	case PNG, JPEG:
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		for i := 0; i < len(img.Pix); i += 4 {
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
		}
		if f == PNG {
			_ = png.Encode(&b, img)
		} else {
			_ = jpeg.Encode(&b, img, &jpeg.Options{Quality: 90})
		}
	case WebP:
		return webp(c, width, height)
	case SVG:
		_, _ = fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d"><rect width="%d" height="%d" fill="#%02x%02x%02x"/></svg>`,
			width, height, width, height, width, height, c.R, c.G, c.B)
	}
	return b.Bytes()
}

// webp is a lossless WebP of one color. Its bitstream is the signature,
// the size, no alpha hint, version 0, no transform, no color cache and no
// meta codes, then five prefix codes that each hold a single 8-bit symbol:
// green, red, blue, alpha and distance. A one-symbol code reads zero bits,
// so every pixel decodes to that color with no pixel data at all.
func webp(c color.RGBA, width, height int) []byte {
	var w bitWriter
	w.put(0x2f, 8)
	w.put(width-1, 14)
	w.put(height-1, 14)
	w.put(0, 1+3+1+1+1)
	for _, symbol := range []uint8{c.G, c.R, c.B, c.A, 0} {
		w.put(0b101, 3)
		w.put(int(symbol), 8)
	}
	chunk := len(w.buf)
	if chunk%2 == 1 {
		w.buf = append(w.buf, 0)
	}
	out := append([]byte("RIFF"), le32(12+len(w.buf))...)
	out = append(out, "WEBPVP8L"...)
	out = append(out, le32(chunk)...)
	return append(out, w.buf...)
}

func le32(n int) []byte {
	return []byte{byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24)} //nolint:gosec // a WebP header's lengths are a few dozen bytes
}

// bitWriter packs values least significant bit first, as VP8L reads them.
type bitWriter struct {
	buf  []byte
	used int
}

func (w *bitWriter) put(v, n int) {
	for i := range n {
		if w.used%8 == 0 {
			w.buf = append(w.buf, 0)
		}
		w.buf[len(w.buf)-1] |= byte((v>>i)&1) << (w.used % 8)
		w.used++
	}
}
