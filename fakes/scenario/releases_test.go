package scenario

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestValidateChecksTheReleases(t *testing.T) {
	good := Release{Name: "A.2021.1080p", TMDBID: 1, Video: Video{Width: 1920, Height: 1080, Seconds: 6}}
	s := Scenario{Name: "x", Nodes: []Node{{Name: "home"}}, Tracker: Tracker{Interval: -time.Second}, Releases: []Release{
		good,
		good,
		{Name: "a b.mkv", TMDBID: 0, Video: Video{Width: 1919, Height: 1080, Seconds: 0}, Seeders: -1},
	}}
	want := []string{
		"scenario x: release A.2021.1080p is listed twice",
		`scenario x: release name "a b.mkv" does not match ^[A-Za-z0-9][A-Za-z0-9._-]*$`,
		"scenario x: release a b.mkv has tmdb_id 0; it is at least 1",
		"scenario x: release a b.mkv video is 1919x1080; width and height are even, up to 7680x4320",
		"scenario x: release a b.mkv video lasts 0 seconds; it lasts 1 to 60",
		"scenario x: release a b.mkv has -1 seeders",
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
