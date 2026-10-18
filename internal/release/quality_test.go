package release_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/release"
)

func TestQualityTable(t *testing.T) {
	want := []struct {
		key, name string
		floor     float64
	}{
		{"sdtv", "SDTV", 0},
		{"dvd", "DVD", 0},
		{"webrip-480p", "WEBRip-480p", 0},
		{"hdtv-720p", "HDTV-720p", 17.1},
		{"webrip-720p", "WEBRip-720p", 12.5},
		{"webdl-720p", "WEBDL-720p", 12.5},
		{"bluray-720p", "Bluray-720p", 25.7},
		{"hdtv-1080p", "HDTV-1080p", 33.8},
		{"webrip-1080p", "WEBRip-1080p", 12.5},
		{"webdl-1080p", "WEBDL-1080p", 12.5},
		{"bluray-1080p", "Bluray-1080p", 50.8},
		{"remux-1080p", "Remux-1080p", 102},
		{"hdtv-2160p", "HDTV-2160p", 85},
		{"webrip-2160p", "WEBRip-2160p", 34.5},
		{"webdl-2160p", "WEBDL-2160p", 34.5},
		{"bluray-2160p", "Bluray-2160p", 102},
		{"remux-2160p", "Remux-2160p", 187.4},
		{"cam", "CAM", 0},
	}
	all := release.Qualities()
	if len(all) != len(want) {
		t.Fatalf("Qualities() has %d qualities, want %d", len(all), len(want))
	}
	for i, q := range all {
		info := q.Info()
		if info.Key != want[i].key || info.Name != want[i].name || info.Floor != want[i].floor {
			t.Errorf("quality %d = %s %s %v, want %s %s %v", i+1, info.Key, info.Name, info.Floor, want[i].key, want[i].name, want[i].floor)
		}
		if got, err := release.ParseQuality(info.Key); err != nil || got != q {
			t.Errorf("ParseQuality(%q) = %v, %v, want %v", info.Key, got, err, q)
		}
		if info.Cinema() != (q == release.CAM) {
			t.Errorf("%s.Cinema() = %v", q, info.Cinema())
		}
	}
	for _, bad := range []string{"", "WEBDL-1080p", "webdl-480p", "bluray-576p", "any"} {
		if _, err := release.ParseQuality(bad); err == nil {
			t.Errorf("ParseQuality(%q) succeeded", bad)
		}
	}
}

var sources = []struct {
	name   string
	source release.Source
}{
	{"TV", release.TV}, {"HDTV", release.HDTV}, {"DVD", release.DVD}, {"WEBRip", release.WEBRip},
	{"WEBDL", release.WEBDL}, {"Bluray", release.Bluray}, {"Remux", release.Remux}, {"Cinema", release.Cinema},
}

var resolutions = []release.Resolution{release.R2160p, release.R1080p, release.R720p, release.R576p, release.R480p}

func TestQualityOfEveryPair(t *testing.T) {
	// Columns: 2160p, 1080p, 720p, 576p, 480p, then no resolution.
	want := map[string][6]string{
		"TV":     {"hdtv-2160p", "hdtv-1080p", "hdtv-720p", "sdtv", "sdtv", "sdtv"},
		"HDTV":   {"hdtv-2160p", "hdtv-1080p", "hdtv-720p", "sdtv", "sdtv", "hdtv-720p"},
		"DVD":    {"hdtv-2160p", "hdtv-1080p", "hdtv-720p", "dvd", "dvd", "dvd"},
		"WEBRip": {"webrip-2160p", "webrip-1080p", "webrip-720p", "webrip-480p", "webrip-480p", "webrip-480p"},
		"WEBDL":  {"webdl-2160p", "webdl-1080p", "webdl-720p", "sdtv", "sdtv", "webdl-720p"},
		"Bluray": {"bluray-2160p", "bluray-1080p", "bluray-720p", "sdtv", "sdtv", "bluray-720p"},
		"Remux":  {"remux-2160p", "remux-1080p", "hdtv-720p", "sdtv", "sdtv", "remux-1080p"},
		"Cinema": {"cam", "cam", "cam", "cam", "cam", "cam"},
		"":       {"hdtv-2160p", "hdtv-1080p", "hdtv-720p", "sdtv", "sdtv", ""},
	}
	check := func(name string, s release.Fact[release.Source], r release.Fact[release.Resolution], want string) {
		t.Helper()
		q, ok := release.QualityOf(s, r)
		got := ""
		if ok {
			got = q.String()
		}
		res, _ := r.Get()
		if got != want {
			t.Errorf("QualityOf(%q, %q) = %q, want %q", name, res, got, want)
		}
	}
	for _, s := range append(sources, struct {
		name   string
		source release.Source
	}{name: ""}) {
		fact := release.Known(s.source)
		if s.name == "" {
			fact = release.Fact[release.Source]{}
		}
		for i, r := range resolutions {
			check(s.name, fact, release.Known(r), want[s.name][i])
		}
		check(s.name, fact, release.Fact[release.Resolution]{}, want[s.name][5])
	}
}
