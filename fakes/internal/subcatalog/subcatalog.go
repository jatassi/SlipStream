// Package subcatalog is the Scenario's subtitle_providers catalog as the
// subtitle provider fakes serve it, and the bytes of each subtitle. Its
// cues follow a file's seeded speech through an offset, so whether a
// subtitle passes Timing sync against a file is a fact of the Scenario.
package subcatalog

import (
	"archive/zip"
	"bytes"
	"fmt"
	"hash/fnv"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/speech"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/subtitles/timing"
	"github.com/jatassi/slipstream/internal/subtitles/timing/timingtest"
)

type Format uint8

const (
	SRT Format = iota + 1
	ASS
	SSA
	// Image is PGS, an image subtitle.
	Image
)

var exts = map[Format]string{SRT: "srt", ASS: "ass", SSA: "ssa", Image: "sup"}

func (f Format) Ext() string { return exts[f] }

// Subtitle is one catalog entry. A film's has no Episodes; a pack's has
// several, one archive member each.
type Subtitle struct {
	// ID names it inside its provider.
	ID               string
	IMDb, TMDB, TVDB string
	// Title and Year are its Work's, for a search by text.
	Title    string
	Year     int
	Season   int
	Episodes []int
	// Language is an ISO 639-1 code, or pt-BR.
	Language string
	HI, AI   bool
	Release  string
	Format   Format
	// Speech is the speech each member's cues follow, in Members' order.
	Speech []scenario.MediaSpeech
	// Offset shifts every cue from where the speech is.
	Offset  time.Duration
	Unaired bool
}

func (s Subtitle) Pack() bool { return len(s.Episodes) > 1 }

// Member is one file of a subtitle's download.
type Member struct {
	Name string
	// Episode is the episode it holds, 0 for a film.
	Episode int
	Bytes   []byte
}

// Members are the files a download holds, in episode order: a film's one
// file, or each episode's.
func Members(s Subtitle) []Member {
	if len(s.Episodes) == 0 {
		return []Member{{Name: s.Release + "." + s.Format.Ext(), Bytes: s.text(0)}}
	}
	if !s.Pack() {
		return []Member{{Name: s.Release + "." + s.Format.Ext(), Episode: s.Episodes[0], Bytes: s.text(0)}}
	}
	out := make([]Member, len(s.Episodes))
	for i, e := range s.Episodes {
		out[i] = Member{Name: fmt.Sprintf("%02d %s.%s", e, s.Release, s.Format.Ext()), Episode: e, Bytes: s.text(i)}
	}
	return out
}

// text is member i's subtitle: a person's cues for its speech, shifted.
func (s Subtitle) text(i int) []byte {
	sp := s.Speech[i]
	return Text(s.Format, timingtest.Shift(speech.Schedule(sp).Cues(sp.Seed+1), s.Offset))
}

// junkSize is the notice member SubDL packs open with, as recorded.
const junkSize = 296

// Junk is the notice member a SubDL pack opens with.
func Junk(s Subtitle) Member {
	b := []byte("1\n00:00:00,000 --> 00:00:05,000\nDownloaded from SubDL\n")
	return Member{Name: fmt.Sprintf("00 %s SEASON %d.srt", s.Release, s.Season), Bytes: append(b, bytes.Repeat([]byte(" "), junkSize-len(b)-1)...)}
}

// Zip is ms in a zip, in order.
func Zip(ms []Member) ([]byte, error) {
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, m := range ms {
		w, err := z.Create(m.Name)
		if err != nil {
			return nil, fmt.Errorf("zip %s: %w", m.Name, err)
		}
		if _, err := w.Write(m.Bytes); err != nil {
			return nil, fmt.Errorf("zip %s: %w", m.Name, err)
		}
	}
	if err := z.Close(); err != nil {
		return nil, fmt.Errorf("close the zip: %w", err)
	}
	return b.Bytes(), nil
}

