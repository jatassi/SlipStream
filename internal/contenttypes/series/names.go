package series

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/release"
)

// marker is a title's first season and episode marker: S01E03, a
// multi-episode S01E03E04 or S01E03-E04, or a season pack's S01.
var marker = regexp.MustCompile(`(?i)\bS(\d{1,4})(?:E(\d{1,4})((?:-?E\d{1,4})*))?\b`)

var episodeNumber = regexp.MustCompile(`\d+`)

// ReadName reads a title's first marker as its one Mark.
func (*Series) ReadName(p release.Prepared) release.Name {
	m := marker.FindStringSubmatch(p.Title)
	if m == nil {
		return release.Name{}
	}
	season, _ := strconv.Atoi(m[1])
	mk := marked{season: season}
	span := release.Level
	if m[2] != "" {
		mk.first, _ = strconv.Atoi(m[2])
		mk.last, span = mk.first, release.One
		if more := episodeNumber.FindAllString(m[3], -1); len(more) > 0 {
			mk.last, _ = strconv.Atoi(more[len(more)-1])
			span = release.Several
		}
	}
	return release.Name{Marks: []release.Mark{{Words: strings.ToUpper(m[0]), Key: mk.key()}}, Span: span}
}

// CleanTitle is the title as given.
func (*Series) CleanTitle(title string) string { return title }

// marked is a Mark's key read back: a season's episodes first to last, or
// the whole season when first is 0.
type marked struct{ season, first, last int }

func (m marked) key() string {
	switch m.first {
	case 0:
		return fmt.Sprintf("s%d", m.season)
	case m.last:
		return fmt.Sprintf("s%de%d", m.season, m.first)
	}
	return fmt.Sprintf("s%de%d-%d", m.season, m.first, m.last)
}

func markedBy(key string) marked {
	var m marked
	if n, _ := fmt.Sscanf(key, "s%de%d-%d", &m.season, &m.first, &m.last); n == 2 {
		m.last = m.first
	}
	return m
}
