// Package yamltree writes the part of a config tree that differs from a
// base, so a file names only what a Scenario or a fake set.
package yamltree

import (
	"bytes"
	"fmt"

	"go.yaml.in/yaml/v3"
)

// Diff is v as a YAML mapping holding only the leaves whose value differs
// from base's, or nil when none does. v and base have the same type.
func Diff(v, base any) (*yaml.Node, error) {
	var a, b yaml.Node
	if err := a.Encode(v); err != nil {
		return nil, fmt.Errorf("encode %T: %w", v, err)
	}
	if err := b.Encode(base); err != nil {
		return nil, fmt.Errorf("encode %T: %w", base, err)
	}
	return diff(&a, &b), nil
}

func diff(a, b *yaml.Node) *yaml.Node {
	out := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for i := 0; i < len(a.Content); i += 2 {
		key, value := a.Content[i], a.Content[i+1]
		baseValue := lookup(b, key.Value)
		if value.Kind == yaml.MappingNode {
			if sub := diff(value, baseValue); sub != nil {
				out.Content = append(out.Content, key, sub)
			}
			continue
		}
		if !same(value, baseValue) {
			out.Content = append(out.Content, key, value)
		}
	}
	if len(out.Content) == 0 {
		return nil
	}
	return out
}

func lookup(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return &yaml.Node{Kind: yaml.MappingNode}
}

func same(a, b *yaml.Node) bool {
	x, errA := yaml.Marshal(a)
	y, errB := yaml.Marshal(b)
	return errA == nil && errB == nil && bytes.Equal(x, y)
}
