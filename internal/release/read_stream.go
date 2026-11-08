package release

import (
	"slices"

	"github.com/jatassi/slipstream/internal/regex"
	"github.com/jatassi/slipstream/internal/release/internal/upstream"
)

// A stream value is read when a TRaSH Guides format that names it reads
// the title, by the format's Release Title conditions alone: one of the
// conditions that name the value matches, and every required one holds.
// The format's other conditions test facts other readers own, such as
// x264's "Not Remux", so they take no part.
type trashRule[T any] struct {
	value  T
	format upstream.Format
	// by are the conditions that name value. None is every condition
	// that isn't negated.
	by []string
}

// trashTest is a trashRule compiled: indexes into trashPatterns.
type trashTest[T any] struct {
	value    T
	named    []int
	required []trashCondition
}

type trashCondition struct {
	at     int
	negate bool
}

const releaseTitle = "ReleaseTitleSpecification"

// trashPatterns are the distinct condition patterns, each run once per name.
var trashPatterns []fixed

func trashTests[T any](rules []trashRule[T]) []trashTest[T] {
	out := make([]trashTest[T], len(rules))
	for i, r := range rules {
		out[i].value = r.value
		for _, c := range r.format.Conditions {
			if c.Kind != releaseTitle {
				continue
			}
			at := trashPattern(r.format.Key+": "+c.Name, c.Value)
			if !c.Negate && (len(r.by) == 0 || slices.Contains(r.by, c.Name)) {
				out[i].named = append(out[i].named, at)
			}
			if c.Required {
				out[i].required = append(out[i].required, trashCondition{at, c.Negate})
			}
		}
	}
	return out
}

// trashPattern is where text is in trashPatterns, compiled as Radarr's
// RegexSpecificationBase compiles a condition.
func trashPattern(key, text string) int {
	if at := slices.IndexFunc(trashPatterns, func(f fixed) bool { return f.re.String() == text }); at >= 0 {
		return at
	}
	trashPatterns = append(trashPatterns, compile(upstream.Pattern{Key: key, Text: text, Options: upstream.IgnoreCase | upstream.Compiled}))
	return len(trashPatterns) - 1
}

var (
	hdrTests = trashTests([]trashRule[HDR]{
		{"DV", upstream.TrashRadarrDvBoost, nil},
		{"HDR10+", upstream.TrashRadarrHdr10plusBoost, nil},
		// A bare HDR is HDR10, the base layer every HDR format carries.
		{"HDR10", upstream.TrashRadarrHdr, []string{"HDR10", "HDR"}},
		{"HLG", upstream.TrashRadarrHlg, nil},
		// TRaSH's SDR format also reads a 2160p title with no HDR word as
		// SDR. A name reads SDR only when it says so.
		{"SDR", upstream.TrashRadarrSdr, []string{"SDR"}},
	})
	videoTests = trashTests([]trashRule[VideoCodec]{
		{"x264", upstream.TrashRadarrX264, nil},
		{"x265", upstream.TrashRadarrX265, nil},
		{"AV1", upstream.TrashRadarrAv1, nil},
		{"VP9", upstream.TrashRadarrVp9, nil},
		{"MPEG2", upstream.TrashRadarrMpeg2, nil},
	})
	audioTests = trashTests([]trashRule[AudioCodec]{
		{"TrueHD", upstream.TrashRadarrTruehdAtmos, nil},
		{"TrueHD", upstream.TrashRadarrTruehd, nil},
		// DTS:X rides on a DTS-HD MA stream, and DTS-HD HRA and DTS-ES
		// are lossy DTS.
		{"DTS-HD MA", upstream.TrashRadarrDtsX, nil},
		{"DTS-HD MA", upstream.TrashRadarrDtsHdMa, nil},
		{"DTS", upstream.TrashRadarrDtsHdHra, nil},
		{"DTS", upstream.TrashRadarrDtsEs, nil},
		{"DTS", upstream.TrashRadarrDts, nil},
		{"DD+", upstream.TrashRadarrDdplusAtmos, nil},
		{"DD+", upstream.TrashRadarrDdplus, nil},
		{"DD", upstream.TrashRadarrDd, nil},
		{"AAC", upstream.TrashRadarrAac, nil},
		{"FLAC", upstream.TrashRadarrFlac, nil},
		{"PCM", upstream.TrashRadarrPcm, nil},
		{"Opus", upstream.TrashRadarrOpus, nil},
		{"MP3", upstream.TrashRadarrMp3, nil},
	})
	channelTests = trashTests([]trashRule[Channels]{
		{"7.1", upstream.TrashRadarr71Surround, nil},
		{"5.1", upstream.TrashRadarr51Surround, nil},
		{"2.0", upstream.TrashRadarr20Stereo, nil},
		{"1.0", upstream.TrashRadarr10Mono, nil},
	})
	codec = compile(upstream.RadarrQualityParserCodecRegex)
)

// readStream sets the HDR, video, audio and channel facts name gives,
// reading the name the group reads, as Radarr's Release Title condition
// tests its simpleReleaseTitle. TRaSH has no XviD format, so XviD is
// Radarr's CodecRegex reading XviD or DivX.
func readStream(name string, s *scan, f *Facts) {
	r := trashRun{s: s, t: regex.TextOf(name), done: make([]int8, len(trashPatterns))}
	f.HDR = knownList(values(&r, hdrTests))
	f.Audio = knownList(values(&r, audioTests))
	f.Channels = knownList(values(&r, channelTests))
	if v := values(&r, videoTests); len(v) > 0 {
		f.Video = Known(v[0])
		return
	}
	m, ok := s.find(codec, r.t)
	if ok && slices.ContainsFunc([]string{"xvid", "xvidhd", "divx"}, func(g string) bool { _, ok := m.Group(g); return ok }) {
		f.Video = Known[VideoCodec]("XviD")
	}
}

func knownList[T any](v []T) Fact[[]T] {
	if len(v) == 0 {
		return Fact[[]T]{}
	}
	return Known(v)
}

// trashRun runs trashPatterns over one name, each at most once.
type trashRun struct {
	s    *scan
	t    regex.Text
	done []int8 // 0 not run, 1 matched, -1 not matched
}

func (r *trashRun) matches(at int) bool {
	if r.done[at] == 0 {
		r.done[at] = -1
		if r.s.match(trashPatterns[at], r.t) {
			r.done[at] = 1
		}
	}
	return r.done[at] == 1
}

// values are the values of tests the name reads, each once, in tests'
// order.
func values[T comparable](r *trashRun, tests []trashTest[T]) []T {
	var out []T
	for _, t := range tests {
		if slices.Contains(out, t.value) || !slices.ContainsFunc(t.named, r.matches) {
			continue
		}
		if slices.ContainsFunc(t.required, func(c trashCondition) bool { return r.matches(c.at) == c.negate }) {
			continue
		}
		out = append(out, t.value)
	}
	return out
}
