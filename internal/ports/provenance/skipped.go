package provenance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

// SkippedPath is internal/ports/skipped.json from the repository root.
const SkippedPath = "internal/ports/skipped.json"

// Skipped is every Prowlarr indexer SlipStream decided not to port, so
// that prowlarr-sync never reports one as new.
type Skipped struct {
	// Commit is the Prowlarr commit this list, and the hidden list
	// (_definitionBlocklist), were last reviewed at.
	Commit string        `json:"commit"`
	Sites  []SkippedSite `json:"sites"`
}

type SkippedSite struct {
	// Name is the C# class.
	Name string `json:"name"`
	// Path is its file, or its folder with a trailing slash.
	Path string `json:"path"`
	Why  Why    `json:"why"`
	// Reason is the sentence a maintainer reads.
	Reason string `json:"reason"`
}

type Why string

const (
	WhyObsolete Why = "obsolete" // marked Obsolete upstream; the catalog serves its Cardigann definition
	WhyDead     Why = "dead"     // the site shut down, is unusable or bans automation
	WhyGeneric  Why = "generic"  // an engine for many sites, such as Cardigann or Torznab
	WhyUsenet   Why = "usenet"   // SlipStream has no usenet
	WhyUnused   Why = "unused"   // a base no live site builds on
)

var whys = []Why{WhyObsolete, WhyDead, WhyGeneric, WhyUsenet, WhyUnused}

// Skips reports whether file, a path from Prowlarr's root, belongs to a
// skipped indexer.
func (s Skipped) Skips(file string) bool {
	return slices.ContainsFunc(s.Sites, func(x SkippedSite) bool {
		return x.Path == file || strings.HasSuffix(x.Path, "/") && strings.HasPrefix(file, x.Path)
	})
}

// ReadSkipped reads and checks the repository's skipped.json.
func ReadSkipped(root string) (Skipped, error) {
	raw, err := readFile(filepath.Join(root, filepath.FromSlash(SkippedPath)))
	if err != nil {
		return Skipped{}, err
	}
	var s Skipped
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return Skipped{}, fmt.Errorf("read %s: %w", SkippedPath, err)
	}
	if !shaPattern.MatchString(s.Commit) {
		return Skipped{}, fmt.Errorf("%s: commit %q isn't a full Prowlarr commit hash", SkippedPath, s.Commit)
	}
	for i, x := range s.Sites {
		switch {
		case i > 0 && s.Sites[i-1].Name >= x.Name:
			return Skipped{}, fmt.Errorf("%s: %s comes after %s, and the sites are sorted by name", SkippedPath, x.Name, s.Sites[i-1].Name)
		case x.Path == "" || x.Reason == "":
			return Skipped{}, fmt.Errorf("%s: %s needs a path and a reason", SkippedPath, x.Name)
		case !slices.Contains(whys, x.Why):
			return Skipped{}, fmt.Errorf("%s: %s's why %q isn't one of %v", SkippedPath, x.Name, x.Why, whys)
		}
	}
	return s, nil
}

// Marshal is the skipped.json bytes for s.
func (s Skipped) Marshal() []byte {
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("provenance: marshal skipped.json: %v", err))
	}
	return append(out, '\n')
}

// WriteSkipped writes s as the repository's skipped.json.
func WriteSkipped(root string, s Skipped) error {
	return writeFile(filepath.Join(root, filepath.FromSlash(SkippedPath)), s.Marshal())
}

func checkSkipped(root string, prowlarr *Tree) ([]Problem, error) {
	s, err := ReadSkipped(root)
	if err != nil {
		return []Problem{{SkippedPath, err.Error()}}, nil //nolint:nilerr // a skipped.json that doesn't parse is a problem Check reports
	}
	if prowlarr == nil {
		return nil, nil
	}
	var problems []Problem
	for _, x := range s.Sites {
		files, err := prowlarr.Files(context.Background(), s.Commit, x.Path)
		if err != nil {
			return nil, err
		}
		if len(files) == 0 {
			problems = append(problems, Problem{SkippedPath, fmt.Sprintf("%s's path %s isn't in Prowlarr at %s", x.Name, x.Path, s.Commit)})
		}
	}
	return problems, nil
}
