package serverfake_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/internal/serverfake"
	"github.com/jatassi/slipstream/fakes/scenario"
)

func TestARootNamingALibraryGetsALibraryOfItsOwn(t *testing.T) {
	s, err := scenario.Load("moves")
	if err != nil {
		t.Fatal(err)
	}
	for kind, want := range map[string][]string{
		"plex":     {`movie "" [/media/movies-d]`, `series "" [/media/series-d /media/series-e]`, `movie "Movies 4K" [/media/movies-e /media/movies-f]`},
		"jellyfin": {`movie "" [/media/movies-d]`, `movie "Movies 4K" [/media/movies-e /media/movies-f]`},
		"emby":     {`movie "" [/media/movies-d /media/movies-e /media/movies-f]`},
	} {
		p, ok, err := serverfake.Playing(s, kind, []string{"1"})
		if err != nil || !ok {
			t.Fatalf("the moves Scenario plays no %s: %v", kind, err)
		}
		var got []string
		for _, l := range p.Libraries {
			got = append(got, fmt.Sprintf("%s %q %v", l.ContentType, l.Name, l.Folders))
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s plays libraries %q, want %q", kind, got, want)
		}
	}
}
