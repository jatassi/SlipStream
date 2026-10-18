package scenario

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

func TestValidateChecksTheReleases(t *testing.T) {
	good := Release{Name: "A.2021.1080p", TMDBID: 1, Video: Video{Width: 1920, Height: 1080, Seconds: 6}}
	s := Scenario{Name: "x", Nodes: []Node{{Name: "home"}}, TMDB: &TMDB{Movies: []TMDBMovie{{ID: 1, IMDbID: "tt0000001", Title: "A", Released: "2021-01-01"}}}, Tracker: Tracker{Interval: -time.Second}, Releases: []Release{
		good,
		good,
		{Name: "a b.mkv", TMDBID: 0, Video: Video{Width: 1919, Height: 1080, Seconds: 0}, Seeders: -1},
		{Name: "B.2021.720p", TMDBID: 2, Video: Video{Width: 1280, Height: 720, Seconds: 6}},
	}}
	want := []string{
		"scenario x: release A.2021.1080p is listed twice",
		`scenario x: release name "a b.mkv" does not match ^[A-Za-z0-9][A-Za-z0-9._-]*$`,
		"scenario x: release a b.mkv has tmdb_id 0; it is at least 1",
		"scenario x: release a b.mkv video is 1919x1080; width and height are even, up to 7680x4320",
		"scenario x: release a b.mkv video lasts 0 seconds; it lasts 1 to 60",
		"scenario x: release a b.mkv has -1 seeders",
		"scenario x: release B.2021.720p names tmdb movie 2, which tmdb does not list",
		"scenario x: tracker interval -1s is negative",
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("Validate passed")
	}
	if got := strings.Split(err.Error(), "\n"); !reflect.DeepEqual(got, want) {
		t.Fatalf("Validate =\n%s\nwant\n%s", err, strings.Join(want, "\n"))
	}
}

func TestBytesReadBackAsWritten(t *testing.T) {
	for _, written := range []string{"0B", "512B", "1.5KiB", "700MiB", "8.2GiB", "4.4GiB", "2TiB"} {
		var b Bytes
		if err := yaml.Unmarshal([]byte(written), &b); err != nil {
			t.Fatalf("%s: %v", written, err)
		}
		got, err := yaml.Marshal(b)
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(string(got)) != written {
			t.Errorf("%s read as %d bytes writes back as %s", written, b, got)
		}
	}
	var b Bytes
	if err := yaml.Unmarshal([]byte("8.2GiB"), &b); err != nil || b != 8804682957 {
		t.Fatalf("8.2GiB = %d bytes, %v; want 8804682957", b, err)
	}
}
