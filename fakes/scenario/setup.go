package scenario

import (
	"fmt"

	"go.yaml.in/yaml/v3"
)

// Setup is what the world does with a Node once it is ready. The zero value
// leaves it Unclaimed.
type Setup uint8

const (
	SetupNone Setup = iota
	SetupController
)

var setupNames = [...]string{SetupNone: "none", SetupController: "controller"}

func (s Setup) String() string { return setupNames[s] }

func (s Setup) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

func (s *Setup) UnmarshalYAML(n *yaml.Node) error {
	for i, name := range setupNames {
		if n.Kind == yaml.ScalarNode && n.Value == name {
			*s = Setup(i)
			return nil
		}
	}
	return &yaml.TypeError{Errors: []string{fmt.Sprintf("line %d: setup %q is not one of none, controller", n.Line, n.Value)}}
}
