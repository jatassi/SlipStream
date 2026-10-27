package scenario

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/role"
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
			{Name: "Standard", ContentType: "music", Targets: []string{"1080p", "4k"}, Default: true},
			{Name: "Standard", ContentType: "music", Default: true},
			{Name: "Starter", ContentType: "music", Targets: []string{"1080p"}},
		},
		Indexers: []LibraryIndexer{
			{Name: "Site", Site: "fixture-site"},
			{Name: "Site", Site: "other-site"},
		},
	}}
	want := []string{
		"scenario x: library is set, but no node is controller",
		`scenario x: library root folder "Music" does not match ^[a-z0-9][a-z0-9._-]*$`,
		"scenario x: library has two roots in folder music",
		"scenario x: library has two targets named 1080p",
		`scenario x: library target 1080p: "webdl-1440p" is not a quality`,
		"scenario x: library target none: tiers are missing, empty or repeat a quality",
		`scenario x: library target filtered: a filter is Any with values, Only or Not with none, or names a value twice: "sepia" is not an HDR format`,
		"scenario x: library target sized: the Target doesn't accept that quality: a size for bluray-1080p",
		"scenario x: library profile Standard names target 4k, which library.targets does not list and isn't the starter",
		"scenario x: library has two profiles named Standard",
		"scenario x: library has two default profiles for content type music",
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

func TestALibraryRootNamesAStorageNode(t *testing.T) {
	s, err := Load("home")
	if err != nil {
		t.Fatal(err)
	}
	s.Library.Roots = []LibraryRoot{
		{ContentType: "movie", Folder: "movies"},
		{ContentType: "movie", Folder: "films", Node: "seedbox"},
		{ContentType: "movie", Folder: "nas", Node: "nas"},
		{ContentType: "movie", Folder: "full", Volume: &LibraryVolume{TotalGiB: 10, FreeGiB: 12}},
	}
	want := []string{
		"scenario home: library root full's volume has 12 GiB free of 10; want free from 0 to a positive total",
		"scenario home: library root films is on node seedbox, which does not hold storage",
		"scenario home: library root nas is on node nas, which does not hold storage",
	}
	if err := s.Validate(); err == nil || err.Error() != strings.Join(want, "\n") {
		t.Fatalf("Validate = %v\nwant\n%s", err, strings.Join(want, "\n"))
	}
	if got := s.RootNode(s.Library.Roots[0]); got != "home" {
		t.Fatalf("a root naming no Node is on %q, want the controller home", got)
	}
	s.Nodes[1].Roles = append(s.Nodes[1].Roles, "storage")
	s.Library.Roots = s.Library.Roots[:2]
	if err := s.Validate(); err != nil {
		t.Fatalf("a root on seedbox holding storage: %v", err)
	}
	s.Nodes[0].Roles = []role.Name{"metadata"}
	s.Library.Roots = s.Library.Roots[:1]
	if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "library root movies is on node home, which does not hold storage") {
		t.Fatalf("a root on a controller that hands storage to seedbox: %v", err)
	}
	s.Nodes[1].Roles = []role.Name{"searching", "downloading"}
	if err := s.Validate(); err != nil {
		t.Fatalf("a root on a controller no Node takes storage from: %v", err)
	}
}

func TestLibraryWorksNeedTheMoviesAndARoot(t *testing.T) {
	base := func(tmdb *TMDB, w LibraryWorks, roots ...LibraryRoot) Scenario {
		s := singleMachine()
		s.TMDB, s.Releases = tmdb, nil
		s.Library = &Library{Roots: roots, Works: &w}
		return s
	}
	movies := LibraryRoot{ContentType: "movie", Folder: "movies"}
	for _, c := range []struct {
		s    Scenario
		want string
	}{
		{base(quietOrchard(), LibraryWorks{ContentType: "movie"}, movies), ""},
		{base(quietOrchard(), LibraryWorks{ContentType: "music"}, movies), `scenario single-machine: library works content_type is "music"; only movie Works come from a catalog`},
		{base(nil, LibraryWorks{ContentType: "movie"}, movies), "scenario single-machine: library works adds movies, and tmdb serves none"},
		{base(quietOrchard(), LibraryWorks{ContentType: "movie"}), "scenario single-machine: library works adds movies, and library has no movie root"},
	} {
		got := ""
		if err := c.s.Validate(); err != nil {
			got = err.Error()
		}
		if got != c.want {
			t.Errorf("Validate = %q, want %q", got, c.want)
		}
	}
}
