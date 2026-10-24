package scenario

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/role"
)

func singleMachine() Scenario {
	return Scenario{
		Name:        "single-machine",
		Description: "One Node set up as the Controller with the first Admin created through the Setup code.",
		Seed:        "single-machine",
		Admin:       &Admin{Email: "ada@example.test", DisplayName: "Ada Okafor", PIN: "2468"},
		Nodes:       []Node{{Name: "home", Setup: SetupController}},
		Timings:     node.TimingsBlock{Node: node.Timings{StopDeadline: 5 * time.Second}},
		TMDB:        quietOrchard(),
		Releases:    quietOrchardReleases(),
		Tracker:     Tracker{Interval: 5 * time.Second},
		Library:     movieLibrary(),
	}
}

func quietOrchardReleases() []Release {
	return []Release{
		{Name: "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW", TMDBID: 9000001, Video: Video{Width: 1920, Height: 1080, Seconds: 6}, Seeders: 12, Seeded: true, AdvertisedSize: 8804682957},
		{Name: "The.Quiet.Orchard.2021.720p.WEB-DL.H264-FKW", TMDBID: 9000001, Video: Video{Width: 1280, Height: 720, Seconds: 6}, Seeders: 40, AdvertisedSize: 4724464026},
	}
}

func movieLibrary() *Library {
	return &Library{
		StorageLevels: &LibraryLevels{},
		Roots:         []LibraryRoot{{ContentType: "movie", Folder: "movies"}},
		Indexers:      []LibraryIndexer{{Name: "Fixture Site", Site: "fixture-site"}},
	}
}

func quietOrchard() *TMDB {
	return &TMDB{Movies: []TMDBMovie{{ID: 9000001, IMDbID: "tt9000001", Title: "The Quiet Orchard", Released: "2021-04-16", Runtime: 104 * time.Minute}}}
}

func TestLoadLeavesUnsetTimingsZero(t *testing.T) {
	got, err := Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	if want := singleMachine(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Load(single-machine) = %+v, want %+v", got, want)
	}
}

func TestLoadReadsTheHomeScenario(t *testing.T) {
	got, err := Load("home")
	if err != nil {
		t.Fatal(err)
	}
	want := Scenario{
		Name:        "home",
		Description: "Two Nodes. The Controller home holds every Role but Acquire and Relaying. The Worker seedbox pairs by Join Code and holds Searching and Downloading. Grace joins as a Member by Signup link, and fakes/smtp is the email setting.",
		Seed:        "home",
		Admin:       &Admin{Email: "ada@example.test", DisplayName: "Ada Okafor", PIN: "2468"},
		Members:     []Member{{Email: "grace@example.test", DisplayName: "Grace Hopper", PIN: "1357"}},
		Mail:        true,
		Nodes: []Node{
			{Name: "home", Setup: SetupController, Roles: []role.Name{role.Storage, role.Converting, role.Subtitles, role.Metadata, role.MediaServerLink, role.Notifying, role.Serving}},
			{Name: "seedbox", Setup: SetupJoin, Roles: []role.Name{role.Searching, role.Downloading}},
		},
		Timings:  node.TimingsBlock{Node: node.Timings{StopDeadline: 5 * time.Second}},
		TMDB:     quietOrchard(),
		Releases: quietOrchardReleases(),
		Tracker:  Tracker{Interval: 5 * time.Second},
		Library:  movieLibrary(),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load(home) = %+v, want %+v", got, want)
	}
}

func TestLoadRefusesANameOutsideTheCatalog(t *testing.T) {
	_, err := Load("office")
	if want := "scenario office is not in the catalog (cgnat-home, first-run, home, low-space, movies, music, serving, single-machine, sizes, targets)"; err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %s", err, want)
	}
}

