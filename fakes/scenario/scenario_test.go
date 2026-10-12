package scenario

import (
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/storage"
)

func singleMachine() Scenario {
	return Scenario{
		Name:        "single-machine",
		Description: "One Node set up as the Controller with the first Admin created through the Setup code.",
		Seed:        "single-machine",
		Admin:       &Admin{Email: "ada@example.test", DisplayName: "Ada Okafor", PIN: "2468"},
		Nodes:       []Node{{Name: "home", Setup: SetupController}},
		Timings: node.TimingsBlock{
			Node:    node.Timings{StopDeadline: 5 * time.Second},
			Storage: storage.Timings{BusyTimeout: 5 * time.Second},
			HTTP:    api.Timings{ReadHeader: 10 * time.Second, Idle: 2 * time.Minute},
			Live:    live.Timings{Heartbeat: 25 * time.Second, Retry: 3 * time.Second},
		},
	}
}

func TestLoadKeepsProductionTimingsForUnsetKeys(t *testing.T) {
	got, err := Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	if want := singleMachine(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Load(single-machine) = %+v, want %+v", got, want)
	}
}

func TestLoadRefusesANameOutsideTheCatalog(t *testing.T) {
	_, err := Load("home")
	if want := "scenario home is not in the catalog (first-run, single-machine)"; err == nil || err.Error() != want {
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
			"seed: x\nnodes:\n  - name: home\n    setup: join\n",
			`scenario x: line 4: setup "join" is not one of none, controller`,
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
			"two problems at once",
			"seed: x\nnodes: [{name: Home}, {name: Home}]\n",
			"scenario x: node name \"Home\" does not match ^[a-z][a-z0-9-]{0,15}$\n" +
				"scenario x: node name \"Home\" does not match ^[a-z][a-z0-9-]{0,15}$\n" +
				"scenario x: node name Home is used twice",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := parse("x", []byte(c.yaml))
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
  storage:
    busy_timeout: 5s
  http:
    read_header: 10s
    idle: 2m0s
  live:
    heartbeat: 25s
    retry: 3s
`
	if got := string(s.YAML()); got != want {
		t.Fatalf("YAML() =\n%s\nwant\n%s", got, want)
	}
	back, err := parse("single-machine", s.YAML())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, s) {
		t.Fatalf("read back %+v, want %+v", back, s)
	}
}