// Text writes spans as a subtitle in f, one numbered line per cue. ASS
// carries a [Fonts] section, which a reader must keep.
func Text(f Format, spans []timing.Span) []byte {
	var b strings.Builder
	switch f {
	case SRT:
		for i, s := range spans {
			fmt.Fprintf(&b, "%d\n%s --> %s\nLine %d\n\n", i+1, srtStamp(s.Start), srtStamp(s.End), i+1)
		}
	case ASS:
		b.WriteString("[Script Info]\nScriptType: v4.00+\nPlayResX: 1920\nPlayResY: 1080\n\n[V4+ Styles]\n")
		b.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
		b.WriteString("Style: Default,Arial,48,&H00FFFFFF,&H000000FF,&H00000000,&H00000000,0,0,0,0,100,100,0,0,1,2,2,2,10,10,10,1\n\n[Events]\n")
		b.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
		for i, s := range spans {
			fmt.Fprintf(&b, "Dialogue: 0,%s,%s,Default,,0,0,0,,Line %d\n", assStamp(s.Start), assStamp(s.End), i+1)
		}
		b.WriteString("\n[Fonts]\nfontname: CustomFont_0.ttf\nM(\\'!!!!!!!!@!%!!!@!!!Q`!!$!-1T5\"0!!!!!X!!!!!!!!!!!!!!!!!!!!!!!!\n")
	case SSA:
		b.WriteString("[Script Info]\nScriptType: v4.00\n\n[V4 Styles]\n")
		b.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, TertiaryColour, BackColour, Bold, Italic, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, AlphaLevel, Encoding\n")
		b.WriteString("Style: Default,Arial,20,16777215,65535,65535,-2147483640,-1,0,1,1,2,2,30,30,10,0,0\n\n[Events]\n")
		b.WriteString("Format: Marked, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
		for i, s := range spans {
			fmt.Fprintf(&b, "Dialogue: Marked=0,%s,%s,Default,,0000,0000,0000,,Line %d\n", assStamp(s.Start), assStamp(s.End), i+1)
		}
	case Image:
		return []byte("PG\x00\x00\x00\x00\x00\x00\x00\x00\x16\x00\x00")
	}
	return []byte(b.String())
}

// srtStamp writes t in SRT's HH:MM:SS,mmm, never below zero.
func srtStamp(t time.Duration) string {
	ms := max(t, 0).Milliseconds()
	return fmt.Sprintf("%02d:%02d:%02d,%03d", ms/3600000, ms/60000%60, ms/1000%60, ms%1000)
}

// assStamp writes t in ASS's H:MM:SS.cc, never below zero.
func assStamp(t time.Duration) string {
	cs := max(t, 0).Milliseconds() / 10
	return fmt.Sprintf("%d:%02d:%02d.%02d", cs/360000, cs/6000%60, cs/100%60, cs%100)
}

// Of is the Scenario's catalog subtitles at provider.
func Of(sc scenario.Scenario, provider string) []Subtitle {
	var out []Subtitle
	for _, c := range sc.SubtitleProviders.Catalog {
		if c.Provider == provider {
			out = append(out, entry(sc, c))
		}
	}
	return out
}

var formats = map[string]Format{"": SRT, "srt": SRT, "ass": ASS, "ssa": SSA, "sup": Image}

func entry(sc scenario.Scenario, c scenario.CatalogSubtitle) Subtitle {
	s := Subtitle{
		ID: c.ID, Season: c.Season, Episodes: c.Episodes, Language: c.Language, HI: c.HI, AI: c.AI,
		Release: c.Release, Format: formats[c.Format], Offset: c.Offset, Unaired: c.Unaired, Title: c.Work,
	}
	if sc.TMDB != nil {
		if i := slices.IndexFunc(sc.TMDB.Movies, func(m scenario.TMDBMovie) bool { return m.Title == c.Work }); i >= 0 {
			m := sc.TMDB.Movies[i]
			s.IMDb, s.TMDB = m.IMDbID, strconv.FormatInt(m.ID, 10)
			s.Year, _ = strconv.Atoi(strings.SplitN(m.Released, "-", 2)[0])
		}
	}
	if sc.TVDB != nil && s.IMDb == "" {
		if i := slices.IndexFunc(sc.TVDB.Series, func(t scenario.TVDBSeries) bool { return t.Name == c.Work }); i >= 0 {
			t := sc.TVDB.Series[i]
			s.IMDb, s.TVDB = t.IMDb, strconv.FormatInt(t.ID, 10)
			if t.TMDB > 0 {
				s.TMDB = strconv.FormatInt(t.TMDB, 10)
			}
		}
	}
	for i := range max(len(c.Episodes), 1) {
		if len(c.Sources) > 0 {
			f, _ := sc.LibraryFile(c.Sources[i])
			s.Speech = append(s.Speech, *f.SpeechTrack())
			continue
		}
		h := fnv.New64a()
		_, _ = h.Write([]byte(c.ID))
		s.Speech = append(s.Speech, scenario.MediaSpeech{Seed: h.Sum64()>>1 + uint64(i), Length: 2 * time.Minute, Density: 0.4}) //nolint:gosec // G115: i is a member index
	}
	return s
}
