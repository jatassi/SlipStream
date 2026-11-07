package series

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
)

// record is one series as its sources have it in one episode order:
// Fetched.Detail on the wire, gzip-encoded JSON, so a 10,000-episode
// series fits the call layer's cap. Seasons and Episodes are the order's,
// each keyed by its TVDB id, which is its level's or Item's key.
type record struct {
	Order    tvdb.Order   `json:"order"`
	Orders   []tvdb.Order `json:"orders"`
	Show     show         `json:"show"`
	Seasons  []season     `json:"seasons"`
	Episodes []episode    `json:"episodes"`
}

// show is the series' own facts. Empty text is a fact its sources lack.
type show struct {
	TVDBID           db.TVDBID      `json:"tvdbId"`
	TMDBID           *db.TMDBID     `json:"tmdbId,omitempty"`
	IMDbID           string         `json:"imdbId,omitempty"`
	EnglishTitle     string         `json:"englishTitle"`
	OriginalTitle    string         `json:"originalTitle,omitempty"`
	OriginalLanguage string         `json:"originalLanguage,omitempty"`
	Country          string         `json:"country,omitempty"`
	Network          string         `json:"network,omitempty"`
	Creators         []person       `json:"creators,omitempty"`
	Cast             []person       `json:"cast,omitempty"`
	Status           tvdb.Status    `json:"status"`
	FirstAired       string         `json:"firstAired,omitempty"`
	LastAired        string         `json:"lastAired,omitempty"`
	NextAired        string         `json:"nextAired,omitempty"`
	AverageRuntime   time.Duration  `json:"averageRuntime,omitempty"`
	Certification    *certification `json:"certification,omitempty"`
	Genres           []string       `json:"genres,omitempty"`
	Aliases          []string       `json:"aliases,omitempty"`
	Overview         string         `json:"overview,omitempty"`
	Trailer          *trailer       `json:"trailer,omitempty"`
	// Awards is nil when OMDb wasn't read, so Store keeps the stored line,
	// and empty when OMDb has none.
	Awards *string `json:"awards,omitempty"`
}

// person is one credit from TMDB. Photo is a TMDB image path.
type person struct {
	Name  string `json:"name"`
	Role  string `json:"role,omitempty"`
	Photo string `json:"photo,omitempty"`
}

// certification is the rating a series carries in Country's system.
type certification struct {
	Code    string `json:"code"`
	Country string `json:"country"`
}

// trailer names a video on Site, such as YouTube, by its Key there.
type trailer struct {
	Site string `json:"site"`
	Key  string `json:"key"`
}

type season struct {
	ID       db.SeasonID `json:"id"`
	Number   int64       `json:"number"`
	Name     string      `json:"name,omitempty"`
	Overview string      `json:"overview,omitempty"`
}

// episode is one episode, numbered in the record's order. AiredSeason and
// AiredNumber are its numbers in Aired order, nil when Aired order doesn't
// list it, because indexers name releases by them whatever the order.
type episode struct {
	ID          db.EpisodeID  `json:"id"`
	Season      int64         `json:"season"`
	Number      int64         `json:"number"`
	Absolute    *int64        `json:"absolute,omitempty"`
	AiredSeason *int64        `json:"airedSeason,omitempty"`
	AiredNumber *int64        `json:"airedNumber,omitempty"`
	AirDate     string        `json:"airDate,omitempty"`
	Runtime     time.Duration `json:"runtime,omitempty"`
	Overview    string        `json:"overview,omitempty"`
	IMDbID      string        `json:"imdbId,omitempty"`
}

func encode(r record) ([]byte, error) {
	var b bytes.Buffer
	z := gzip.NewWriter(&b)
	if err := json.NewEncoder(z).Encode(r); err != nil {
		return nil, fmt.Errorf("encode the record of series %d: %w", r.Show.TVDBID, err)
	}
	if err := z.Close(); err != nil {
		return nil, fmt.Errorf("encode the record of series %d: %w", r.Show.TVDBID, err)
	}
	return b.Bytes(), nil
}

func decode(detail []byte) (record, error) {
	z, err := gzip.NewReader(bytes.NewReader(detail))
	if err != nil {
		return record{}, fmt.Errorf("decode a series record: %w", err)
	}
	raw, err := io.ReadAll(z)
	if err != nil {
		return record{}, fmt.Errorf("decode a series record: %w", err)
	}
	var r record
	if err := json.Unmarshal(raw, &r); err != nil {
		return record{}, fmt.Errorf("decode a series record: %w", err)
	}
	if !slices.Contains(r.Orders, r.Order) {
		return record{}, fmt.Errorf("series %d is in %s order, which isn't one of its orders %v", r.Show.TVDBID, r.Order, r.Orders)
	}
	return r, nil
}
