package tokens

import (
	"cmp"
	"fmt"
	"math"
	"regexp"
	"strconv"
)

// HSLA is a color as hue in degrees, saturation and lightness in percent,
// and alpha.
type HSLA struct {
	H, S, L, A float64
}

func (c HSLA) String() string {
	return fmt.Sprintf("hsl(%s %s%% %s%% / %s)", num(c.H), num(c.S), num(c.L), num(c.A))
}

// RGBA is a color as sRGB channels from 0 to 255, and alpha.
type RGBA struct {
	R, G, B, A float64
}

// RGBA converts as Python's colorsys.hls_to_rgb does, so the contrast proof
// keeps the numbers of the script it replaced.
func (c HSLA) RGBA() RGBA {
	h, l, s := c.H/360, c.L/100, c.S/100
	if s == 0 {
		return RGBA{l * 255, l * 255, l * 255, c.A}
	}
	m2 := l + s - l*s
	if l <= 0.5 {
		m2 = l * (1 + s)
	}
	m1 := 2*l - m2
	return RGBA{hueChannel(m1, m2, h+1.0/3) * 255, hueChannel(m1, m2, h) * 255, hueChannel(m1, m2, h-1.0/3) * 255, c.A}
}

func hueChannel(m1, m2, hue float64) float64 {
	hue = pyMod1(hue)
	switch {
	case hue < 1.0/6:
		return m1 + (m2-m1)*hue*6
	case hue < 0.5:
		return m2
	case hue < 2.0/3:
		return m1 + (m2-m1)*(2.0/3-hue)*6
	default:
		return m1
	}
}

// pyMod1 is Python's x % 1.0, whose result takes the divisor's sign.
func pyMod1(x float64) float64 {
	m := math.Mod(x, 1)
	if m < 0 {
		m++
	}
	return m
}

var (
	hslCSS  = regexp.MustCompile(`^hsl\(([0-9.]+) ([0-9.]+)% ([0-9.]+)%(?: / ([0-9.]+))?\)$`)
	rgbaCSS = regexp.MustCompile(`^rgba\((\d+), (\d+), (\d+), ([0-9.]+)\)$`)
	hexCSS  = regexp.MustCompile(`^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$`)
)

func parseHSL(s string) (HSLA, error) {
	m := hslCSS.FindStringSubmatch(s)
	if m == nil {
		return HSLA{}, fmt.Errorf("%q is not hsl(<hue> <sat>%% <light>%%[ / <alpha>])", s)
	}
	v, err := floats(m[1], m[2], m[3], cmp.Or(m[4], "1"))
	if err != nil {
		return HSLA{}, fmt.Errorf("%q: %w", s, err)
	}
	return HSLA{v[0], v[1], v[2], v[3]}, nil
}

// parseLiteral reads a fixed color written as #rrggbb, rgba() or hsl().
func parseLiteral(s string) (RGBA, error) {
	if m := hexCSS.FindStringSubmatch(s); m != nil {
		var c [3]float64
		for i, h := range m[1:] {
			n, err := strconv.ParseUint(h, 16, 8)
			if err != nil {
				return RGBA{}, fmt.Errorf("%q: %w", s, err)
			}
			c[i] = float64(n)
		}
		return RGBA{c[0], c[1], c[2], 1}, nil
	}
	if m := rgbaCSS.FindStringSubmatch(s); m != nil {
		v, err := floats(m[1:]...)
		if err != nil {
			return RGBA{}, fmt.Errorf("%q: %w", s, err)
		}
		return RGBA{v[0], v[1], v[2], v[3]}, nil
	}
	c, err := parseHSL(s)
	if err != nil {
		return RGBA{}, fmt.Errorf("%q is not #rrggbb, rgba() or hsl()", s)
	}
	return c.RGBA(), nil
}

func floats(texts ...string) ([]float64, error) {
	out := make([]float64, len(texts))
	for i, t := range texts {
		f, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return nil, fmt.Errorf("parse %q: %w", t, err)
		}
		out[i] = f
	}
	return out, nil
}
