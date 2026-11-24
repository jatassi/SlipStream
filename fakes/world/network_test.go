package world_test

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/gateway"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/netfacts/script"
)

var networkFakes = []string{"stun", "stun-2", "gateway", "gateway-igd", "dns", "outside", "proxy"}

// A launch also proves each Node's Census empty, since admit refuses
// any other.
func TestTheNetworkFakesPlayOnlyAScenarioWithANetwork(t *testing.T) {
	t.Parallel()
	home, err := scenario.Load("home")
	if err != nil {
		t.Fatal(err)
	}
	network, err := world.NetworkHome()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		scenario scenario.Scenario
		network  bool
	}{
		{home, false},
		{network, true},
	} {
		t.Run(c.scenario.Name+map[bool]string{true: " with a network"}[c.network], func(t *testing.T) {
			t.Parallel()
			ctx := worldtest.Context(t)
			run, err := world.Launch(ctx, world.Options{Binary: testbin.Slipstream(t), Scenario: c.scenario, RunDir: filepath.Join(t.TempDir(), "run"), Origin: worldtest.Origin(t)})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _, _ = run.Stop(ctx) })
			checkNetworkWorld(t, run.Manifest(), c.network)
			if check := world.Doctor(ctx, run.Manifest().RunDir, ".").Checks[3]; !check.OK {
				t.Errorf("doctor's %q failed with %v", check.Name, check.Failures)
			}
		})
	}
}

// checkNetworkWorld checks m runs the network fakes, trusts the outside
// dialer and holds a script per Node only when network says so.
func checkNetworkWorld(t *testing.T, m world.Manifest, network bool) {
	t.Helper()
	var played []string
	for _, f := range m.Fakes {
		if slices.Contains(networkFakes, f.Name) {
			played = append(played, f.Name)
		}
	}
	if want := map[bool][]string{true: networkFakes}[network]; !slices.Equal(played, want) {
		t.Errorf("network fakes = %v, want %v", played, want)
	}
	if got := slices.Contains(m.Divergences, "outside-dialer"); got != network {
		t.Errorf("divergences %v name outside-dialer: %t, want %t", m.Divergences, got, network)
	}
	if !network {
		return
	}
	for _, n := range m.Nodes {
		if _, err := script.Read(gateway.ScriptPath(filepath.Join(m.ScratchDir, "fakes", gateway.Fake.Name), n.Name)); err != nil {
			t.Errorf("node %s's script: %v", n.Name, err)
		}
	}
}
