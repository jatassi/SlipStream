//go:build ignore

// Writes the ambient sampler's fixture images at the sample's own 24 × 36,
// so the canvas copies them without resampling.
//
//	go run ./web/src/test/ambient/fixtures.go
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
)

func main() {
	white := color.NRGBA{255, 255, 255, 255}
	fixtures := map[string]func(y int) color.NRGBA{
		"white.png": func(int) color.NRGBA { return white },
		"black.png": func(int) color.NRGBA { return color.NRGBA{0, 0, 0, 255} },
		"red.png":   func(int) color.NRGBA { return color.NRGBA{255, 0, 0, 255} },
		"two-tone.png": func(y int) color.NRGBA {
			if y < 18 {
				return color.NRGBA{40, 90, 200, 255}
			}
			return white
		},
	}
	for name, at := range fixtures {
		img := image.NewNRGBA(image.Rect(0, 0, 24, 36))
		for y := range 36 {
			for x := range 24 {
				img.SetNRGBA(x, y, at(y))
			}
		}
		f, err := os.Create(filepath.Join("web", "src", "test", "ambient", name))
		if err != nil {
			log.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			log.Fatal(err)
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}
}
