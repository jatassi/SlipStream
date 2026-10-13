package music

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/fakes/contenttypes/music/internal/db"
	"github.com/jatassi/slipstream/fakes/contenttypes/music/internal/musicbrainz"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

var Def = contenttype.Define("music", schemaFiles, DefaultEndpoints, New)

type Endpoints struct {
	Metadata endpoint.URL `yaml:"metadata"`
}

func DefaultEndpoints() Endpoints {
	return Endpoints{Metadata: endpoint.MustParseURL("https://musicbrainz.invalid/ws/2")}
}

var Description = works.Description{
	Name:   "Music",
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

type Music struct {
	source *musicbrainz.Client
}

func New(d contenttype.Deps, e Endpoints) contenttype.Type {
	return &Music{source: musicbrainz.New(d.HTTP, e.Metadata, d.UserAgent)}
}

func (*Music) Describe() works.Description { return Description }

func (m *Music) Search(ctx context.Context, query string) ([]works.Candidate, error) {
	artists, err := m.source.SearchArtists(ctx, query)
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

func (m *Music) Fetch(ctx context.Context, key works.Key) (works.Fetched, error) {
	artist, err := m.source.Artist(ctx, string(key))
	if err != nil {
		return works.Fetched{}, err
	}
	albums, err := m.source.Albums(ctx, artist.ID)
	if err != nil {
		return works.Fetched{}, err
	}
	f := works.Fetched{Label: artist.Name, SortLabel: artist.SortName}
	d := detail{Country: artist.Country, Disambiguation: artist.Disambiguation}
	for _, album := range albums {
		release, ok, err := m.source.Canonical(ctx, album.ID)
		if err != nil {
			return works.Fetched{}, err
		}
		if !ok {
			continue
		}
		if album.FirstRelease != "" {
			if _, err := works.ParseDate(album.FirstRelease); err != nil {
				return works.Fetched{}, fmt.Errorf("album %s: %w: %w", album.ID, err, works.ErrInvalid)
			}
		}
		level := works.Entry{Key: works.Key(album.ID), Label: album.Title}
		ad := albumDetail{Key: level.Key, Released: album.FirstRelease, PrimaryType: album.PrimaryType, Release: db.ReleaseID(release.ID)}
		for _, medium := range release.Media {
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

func (*Music) Store(ctx context.Context, tx *storage.Tx[file], at works.Placed, raw []byte) error {
	var d detail
	if err := json.Unmarshal(raw, &d); err != nil {
		return fmt.Errorf("decode the detail of work %s: %w", at.Work, err)
	}
	q := db.New(tx)
	if err := q.UpsertArtist(ctx, db.UpsertArtistParams{WorkID: at.Work, Country: d.Country, Disambiguation: d.Disambiguation}); err != nil {
		return err
	}
	for _, a := range d.Albums {
		if err := q.UpsertAlbum(ctx, db.UpsertAlbumParams{LevelID: at.Levels[a.Key], WorkID: at.Work, Released: a.Released, PrimaryType: a.PrimaryType, ReleaseID: a.Release}); err != nil {
			return err
		}
		for _, t := range a.Tracks {
			if err := q.UpsertTrack(ctx, db.UpsertTrackParams{ItemID: at.Items[t.Key], WorkID: at.Work, Disc: t.Disc, Number: t.Number, LengthMs: t.LengthMs}); err != nil {
				return err
			}
		}
	}
	return nil
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

func artistFields(country string) []works.FieldValue {
	if country == "" {
		return nil
	}
	return []works.FieldValue{works.TextField("country", country)}
}
