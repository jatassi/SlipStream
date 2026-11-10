package scenario

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

func TestValidateChecksTheEpisodeReleases(t *testing.T) {
	video := Video{Width: 1920, Height: 1080, Seconds: 6}
	s := Scenario{Name: "x", Nodes: []Node{{Name: "home"}}, TVDB: &TVDB{Series: []TVDBSeries{{
		ID: 1001, Name: "The Lantern Keepers", Status: "continuing",
		Seasons: []TVDBSeason{{Number: 0, Episodes: []TVDBEpisode{{ID: 100101, Number: 1, Name: "Before"}}}, {Number: 1, Episodes: []TVDBEpisode{{ID: 100111, Number: 1, Name: "First"}}}},
	}}}, Releases: []Release{
		{Name: "Lantern.S01E01", TVDB: &ReleaseEpisode{Series: 1001, Season: 1, Episode: 1}, Video: video},
		{Name: "Lantern.S00E01", TVDB: &ReleaseEpisode{Series: 1001, Season: 0, Episode: 1}, Video: video},
		{Name: "Lantern.S01", TVDB: &ReleaseEpisode{Series: 1001, Season: 1}, Video: video},
		{Name: "Lantern.S01E02", TVDB: &ReleaseEpisode{Series: 1001, Season: 1, Episode: 2}, Video: video},
		{Name: "Lantern.S02", TVDB: &ReleaseEpisode{Series: 1001, Season: 2}, Video: video},
		{Name: "Harbor.S01E01", TVDB: &ReleaseEpisode{Series: 1002, Season: 1, Episode: 1}, Video: video},
	}}
	want := []string{
		"scenario x: release Lantern.S01E02 names episode 2 of season 1 of tvdb series 1001, which it does not list",
		"scenario x: release Lantern.S02 names season 2 of tvdb series 1001, which it does not list",
		"scenario x: release Harbor.S01E01 names tvdb series 1002, which tvdb does not list",
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("Validate passed")
	}
	if got := strings.Split(err.Error(), "\n"); !reflect.DeepEqual(got, want) {
		t.Fatalf("Validate =\n%s\nwant\n%s", err, strings.Join(want, "\n"))
	}
}

