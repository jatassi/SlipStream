package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const pinsFile = "cmd/arr-sync/pins.json"

// pins is cmd/arr-sync/pins.json, the one hand-edited input: which
// upstream commits to read, and what the lever writes from each. A resync
// edits a commit here and runs make arr-sync.
type pins struct {
	Upstreams map[string]upstream `json:"upstreams"`
	// CoreWords renames the words of upstream identifiers the lever writes
	// into core packages, which name no Content Type.
	CoreWords map[string]string `json:"core_words"`
	Patterns  []patternFile     `json:"patterns"`
	Fixtures  []fixtureSet      `json:"fixtures"`
	Dialects  dialectSources    `json:"dialects"`
	Trash     trashSources      `json:"trash"`
	Fuzz      fuzzSources       `json:"fuzz"`
	// Ports maps each owner's hand-ported Go files to the upstream files
	// they port, as "upstream:path".
	Ports map[string]map[string][]string `json:"ports"`
}

type upstream struct {
	Name    string `json:"name"`
	Repo    string `json:"repo"`
	Commit  string `json:"commit"`
	License string `json:"license"`
	SPDX    string `json:"spdx"`
	// Paths are the sparse checkout: the only paths fetched.
	Paths []string `json:"paths"`
	// Notice, when set, is why SlipStream carries this upstream's code:
	// the lever writes it to internal/notices/<key>/VENDORED.md beside the
	// upstream LICENSE.
	Notice string `json:"notice,omitempty"`
}

type patternFile struct {
	From  string `json:"from"`
	File  string `json:"file"`
	Owner string `json:"owner"`
	// Fields moves single declarations to another owner.
	Fields map[string]string `json:"fields,omitempty"`
	// Skip names methods whose inline patterns the lever doesn't read.
	Skip []string `json:"skip,omitempty"`
	// Strings names fields whose initializers list strings, such as a
	// HashSet of extensions or a Dictionary's keys, which the lever writes
	// as []string tables.
	Strings []string `json:"strings,omitempty"`
	// Adds names methods whose if-statements add an enum value when a
	// Contains call or a match's named group says so, which the lever
	// writes as []Adds tables, one per kind of test.
	Adds []string `json:"adds,omitempty"`
	// IsoLanguages makes the lever read the file's IsoLanguage rows as an
	// []IsoLanguage table.
	IsoLanguages bool `json:"iso_languages,omitempty"`
}

type fixtureSet struct {
	From  string   `json:"from"`
	Dir   string   `json:"dir"`
	Owner string   `json:"owner"`
	Files []string `json:"files"`
}

type dialectSources struct {
	Out       string       `json:"out"`
	Enums     []enumSource `json:"enums"`
	Languages []fileSource `json:"languages"`
}

type enumSource struct {
	From string `json:"from"`
	File string `json:"file"`
	Enum string `json:"enum"`
}

type fileSource struct {
	From string `json:"from"`
	File string `json:"file"`
}

type trashSources struct {
	From string `json:"from"`
	// Root is the directory the snapshot and the readers' keys start at.
	Root string `json:"root"`
	// Snapshot are the directories copied verbatim to Out.
	Snapshot []string `json:"snapshot"`
	Out      string   `json:"out"`
	Titles   string   `json:"titles"`
	// Readers are the formats whose conditions the stream readers take,
	// written as a table into ReadersOwner.
	Readers      []string `json:"readers"`
	ReadersOwner string   `json:"readers_owner"`
}

type fuzzSources struct {
	Out     string       `json:"out"`
	Sources []fuzzSource `json:"sources"`
}

type fuzzSource struct {
	From string `json:"from"`
	// Reader is how titles are read from Files: yaml-keys, python-cases or
	// json-titles.
	Reader string   `json:"reader"`
	Files  []string `json:"files"`
}

func readPins(root *os.Root) (pins, error) {
	var p pins
	src, err := root.ReadFile(filepath.FromSlash(pinsFile))
	if err != nil {
		return p, fmt.Errorf("read the pins: %w", err)
	}
	if err := json.Unmarshal(src, &p); err != nil {
		return p, fmt.Errorf("%s: %w", pinsFile, err)
	}
	return p, p.check()
}

func (p pins) check() error {
	from := func(key string) error {
		if _, ok := p.Upstreams[key]; !ok {
			return fmt.Errorf("%s names upstream %q, which it doesn't pin", pinsFile, key)
		}
		return nil
	}
	var keys []string
	for _, f := range p.Patterns {
		keys = append(keys, f.From)
	}
	for _, f := range p.Fixtures {
		keys = append(keys, f.From)
	}
	for _, e := range p.Dialects.Enums {
		keys = append(keys, e.From)
	}
	for _, f := range p.Dialects.Languages {
		keys = append(keys, f.From)
	}
	for _, f := range p.Fuzz.Sources {
		keys = append(keys, f.From)
	}
	for _, files := range p.Ports {
		for _, refs := range files {
			for _, ref := range refs {
				key, _, _ := strings.Cut(ref, ":")
				keys = append(keys, key)
			}
		}
	}
	keys = append(keys, p.Trash.From)
	for _, k := range keys {
		if err := from(k); err != nil {
			return err
		}
	}
	return nil
}

// isCore is whether the lever writes into a core package, whose
// identifiers name no Content Type.
func isCore(owner string) bool {
	return !strings.HasPrefix(owner, "internal/contenttypes/")
}
