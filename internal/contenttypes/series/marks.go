package series

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/release"
)

type markKind uint8

const (
	markEpisodes markKind = iota + 1
	markSeason
	markSeasons
	markPart
	markAbsolute
	markSpecial
	markAired
)

// marked is one Mark read back: a season's episodes first to last, a
// whole season, seasons season to last, part of a season, absolute
// numbers first to last, an absolute special such as 12.5, or an air
// date with its part, 0 when it names none.
type marked struct {
	kind                markKind
	season, first, last int
	special             float64
	aired               time.Time
}

// marks are the Marks a parsed name numbers, with its Span.
func (r *parsed) marks() ([]release.Mark, release.Span) {
	var ms []marked
	switch {
	case len(r.episodes) > 0:
		ms = append(ms, marked{kind: markEpisodes, season: r.seasons[0], first: r.episodes[0], last: r.episodes[len(r.episodes)-1]})
	case r.fullSeason && len(r.seasons) > 1:
		ms = append(ms, marked{kind: markSeasons, season: r.seasons[0], last: r.seasons[len(r.seasons)-1]})
	case r.fullSeason:
		ms = append(ms, marked{kind: markSeason, season: r.seasons[0]})
	case r.part > 0:
		ms = append(ms, marked{kind: markPart, season: r.seasons[0], first: r.part})
	case !r.airDate.IsZero():
		ms = append(ms, marked{kind: markAired, aired: r.airDate, first: r.dailyPart})
	}
	switch {
	case len(r.absolutes) > 0:
		ms = append(ms, marked{kind: markAbsolute, first: r.absolutes[0], last: r.absolutes[len(r.absolutes)-1]})
	case r.specialAbsolute != 0:
		ms = append(ms, marked{kind: markSpecial, special: r.specialAbsolute})
	}
	out := make([]release.Mark, len(ms))
	for i, m := range ms {
		out[i] = release.Mark{Words: m.words(), Key: m.key()}
	}
	return out, r.span()
}

// span is Sonarr's ReleaseType, with a partial season as several
// episodes of one season.
func (r *parsed) span() release.Span {
	switch {
	case len(r.episodes) > 1 || len(r.absolutes) > 1:
		return release.Several
	case len(r.episodes) == 1 || len(r.absolutes) == 1 || r.specialAbsolute != 0 || !r.airDate.IsZero():
		return release.One
	case r.fullSeason && len(r.seasons) > 1:
		return release.Levels
	case r.fullSeason:
		return release.Level
	case r.part > 0:
		return release.Several
	}
	return release.SpanUnknown
}

func (m marked) key() string {
	switch m.kind {
	case markEpisodes:
		if m.first == m.last {
			return fmt.Sprintf("s%de%d", m.season, m.first)
		}
		return fmt.Sprintf("s%de%d-%d", m.season, m.first, m.last)
	case markSeason:
		return fmt.Sprintf("s%d", m.season)
	case markSeasons:
		return fmt.Sprintf("s%d-%d", m.season, m.last)
	case markPart:
		return fmt.Sprintf("s%dp%d", m.season, m.first)
	case markAbsolute:
		if m.first == m.last {
			return fmt.Sprintf("a%d", m.first)
		}
		return fmt.Sprintf("a%d-%d", m.first, m.last)
	case markSpecial:
		return "a" + strconv.FormatFloat(m.special, 'f', -1, 64)
	case markAired:
		if m.first == 0 {
			return "d" + m.aired.Format(time.DateOnly)
		}
		return fmt.Sprintf("d%s.%d", m.aired.Format(time.DateOnly), m.first)
	}
	return ""
}

// words is m as the many-Items layout names it: S01E02, S01E02-E04, S01,
// S01-S03, S01 Part 2, an absolute number alone, or an air date with its
// part: 2024-03-15 Part 2.
func (m marked) words() string {
	switch m.kind {
	case markEpisodes:
		if m.first == m.last {
			return fmt.Sprintf("S%02dE%02d", m.season, m.first)
		}
		return fmt.Sprintf("S%02dE%02d-E%02d", m.season, m.first, m.last)
	case markSeason:
		return fmt.Sprintf("S%02d", m.season)
	case markSeasons:
		return fmt.Sprintf("S%02d-S%02d", m.season, m.last)
	case markPart:
		return fmt.Sprintf("S%02d Part %d", m.season, m.first)
	case markAbsolute:
		if m.first == m.last {
			return strconv.Itoa(m.first)
		}
		return fmt.Sprintf("%d-%d", m.first, m.last)
	case markSpecial:
		return strconv.FormatFloat(m.special, 'f', -1, 64)
	case markAired:
		if m.first == 0 {
			return m.aired.Format(time.DateOnly)
		}
		return fmt.Sprintf("%s Part %d", m.aired.Format(time.DateOnly), m.first)
	}
	return ""
}

// markedBy reads a key back. Each layout is tried longest first, as
// Sscanf leaves what follows the last verb unread.
func markedBy(key string) marked {
	var m marked
	scan := func(layout string, kind markKind, to ...any) bool {
		n, err := fmt.Sscanf(key, layout, to...)
		if err != nil || n != len(to) {
			return false
		}
		m.kind = kind
		return true
	}
	switch {
	case strings.HasPrefix(key, "d"):
		date, part, _ := strings.Cut(key[1:], ".")
		m.kind = markAired
		m.aired, _ = time.Parse(time.DateOnly, date)
		m.first, _ = strconv.Atoi(part)
	case strings.HasPrefix(key, "a") && strings.Contains(key, "."):
		m.kind = markSpecial
		m.special, _ = strconv.ParseFloat(key[1:], 64)
	case scan("s%de%d-%d", markEpisodes, &m.season, &m.first, &m.last):
	case scan("s%de%d", markEpisodes, &m.season, &m.first):
		m.last = m.first
	case scan("s%dp%d", markPart, &m.season, &m.first):
	case scan("s%d-%d", markSeasons, &m.season, &m.last):
	case scan("s%d", markSeason, &m.season):
	case scan("a%d-%d", markAbsolute, &m.first, &m.last):
	case scan("a%d", markAbsolute, &m.first):
		m.last = m.first
	}
	return m
}
