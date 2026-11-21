package release

import (
	"strconv"
	"strings"
)

// Summary is the facts a Release shows, in the order they read: its
// Quality, its revision when it isn't the original, its first HDR format,
// its first audio codec with its first channel layout, its group and its
// size. The title readers list each list fact best first. A fact the
// title doesn't name is unknown.
type Summary struct {
	Quality  Fact[Quality]
	Revision Fact[Revision]
	HDR      Fact[HDR]
	// Audio is "DD+ 5.1", "TrueHD" with no layout, or "5.1" with no codec.
	Audio Fact[string]
	Group Fact[string]
	Size  Fact[int64]
}

// Summarize is r's facts as a Release shows them.
func Summarize(r Read) Summary {
	f := r.Facts
	s := Summary{Quality: f.Quality, Group: f.Group, Size: f.Size}
	if !f.Revision.Original() {
		s.Revision = Known(f.Revision)
	}
	if hdr, ok := firstOf(f.HDR); ok {
		s.HDR = Known(hdr)
	}
	var sound []string
	if a, ok := firstOf(f.Audio); ok {
		sound = append(sound, string(a))
	}
	if c, ok := firstOf(f.Channels); ok {
		sound = append(sound, string(c))
	}
	if len(sound) > 0 {
		s.Audio = Known(strings.Join(sound, " "))
	}
	return s
}

// Words is s joined with " · ", as a sentence names a Release:
// "WEBDL-1080p · v2 REPACK · Dolby Vision · DD+ 5.1 · FLUX · 6.2 GB".
func (s Summary) Words() string {
	var parts []string
	if q, ok := s.Quality.Get(); ok {
		parts = append(parts, q.Info().Name)
	}
	if r, ok := s.Revision.Get(); ok {
		parts = append(parts, r.String())
	}
	if h, ok := s.HDR.Get(); ok {
		parts = append(parts, h.Words())
	}
	if a, ok := s.Audio.Get(); ok {
		parts = append(parts, a)
	}
	if g, ok := s.Group.Get(); ok {
		parts = append(parts, g)
	}
	if n, ok := s.Size.Get(); ok {
		parts = append(parts, sizeWords(n))
	}
	return strings.Join(parts, " · ")
}

func firstOf[T any](f Fact[[]T]) (T, bool) {
	v, _ := f.Get()
	if len(v) == 0 {
		var zero T
		return zero, false
	}
	return v[0], true
}

// Words is h as a person reads it: Dolby Vision for DV.
func (h HDR) Words() string {
	if h == "DV" {
		return "Dolby Vision"
	}
	return string(h)
}

var sizeUnits = [...]string{"bytes", "KB", "MB", "GB", "TB"}

// sizeWords writes bytes as the web's bytes() does: 1024 to a unit, one
// decimal under 10.
func sizeWords(n int64) string {
	v, unit := float64(n), 0
	for v >= 1024 && unit < len(sizeUnits)-1 {
		v /= 1024
		unit++
	}
	digits := 0
	if unit > 0 && v < 10 {
		digits = 1
	}
	return strconv.FormatFloat(v, 'f', digits, 64) + " " + sizeUnits[unit]
}
