package movie

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/jatassi/slipstream/internal/contenttypes/movie/internal/db"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

const day = 24 * time.Hour

const (
	homeRelease   db.CountsAs = "home"
	cinemaRelease db.CountsAs = "cinema"
)

// The movie settings, stored in movie_settings.
var (
	CountsAs = setting.Choice[db.CountsAs]{Key: "movie.countsAs", Label: "Movies count as out on", Default: homeRelease, Options: []setting.Option[db.CountsAs]{
		{Value: homeRelease, Label: "Home release"},
		{Value: cinemaRelease, Label: "Cinema release"},
	}}
	CinemaDelay = setting.Duration{Key: "movie.cinemaDelay", Label: "A cinema release counts as out after", Default: 90 * day, Max: 365 * day, Step: day}
)

func (*Movie) SettingFields(ctx context.Context, r storage.Reader[file]) ([]setting.Field, error) {
	row, err := db.New(r).GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	return []setting.Field{CountsAs.Field(row.CountsAs), CinemaDelay.Field(row.CinemaDelaySeconds)}, nil
}

func (*Movie) ApplySettings(ctx context.Context, tx *storage.Tx[file], p setting.Patch) (bool, error) {
	q := db.New(tx)
	row, err := q.GetSettings(ctx)
	if err != nil {
		return false, err
	}
	was := db.SetSettingsParams(row)
	next := was
	for _, key := range slices.Sorted(maps.Keys(p)) {
		switch key {
		case CountsAs.Key:
			s, err := CountsAs.ParseJSON(p[key])
			if err != nil {
				return false, err
			}
			next.CountsAs = s.Override()
		case CinemaDelay.Key:
			s, err := CinemaDelay.ParseJSON(p[key])
			if err != nil {
				return false, err
			}
			next.CinemaDelaySeconds = s.Override()
		default:
			return false, &setting.UnknownKeyError{Key: key}
		}
	}
	if reflect.DeepEqual(next, was) {
		return false, nil
	}
	return true, q.SetSettings(ctx, next)
}

// rule is how a movie's release dates make it count as out, under the
// movie settings in a region.
type rule struct {
	countsAs db.CountsAs
	delay    time.Duration
	region   string
}

func ruleIn(ctx context.Context, q *db.Queries, region string) (rule, error) {
	row, err := q.GetSettings(ctx)
	if err != nil {
		return rule{}, err
	}
	return rule{countsAs: CountsAs.Value(row.CountsAs), delay: CinemaDelay.Value(row.CinemaDelaySeconds), region: region}, nil
}

// basis is a release date, the words for the release that set it and the
// delay after it before the movie counts as out.
type basis struct {
	date  string
	words string
	delay time.Duration
}

// available is the release a movie counts as out on. Home: the earliest
// digital or physical release in the region, else in any country; else
// the earliest limited or wide cinema release in the region, else in any
// country, plus the delay; else TMDB's primary release date plus the
// delay. Cinema: the earlier of that cinema release and the home release.
// false: no date at all.
func (r rule) available(rs []releaseDate, primary *string) (basis, bool) {
	home, atHome := earliest(rs, r.region, tmdb.Digital, tmdb.Physical)
	cinema, inCinemas := earliest(rs, r.region, tmdb.TheatricalLimited, tmdb.Theatrical)
	switch {
	case atHome && (r.countsAs == homeRelease || !inCinemas || home.date <= cinema.date):
		return home, true
	case inCinemas && r.countsAs == cinemaRelease:
		return cinema, true
	case inCinemas:
		cinema.delay = r.delay
		return cinema, true
	case primary != nil:
		return basis{date: *primary, words: "Release date", delay: r.delay}, true
	}
	return basis{}, false
}

// day is the day b's movie counts as out and the words for why.
func (b basis) day() (works.Date, string, error) {
	d, ok := dayOf(b.date)
	if !ok {
		return works.Date{}, "", fmt.Errorf("release date %q is not a day", b.date)
	}
	words := b.words
	if b.delay > 0 {
		d = d.AddDate(0, 0, int(b.delay/day))
		words += " plus " + setting.Words(b.delay)
	}
	return works.Date{Year: d.Year(), Month: int(d.Month()), Day: d.Day()}, words, nil
}

var kindWords = map[tmdb.ReleaseType]string{
	tmdb.TheatricalLimited: "Limited cinema release",
	tmdb.Theatrical:        "Cinema release",
	tmdb.Digital:           "Digital release",
	tmdb.Physical:          "Physical release",
}

// earliest is the earliest release of kinds in region, else in any
// country.
func earliest(rs []releaseDate, region string, kinds ...tmdb.ReleaseType) (basis, bool) {
	var inRegion, anywhere *releaseDate
	for i, r := range rs {
		if !slices.Contains(kinds, r.Kind) {
			continue
		}
		if anywhere == nil || r.Date < anywhere.Date {
			anywhere = &rs[i]
		}
		if r.Country == region && (inRegion == nil || r.Date < inRegion.Date) {
			inRegion = &rs[i]
		}
	}
	first := cmp.Or(inRegion, anywhere)
	if first == nil {
		return basis{}, false
	}
	return basis{date: first.Date, words: kindWords[first.Kind] + " in " + countryName(first.Country)}, true
}

// countryName is the English name of an ISO 3166-1 country code, the code
// itself when it names none.
func countryName(code string) string {
	r, err := language.ParseRegion(code)
	if err != nil {
		return code
	}
	return cmp.Or(display.English.Regions().Name(r), code)
}
