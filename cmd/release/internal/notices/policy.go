package notices

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

type Denial struct {
	License string `json:"license"`
	Reason  string `json:"reason"`
}

// Override records the license a maintainer read for one exact version of a
// dependency, so a later version is checked afresh.
type Override struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	License string `json:"license"`
	Reason  string `json:"reason"`
}

type Policy struct {
	Allow     []string   `json:"allow"`
	Deny      []Denial   `json:"deny"`
	Overrides []Override `json:"overrides"`
	Vendored  []Vendored `json:"vendored"`
}

func loadPolicy(path string) (Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("read the license policy: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var p Policy
	if err := dec.Decode(&p); err != nil {
		return Policy{}, fmt.Errorf("%s: %w", policyPath, err)
	}
	return p, nil
}

// enforce applies the overrides to deps, then fails on any override that
// matches nothing shipped and any dependency the allow-list does not cover.
func (p Policy) enforce(deps []Dependency) error {
	var failures []string
	for _, o := range p.Overrides {
		i := slices.IndexFunc(deps, func(d Dependency) bool { return d.Name == o.Name && d.Version == o.Version })
		if i < 0 {
			failures = append(failures, fmt.Sprintf("override %s %s in %s matches no shipped dependency", o.Name, o.Version, policyPath))
			continue
		}
		deps[i].License = o.License
	}
	for _, d := range deps {
		if i := slices.IndexFunc(p.Deny, func(x Denial) bool { return x.License == d.License }); i >= 0 {
			failures = append(failures, fmt.Sprintf("%s: license %s is denied: %s", d, d.License, p.Deny[i].Reason))
			continue
		}
		if !slices.Contains(p.Allow, d.License) {
			failures = append(failures, fmt.Sprintf("%s: license %s is not on the allow-list in %s", d, d.License, policyPath))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "\n"))
	}
	return nil
}
