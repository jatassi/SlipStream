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
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/jatassi/slipstream/fakes/internal/yamltree"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/role"
)

type Scenario struct {
	Name        string `yaml:"-"`
	Description string `yaml:"description"`
	Seed        string `yaml:"seed"`
	Admin       *Admin `yaml:"admin,omitempty"`
	Nodes       []Node `jsonschema:"required,minItems=1,maxItems=31" yaml:"nodes"`
	// Timings holds the durations the Scenario shortens. A zero duration is
	// unset, so the Node keeps its production value.
	Timings node.TimingsBlock `yaml:"timings"`
	// Music is the catalog fake-music-meta serves. A Scenario that sets it
	// runs only on a binary that registers the music Content Type.
	Music *Music `yaml:"music,omitempty"`
	// TMDB is the catalog fake-tmdb serves; nil serves an empty one.
	TMDB *TMDB `yaml:"tmdb,omitempty"`
	// Library is what the world creates on the controller Node after
	// setting it up.
	Library *Library `yaml:"library,omitempty"`
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
	// Roles are the Roles the world assigns a controller or join Node.
	Roles []role.Name `jsonschema:"uniqueItems=true" yaml:"roles,omitempty"`
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
		return Scenario{}, fmt.Errorf("scenario %s is not in the catalog (%s)", name, strings.Join(Catalog(), ", "))
	}
	if err != nil {
		return Scenario{}, fmt.Errorf("scenario %s: %w", name, err)
	}
	return parse(name, data)
}

// Catalog is the name of every catalog Scenario.
func Catalog() []string {
	entries, _ := catalog.ReadDir("catalog")
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, strings.TrimSuffix(e.Name(), ".yaml"))
	}
	return out
}

func parse(name string, data []byte) (Scenario, error) {
	s := Scenario{Name: name}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil {
		if te, ok := errors.AsType[*yaml.TypeError](err); ok {
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
	controllers := s.checkNodes(add)
	switch {
	case s.Admin == nil && len(controllers) > 0:
		add("node %s is controller, so admin is required", controllers[0])
	case s.Admin != nil && len(controllers) == 0:
		add("admin is set, but no node is controller")
	case s.Admin != nil && !pin.MatchString(s.Admin.PIN):
		add("admin pin %q is not four digits", s.Admin.PIN)
	}
	if s.Music != nil {
		s.Music.check(add)
	}
	if s.TMDB != nil {
		s.TMDB.check(add)
	}
	if s.Library != nil {
		if len(controllers) == 0 {
			add("library is set, but no node is controller")
		}
		s.Library.check(add)
	}
	if problems != nil {
		return errors.New(strings.Join(problems, "\n"))
	}
	return nil
}

// checkNodes checks each Node and the rules across them, and returns the
// controller Nodes.
func (s Scenario) checkNodes(add func(format string, args ...any)) []string {
	seen := map[string]bool{}
	var controllers, relaying []string
	for _, n := range s.Nodes {
		if !nodeName.MatchString(n.Name) {
			add("node name %q does not match %s", n.Name, nodeName)
		}
		if seen[n.Name] {
			add("node name %s is used twice", n.Name)
		}
		seen[n.Name] = true
		switch n.Setup {
		case SetupController:
			controllers = append(controllers, n.Name)
		case SetupJoin:
			if len(controllers) == 0 {
				add("node %s is join, so an earlier node must be controller", n.Name)
			}
		case SetupNone:
		}
		n.checkRoles(add)
		if slices.Contains(n.Roles, role.Relaying) {
			relaying = append(relaying, n.Name)
		}
	}
	if len(controllers) > 1 {
		add("nodes %s are all controller; a Scenario has at most one", strings.Join(controllers, ", "))
	}
	if len(relaying) > 1 {
		add("nodes %s all hold relaying; at most one Node does", strings.Join(relaying, ", "))
	}
	return controllers
}

func (n Node) checkRoles(add func(format string, args ...any)) {
	if n.Setup == SetupNone && len(n.Roles) > 0 {
		add("node %s has roles, but only controller and join nodes hold Roles", n.Name)
	}
	for i, r := range n.Roles {
		if _, err := role.ParseName(string(r)); err != nil {
			add("node %s role %s", n.Name, err)
		} else if slices.Contains(n.Roles[:i], r) {
			add("node %s holds role %s twice", n.Name, r)
		}
	}
}

// Hash is the hex SHA-256 of the Scenario's canonical JSON, so a Scenario
// built in code hashes like the same one read from a file. The name is not
// part of it.
func (s Scenario) Hash() string {
	var doc any
	if err := s.document().Decode(&doc); err != nil {
		panic("scenario: " + err.Error())
	}
	data, err := json.Marshal(doc)
	if err != nil {
		panic("scenario: " + err.Error())
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// YAML is the Scenario as the strict decoder reads it back.
func (s Scenario) YAML() []byte {
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(s.document()); err != nil {
		panic("scenario: " + err.Error())
	}
	return b.Bytes()
}

// SetTimings is the timings block holding only the keys t sets, or nil when
// it sets none.
func SetTimings(t node.TimingsBlock) *yaml.Node {
	n, err := yamltree.Diff(t, node.TimingsBlock{})
	if err != nil {
		panic("scenario: " + err.Error())
	}
	return n
}

func (s Scenario) document() *yaml.Node {
	type plain Scenario
	var doc yaml.Node
	if err := doc.Encode(plain(s)); err != nil {
		panic("scenario: " + err.Error())
	}
	for i := 0; i < len(doc.Content); i += 2 {
		if doc.Content[i].Value != "timings" {
			continue
		}
		if t := SetTimings(s.Timings); t != nil {
			doc.Content[i+1] = t
		} else {
			doc.Content = slices.Delete(doc.Content, i, i+2)
		}
		break
	}
	return &doc
}
