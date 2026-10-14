package fault

import (
	"fmt"
	"reflect"
	"strings"

	"go.yaml.in/yaml/v3"
)

// MarshalYAML writes Fault as its canonical text.
func (r Rule) MarshalYAML() (any, error) {
	fields := ruleFields(r)
	fields.Fault = nil
	var n yaml.Node
	if err := n.Encode(fields); err != nil {
		return nil, fmt.Errorf("rule %q: %w", r.ID, err)
	}
	for i := 0; i < len(n.Content); i += 2 {
		if n.Content[i].Value == "fault" && r.Fault != nil {
			n.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Value: r.Fault.String()}
		}
	}
	return &n, nil
}

// UnmarshalYAML reads a rule as a Scenario's faults block holds it,
// refusing a key Rule lacks the way a strict decoder does, since a
// decoder's strictness doesn't reach an unmarshaler.
func (r *Rule) UnmarshalYAML(value *yaml.Node) error {
	if errs := unknownKeys(value, reflect.TypeFor[Rule]()); len(errs) > 0 {
		return &yaml.TypeError{Errors: errs}
	}
	rest := *value
	rest.Content = nil
	var text *yaml.Node
	for i := 0; i+1 < len(value.Content); i += 2 {
		if value.Content[i].Value == "fault" {
			text = value.Content[i+1]
			continue
		}
		rest.Content = append(rest.Content, value.Content[i], value.Content[i+1])
	}
	if err := rest.Decode((*ruleFields)(r)); err != nil {
		return err //nolint:wrapcheck // a *yaml.TypeError the decoder collects with the Scenario's own
	}
	if text == nil {
		return nil
	}
	f, err := Parse(text.Value)
	if err != nil {
		return &yaml.TypeError{Errors: []string{fmt.Sprintf("line %d: rule %q: %v", text.Line, r.ID, err)}}
	}
	r.Fault = f
	return nil
}

// unknownKeys names each key of the mapping n that no yaml-tagged field of
// the struct t takes, at any depth.
func unknownKeys(n *yaml.Node, t reflect.Type) []string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if n.Kind != yaml.MappingNode || t.Kind() != reflect.Struct {
		return nil
	}
	fields := map[string]reflect.Type{}
	for i := range t.NumField() {
		f := t.Field(i)
		if name, _, _ := strings.Cut(f.Tag.Get("yaml"), ","); name != "" && name != "-" {
			fields[name] = f.Type
		}
	}
	var errs []string
	for i := 0; i+1 < len(n.Content); i += 2 {
		key := n.Content[i]
		ft, ok := fields[key.Value]
		if !ok {
			errs = append(errs, fmt.Sprintf("line %d: field %s not found in type %s", key.Line, key.Value, t))
			continue
		}
		errs = append(errs, unknownKeys(n.Content[i+1], ft)...)
	}
	return errs
}
