package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var (
	headerCommit = regexp.MustCompile(`(?:from|and) (.+?) ([0-9a-f]{40})`)
	vendoredPair = regexp.MustCompile(`(?m)^Source: (\S+)\nCommit: ([0-9a-f]{40})$`)
)

// drift lists every file the lever wrote that no longer matches pins.json
// or its own digest: a pin bumped without a rerun, or a hand edit. CI runs
// it through the package's tests, with no network.
func drift(root *os.Root, p pins) ([]string, error) {
	files, err := filesUnder(root, generatedRoots(p))
	if err != nil {
		return nil, err
	}
	pinned := pinnedCommits{pins: p, byName: map[string]string{}, byRepo: map[string]string{}}
	for _, u := range p.Upstreams {
		pinned.byName[u.Name], pinned.byRepo[repoURL(u.Repo)] = u.Commit, u.Commit
	}
	var problems []string
	for _, file := range files {
		b, err := root.ReadFile(filepath.FromSlash(file))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", file, err)
		}
		problems = append(problems, pinned.check(file, b)...)
	}
	return problems, nil
}

// generatedRoots are the files and directories whose drift matters: the
// tables, the fixtures, the provenance records and the notices.
func generatedRoots(p pins) []string {
	owners := []string{p.Trash.ReadersOwner}
	for _, pf := range p.Patterns {
		owners = append(owners, pf.Owner)
		for _, o := range pf.Fields {
			owners = append(owners, o)
		}
	}
	var roots []string
	for _, o := range owners {
		roots = append(roots, path.Join(o, "internal/upstream"))
	}
	for _, set := range p.Fixtures {
		roots = append(roots, path.Join(set.Owner, "testdata/upstream", set.From))
	}
	for o := range p.Ports {
		roots = append(roots, path.Join(o, "upstream.json"))
	}
	for key, u := range p.Upstreams {
		if u.Notice != "" {
			roots = append(roots, path.Join(noticesDir, key))
		}
	}
	roots = append(roots, p.Dialects.Out, path.Join(p.Trash.Out, "VENDORED.md"), path.Join(p.Fuzz.Out, "VENDORED.md"))
	slices.Sort(roots)
	return slices.Compact(roots)
}

type pinnedCommits struct {
	pins           pins
	byName, byRepo map[string]string
}

func (c pinnedCommits) check(file string, b []byte) []string {
	switch {
	case strings.HasSuffix(file, ".go"):
		return c.goFile(file, b)
	case path.Base(file) == "VENDORED.md":
		var problems []string
		for _, m := range vendoredPair.FindAllStringSubmatch(string(b), -1) {
			if c.byRepo[m[1]] != m[2] {
				problems = append(problems, fmt.Sprintf("%s names %s at %s, which isn't the pinned commit. Run make arr-sync.", file, m[1], m[2]))
			}
		}
		return problems
	case strings.HasSuffix(file, ".json") && !handKept(file):
		return c.jsonFile(file, b)
	}
	return nil
}

func (c pinnedCommits) goFile(file string, b []byte) []string {
	first, rest, _ := bytes.Cut(b, []byte("\n"))
	second, body, _ := bytes.Cut(rest, []byte("\n"))
	body, blank := bytes.CutPrefix(body, []byte("\n"))
	if !bytes.HasPrefix(first, []byte(generatedPrefix)) || !bytes.HasSuffix(first, []byte(". DO NOT EDIT.")) || !blank {
		return []string{file + " has no cmd/arr-sync header. Run make arr-sync."}
	}
	var problems []string
	sum := sha256.Sum256(body)
	if string(second) != digestPrefix+hex.EncodeToString(sum[:]) {
		problems = append(problems, file+" doesn't match its digest because it was edited by hand. Run make arr-sync.")
	}
	for _, m := range headerCommit.FindAllStringSubmatch(string(first), -1) {
		if c.byName[m[1]] != m[2] {
			problems = append(problems, fmt.Sprintf("%s was written from %s %s, which isn't the pinned commit. Run make arr-sync.", file, m[1], m[2]))
		}
	}
	return problems
}

func (c pinnedCommits) jsonFile(file string, b []byte) []string {
	var fixture struct{ Upstream, Commit string }
	if err := json.Unmarshal(b, &fixture); err != nil {
		return []string{fmt.Sprintf("%s isn't JSON: %v", file, err)}
	}
	if fixture.Commit != "" {
		if c.byName[fixture.Upstream] != fixture.Commit {
			return []string{fmt.Sprintf("%s was read from %s %s, which isn't the pinned commit. Run make arr-sync.", file, fixture.Upstream, fixture.Commit)}
		}
		return nil
	}
	var rec record
	if err := json.Unmarshal(b, &rec); err != nil {
		return []string{fmt.Sprintf("%s isn't an upstream record: %v", file, err)}
	}
	var problems []string
	for key, pr := range rec {
		if c.pins.Upstreams[key].Commit != pr.Commit {
			problems = append(problems, fmt.Sprintf("%s records %s at %s, which isn't the pinned commit. Run make arr-sync.", file, key, pr.Commit))
		}
	}
	return problems
}
