package release

import (
	"strconv"
	"strings"
)

// Describe is the facts in words, joined with " · ": Quality, the
// revision when it isn't the original, HDR, audio and channels, group,
// size. "WEBDL-1080p · v2 REPACK · Dolby Vision · DD+ 5.1 · FLUX · 6.2 GB".
// It names the first HDR format, audio codec and channel layout in each
// fact, which the title readers list best first, and leaves out each fact
// that's unknown or empty.
func Describe(r Read) string {
	f := r.Facts
	var parts []string
	if q, ok := f.Quality.Get(); ok {
		parts = append(parts, q.Info().Name)
	}
	if !f.Revision.Original() {
		parts = append(parts, f.Revision.String())
	}
	if hdr, ok := firstOf(f.HDR); ok {
		parts = append(parts, hdr.Words())
	}
	var sound []string
	if a, ok := firstOf(f.Audio); ok {
		sound = append(sound, string(a))
	}
	if c, ok := firstOf(f.Channels); ok {
		sound = append(sound, string(c))
	}
	if len(sound) > 0 {
		parts = append(parts, strings.Join(sound, " "))
	}
	if g, ok := f.Group.Get(); ok {
		parts = append(parts, g)
	}
	if n, ok := f.Size.Get(); ok {
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
