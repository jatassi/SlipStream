// Command gen draws the tray icons from brand/app-icon.svg and
// brand/mark.svg's geometry: tray.ico for the Windows notification area
// and tray-template.png for the macOS menu bar, which AppKit tints.
// Run it from the desktop package: go run ./testdata/gen
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

// The mark in brand/mark.svg's 64-unit viewBox: an S of three lines and two
// half circles, stroked 8 wide with round caps, and two dots.
const (
	strokeHalf = 4
	dotRadius  = 5.5
	arcRadius  = 9
)

type point struct{ x, y float64 }

var (
	lines = [][2]point{{{40, 14}, {25, 14}}, {{25, 32}, {39, 32}}, {{39, 50}, {24, 50}}}
	// Each arc is the half of its circle on one side of the center: the
	// upper turn bulges left, the lower one right.
	arcs = []struct {
		center point
		left   bool
	}{{point{25, 23}, true}, {point{39, 41}, false}}
	dots = []point{{52, 14}, {12, 50}}
)

var (
	white = color.NRGBA{0xf2, 0xf4, 0xf7, 0xff}
	blue  = color.NRGBA{0x74, 0xaf, 0xfb, 0xff}
	black = color.NRGBA{0x0c, 0x0d, 0x0e, 0xff}
	ink   = color.NRGBA{0, 0, 0, 0xff}
)

func main() {
	var ico []*image.NRGBA
	for _, size := range []int{16, 20, 24, 32, 40, 48, 64} {
		ico = append(ico, render(size, appIcon))
	}
	write("tray.ico", icoFile(ico))
	write("tray-template.png", encode(render(32, template)))
}

func write(name string, data []byte) {
	if err := os.WriteFile(name, data, 0o644); err != nil {
		log.Fatal(err)
	}
}

// appIcon is brand/app-icon.svg: the mark at 70% on a dark rounded square.
func appIcon(p point) (color.NRGBA, bool) {
	if !inRoundedSquare(p, 64, 14) {
		return color.NRGBA{}, false
	}
	m := point{(p.x - 9.6) / 0.7, (p.y - 9.6) / 0.7}
	switch {
	case onDot(m):
		return blue, true
	case onStroke(m):
		return white, true
	}
	return black, true
}

// template is the mark alone in one ink: AppKit draws a template image's
// alpha in the menu bar's own color.
func template(p point) (color.NRGBA, bool) {
	return ink, onDot(p) || onStroke(p)
}

func inRoundedSquare(p point, size, r float64) bool {
	cx := math.Max(r, math.Min(size-r, p.x))
	cy := math.Max(r, math.Min(size-r, p.y))
	return p.x >= 0 && p.y >= 0 && p.x <= size && p.y <= size && math.Hypot(p.x-cx, p.y-cy) <= r
}

func onDot(p point) bool {
	for _, d := range dots {
		if math.Hypot(p.x-d.x, p.y-d.y) <= dotRadius {
			return true
		}
	}
	return false
}

func onStroke(p point) bool {
	for _, l := range lines {
		if segmentDistance(p, l[0], l[1]) <= strokeHalf {
			return true
		}
	}
	for _, a := range arcs {
		if (p.x <= a.center.x) == a.left && math.Abs(math.Hypot(p.x-a.center.x, p.y-a.center.y)-arcRadius) <= strokeHalf {
			return true
		}
	}
	return false
}

func segmentDistance(p, a, b point) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	t := math.Max(0, math.Min(1, ((p.x-a.x)*dx+(p.y-a.y)*dy)/(dx*dx+dy*dy)))
	return math.Hypot(p.x-a.x-t*dx, p.y-a.y-t*dy)
}

// render draws shade at size pixels square, averaging 8x8 samples a pixel.
func render(size int, shade func(point) (color.NRGBA, bool)) *image.NRGBA {
	const samples = 8
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	scale := 64 / float64(size)
	for y := range size {
		for x := range size {
			var r, g, b, a float64
			for sy := range samples {
				for sx := range samples {
					p := point{(float64(x) + (float64(sx)+0.5)/samples) * scale, (float64(y) + (float64(sy)+0.5)/samples) * scale}
					if c, ok := shade(p); ok {
						r, g, b, a = r+float64(c.R), g+float64(c.G), b+float64(c.B), a+1
					}
				}
			}
			if a > 0 {
				img.SetNRGBA(x, y, color.NRGBA{uint8(r / a), uint8(g / a), uint8(b / a), uint8(math.Round(255 * a / samples / samples))})
			}
		}
	}
	return img
}

func encode(img image.Image) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		log.Fatal(err)
	}
	return b.Bytes()
}

// icoFile is an ICO of PNG images, which Windows reads since Vista.
func icoFile(images []*image.NRGBA) []byte {
	var head, body bytes.Buffer
	le := func(b *bytes.Buffer, v any) {
		if err := binary.Write(b, binary.LittleEndian, v); err != nil {
			log.Fatal(err)
		}
	}
	le(&head, [3]uint16{0, 1, uint16(len(images))})
	offset := 6 + 16*len(images)
	for _, img := range images {
		data := encode(img)
		size := uint8(img.Bounds().Dx())
		le(&head, [4]uint8{size, size, 0, 0})
		le(&head, [2]uint16{1, 32})
		le(&head, [2]uint32{uint32(len(data)), uint32(offset + body.Len())})
		body.Write(data)
	}
	return append(head.Bytes(), body.Bytes()...)
}