func TestParseRefuses(t *testing.T) {
	cases := []struct {
		name, yaml, want string
	}{
		{
			"unknown key",
			"seed: x\nnodes: [{name: home, setup: none}]\nfavourite: 1\n",             //nolint:misspell // an unknown key is the point
			"scenario x: line 3: field favourite not found in type scenario.Scenario", //nolint:misspell // an unknown key is the point
		},
		{
			"unknown timings key",
			"seed: x\nnodes: [{name: home, setup: none}]\ntimings:\n  node:\n    stop_dedline: 5s\n",
			"scenario x: line 5: field stop_dedline not found in type node.Timings",
		},
		{
			"duplicate key",
			"seed: x\nnodes: [{name: home, setup: none}]\nseed: y\n",
			`scenario x: line 3: mapping key "seed" already defined at line 1`,
		},
		{
			"setup outside the closed set",
			"seed: x\nnodes:\n  - name: home\n    setup: relay\n",
			`scenario x: line 4: setup "relay" is not one of none, controller, join`,
		},
		{
			"name in the file",
			"name: other\nseed: x\nnodes: [{name: home, setup: none}]\n",
			"scenario x: line 1: field name not found in type scenario.Scenario",
		},
		{
			"duration that is not one",
			"seed: x\nnodes: [{name: home, setup: none}]\ntimings: {node: {stop_deadline: soon}}\n",
			"scenario x: line 3: cannot unmarshal !!str `soon` into time.Duration",
		},
		{
			"advertised size without a unit",
			"seed: x\nnodes: [{name: home, setup: none}]\nreleases: [{name: A, tmdb_id: 1, video: {width: 2, height: 2, seconds: 1}, advertised_size: 8.2}]\n",
			`scenario x: line 3: size "8.2" is not a number and a unit such as 8.2GiB`,
		},
		{
			"node name",
			"seed: x\nnodes: [{name: Home_1, setup: none}]\n",
			`scenario x: node name "Home_1" does not match ^[a-z][a-z0-9-]{0,15}$`,
		},
		{
			"pin",
			"seed: x\nadmin: {email: ada@example.test, display_name: Ada, pin: \"12a4\"}\nnodes: [{name: home, setup: controller}]\n",
			`scenario x: admin pin "12a4" is not four digits`,
		},
		{
			"members without an admin",
			"seed: x\nmembers: [{email: grace@example.test, display_name: Grace, pin: \"1357\"}]\nnodes: [{name: home, setup: none}]\n",
			"scenario x: members are set, but no admin issues their Signup links",
		},
		{
			"mail without an admin",
			"seed: x\nmail: true\nnodes: [{name: home, setup: none}]\n",
			"scenario x: mail is set, but no admin saves it",
		},
		{
			"member email used twice",
			"seed: x\nadmin: {email: ada@example.test, display_name: Ada, pin: \"2468\"}\nmembers: [{email: Ada@Example.test, display_name: Ada Again, pin: \"1357\"}]\nnodes: [{name: home, setup: controller}]\n",
			"scenario x: member email Ada@Example.test is used twice",
		},
		{
			"member pin",
			"seed: x\nadmin: {email: ada@example.test, display_name: Ada, pin: \"2468\"}\nmembers: [{email: grace@example.test, display_name: Grace, pin: \"135\"}]\nnodes: [{name: home, setup: controller}]\n",
			`scenario x: member grace@example.test pin "135" is not four digits`,
		},
		{
			"two controllers",
			"seed: x\nadmin: {email: ada@example.test, display_name: Ada, pin: \"2468\"}\nnodes: [{name: home, setup: controller}, {name: work, setup: controller}]\n",
			"scenario x: nodes home, work are all controller; a Scenario has at most one",
		},
		{
			"admin without a controller",
			"seed: x\nadmin: {email: ada@example.test, display_name: Ada, pin: \"2468\"}\nnodes: [{name: home, setup: none}]\n",
			"scenario x: admin is set, but no node is controller",
		},
		{
			"controller without an admin",
			"seed: x\nnodes: [{name: home, setup: controller}]\n",
			"scenario x: node home is controller, so admin is required",
		},
		{
			"join without an earlier controller",
			"seed: x\nadmin: {email: ada@example.test, display_name: Ada, pin: \"2468\"}\nnodes: [{name: box, setup: join}, {name: home, setup: controller}]\n",
			"scenario x: node box is join, so an earlier node must be controller",
		},
		{
			"tmdb movie",
			"seed: x\nnodes: [{name: home, setup: none}]\ntmdb:\n  movies:\n    - {id: 0, imdb_id: nm1, title: \"\", released: 2021-4-16, runtime: 90s}\n    - {id: 0, imdb_id: tt0000001, title: Again, released: 2021-02-30, runtime: 90m}\n",
			"scenario x: tmdb movie \"\" has id 0; ids start at 1\n" +
				"scenario x: tmdb movie 0 imdb_id \"nm1\" is not tt and seven or more digits\n" +
				"scenario x: tmdb movie 0 has no title\n" +
				"scenario x: tmdb movie 0 released \"2021-4-16\" is not YYYY-MM-DD\n" +
				"scenario x: tmdb movie 0 runtime 1m30s is not whole minutes\n" +
				"scenario x: tmdb movie \"Again\" has id 0; ids start at 1\n" +
				"scenario x: tmdb movie 0 is listed twice\n" +
				"scenario x: tmdb movie 0 released \"2021-02-30\" is not YYYY-MM-DD",
		},
		{
			"roles on an Unclaimed node",
			"seed: x\nnodes: [{name: home, setup: none, roles: [searching]}]\n",
			"scenario x: node home has roles, but only controller and join nodes hold Roles",
		},
		{
			"a role twice",
			"seed: x\nadmin: {email: ada@example.test, display_name: Ada, pin: \"2468\"}\nnodes: [{name: home, setup: controller, roles: [searching, storage, searching]}]\n",
			"scenario x: node home holds role searching twice",
		},
		{
			"a role that is not one",
			"seed: x\nadmin: {email: ada@example.test, display_name: Ada, pin: \"2468\"}\nnodes: [{name: home, setup: controller, roles: [acquire]}]\n",
			`scenario x: node home role "acquire" is not a Role`,
		},
		{
			"two relaying nodes",
			"seed: x\nadmin: {email: ada@example.test, display_name: Ada, pin: \"2468\"}\nnodes: [{name: home, setup: controller, roles: [relaying]}, {name: box, setup: join, roles: [relaying]}]\n",
			"scenario x: nodes home, box all hold relaying; at most one Node does",
		},
		{
			"names used twice",
			"seed: x\nnodes: [{name: home}, {name: home}]\n",
			"scenario x: node name home is used twice",
		},
		{
			"no nodes",
			"seed: x\nnodes: []\n",
			"scenario x: has 0 nodes; a Scenario has 1 to 31",
		},
		{
			"an unknown key in a fault rule and its query",
			"seed: x\nnodes: [{name: home}]\nfaults:\n  - id: a\n    target: tmdb\n    fault: hang\n    count: 2\n    after: {kind: cue, nodes: home}\n",
			"scenario x: line 7: field count not found in type fault.Rule\n" +
				"scenario x: line 8: field nodes not found in type events.Query",
		},
		{
			"a fault the grammar refuses",
			"seed: x\nnodes: [{name: home}]\nfaults:\n  - {id: a, target: tmdb, fault: throttle=fast}\n",
			`scenario x: line 4: rule "a": fault throttle=fast: rate "fast" needs /s`,
		},
		{
			"fault rules without an id, twice the same or without a fault",
			"seed: x\nnodes: [{name: home}]\nfaults:\n  - {target: tmdb, fault: hang}\n  - {id: a, target: tmdb, fault: reset}\n  - {id: a, target: tmdb}\n",
			"scenario x: fault 1 has no id\n" +
				"scenario x: fault id a is used twice\n" +
				"scenario x: fault a names no fault",
		},
		{
			"two problems at once",
			"seed: x\nnodes: [{name: Home}, {name: Home}]\n",
			"scenario x: node name \"Home\" does not match ^[a-z][a-z0-9-]{0,15}$\n" +
				"scenario x: node name \"Home\" does not match ^[a-z][a-z0-9-]{0,15}$\n" +
				"scenario x: node name Home is used twice",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse("x", []byte(c.yaml))
			if err == nil || err.Error() != c.want {
				t.Fatalf("err = %v, want %s", err, c.want)
			}
		})
	}
}

