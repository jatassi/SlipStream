package tvdb

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/works"
)

// envelope is every TVDB answer: the record or list under data, and the
// page links of a list.
type envelope[T any] struct {
	Data  T      `json:"data"`
	Links *links `json:"links"`
}

type links struct {
	Next       *string `json:"next"`
	TotalItems int     `json:"total_items"`
	PageSize   int     `json:"page_size"`
}

// token reads a 200 login's token and the expiry its JWT payload claims.
func token(body []byte) (string, time.Time, error) {
	var login envelope[struct {
		Token string `json:"token"`
	}]
	if err := json.Unmarshal(body, &login); err != nil {
		return "", time.Time{}, fmt.Errorf("decoding the login: %w", err)
	}
	parts := strings.Split(login.Data.Token, ".")
	if len(parts) != 3 {
		return "", time.Time{}, errors.New("the token is not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", time.Time{}, fmt.Errorf("decoding the token's payload: %w", err)
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return "", time.Time{}, errors.New("the token claims no expiry")
	}
	return login.Data.Token, time.Unix(claims.Exp, 0), nil
}

// refused is the PIN when a refused login's message names it, else the
// key.
func refused(_ int, body []byte) metadata.Credential {
	var b struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &b) == nil && strings.Contains(strings.ToLower(b.Message), "pin") {
		return metadata.CredentialPin
	}
	return metadata.CredentialKey
}

func invalid(route string, err error) error {
	return &works.SourceError{Source: "TVDB", Words: "sent a " + route + " response SlipStream can't read: " + err.Error(), Err: works.ErrInvalid}
}

type searchJSON struct {
	TVDBID       string            `json:"tvdb_id"`
	Name         string            `json:"name"`
	Translations map[string]string `json:"translations"`
	Overview     string            `json:"overview"`
	Overviews    map[string]string `json:"overviews"`
	Year         string            `json:"year"`
	Network      string            `json:"network"`
	ImageURL     string            `json:"image_url"`
	Aliases      []string          `json:"aliases"`
}

type seriesJSON struct {
	ID               int64             `json:"id"`
	Name             string            `json:"name"`
	Image            string            `json:"image"`
	Aliases          []aliasJSON       `json:"aliases"`
	FirstAired       string            `json:"firstAired"`
	LastAired        string            `json:"lastAired"`
	NextAired        string            `json:"nextAired"`
	Status           statusJSON        `json:"status"`
	OriginalCountry  string            `json:"originalCountry"`
	OriginalLanguage string            `json:"originalLanguage"`
	AverageRuntime   *int              `json:"averageRuntime"`
	Overview         string            `json:"overview"`
	OriginalNetwork  *companyJSON      `json:"originalNetwork"`
	LatestNetwork    *companyJSON      `json:"latestNetwork"`
	Genres           []genreJSON       `json:"genres"`
	RemoteIDs        []remoteIDJSON    `json:"remoteIds"`
	Seasons          []seasonJSON      `json:"seasons"`
	SeasonTypes      []seasonTypeJSON  `json:"seasonTypes"`
	Translations     *translationsJSON `json:"translations"`
}

type aliasJSON struct {
	Language string `json:"language"`
	Name     string `json:"name"`
}

type statusJSON struct {
	Name string `json:"name"`
}

type companyJSON struct {
	Name string `json:"name"`
}

type genreJSON struct {
	Name string `json:"name"`
}

type remoteIDJSON struct {
	ID         string `json:"id"`
	SourceName string `json:"sourceName"`
}

type seasonJSON struct {
	ID     int64          `json:"id"`
	Number int            `json:"number"`
	Name   string         `json:"name"`
	Image  string         `json:"image"`
	Type   seasonTypeJSON `json:"type"`
}

type seasonTypeJSON struct {
	Type string `json:"type"`
}

type translationsJSON struct {
	Names     []translationJSON `json:"nameTranslations"`
	Overviews []translationJSON `json:"overviewTranslations"`
}

type translationJSON struct {
	Language string `json:"language"`
	Name     string `json:"name"`
	Overview string `json:"overview"`
	IsAlias  bool   `json:"isAlias"`
}

