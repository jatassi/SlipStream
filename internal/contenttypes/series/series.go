// Package series is the TV Content Type: a series is the Work, its seasons
// the levels with specials aside, and its episodes the Items, read from
// TVDB with TMDB and OMDb beside it.
package series

import (
	"context"
	"embed"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

var Def = contenttype.Define("series", schemaFiles, DefaultEndpoints, contenttype.Halves[Endpoints]{
	Provider: func(d contenttype.ProviderDeps, _ Endpoints) metadata.Source {
		return &Source{tvdb: d.TVDB, tmdb: d.TMDB, omdb: d.OMDb}
	},
	Library: func(d contenttype.LibraryDeps, e Endpoints) contenttype.Library {
		return &Series{region: d.Region, web: e}
	},
})

var Description = works.Description{
	Name:         "TV",
	Icon:         works.IconTV,
	Source:       "TVDB",
	Work:         works.Noun{One: "Series", Many: "Series"},
	Levels:       []works.Noun{{One: "Season", Many: "Seasons"}},
	Aside:        works.Noun{One: "Special", Many: "Specials"},
	Item:         works.Noun{One: "Episode", Many: "Episodes"},
	Items:        works.ItemsMany,
	Availability: "Next airing",
	Fields: []works.FieldSpec{
		{Key: "years", Label: "Years", Kind: works.KindText, In: in(works.PlaceFacts)},
		{Key: "year", Label: "First aired year", Kind: works.KindNumber, In: in(works.PlaceDetails), List: works.ListColumn | works.ListSort | works.ListFilter},
		{
			Key: "network", Label: "Network", Kind: works.KindText, In: in(works.PlaceFacts, works.PlaceAbout),
			Art: networkLogoKind.Key, List: works.ListColumn | works.ListSort | works.ListFilter,
		},
		{
			Key: "certification", Label: "Certification", Kind: works.KindText, In: in(works.PlaceFacts, works.PlaceAbout),
			With: "certification_country", List: works.ListColumn | works.ListSort | works.ListFilter,
		},
		{Key: "certification_country", Label: "Certification country", Kind: works.KindText, In: in(works.PlaceDetails)},
		{Key: "genres", Label: "Genres", Kind: works.KindTexts, In: in(works.PlaceFacts, works.PlaceAbout), List: works.ListFilter},
		{Key: "status", Label: "Production status", Kind: works.KindText, In: in(works.PlaceFacts, works.PlaceAbout), List: works.ListColumn | works.ListFilter},
		{Key: "seasons", Label: "Seasons", Kind: works.KindNumber, In: in(works.PlaceAbout), List: works.ListColumn | works.ListSort},
		{Key: "episodes", Label: "Episodes", Kind: works.KindNumber, In: in(works.PlaceAbout), List: works.ListColumn | works.ListSort},
		{Key: "overview", Label: "Overview", Kind: works.KindText, In: in(works.PlaceOverview)},
		{Key: "creators", Label: "Created by", Kind: works.KindPeople, In: in(works.PlaceCredits)},
		{Key: "credits", Label: "Cast", Kind: works.KindPeople, In: in(works.PlaceCredits)},
		{Key: "original_title", Label: "Original title", Kind: works.KindText, In: in(works.PlaceAbout)},
		{Key: "original_language", Label: "Original language", Kind: works.KindText, In: in(works.PlaceAbout)},
		{Key: "country", Label: "Country", Kind: works.KindText, In: in(works.PlaceAbout)},
		{Key: "first_aired", Label: "First aired", Kind: works.KindDate, In: in(works.PlaceAbout)},
		{Key: "last_aired", Label: "Last aired", Kind: works.KindDate, In: in(works.PlaceAbout)},
		{Key: "next_aired", Label: "Next airing", Kind: works.KindDate, In: in(works.PlaceAbout)},
		{Key: "awards", Label: "Awards", Kind: works.KindText, In: in(works.PlaceAbout)},
		{Key: "english_title", Label: "English title", Kind: works.KindText, In: in(works.PlaceDetails)},
		{Key: "episode_order", Label: "Episode order", Kind: works.KindText, In: in(works.PlaceDetails)},
		{Key: "tvdb_id", Label: "TVDB id", Kind: works.KindText, In: in(works.PlaceDetails)},
		{Key: "tmdb_id", Label: "TMDB id", Kind: works.KindText, In: in(works.PlaceDetails)},
		{Key: "imdb_id", Label: "IMDb id", Kind: works.KindText, In: in(works.PlaceDetails)},
		{Key: "trailer_site", Label: "Trailer site", Kind: works.KindText, In: in(works.PlaceDetails)},
		{Key: "trailer_key", Label: "Trailer key", Kind: works.KindText, In: in(works.PlaceDetails)},
		{Key: "season", Label: "Season", Tier: works.TierLevel(1), Kind: works.KindNumber, In: in(works.PlaceDetails)},
		{Key: "overview", Label: "Overview", Tier: works.TierLevel(1), Kind: works.KindText, In: in(works.PlaceOverview)},
		{Key: "code", Label: "Episode", Tier: works.TierItem, Kind: works.KindText, In: in(works.PlaceCode)},
		{Key: "air_date", Label: "Air date", Tier: works.TierItem, Kind: works.KindDate, In: in(works.PlaceAbout)},
		{Key: "runtime", Label: "Runtime", Tier: works.TierItem, Kind: works.KindDuration, In: in(works.PlaceAbout)},
		{Key: "overview", Label: "Overview", Tier: works.TierItem, Kind: works.KindText, In: in(works.PlaceOverview)},
		{Key: "season", Label: "Season", Tier: works.TierItem, Kind: works.KindNumber, In: in(works.PlaceDetails)},
		{Key: "episode", Label: "Episode number", Tier: works.TierItem, Kind: works.KindNumber, In: in(works.PlaceDetails)},
		{Key: "absolute", Label: "Absolute number", Tier: works.TierItem, Kind: works.KindNumber, In: in(works.PlaceDetails)},
		{Key: "imdb_id", Label: "IMDb id", Tier: works.TierItem, Kind: works.KindText, In: in(works.PlaceDetails)},
	},
	Artwork: []artwork.Kind{posterKind, backdropKind, titleTreatmentKind, networkLogoKind, seasonPosterKind, stillKind},
	Art:     works.ArtRoles{Poster: posterKind.Key, Backdrop: backdropKind.Key, Title: titleTreatmentKind.Key},
	Ratings: []works.RatingSource{"imdb", "tmdb", "rottentomatoes", "metacritic"},
}

var (
	posterKind         = artwork.Kind{Key: "poster", Label: "Poster", Tier: artwork.TierWork, Shape: artwork.ShapePoster, Prefetch: artwork.SizeMedium}
	backdropKind       = artwork.Kind{Key: "backdrop", Label: "Backdrop", Tier: artwork.TierWork, Shape: artwork.ShapeBackdrop, Prefetch: artwork.SizeLarge}
	titleTreatmentKind = artwork.Kind{Key: "title_treatment", Label: "Title treatment", Tier: artwork.TierWork, Shape: artwork.ShapeLogo, Prefetch: artwork.SizeMedium}
	networkLogoKind    = artwork.Kind{Key: "network_logo", Label: "Network logo", Tier: artwork.TierWork, Shape: artwork.ShapeLogo}
	seasonPosterKind   = artwork.Kind{Key: "season_poster", Label: "Season poster", Tier: artwork.TierLevel(1), Shape: artwork.ShapePoster}
	stillKind          = artwork.Kind{Key: "still", Label: "Still", Tier: artwork.TierItem, Shape: artwork.ShapeBackdrop}
)

func in(places ...works.Place) []works.Place { return places }

// Series is the library half.
type Series struct {
	region func(context.Context, storage.Reader[file]) (string, error)
	web    Endpoints
}

func (*Series) Describe() works.Description { return Description }

// SearchTerms answers no Terms, so no episode is searched, until S26
// (Indexers and Trackers) gives the type its terms.
func (*Series) SearchTerms(context.Context, storage.Reader[file], works.ItemID) (indexers.Terms, error) {
	return indexers.Terms{}, nil
}