func TestValidateRefusesThirtyTwoNodes(t *testing.T) {
	s := Scenario{Name: "big"}
	for i := range 32 {
		s.Nodes = append(s.Nodes, Node{Name: "n" + strconv.Itoa(i)})
	}
	err := s.Validate()
	if want := "scenario big: has 32 nodes; a Scenario has 1 to 31"; err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %s", err, want)
	}
}

func TestValidateRefusesAnUnnamedScenario(t *testing.T) {
	err := Scenario{Nodes: []Node{{Name: "home"}}}.Validate()
	if want := "scenario has no name"; err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %s", err, want)
	}
}

func TestHashOfABuiltScenarioMatchesTheFile(t *testing.T) {
	loaded, err := Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	built := singleMachine()
	if loaded.Hash() != built.Hash() {
		t.Fatalf("built hash %s, loaded hash %s", built.Hash(), loaded.Hash())
	}
	built.Timings.Node.StopDeadline = 6 * time.Second
	if loaded.Hash() == built.Hash() {
		t.Fatalf("a different stop_deadline kept hash %s", built.Hash())
	}
}

func TestYAMLReadsBackAsTheSameScenario(t *testing.T) {
	s := singleMachine()
	want := `description: One Node set up as the Controller with the first Admin created through the Setup code.
seed: single-machine
admin:
  email: ada@example.test
  display_name: Ada Okafor
  pin: "2468"
nodes:
  - name: home
    setup: controller
timings:
  node:
    stop_deadline: 5s
tmdb:
  movies:
    - id: 9000001
      imdb_id: tt9000001
      title: The Quiet Orchard
      released: "2021-04-16"
      runtime: 1h44m0s
releases:
  - name: The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW
    tmdb_id: 9000001
    video:
      width: 1920
      height: 1080
      seconds: 6
    seeders: 12
    seeded: true
    advertised_size: 8.2GiB
  - name: The.Quiet.Orchard.2021.720p.WEB-DL.H264-FKW
    tmdb_id: 9000001
    video:
      width: 1280
      height: 720
      seconds: 6
    seeders: 40
    advertised_size: 4.4GiB
tracker:
  interval: 5s
library:
  storage_levels:
    warn_below: 0B
    error_below: 0B
  roots:
    - content_type: movie
      folder: movies
  indexers:
    - name: Fixture Site
      site: fixture-site
`
	if got := string(s.YAML()); got != want {
		t.Fatalf("YAML() =\n%s\nwant\n%s", got, want)
	}
	back, err := Parse("single-machine", s.YAML())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, s) {
		t.Fatalf("read back %+v, want %+v", back, s)
	}
}

