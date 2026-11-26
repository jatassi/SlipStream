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

func TestAServerReportsTheScenariosStreamsInItsOwnNames(t *testing.T) {
	s, err := scenario.Load("moves")
	if err != nil {
		t.Fatal(err)
	}
	s.Library.Folders = append(s.Library.Folders, scenario.LibraryFolder{Folder: "movies-d", Files: []scenario.MediaFile{{
		Path:      "Heat (1995)/Heat (1995).mkv",
		Video:     scenario.MediaVideo{Width: 3840, Height: 2160, Seconds: 1, HDR: "dv8"},
		Audio:     []scenario.MediaAudio{{Codec: "eac3", Channels: 6, Language: "eng", Default: true}},
		Subtitles: []scenario.MediaSubtitle{{Language: "fre"}},
	}}})
	for kind, subrip := range map[string]string{"plex": "srt", "jellyfin": "subrip", "emby": "subrip"} {
		p, _, err := serverfake.Playing(s, kind, []string{"1"})
		if err != nil {
			t.Fatal(err)
		}
		want := fmt.Sprintf(`[/media/movies-d/Heat (1995)/Heat (1995).mkv [{video hevc Main 10 0 2160  false false} {audio eac3  6 0 eng true false} {subtitle %s  0 0 fre false false}]]`, subrip)
		var got []string
		for _, f := range p.Streamed {
			got = append(got, fmt.Sprintf("%s %v", f.Path, f.Streams))
		}
		if fmt.Sprint(got) != want {
			t.Errorf("%s reports %v, want %s", kind, got, want)
		}
	}
}