const (
	sourceIMDb = "IMDB"
	sourceTMDB = "TheMovieDB.com"
)

var statuses = map[string]Status{"Continuing": Continuing, "Ended": Ended, "Upcoming": Upcoming}

func (s seriesJSON) series(lang string) (Series, error) {
	status, ok := statuses[s.Status.Name]
	if !ok {
		return Series{}, invalid("series", fmt.Errorf("status %q is none of Continuing, Ended and Upcoming", s.Status.Name))
	}
	names, overviews := s.Translations.byLanguage()
	out := Series{
		ID: SeriesID(s.ID), Name: translated(names, lang, s.Name), English: translated(names, english, s.Name), Original: s.Name,
		OriginalLanguage: s.OriginalLanguage, Country: s.OriginalCountry, Status: status,
		FirstAired: s.FirstAired, LastAired: s.LastAired, NextAired: s.NextAired,
		Overview: translated(overviews, lang, s.Overview), Poster: artworkPath(s.Image),
	}
	for _, a := range s.Aliases {
		out.Aliases = append(out.Aliases, a.Name)
	}
	for _, n := range []*companyJSON{s.OriginalNetwork, s.LatestNetwork} {
		if n != nil && n.Name != "" {
			out.Network = n.Name
			break
		}
	}
	if s.AverageRuntime != nil {
		out.AverageRuntime = time.Duration(*s.AverageRuntime) * time.Minute
	}
	for _, g := range s.Genres {
		out.Genres = append(out.Genres, g.Name)
	}
	var err error
	if out.IMDb, out.TMDB, err = remoteIDs(s.RemoteIDs); err != nil {
		return Series{}, invalid("series", err)
	}
	for _, t := range s.SeasonTypes {
		if o := Order(t.Type); slices.Contains(orders, o) && !slices.Contains(out.Orders, o) {
			out.Orders = append(out.Orders, o)
		}
	}
	for _, season := range s.Seasons {
		if o := Order(season.Type.Type); slices.Contains(orders, o) {
			out.Seasons = append(out.Seasons, Season{ID: SeasonID(season.ID), Number: season.Number, Order: o, Name: season.Name, Poster: artworkPath(season.Image)})
		}
	}
	return out, nil
}

// byLanguage are t's names, aliases left out, and overviews by language.
func (t *translationsJSON) byLanguage() (names, overviews map[string]string) {
	names, overviews = map[string]string{}, map[string]string{}
	if t == nil {
		return names, overviews
	}
	for _, n := range t.Names {
		if !n.IsAlias {
			names[n.Language] = n.Name
		}
	}
	for _, o := range t.Overviews {
		overviews[o.Language] = o.Overview
	}
	return names, overviews
}

func remoteIDs(ids []remoteIDJSON) (imdb string, tmdb int64, err error) {
	for _, r := range ids {
		switch r.SourceName {
		case sourceIMDb:
			imdb = r.ID
		case sourceTMDB:
			if tmdb, err = strconv.ParseInt(r.ID, 10, 64); err != nil {
				return "", 0, fmt.Errorf("TMDB id %q is not a number", r.ID)
			}
		}
	}
	return imdb, tmdb, nil
}

type episodeJSON struct {
	ID             int64  `json:"id"`
	SeasonNumber   int    `json:"seasonNumber"`
	Number         int    `json:"number"`
	AbsoluteNumber int    `json:"absoluteNumber"`
	Aired          string `json:"aired"`
	Runtime        *int   `json:"runtime"`
	Name           string `json:"name"`
	Overview       string `json:"overview"`
	Image          string `json:"image"`
}

func (e episodeJSON) episode() Episode {
	out := Episode{
		ID: EpisodeID(e.ID), Season: e.SeasonNumber, Number: e.Number, Absolute: e.AbsoluteNumber,
		Aired: e.Aired, Name: e.Name, Overview: e.Overview, Still: artworkPath(e.Image),
	}
	if e.Runtime != nil {
		out.Runtime = time.Duration(*e.Runtime) * time.Minute
	}
	return out
}

type updateJSON struct {
	EntityType string `json:"entityType"`
	RecordID   int64  `json:"recordId"`
}
