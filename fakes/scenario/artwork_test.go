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
	artworkStore = artwork.New(fstest.MapFS{"st0red.jpg": {Data: []byte("x")}, "pers0n.jpg": {Data: []byte("y")}})
	s := Scenario{Name: "x", Seed: "x", Nodes: []Node{{Name: "home"}}, TMDB: &TMDB{Movies: []TMDBMovie{{
		ID: 1, IMDbID: "tt0000001", Title: "A", Runtime: 90 * time.Minute,
		Images:  []TMDBImage{{Shape: "poster", Path: "/st0red.jpg", Width: 500, Height: 750}, {Shape: "logo", Path: "/st0red.jpg"}},
		Credits: TMDBCredits{Cast: []TMDBCast{{ID: 2, Name: "B", Profile: "/pers0n.jpg"}}, Crew: []TMDBCrew{{ID: 3, Name: "C", Department: "Directing", Job: "Director", Profile: "/pers0n.jpg"}}},
	}}}}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
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
		if s.TMDB == nil {
			continue
		}
		for _, m := range s.TMDB.Movies {
			for _, im := range m.Images {
				out = append(out, im.Path)
			}
			for _, c := range m.Credits.Cast {
				out = append(out, c.Profile)
			}
			for _, c := range m.Credits.Crew {
				out = append(out, c.Profile)
			}
		}
	}
	return out
}

// TestEveryStoredImageIsUsed keeps the artwork store to the files some
// catalog Scenario names, each one an image the image fake can scale.
func TestEveryStoredImageIsUsed(t *testing.T) {
	used := named(t)
	for _, p := range artwork.Embedded.Paths() {
		if !slices.Contains(used, p) {
			t.Errorf("fakes/internal/artwork/files%s is in no catalog Scenario; delete it", p)
		}
		data, _ := artwork.Embedded.Lookup(p)
		if strings.HasSuffix(p, ".svg") {
			continue
		}
		if _, _, err := image.DecodeConfig(bytes.NewReader(data)); !ArtworkFile.MatchString(p) || err != nil {
			t.Errorf("fakes/internal/artwork/files%s is not a JPEG or PNG named as TMDB names files: %v", p, err)
		}
	}
}
