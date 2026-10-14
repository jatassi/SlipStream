package world_test

import (
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/fakes/world"
)

func TestExpand(t *testing.T) {
	m := world.Manifest{
		RunDir: "/runs/a",
		APIKey: "ssk_x",
		Nodes:  []world.Node{{Name: "home", URL: "http://127.0.0.2:8484", PID: 42}},
		Fakes:  []world.Fake{{Name: "tmdb", URL: "https://127.0.0.1:9000"}},
		Fronts: []world.Front{
			{Target: "tmdb", Node: "home", Address: "127.0.0.1:9001", URL: "https://127.0.0.1:9001"},
			{Target: "tmdb", Node: "world", Address: "127.0.0.1:9000", URL: "https://127.0.0.1:9000"},
		},
		Releases: []world.Release{{Name: "The.Quiet.Orchard.2021", BLAKE3: "b3"}},
	}
	got, err := world.Expand(m, []string{"curl", "-H", "X-Api-Key: {apiKey}", "{nodes.home.url}/api/v1/system", "{fakes.tmdb.url}", "{nodes.home.pid}", `{{"a":{runDir}}`, "{{apiKey}", "{ not a token }", "{fronts.tmdb.home.url}", "{releases.The.Quiet.Orchard.2021.blake3}"})
	want := []string{"curl", "-H", "X-Api-Key: ssk_x", "http://127.0.0.2:8484/api/v1/system", "https://127.0.0.1:9000", "42", `{"a":/runs/a}`, "{apiKey}", "{ not a token }", "https://127.0.0.1:9001", "b3"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Expand = %q, %v\nwant %q", got, err, want)
	}
	for arg, msg := range map[string]string{
		"{nodes.nas.url}":     "{nodes.nas.url} names nothing in the manifest",
		"{runDir.length}":     "{runDir.length} names nothing in the manifest",
		"x{nodes.home}y":      "{nodes.home} names a list or an object, not a value",
		"{nodes}":             "{nodes} names a list or an object, not a value",
		"{fronts.tmdb.url}":   "{fronts.tmdb.url} names nothing in the manifest",
		"{mediaDir} {binary}": "{mediaDir} names nothing in the manifest",
	} {
		if _, err := world.Expand(m, []string{arg}); err == nil || err.Error() != msg {
			t.Errorf("Expand(%q) = %v, want %s", arg, err, msg)
		}
	}
}

func TestLiteralExpandsToItself(t *testing.T) {
	m := world.Manifest{APIKey: "ssk_x"}
	for _, s := range []string{`{"contentType":"movie"}`, "{apiKey}", "{{apiKey}", "{{", "{{{", "{{{apiKey}", "{ apiKey }", "{nodes.home.url}/x"} {
		if got, err := world.Expand(m, []string{world.Literal(s)}); err != nil || got[0] != s {
			t.Errorf("Expand(Literal(%q)) = %q, %v", s, got, err)
		}
	}
}
