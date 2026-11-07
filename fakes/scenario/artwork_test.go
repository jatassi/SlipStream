package scenario

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/artwork"
)

func TestARecordedImageInTheStoreValidates(t *testing.T) {
	stored := artworkStore
	t.Cleanup(func() { artworkStore = stored })
	artworkStore = artwork.New(fstest.MapFS{
		"st0red.jpg": {Data: []byte("x")}, "pers0n.jpg": {Data: []byte("y")},
		"tvdb/v4/series/7/posters/p.jpg": {Data: []byte("z")}, "tvdb/v4/season/8/posters/s.png": {Data: []byte("z")}, "tvdb/episodes/7/9.jpg": {Data: []byte("z")},
	})
	s := Scenario{Name: "x", Seed: "x", Nodes: []Node{{Name: "home"}}, TMDB: &TMDB{Movies: []TMDBMovie{{
		ID: 1, IMDbID: "tt0000001", Title: "A", Runtime: 90 * time.Minute,
		Images:  []TMDBImage{{Shape: "poster", Path: "/st0red.jpg", Width: 500, Height: 750}, {Shape: "logo", Path: "/st0red.jpg"}},
		Credits: TMDBCredits{Cast: []TMDBCast{{ID: 2, Name: "B", Profile: "/pers0n.jpg"}}, Crew: []TMDBCrew{{ID: 3, Name: "C", Department: "Directing", Job: "Director", Profile: "/pers0n.jpg"}}},
	}}}, TVDB: &TVDB{Series: []TVDBSeries{{
		ID: 7, Name: "A", Status: "ended", Image: "/banners/v4/series/7/posters/p.jpg",
		Seasons: []TVDBSeason{{Number: 1, Image: "/banners/v4/season/8/posters/s.png", Episodes: []TVDBEpisode{{ID: 9, Number: 1, Name: "B", Image: "/banners/episodes/7/9.jpg"}}}},
	}}}}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestARecordedTVDBImageOutsideTheStoreFails(t *testing.T) {
	stored := artworkStore
	t.Cleanup(func() { artworkStore = stored })
	artworkStore = artwork.New(fstest.MapFS{})
	s := Scenario{Name: "x", Seed: "x", Nodes: []Node{{Name: "home"}}, TVDB: &TVDB{Series: []TVDBSeries{{
		ID: 7, Name: "A", Status: "ended", Image: "/banners/v4/series/7/posters/p.jpg",
		Seasons: []TVDBSeason{{Number: 1, Image: "https://artworks.thetvdb.com/banners/s.jpg", Episodes: []TVDBEpisode{{ID: 9, Number: 1, Name: "B", Image: "/banners/../9.jpg"}}}},
	}}}}
	err := s.Validate()
	for _, want := range []string{
		"tvdb series 7 image /banners/v4/series/7/posters/p.jpg is not in fakes/internal/artwork/files/tvdb",
		`tvdb series 7 season 1 image "https://artworks.thetvdb.com/banners/s.jpg" is not a TVDB artwork path`,
		`tvdb series 7 episode 9 image "/banners/../9.jpg" is not a TVDB artwork path`,
	} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Validate = %v; want it to hold %q", err, want)
		}
	}
}

// named is every file path a catalog Scenario names.
func named(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, name := range Catalog() {
		s, err := Load(name)
		if err != nil {
			t.Fatal(err)
		}
		if s.TVDB != nil {
			for _, series := range s.TVDB.Series {
				out = append(out, tvdbNamed(series)...)
			}
		}
		if s.TMDB == nil {
			continue
		}
		for _, m := range s.TMDB.Movies {
			out = append(out, tmdbNamed(m.Images, m.Credits)...)
		}
	}
	return out
}

func tvdbNamed(series TVDBSeries) []string {
	out := []string{series.Image}
	for _, season := range series.Seasons {
		out = append(out, season.Image)
		for _, e := range season.Episodes {
			out = append(out, e.Image)
		}
	}
	return out
}

func tmdbNamed(ims []TMDBImage, credits TMDBCredits) []string {
	var out []string
	for _, im := range ims {
		out = append(out, im.Path)
	}
	for _, c := range credits.Cast {
		out = append(out, c.Profile)
	}
	for _, c := range credits.Crew {
		out = append(out, c.Profile)
	}
	return out
}

// TestEveryStoredImageIsUsed keeps the artwork store to the files some
// catalog Scenario names, each one an image the image fake can scale.
func TestEveryStoredImageIsUsed(t *testing.T) {
	used := named(t)
	for _, p := range artwork.Embedded.Paths() {
		if !slices.Contains(used, p) {
			t.Errorf("the store's %s is in no catalog Scenario; delete it", p)
		}
		data, _ := artwork.Embedded.Lookup(p)
		if strings.HasSuffix(p, ".svg") {
			continue
		}
		if _, _, err := image.DecodeConfig(bytes.NewReader(data)); !ArtworkFile.MatchString(p) && !TVDBArtworkFile.MatchString(p) || err != nil {
			t.Errorf("the store's %s is not a JPEG or PNG named as TMDB or TVDB names files: %v", p, err)
		}
	}
}
