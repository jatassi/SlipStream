// Package scenario is the Fake world's one description of a world: its
// Nodes, its first Admin and the timings every Node runs with. A Scenario is
// a catalog file read strictly, or a value a Go test builds.
package scenario

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/jatassi/slipstream/internal/node"
)

type Scenario struct {
	Name        string            `json:"-"            yaml:"-"`
	Description string            `yaml:"description"`
	Seed        string            `yaml:"seed"`
	Admin       *Admin            `yaml:"admin,omitempty"`
	Nodes       []Node            `jsonschema:"required,minItems=1,maxItems=31" yaml:"nodes"`
	Timings     node.TimingsBlock `yaml:"timings"`
}

// Admin is the first Admin a controller Node is set up with.
type Admin struct {
	Email       string `jsonschema:"required"                   yaml:"email"`
	DisplayName string `jsonschema:"required"                   yaml:"display_name"`
	PIN         string `jsonschema:"required,pattern=^[0-9]{4}$" yaml:"pin"`
}

type Node struct {
	Name  string `jsonschema:"required,pattern=^[a-z][a-z0-9-]{0\\,15}$" yaml:"name"`
	Setup Setup  `yaml:"setup"`
}

const MaxNodes = 31

var (
	nodeName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,15}$`)
	pin      = regexp.MustCompile(`^\d{4}$`)
)

//go:embed catalog/*.yaml
var catalog embed.FS

// Load reads the catalog Scenario name, strictly, and validates it.
func Load(name string) (Scenario, error) {
	data, err := catalog.ReadFile("catalog/" + name + ".yaml")
	if errors.Is(err, fs.ErrNotExist) {
		return Scenario{}, fmt.Errorf("scenario %s is not in the catalog (%s)", name, strings.Join(names(), ", "))
	}
	if err != nil {
		return Scenario{}, fmt.Errorf("scenario %s: %w", name, err)
	}
	return parse(name, data)
}

func names() []string {
	entries, _ := catalog.ReadDir("catalog")
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, strings.TrimSuffix(e.Name(), ".yaml"))
	}
	return out
}

// parse decodes data over the production timings, so an unset timings key
// keeps its production value. Unknown and duplicate keys fail with their
// line.
func parse(name string, data []byte) (Scenario, error) {
	s := Scenario{Name: name, Timings: node.Defaults().Timings}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil {
		var te *yaml.TypeError
		if errors.As(err, &te) {
			lines := make([]string, len(te.Errors))
			for i, e := range te.Errors {
				lines[i] = "scenario " + name + ": " + e
			}
			return Scenario{}, errors.New(strings.Join(lines, "\n"))
		}
		return Scenario{}, fmt.Errorf("scenario %s: %w", name, err)
	}
	return s, s.Validate()
}

// Validate checks the rules the types cannot hold.
func (s Scenario) Validate() error {
	if s.Name == "" {
		return errors.New("scenario has no name")
	}
	var problems []string
	add := func(format string, args ...any) {
		problems = append(problems, "scenario "+s.Name+": "+fmt.Sprintf(format, args...))
	}
	if len(s.Nodes) == 0 || len(s.Nodes) > MaxNodes {
		add("has %d nodes; a Scenario has 1 to %d", len(s.Nodes), MaxNodes)
	}
	seen := map[string]bool{}
	var controllers []string
	for _, n := range s.Nodes {
		if !nodeName.MatchString(n.Name) {
			add("node name %q does not match %s", n.Name, nodeName)
		}
		if seen[n.Name] {
			add("node name %s is used twice", n.Name)
		}
		seen[n.Name] = true
		if n.Setup == SetupController {
			controllers = append(controllers, n.Name)
		}
	}
	if len(controllers) > 1 {
		add("nodes %s are all controller; a Scenario has at most one", strings.Join(controllers, ", "))
	}
	switch {
	case s.Admin == nil && len(controllers) > 0:
		add("node %s is controller, so admin is required", controllers[0])
	case s.Admin != nil && len(controllers) == 0:
		add("admin is set, but no node is controller")
	case s.Admin != nil && !pin.MatchString(s.Admin.PIN):
		add("admin pin %q is not four digits", s.Admin.PIN)
	}
	if problems != nil {
		return errors.New(strings.Join(problems, "\n"))
	}
	return nil
}

// Hash is the hex SHA-256 of the Scenario's canonical JSON, so a Scenario
// built in code hashes like the same one read from a file. The name is not
// part of it.
func (s Scenario) Hash() string {
	data, err := json.Marshal(s)
	if err != nil {
		panic("scenario: " + err.Error())
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// YAML is the Scenario as the strict decoder reads it back, with every
// timings key written out.
func (s Scenario) YAML() []byte {
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(s); err != nil {
		panic("scenario: " + err.Error())
	}
	return b.Bytes()
}
