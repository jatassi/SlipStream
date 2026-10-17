package scenario

import (
	"reflect"
	"strings"
	"testing"
)

func TestValidateChecksTheLibrary(t *testing.T) {
	s := Scenario{Name: "x", Nodes: []Node{{Name: "home"}}, Library: &Library{
		Roots: []LibraryRoot{
			{ContentType: "music", Folder: "music"},
			{ContentType: "music", Folder: "Music"},
			{ContentType: "fixture", Folder: "music"},
		},
		Targets: []LibraryTarget{
			{Name: "1080p", Tiers: [][]QualityKey{{"webdl-1080p"}}, Cutoff: "webdl-1080p"},
			{Name: "1080p", Tiers: [][]QualityKey{{"webdl-1440p"}}, Cutoff: "webdl-1080p"},
			{Name: "none", Cutoff: "webdl-1080p"},
			{Name: "filtered", Tiers: [][]QualityKey{{"webdl-1080p"}}, Cutoff: "webdl-1080p", Filters: []LibraryFilter{{Attribute: "hdr", Mode: "only", Values: []string{"sepia"}}}},
			{Name: "sized", Tiers: [][]QualityKey{{"webdl-1080p"}}, Cutoff: "webdl-1080p", Sizes: []LibrarySize{{Quality: "bluray-1080p", Min: 1}}},
		},
		Profiles: []LibraryProfile{
			{Name: "Standard", ContentType: "music", Targets: []string{"1080p", "4k"}},
			{Name: "Standard", ContentType: "music"},
		},
		Indexers: []LibraryIndexer{
			{Name: "Site", Site: "fixture-site"},
			{Name: "Site", Site: "other-site"},
		},
	}}
	want := []string{
		"scenario x: library is set, but no node is controller",
		"scenario x: library has two roots for content type music",
		`scenario x: library root folder "Music" does not match ^[a-z0-9][a-z0-9._-]*$`,
		"scenario x: library has two roots in folder music",
		"scenario x: library has two targets named 1080p",
		`scenario x: library target 1080p: "webdl-1440p" is not a quality`,
		"scenario x: library target none: tiers are missing, empty or repeat a quality",
		`scenario x: library target filtered: a filter is Any with values, Only or Not with none, or names a value twice: "sepia" is not an HDR format`,
		"scenario x: library target sized: the Target doesn't accept that quality: a size for bluray-1080p",
		"scenario x: library profile Standard names target 4k, which library.targets does not list",
		"scenario x: library has two profiles named Standard",
		"scenario x: library profile Standard has no targets",
		"scenario x: library has two indexers named Site",
		`scenario x: library indexer Site is on site "other-site", which no fake serves`,
		"scenario x: library has indexers, and no node holds searching",
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("Validate passed")
	}
	if got := strings.Split(err.Error(), "\n"); !reflect.DeepEqual(got, want) {
		t.Fatalf("Validate =\n%s\nwant\n%s", err, strings.Join(want, "\n"))
	}
	if got := s.ContentTypes(); !reflect.DeepEqual(got, []string{"fixture", "music"}) {
		t.Fatalf("ContentTypes = %v, want [fixture music]", got)
	}
}

func TestValidateRefusesTwoSearchingHoldersForIndexers(t *testing.T) {
	s, err := Load("home")
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Searching(); got != "seedbox" {
		t.Fatalf("home's Searching holder = %q, want seedbox", got)
	}
	s.Nodes[0].Roles = nil
	want := "scenario home: library has indexers, and nodes home, seedbox all hold searching; exactly one does"
	if err := s.Validate(); err == nil || err.Error() != want {
		t.Fatalf("Validate = %v, want %s", err, want)
	}
}
