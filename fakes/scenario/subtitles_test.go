package scenario

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestValidateChecksSpeechAndTheSubtitleCatalog(t *testing.T) {
	ok := MediaVideo{Width: 64, Height: 64, Seconds: 1, HDR: "sdr"}
	speech := &MediaSpeech{Seed: 7, Density: 0.4, Length: time.Minute}
	s := Scenario{
		Name: "x", Nodes: []Node{{Name: "home", Setup: SetupController}}, Admin: &Admin{Email: "a@example.test", DisplayName: "A", PIN: "1234"},
		TMDB: &TMDB{Movies: []TMDBMovie{{ID: 1, IMDbID: "tt0000001", Title: "Heat", Runtime: time.Hour}}},
		TVDB: &TVDB{Series: []TVDBSeries{{ID: 2, Name: "Lanterns", Status: "ended", Seasons: []TVDBSeason{{Number: 1, Episodes: []TVDBEpisode{{ID: 3, Number: 1, Name: "One"}}}}}}},
		Library: &Library{Folders: []LibraryFolder{{Folder: "films", Files: []MediaFile{
			{Path: "heat.mkv", Video: ok, Audio: []MediaAudio{{Codec: "pcm_s16le", Channels: 1, Speech: speech}}},
			{Path: "tone.mkv", Video: ok, Audio: []MediaAudio{
				{Codec: "pcm_s16le", Channels: 1},
				{Codec: "aac", Channels: 2, Speech: speech},
				{Codec: "pcm_s16le", Channels: 1, Speech: &MediaSpeech{Seed: 1, Density: 0.4, Length: 10 * time.Second}},
				{Codec: "pcm_s16le", Channels: 1, Speech: &MediaSpeech{Seed: 1, Density: 0.95, Length: time.Minute}},
			}},
		}}}},
		SubtitleProviders: SubtitleProviders{Catalog: []CatalogSubtitle{
			{ID: "heat", Provider: "subdl", Work: "Heat", Language: "en", Release: "Heat.1995", Sources: []string{"films/heat.mkv"}},
			{ID: "heat", Provider: "gestdown", Work: "Heat", Language: "en", Release: "Heat.1995", Unaired: true},
			{ID: "odd", Provider: "opensubtitles", Work: "Nothing", Language: "xx", Release: "Odd", Format: "vtt", Unaired: true, Sources: []string{"films/none.mkv"}},
			{ID: "pack", Provider: "subsource", Work: "Lanterns", Season: 1, Episodes: []int{1, 2}, Language: "en", Release: "Lanterns.S01", Sources: []string{"films/heat.mkv"}},
		}},
	}
	want := []string{
		"scenario x: subtitle_providers catalog heat is listed twice",
		"scenario x: subtitle_providers catalog heat is for the film Heat, which takes no season or episodes and no gestdown",
		`scenario x: subtitle_providers catalog odd provider is "opensubtitles", not subdl, subsource or gestdown`,
		`scenario x: subtitle_providers catalog odd language: "xx" is not a subtitle language`,
		`scenario x: subtitle_providers catalog odd format is "vtt", not srt, ass, ssa or sup`,
		"scenario x: subtitle_providers catalog odd is unaired, which only gestdown plays",
		`scenario x: subtitle_providers catalog odd work "Nothing" is no tmdb movie title and no tvdb series name`,
		"scenario x: subtitle_providers catalog odd source films/none.mkv is no library file with a speech track",
		"scenario x: subtitle_providers catalog pack: subsource finds a series by its IMDb id, and tvdb series Lanterns has none",
		"scenario x: subtitle_providers catalog pack episode 2 is not in Lanterns season 1",
		"scenario x: subtitle_providers catalog pack has 1 sources for 2 files",
		"scenario x: library folder films file tone.mkv audio 1 is pcm_s16le, which only a speech track takes",
		"scenario x: library folder films file tone.mkv audio 2 has speech, which only pcm_s16le carries",
		"scenario x: library folder films file tone.mkv audio 3 speech lasts 10s; it lasts 30s to 5m0s",
		"scenario x: library folder films file tone.mkv audio 4 speech density is 0.95; it is 0.05 to 0.9",
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("Validate passed")
	}
	var got []string
	for _, line := range strings.Split(err.Error(), "\n") {
		if strings.Contains(line, "audio") || strings.Contains(line, "subtitle_providers") {
			got = append(got, line)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Validate =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
