package music

import (
	"cmp"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/fakes/contenttypes/music/internal/db"
	"github.com/jatassi/slipstream/fakes/contenttypes/music/internal/musicbrainz"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

var Def = contenttype.Define("music", schemaFiles, DefaultEndpoints, contenttype.Halves[Endpoints]{
	Provider: func(d contenttype.ProviderDeps, e Endpoints) metadata.Source {
		return &Source{mb: musicbrainz.New(d.Lines, e.Metadata)}
	},
	Library: func(contenttype.LibraryDeps) contenttype.Library { return &Music{} },
})

type Endpoints struct {
	Metadata endpoint.URL `yaml:"metadata"`
	CoverArt endpoint.URL `yaml:"coverart"`
}

func DefaultEndpoints() Endpoints {
	return Endpoints{
		Metadata: endpoint.MustParseURL("https://musicbrainz.invalid/ws/2"),
		CoverArt: endpoint.MustParseURL("https://coverart.invalid"),
	}
}

var Description = works.Description{
	Name:   "Music",
	Icon:   works.IconNote,
	Work:   works.Noun{One: "Artist", Many: "Artists"},
	Levels: []works.Noun{{One: "Album", Many: "Albums"}},
	Item:   works.Noun{One: "Track", Many: "Tracks"},
	Fields: []works.FieldSpec{
		{Key: "country", Label: "Country", Tier: works.TierWork, Kind: works.KindText},
		{Key: "released", Label: "Released", Tier: works.TierLevel(1), Kind: works.KindDate},
		{Key: "kind", Label: "Kind", Tier: works.TierLevel(1), Kind: works.KindText},
		{Key: "disc", Label: "Disc", Tier: works.TierItem, Kind: works.KindNumber},
		{Key: "length", Label: "Length", Tier: works.TierItem, Kind: works.KindDuration},
	},
}

// Source is the provider half: it reads MusicBrainz on any Metadata
// holder.
type Source struct {
	mb *musicbrainz.Client
}

// Music is the library half.
type Music struct{}

func (*Music) Describe() works.Description { return Description }

func (s *Source) Search(ctx context.Context, ask metadata.Ask, query string) ([]works.Candidate, error) {
	artists, err := s.mb.SearchArtists(ctx, ask, query)
	if err != nil {
		return nil, err
	}
	out := make([]works.Candidate, len(artists))
	for i, a := range artists {
		out[i] = works.Candidate{Key: works.Key(a.ID), Label: a.Name, Fields: artistFields(a.Country)}
	}
	return out, nil
}

type detail struct {
	Country        string        `json:"country"`
	Disambiguation string        `json:"disambiguation"`
	Albums         []albumDetail `json:"albums"`
}

type albumDetail struct {
	Key         works.Key     `json:"key"`
	Released    string        `json:"released"`
	PrimaryType string        `json:"primaryType"`
	Release     db.ReleaseID  `json:"release"`
	Tracks      []trackDetail `json:"tracks"`
}

type trackDetail struct {
	Key      works.Key `json:"key"`
	Disc     int64     `json:"disc"`
	Number   int64     `json:"number"`
	LengthMs *int64    `json:"lengthMs"`
}

func (s *Source) Fetch(ctx context.Context, ask metadata.Ask, key works.Key) (works.Fetched, error) {
	artist, err := s.mb.Artist(ctx, ask, string(key))
	if err != nil {
		return works.Fetched{}, err
	}
	albums, err := s.mb.Albums(ctx, ask, artist.ID)
	if err != nil {
		return works.Fetched{}, err
	}
	f := works.Fetched{Label: artist.Name, SortLabel: artist.SortName}
	d := detail{Country: artist.Country, Disambiguation: artist.Disambiguation}
	for _, album := range albums {
		canonical, ok, err := s.mb.Canonical(ctx, ask, album.ID)
		if err != nil {
			return works.Fetched{}, err
		}
		if !ok {
			continue
		}
		released := album.FirstRelease
		if _, err := works.ParseDate(released); err != nil {
			released = ""
		}
		level := works.Entry{Key: works.Key(album.ID), Label: album.Title}
		ad := albumDetail{Key: level.Key, Released: released, PrimaryType: album.PrimaryType, Release: db.ReleaseID(canonical.ID)}
		slices.SortFunc(canonical.Media, func(a, b musicbrainz.Medium) int { return cmp.Compare(a.Position, b.Position) })
		for _, medium := range canonical.Media {
			slices.SortFunc(medium.Tracks, func(a, b musicbrainz.Track) int { return cmp.Compare(a.Position, b.Position) })
			for _, t := range medium.Tracks {
				track := works.Entry{Key: works.Key(album.ID + "/" + t.Recording.ID), Label: t.Title}
				level.Children = append(level.Children, track)
				ad.Tracks = append(ad.Tracks, trackDetail{Key: track.Key, Disc: int64(medium.Position), Number: int64(t.Position), LengthMs: t.Length})
			}
		}
		f.Children = append(f.Children, level)
		d.Albums = append(d.Albums, ad)
	}
	if f.Detail, err = json.Marshal(d); err != nil {
		return works.Fetched{}, fmt.Errorf("encode artist %s: %w", key, err)
	}
	return f, nil
}

func (*Music) Store(ctx context.Context, tx *storage.Tx[file], at works.Placed, raw []byte) (bool, error) {
	var d detail
	if err := json.Unmarshal(raw, &d); err != nil {
		return false, fmt.Errorf("decode the detail of work %s: %w", at.Work, err)
	}
	q := db.New(tx)
	rows, err := q.UpsertArtist(ctx, db.UpsertArtistParams{WorkID: at.Work, Country: d.Country, Disambiguation: d.Disambiguation})
	if err != nil {
		return false, err
	}
	for _, a := range d.Albums {
		n, err := q.UpsertAlbum(ctx, db.UpsertAlbumParams{LevelID: at.Levels[a.Key], WorkID: at.Work, Released: a.Released, PrimaryType: a.PrimaryType, ReleaseID: a.Release})
		if err != nil {
			return false, err
		}
		rows += n
		for _, t := range a.Tracks {
			n, err := q.UpsertTrack(ctx, db.UpsertTrackParams{ItemID: at.Items[t.Key], WorkID: at.Work, Disc: t.Disc, Number: t.Number, LengthMs: t.LengthMs})
			if err != nil {
				return false, err
			}
			rows += n
		}
	}
	return rows > 0, nil
}

// SettingFields answers none: music keeps no settings.
func (*Music) SettingFields(context.Context, storage.Reader[file]) ([]setting.Field, error) {
	return nil, nil
}

func (*Music) ApplySettings(_ context.Context, _ *storage.Tx[file], p setting.Patch) (bool, error) {
	if len(p) > 0 {
		return false, &setting.UnknownKeyError{Key: slices.Min(slices.Collect(maps.Keys(p)))}
	}
	return false, nil
}

func (*Music) Fields(ctx context.Context, r storage.Reader[file], w works.WorkID) (works.FieldSet, error) {
	q := db.New(r)
	artist, err := q.GetArtist(ctx, w)
	if err != nil {
		return works.FieldSet{}, err
	}
	albums, err := q.ListAlbums(ctx, w)
	if err != nil {
		return works.FieldSet{}, err
	}
	tracks, err := q.ListTracks(ctx, w)
	if err != nil {
		return works.FieldSet{}, err
	}
	fs := works.FieldSet{Work: artistFields(artist.Country), Levels: map[works.LevelID][]works.FieldValue{}, Items: map[works.ItemID][]works.FieldValue{}}
	for _, a := range albums {
		values := []works.FieldValue{works.TextField("kind", a.PrimaryType)}
		if a.Released != "" {
			released, err := works.ParseDate(a.Released)
			if err != nil {
				return works.FieldSet{}, fmt.Errorf("album %s: %w", a.LevelID, err)
			}
			values = append(values, works.DateField("released", released))
		}
		fs.Levels[a.LevelID] = values
	}
	for _, t := range tracks {
		values := []works.FieldValue{works.NumberField("disc", float64(t.Disc))}
		if t.LengthMs != nil {
			values = append(values, works.DurationField("length", time.Duration(*t.LengthMs)*time.Millisecond))
		}
		fs.Items[t.ItemID] = values
	}
	return fs, nil
}

func (*Music) NamingTokens(ctx context.Context, r storage.Reader[file], item works.ItemID) (map[string]string, error) {
	artist, track, err := works.ItemOf(ctx, r, item)
	if err != nil {
		return nil, err
	}
	t, err := db.New(r).GetTrack(ctx, item)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"Artist Name": artist.Label,
		"Track Title": track.Label,
		"Disc":        strconv.FormatInt(t.Disc, 10),
		"Track":       fmt.Sprintf("%02d", t.Number),
	}, nil
}

