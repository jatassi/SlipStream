package scenario

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Setup is what the world does with a Node once it is ready. The zero value
// leaves it Unclaimed, controller claims it with the Scenario's Admin, and
// join pairs it to the controller Node by Join Code.
type Setup uint8

const (
	SetupNone Setup = iota
	SetupController
	SetupJoin
)

var setupNames = [...]string{SetupNone: "none", SetupController: "controller", SetupJoin: "join"}

var setupList = strings.Join(setupNames[:], ", ")

func (s Setup) String() string { return setupNames[s] }

func (s Setup) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

func (s *Setup) UnmarshalText(b []byte) error {
	for v := range Setup(len(setupNames)) {
		if setupNames[v] == string(b) {
			*s = v
			return nil
		}
	}
	return fmt.Errorf("setup %q is not one of %s", b, setupList)
}

func (s *Setup) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		if err := s.UnmarshalText([]byte(n.Value)); err == nil {
			return nil
		}
	}
	return &yaml.TypeError{Errors: []string{fmt.Sprintf("line %d: setup %q is not one of %s", n.Line, n.Value, setupList)}}
}