func TestValidateChecksTheReleases(t *testing.T) {
	good := Release{Name: "A.2021.1080p", TMDBID: 1, Video: Video{Width: 1920, Height: 1080, Seconds: 6}}
	s := Scenario{Name: "x", Nodes: []Node{{Name: "home"}}, TMDB: &TMDB{Movies: []TMDBMovie{{ID: 1, IMDbID: "tt0000001", Title: "A", Released: "2021-01-01"}}}, Tracker: Tracker{Interval: -time.Second}, Releases: []Release{
		good,
		good,
		{Name: "a b.mkv", TMDBID: 0, Video: Video{Width: 1919, Height: 1080, Seconds: 0}, Seeders: -1, BadPeer: "evil"},
		{Name: "B.2021.720p", TMDBID: 2, Video: Video{Width: 1280, Height: 720, Seconds: 6}},
		{Name: "A.2021.2160p", TMDBID: 1, Video: Video{Width: 3840, Height: 2160, Seconds: 6}, Link: "ftp", Trackerless: true},
		{Name: "A.2021.480p", TMDBID: 1, Video: Video{Width: 640, Height: 480, Seconds: 6}, Link: MagnetLink, Trackerless: true},
		{Name: "A.2021.576p", TMDBID: 1, Video: Video{Width: 720, Height: 576, Seconds: 6}, Link: RedirectLink, Trackerless: true, Public: true},
		{Name: "A.2021.360p", TMDBID: 1, Video: Video{Width: 640, Height: 360, Seconds: 6}, Sites: []string{"heron", "osprey"}},
		{Name: "A.2021.240p", TMDBID: 1, Video: Video{Width: 426, Height: 240, Seconds: 6}, Seeded: true, Unregistered: true},
	}}
	want := []string{
		"scenario x: release A.2021.1080p is listed twice",
		`scenario x: release name "a b.mkv" does not match ^[A-Za-z0-9][A-Za-z0-9._-]*$`,
		"scenario x: release a b.mkv has tmdb_id 0; it is at least 1",
		"scenario x: release a b.mkv video is 1919x1080; width and height are even, up to 7680x4320",
		"scenario x: release a b.mkv video lasts 0 seconds; it lasts 1 to 60",
		"scenario x: release a b.mkv has -1 seeders",
		`scenario x: release a b.mkv bad_peer "evil" is not corrupt, stall, choke or drop`,
		"scenario x: release B.2021.720p names tmdb movie 2, which tmdb does not list",
		`scenario x: release A.2021.2160p link "ftp" is not torrent, magnet or redirect`,
		"scenario x: release A.2021.2160p is trackerless, which is for link magnet or redirect",
		"scenario x: release A.2021.2160p is trackerless and private; a trackerless release is public",
		"scenario x: release A.2021.480p is trackerless and private; a trackerless release is public",
		`scenario x: release A.2021.360p names site "osprey", which fixture-site doesn't serve`,
		"scenario x: release A.2021.240p is seeded and unregistered; fake-tracker refuses its seeder's announces",
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

func TestValidateChecksTheTracker(t *testing.T) {
	for tracker, want := range map[*Tracker]string{
		{Interval: time.Minute, MinInterval: -time.Second}:    "tracker min_interval -1s is negative",
		{Interval: time.Minute, MinInterval: 2 * time.Minute}: "tracker min_interval 2m0s is above its interval 1m0s",
		{Mode: "ocelot"}: `tracker mode "ocelot" is not plain, unit3d or gazelle`,
		{Mode: UNIT3D, AllowPeerIDs: []string{"-qB"}}:                                `tracker allow_peer_ids is for gazelle mode, not "unit3d"`,
		{Mode: Gazelle, AllowPeerIDs: []string{"-LT", ""}}:                           "tracker allow_peer_ids lists an empty prefix, which allows every peer ID",
		{Grace: -time.Minute}:                                                        "tracker grace -1m0s is negative",
		{Sites: map[string]SiteRule{"brook": {MinSeedTime: time.Hour}}}:              `tracker sites names "brook", which states no rule`,
		{Sites: map[string]SiteRule{"heron": {MinRatio: 1, MinSeedTime: time.Hour}}}: "tracker sites heron states a ratio and a seed time, and site heron states a seed time only",
		{Sites: map[string]SiteRule{"tamarack": {MinSeedTime: 3 * time.Minute}}}:     "tracker sites tamarack states a seed time only, and site tamarack states a ratio and a seed time",
	} {
		s := Scenario{Name: "x", Nodes: []Node{{Name: "home"}}, Tracker: *tracker}
		if err := s.Validate(); err == nil || err.Error() != "scenario x: "+want {
			t.Errorf("Validate(%+v) = %v, want %q", *tracker, err, want)
		}
	}
	s := Scenario{Name: "x", Nodes: []Node{{Name: "home"}}, Tracker: Tracker{Interval: time.Hour, MinInterval: 30 * time.Minute, Mode: Gazelle, AllowPeerIDs: []string{"-LT", "-qB"}, FailWithoutInterval: true}}
	if err := s.Validate(); err != nil {
		t.Errorf("Validate(%+v) = %v", s.Tracker, err)
	}
	s.Tracker = Tracker{Mode: UNIT3D, Prewarn: 2 * time.Minute, Grace: 2 * time.Minute, InactiveAfter: time.Minute, Sites: map[string]SiteRule{
		"heron": {MinSeedTime: 3 * time.Minute}, "tamarack": {MinRatio: 1, MinSeedTime: 3 * time.Minute},
	}}
	if err := s.Validate(); err != nil {
		t.Errorf("Validate(%+v) = %v", s.Tracker, err)
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

func TestValidateChecksTheLeecher(t *testing.T) {
	movies := &TMDB{Movies: []TMDBMovie{{ID: 1, IMDbID: "tt0000001", Title: "A", Released: "2021-01-01"}}}
	releases := []Release{
		{Name: "A.2021.1080p", TMDBID: 1, Video: Video{Width: 1920, Height: 1080, Seconds: 6}, Seeded: true},
		{Name: "A.2021.720p", TMDBID: 1, Video: Video{Width: 1280, Height: 720, Seconds: 6}},
	}
	for leecher, want := range map[Leecher]string{
		{Node: "home", Release: "A.2021.1080p"}:     "",
		{Node: "seedbox", Release: "A.2021.1080p"}:  "scenario x: leecher names node seedbox, which nodes does not list",
		{Node: "home", Release: "A.2021.720p"}:      "scenario x: leecher names release A.2021.720p, which releases does not list as seeded",
		{Node: "home", Release: "B.2021.1080p.WEB"}: "scenario x: leecher names release B.2021.1080p.WEB, which releases does not list as seeded",
	} {
		s := Scenario{Name: "x", Nodes: []Node{{Name: "home"}}, TMDB: movies, Releases: releases, Leecher: &leecher}
		if err := s.Validate(); want == "" && err != nil || want != "" && (err == nil || err.Error() != want) {
			t.Errorf("Validate(leecher %+v) = %v, want %q", leecher, err, want)
		}
	}
}
