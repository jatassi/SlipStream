package release_test

import (
	"reflect"
	"testing"
)

func TestIDTags(t *testing.T) {
	for name, want := range map[string]map[string]string{
		"The Matrix (1999) {tmdb-603}":                  {"tmdbid": "603"},
		"The Matrix (1999) [tmdbid-603] 1080p":          {"tmdbid": "603"},
		"The Matrix (1999) [imdbid-tt0133093]":          {"imdbid": "tt0133093"},
		"The Matrix (1999) {imdb-tt0133093} {tmdb-603}": {"imdbid": "tt0133093", "tmdbid": "603"},
		"The Simpsons (1989) {tvdb-71663}":              {"tvdbid": "71663"},
		"The Simpsons (1989) [tvdbid-71663] S01E01":     {"tvdbid": "71663"},
		"The Matrix 1999 tt12345 1080p":                 nil,
		"The Matrix (1999) {tmdb-0}":                    nil,
		"The Matrix (1999) {tmdb-99999999999}":          nil,
		"The.Matrix.1999.1080p.BluRay.x264-GRP":         nil,
	} {
		if got := titleFacts(name).IDs; !reflect.DeepEqual(got, want) {
			t.Errorf("%s reads ids %v, want %v", name, got, want)
		}
	}
}
