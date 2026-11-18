// Package subcatalog is the catalog the subtitle provider fakes serve,
// and Render, which writes a catalog subtitle's bytes. Its cues follow a
// seeded speech schedule through an offset, so whether a subtitle passes
// Timing sync against a file is a fact of the catalog.
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

type Catalog struct{ Subtitles []Subtitle }

// Subtitle is one catalog entry. A film's has no Episodes; a pack's has
// several, one archive member each.
type Subtitle struct {
	// ID names it inside its provider.
	ID               string
	IMDb, TMDB, TVDB string
	Season           int
	Episodes         []int
	// Language is an ISO 639-1 code.
	Language string
	HI, AI   bool
	Release  string
	Format   Format
	Speech   Speech
	// Offset shifts every cue from where the speech is.
	Offset time.Duration
}

func (s Subtitle) Pack() bool { return len(s.Episodes) > 1 }

// Speech is the seeded schedule a file's speech, and so its subtitles'
// cues, follow.
type Speech struct {
	Seed    uint64
	Length  time.Duration
	Density float64
}

// Schedule is where speech falls in the file of episode, 0 for a film.
func (sp Speech) Schedule(episode int) timingtest.Schedule {
	return timingtest.NewSchedule(sp.Seed+uint64(episode), sp.Length, sp.Density) //nolint:gosec // G115: an episode number is never negative
}

// Cues are the times a person gave episode's subtitle, before Offset.
func (sp Speech) Cues(episode int) []timing.Span {
	return sp.Schedule(episode).Cues(sp.Seed + uint64(episode) + 1) //nolint:gosec // G115: an episode number is never negative
}

// Member is one file of a subtitle's download.
type Member struct {
	Name  string
	Bytes []byte
}

// junkSize is the notice member SubDL packs open with, as recorded.
const junkSize = 296

// Members are the files a download holds, in archive order: a film's one
// file, or a pack's notice and then each episode's file.
func Members(s Subtitle) []Member {
	if !s.Pack() {
		episode := 0
		if len(s.Episodes) == 1 {
			episode = s.Episodes[0]
		}
		return []Member{{Name: s.Release + "." + s.Format.Ext(), Bytes: Text(s.Format, timingtest.Shift(s.Speech.Cues(episode), s.Offset))}}
	}
	out := []Member{{Name: fmt.Sprintf("00 %s SEASON %d.srt", s.Release, s.Season), Bytes: junk()}}
	for _, e := range s.Episodes {
		out = append(out, Member{Name: fmt.Sprintf("%02d %s.%s", e, s.Release, s.Format.Ext()), Bytes: Text(s.Format, timingtest.Shift(s.Speech.Cues(e), s.Offset))})
	}
	return out
}

// Render is the subtitle's download: a zip of its Members.
func Render(s Subtitle) ([]byte, error) {
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, m := range Members(s) {
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

func junk() []byte {
	b := []byte("1\n00:00:00,000 --> 00:00:05,000\nDownloaded from SubDL\n")
	return append(b, bytes.Repeat([]byte(" "), junkSize-len(b)-1)...)
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

// speech is the file schedule a Scenario's Work gets: two minutes, two
// fifths speech, seeded by its id.
func speech(id string) Speech {
	h := fnv.New64a()
	_, _ = h.Write([]byte(id))
	return Speech{Seed: h.Sum64() >> 1, Length: 2 * time.Minute, Density: 0.4}
}

// FromScenario is the catalog every Work the Scenario's metadata fakes
// know gets until a Scenario names its own: for each film an English SRT
// in time, the same machine translated, and an image subtitle; for each
// episode an English SRT in time; and for each season of two or more
// episodes an English pack, which every other season's lists as not a
// full season.
func FromScenario(sc scenario.Scenario) Catalog {
	var c Catalog
	if sc.TMDB != nil {
		for _, m := range sc.TMDB.Movies {
			c.Subtitles = append(c.Subtitles, films(m)...)
		}
	}
	if sc.TVDB != nil {
		for _, series := range sc.TVDB.Series {
			if series.IMDb != "" {
				c.Subtitles = append(c.Subtitles, episodes(series)...)
			}
		}
	}
	return c
}

func films(m scenario.TMDBMovie) []Subtitle {
	base := Subtitle{IMDb: m.IMDbID, TMDB: strconv.FormatInt(m.ID, 10), Language: "en", Release: strings.ReplaceAll(m.Title, " ", ".") + ".1080p.WEB-DL-SLIP", Speech: speech(m.IMDbID)}
	var out []Subtitle
	for i, v := range []func(*Subtitle){
		func(s *Subtitle) { s.Format = SRT },
		func(s *Subtitle) { s.Format, s.AI = SRT, true },
		func(s *Subtitle) { s.Format = Image },
	} {
		s := base
		s.ID = fmt.Sprintf("%s-%d", m.IMDbID, i)
		v(&s)
		out = append(out, s)
	}
	return out
}

func episodes(series scenario.TVDBSeries) []Subtitle {
	sp := speech(series.IMDb)
	name := strings.ReplaceAll(series.Name, " ", ".")
	tvdb := strconv.FormatInt(series.ID, 10)
	var out []Subtitle
	for _, season := range series.Seasons {
		var numbers []int
		for _, e := range season.Episodes {
			out = append(out, Subtitle{
				ID: fmt.Sprintf("%s-s%02de%02d", series.IMDb, season.Number, e.Number), IMDb: series.IMDb, TVDB: tvdb,
				Season: season.Number, Episodes: []int{e.Number}, Language: "en", Format: SRT, Speech: sp,
				Release: fmt.Sprintf("%s.S%02dE%02d.1080p.WEB-DL-SLIP", name, season.Number, e.Number),
			})
			numbers = append(numbers, e.Number)
		}
		if len(numbers) < 2 {
			continue
		}
		slices.Sort(numbers)
		out = append(out, Subtitle{
			ID: fmt.Sprintf("%s-s%02d", series.IMDb, season.Number), IMDb: series.IMDb, TVDB: tvdb,
			Season: season.Number, Episodes: numbers, Language: "en", Format: SRT, Speech: sp,
			Release: fmt.Sprintf("%s.S%02d.1080p.WEB-DL-SLIP", name, season.Number),
		})
	}
	return out
}