func (*Music) DefaultTemplates() naming.Templates {
	return naming.Templates{Folder: "{Artist Name}", File: "{Artist Name} - {Disc}-{Track} - {Track Title} {Edition} {Target}"}
}

// SearchTerms answers indexers' empty answer: music isn't searched.
func (*Music) SearchTerms(context.Context, storage.Reader[file], works.ItemID) (indexers.Terms, error) {
	return indexers.Terms{}, nil
}

// Timings answers each track's length and its album's release date.
func (*Music) Timings(ctx context.Context, r storage.Reader[file], ws []works.WorkID) (map[works.ItemID]targets.Timing, error) {
	q := db.New(r)
	albums, err := q.ListAlbumsOf(ctx, ws)
	if err != nil {
		return nil, err
	}
	released := map[works.LevelID]release.Fact[works.Date]{}
	for _, a := range albums {
		if a.Released == "" {
			continue
		}
		d, err := works.ParseDate(a.Released)
		if err != nil {
			return nil, fmt.Errorf("album %s: %w", a.LevelID, err)
		}
		released[a.LevelID] = release.Known(d)
	}
	tracks, err := q.ListTracksOf(ctx, ws)
	if err != nil {
		return nil, err
	}
	length := map[works.ItemID]release.Fact[time.Duration]{}
	for _, t := range tracks {
		if t.LengthMs != nil {
			length[t.ItemID] = release.Known(time.Duration(*t.LengthMs) * time.Millisecond)
		}
	}
	items, err := works.ItemsOf(ctx, r, ws)
	if err != nil {
		return nil, err
	}
	out := map[works.ItemID]targets.Timing{}
	for _, of := range items {
		for _, it := range of {
			out[it.ID] = targets.Timing{Released: released[it.LevelID], Runtime: length[it.ID]}
		}
	}
	return out, nil
}

func (*Music) Cuts() []release.Cut {
	return []release.Cut{{Key: "deluxe", Name: "Deluxe", Aliases: []string{"Deluxe", "Deluxe Edition"}}}
}

func artistFields(country string) []works.FieldValue {
	if country == "" {
		return nil
	}
	return []works.FieldValue{works.TextField("country", country)}
}