func TestAScenarioWithoutTimingsHashesLikeTheFileWithout(t *testing.T) {
	built := Scenario{Name: "bare", Seed: "bare", Nodes: []Node{{Name: "home"}}}
	read, err := Parse("bare", []byte("seed: bare\nnodes: [{name: home, setup: none}]\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(read, built) || read.Hash() != built.Hash() {
		t.Fatalf("read %+v hash %s, built %+v hash %s", read, read.Hash(), built, built.Hash())
	}
	if want := "description: \"\"\nseed: bare\nnodes:\n  - name: home\n    setup: none\n"; string(built.YAML()) != want {
		t.Fatalf("YAML() = %q, want %q", built.YAML(), want)
	}
}

func TestYAMLWritesOnlyTheTimingsAScenarioSets(t *testing.T) {
	s := singleMachine()
	s.Timings.Live = live.Timings{Retry: 2 * time.Second}
	back, err := Parse(s.Name, s.YAML())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, s) {
		t.Fatalf("read back %+v, want %+v", back, s)
	}
	want := "timings:\n  node:\n    stop_deadline: 5s\n  live:\n    retry: 2s\n"
	if got := string(s.YAML()); !strings.Contains(got, want+"tmdb:\n") {
		t.Fatalf("YAML() =\n%s\nwant it to hold\n%s", got, want)
	}
}

// TestFaultsLiveOnlyInTheFaultsBlock keeps every fault in the one
// vocabulary: no key beneath the Scenario but its faults block names one.
func TestFaultsLiveOnlyInTheFaultsBlock(t *testing.T) {
	var private []string
	var walk func(t reflect.Type, path string)
	walk = func(t reflect.Type, path string) {
		for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Map {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return
		}
		for i := range t.NumField() {
			f := t.Field(i)
			name, _, _ := strings.Cut(f.Tag.Get("yaml"), ",")
			if name == "" || name == "-" {
				continue
			}
			key := strings.TrimPrefix(path+"."+name, ".")
			if key == "faults" {
				continue
			}
			if slices.ContainsFunc(strings.Split(name, "_"), func(w string) bool { return strings.HasPrefix(w, "fault") }) {
				private = append(private, key)
			}
			walk(f.Type, key)
		}
	}
	walk(reflect.TypeFor[Scenario](), "")
	if private != nil {
		t.Fatalf("scenario keys %v name faults outside the faults block; write them as fault rules", private)
	}
}
