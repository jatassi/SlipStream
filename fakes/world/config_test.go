package world

import (
	"log/slog"
	"net/netip"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/nodeenv"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/internal/setup"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node"
)

func TestDivergencesRefuseAPlantedKey(t *testing.T) {
	r := &Run{used: map[string]bool{}}
	resolved := node.Resolved{Keys: []node.Key{
		{Key: "seed", Value: "first-run/home", Default: "", Source: "config.yaml"},
		{Key: "torrent.dht", Value: "true", Default: "false", Source: "config.yaml"},
	}, Census: []string{}}
	err := r.admit("home", resolved, nodeenv.World("/runs/r1/ca.pem"))
	if want := "node home sets torrent.dht, which the divergences list in fakes/world does not name"; err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %s", err, want)
	}
}

func TestEveryDivergenceIsUsedByTheCatalog(t *testing.T) {
	var defaults yaml.Node
	if err := defaults.Encode(node.Defaults(contenttype.Set{}, install.Dev)); err != nil {
		t.Fatal(err)
	}
	var probe node.Resolved
	for _, key := range flatten(&defaults, "") {
		probe.Keys = append(probe.Keys, node.Key{Key: key})
	}
	listen := setup.ListenKeys(probe)
	if want := []string{"torrent.listen", "http.listen", "link.listen"}; !slices.Equal(listen, want) {
		t.Fatalf("listen keys = %v, want %v", listen, want)
	}
	used := map[string]bool{}
	for _, name := range scenario.Catalog() {
		s, err := scenario.Load(name)
		if err != nil {
			t.Fatal(err)
		}
		endpoints := fakeEndpoints(s)
		for i := range s.Nodes {
			ids, uncovered := diverge(settings(configKeys(t, s, i, listen, endpoints), nodeenv.World("ca.pem")))
			if uncovered != nil {
				t.Fatalf("catalog %s node %s sets %v, which no divergence names", name, s.Nodes[i].Name, uncovered)
			}
			for id := range ids {
				used[id] = true
			}
		}
	}
	var all []string
	for _, d := range divergences {
		all = append(all, d.id)
	}
	if got := usedIDs(used); !reflect.DeepEqual(got, all) {
		t.Fatalf("the catalog uses divergences %v of %v; delete the rest", got, all)
	}
}

func fakeEndpoints(s scenario.Scenario) node.EndpointsBlock {
	e := node.Defaults(contenttype.Set{}, install.Dev).Endpoints
	at := fake.Address{AddrPort: netip.MustParseAddrPort("127.0.0.1:1"), TLS: true}
	for _, spec := range fakes {
		if spec.ContentType == "" || slices.Contains(s.ContentTypes(), spec.ContentType) {
			spec.Endpoints(&e, at)
		}
	}
	return e
}

func configKeys(t *testing.T, s scenario.Scenario, i int, listen []string, endpoints node.EndpointsBlock) []node.Key {
	t.Helper()
	data, err := setup.Config(s, i, listen, netip.MustParseAddrPort("127.0.0.1:2"), endpoints)
	if err != nil {
		t.Fatal(err)
	}
	var config yaml.Node
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	var keys []node.Key
	for _, key := range flatten(config.Content[0], "") {
		keys = append(keys, node.Key{Key: key})
	}
	return keys
}

func flatten(n *yaml.Node, prefix string) []string {
	var keys []string
	for i := 0; i < len(n.Content); i += 2 {
		key, value := prefix+n.Content[i].Value, n.Content[i+1]
		if value.Kind == yaml.MappingNode {
			keys = append(keys, flatten(value, key+".")...)
			continue
		}
		keys = append(keys, key)
	}
	return keys
}

func TestSupervisorGoneNamesEachNodesStopCommand(t *testing.T) {
	err := (&SupervisorGoneError{PID: 4242, Spawns: []Spawn{
		{Node: "a", PID: 4250, DataDir: "/tmp/ssfw/k3v9x2qa/a"},
		{Node: "b", PID: 4251, DataDir: "/tmp/ssfw/k3v9x2qa/b"},
	}}).Error()
	want := strings.Join([]string{
		"the supervisor (PID 4242) is not running; stop each Node with:",
		"  slipstream stop --data-dir /tmp/ssfw/k3v9x2qa/a",
		"  slipstream stop --data-dir /tmp/ssfw/k3v9x2qa/b",
	}, "\n")
	if err != want {
		t.Fatalf("error =\n%s\nwant\n%s", err, want)
	}
}

// discard is an event log that keeps nothing, for fakes built with no IO.
type discard struct{}

func (discard) Append(rec events.Record) (events.Record, error) { return rec, nil }

// catalogPlan is the fault Plan of a world of every fake running s, with
// each fake built, as at launch, so its routes are declared.
func catalogPlan(t *testing.T, s scenario.Scenario) *fault.Plan {
	t.Helper()
	names := make([]string, len(s.Nodes))
	for i, n := range s.Nodes {
		names[i] = n.Name
	}
	plan := fault.NewPlan(planFronts(fakes, names, s.Releases), nil)
	for _, spec := range fakes {
		if _, err := spec.New(s, fake.NewEnv(s, spec, slog.New(slog.DiscardHandler), plan, discard{})); err != nil {
			t.Fatal(err)
		}
	}
	return plan
}

func TestCatalogFaultsResolve(t *testing.T) {
	for _, name := range scenario.Catalog() {
		s, err := scenario.Load(name)
		if err != nil {
			t.Fatal(err)
		}
		plan := catalogPlan(t, s)
		for _, r := range s.Faults {
			if _, err := plan.Set(r); err != nil {
				t.Errorf("catalog %s fault %s: %v", name, r.ID, err)
			}
		}
	}
	s, err := scenario.Load("music")
	if err != nil {
		t.Fatal(err)
	}
	plan := catalogPlan(t, s)
	for _, c := range []struct {
		rule fault.Rule
		want string
	}{
		{fault.Rule{ID: "x", Target: "music-meta", Route: "artists", Fault: fault.Hang{}}, "music-meta serves no route artists"},
		{fault.Rule{ID: "x", Target: "music-meta", Route: "artist", Params: map[string]string{"id": "1"}, Fault: fault.Hang{}}, "route artist of music-meta has no path value id"},
		{fault.Rule{ID: "x", Target: "music-meta", Route: "artist", Fault: fault.Throttle(1024)}, "throttle=1KiB/s is a connection fault and takes no route or params"},
		{fault.Rule{ID: "x", Target: "musicmeta", Fault: fault.Hang{}}, "no target musicmeta"},
	} {
		if _, err := plan.Set(c.rule); err == nil || err.Error() != c.want {
			t.Errorf("Set(%+v) = %v, want %s", c.rule, err, c.want)
		}
	}
}
