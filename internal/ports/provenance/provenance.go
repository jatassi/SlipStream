// Package provenance is the record of which Prowlarr code each part of
// SlipStream was ported from. Each unit (a port, a family, the port kit
// or an internal/prowlarr package) keeps an upstream.json beside its Go
// files. It maps each Go file to the Prowlarr files it came from at one
// commit, and each of those Go files opens with Header's three lines,
// which name the same files and commit and carry the GPL-3.0 notice.
// Check proves the two agree, and with a Prowlarr clone, that every
// named file exists at its commit. prowlarr-sync reads and bumps the
// same files, and a port's Entry.Upstream embeds its own.
package provenance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	portsDir    = "internal/ports"
	familiesDir = "internal/ports/internal"
	kitDir      = "internal/ports/port"
	sharedDir   = "internal/prowlarr"
	// ManifestName is the file each unit keeps beside its Go files.
	ManifestName = "upstream.json"
)

// notPorts are the directories under internal/ports that are plumbing,
// neither a site nor a family.
var notPorts = []string{"all", "internal", "port", "provenance"}

// Kind is what a unit mirrors upstream.
type Kind string

const (
	Port   Kind = "port"     // one site, internal/ports/<id>
	Family Kind = "family"   // a shared base, internal/ports/internal/<name>
	Kit    Kind = "kit"      // HttpIndexerBase and its friends, internal/ports/port
	Shared Kind = "prowlarr" // ParseUtil and the like, internal/prowlarr/<name>
)

// Manifest is one unit's upstream.json, the format port.ReadUpstream
// reads.
type Manifest struct {
	Commit string `json:"commit"`
	// Date is the commit's committer date, as 2006-01-02.
	Date string `json:"date"`
	// Family is the family a port builds on, "" for none.
	Family string `json:"family"`
	// Files maps each Go file, by name, to the Prowlarr files it was
	// ported from, by path from Prowlarr's root.
	Files map[string][]string `json:"files"`
}

// Unit is one thing with an upstream.json.
type Unit struct {
	// ID is "port:hdbits", "family:gazelle", "kit" or "prowlarr:parseutil".
	ID   string
	Kind Kind
	// Dir is the unit's directory from the repository root, with slashes.
	Dir string
	Manifest
}

// Upstream is every Prowlarr file the unit maps, sorted.
func (u Unit) Upstream() []string {
	var out []string
	for _, files := range u.Files {
		out = append(out, files...)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// GoFiles are the unit's Go files that map upstream file f, with their
// directory, sorted.
func (u Unit) GoFiles(f string) []string {
	var out []string
	for gofile, files := range u.Files {
		if slices.Contains(files, f) {
			out = append(out, path.Join(u.Dir, gofile))
		}
	}
	slices.Sort(out)
	return out
}

// Problem is one way the manifests and the code disagree.
type Problem struct {
	Path    string
	Message string
}

func (p Problem) String() string { return p.Path + ": " + p.Message }

var (
	shaPattern  = regexp.MustCompile(`^[0-9a-f]{40}$`)
	headerFirst = "// Ported from Prowlarr's "
	headerThird = "// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go."
)

// Header is the three lines a ported Go file opens with.
func Header(commit string, files []string) string {
	return headerFirst + strings.Join(files, ", ") + "\n" +
		"// at " + commit + ". Copyright the Prowlarr\n" +
		headerThird + "\n"
}

// readHeader is the commit and files src's header names. ok is false
// when src has none. A file that starts like a header but isn't one is
// an error.
func readHeader(src []byte) (commit string, files []string, ok bool, err error) {
	if !bytes.HasPrefix(src, []byte(headerFirst)) {
		return "", nil, false, nil
	}
	lines := strings.SplitN(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n", 4)
	if len(lines) < 4 {
		return "", nil, false, errors.New("the header isn't three lines")
	}
	named := strings.TrimPrefix(lines[0], headerFirst)
	commit, found := strings.CutPrefix(lines[1], "// at ")
	commit, found2 := strings.CutSuffix(commit, ". Copyright the Prowlarr")
	if named == "" || !found || !found2 || !shaPattern.MatchString(commit) || lines[2] != headerThird {
		return "", nil, false, errors.New("the header isn't the template provenance.Header writes")
	}
	return commit, strings.Split(named, ", "), true, nil
}

// Units reads every unit in the repository at root. A port or family
// directory with no upstream.json is a problem, as is a manifest that
// doesn't parse.
func Units(root string) ([]Unit, []Problem, error) {
	var units []Unit
	var problems []Problem
	dirs, err := unitDirs(root)
	if err != nil {
		return nil, nil, err
	}
	for _, d := range dirs {
		raw, err := readFile(filepath.Join(root, filepath.FromSlash(d.Dir), ManifestName))
		switch {
		case errors.Is(err, fs.ErrNotExist) && d.needed:
			problems = append(problems, Problem{d.Dir, "has no " + ManifestName})
			continue
		case errors.Is(err, fs.ErrNotExist):
			continue
		case err != nil:
			return nil, nil, err
		}
		m, err := parseManifest(raw)
		if err != nil {
			problems = append(problems, Problem{path.Join(d.Dir, ManifestName), err.Error()})
			continue
		}
		d.Manifest = m
		units = append(units, d.Unit)
	}
	return units, problems, nil
}

type unitDir struct {
	Unit
	// needed is true for ports and families, which must have a manifest.
	needed bool
}

func unitDirs(root string) ([]unitDir, error) {
	var out []unitDir
	add := func(dir string, kind Kind, needed bool, id func(name string) string) error {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", dir, err)
		}
		for _, e := range entries {
			if !e.IsDir() || kind == Port && slices.Contains(notPorts, e.Name()) {
				continue
			}
			sub := path.Join(dir, e.Name())
			hasGo, err := holdsGo(filepath.Join(root, filepath.FromSlash(sub)))
			if err != nil {
				return err
			}
			if hasGo {
				out = append(out, unitDir{Unit{ID: id(e.Name()), Kind: kind, Dir: sub}, needed})
			}
		}
		return nil
	}
	named := func(kind Kind) func(string) string { return func(n string) string { return string(kind) + ":" + n } }
	if err := add(portsDir, Port, true, named(Port)); err != nil {
		return nil, err
	}
	if err := add(familiesDir, Family, true, named(Family)); err != nil {
		return nil, err
	}
	if err := add(sharedDir, Shared, false, named(Shared)); err != nil {
		return nil, err
	}
	out = append(out, unitDir{Unit{ID: string(Kit), Kind: Kit, Dir: kitDir}, false})
	slices.SortFunc(out, func(a, b unitDir) int { return strings.Compare(a.ID, b.ID) })
	return out, nil
}

func holdsGo(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", dir, err)
	}
	return slices.ContainsFunc(entries, func(e os.DirEntry) bool {
		return !e.IsDir() && strings.HasSuffix(e.Name(), ".go")
	}), nil
}

func parseManifest(raw []byte) (Manifest, error) {
	var m Manifest
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("read the manifest: %w", err)
	}
	if !shaPattern.MatchString(m.Commit) {
		return Manifest{}, fmt.Errorf("commit %q isn't a full Prowlarr commit hash", m.Commit)
	}
	if _, err := time.Parse(time.DateOnly, m.Date); err != nil {
		return Manifest{}, fmt.Errorf("date %q isn't a date as 2006-01-02", m.Date)
	}
	if len(m.Files) == 0 {
		return Manifest{}, errors.New("files maps no Go file")
	}
	for gofile, files := range m.Files {
		if path.Base(gofile) != gofile || !strings.HasSuffix(gofile, ".go") {
			return Manifest{}, fmt.Errorf("files names %q, which isn't a Go file in this directory", gofile)
		}
		if len(files) == 0 || !slices.IsSorted(files) || len(slices.Compact(slices.Clone(files))) != len(files) {
			return Manifest{}, fmt.Errorf("files maps %s to %q, which isn't a sorted list of distinct Prowlarr files", gofile, files)
		}
	}
	return m, nil
}

// Marshal is the upstream.json bytes for m, the one layout bump and new
// write.
func (m Manifest) Marshal() []byte {
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("provenance: marshal a manifest: %v", err))
	}
	return append(out, '\n')
}

// Check is every way the manifests under root disagree with the code,
// and, when prowlarr isn't nil, with Prowlarr's history.
func Check(root string, prowlarr *Tree) ([]Problem, error) {
	units, problems, err := Units(root)
	if err != nil {
		return nil, err
	}
	families := map[string]bool{}
	for _, u := range units {
		if u.Kind == Family {
			families[strings.TrimPrefix(u.ID, "family:")] = true
		}
	}
	for _, u := range units {
		if u.Family != "" && (u.Kind != Port || !families[u.Family]) {
			problems = append(problems, Problem{path.Join(u.Dir, ManifestName), fmt.Sprintf("family %q isn't a family under %s", u.Family, familiesDir)})
		}
		found, err := checkHeaders(root, u)
		if err != nil {
			return nil, err
		}
		problems = append(problems, found...)
		if prowlarr == nil {
			continue
		}
		missing, err := prowlarr.Missing(u.Commit, u.Upstream())
		if err != nil {
			problems = append(problems, Problem{path.Join(u.Dir, ManifestName), err.Error()})
			continue
		}
		for _, f := range missing {
			problems = append(problems, Problem{path.Join(u.Dir, ManifestName), fmt.Sprintf("%s isn't in Prowlarr at %s", f, u.Commit)})
		}
	}
	skipped, err := checkSkipped(root, prowlarr)
	if err != nil {
		return nil, err
	}
	return append(problems, skipped...), nil
}

// checkHeaders compares each Go file's header with the manifest. Every
// file the manifest maps has the header naming its files and the
// manifest's commit, and no other file has one, so the headers together
// name exactly the manifest's files.
func checkHeaders(root string, u Unit) ([]Problem, error) {
	dir := filepath.Join(root, filepath.FromSlash(u.Dir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", u.Dir, err)
	}
	var problems []Problem
	seen := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		seen[e.Name()] = true
		src, err := readFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		where := path.Join(u.Dir, e.Name())
		commit, files, ok, err := readHeader(src)
		want, mapped := u.Files[e.Name()]
		switch {
		case err != nil:
			problems = append(problems, Problem{where, err.Error()})
		case !ok && mapped:
			problems = append(problems, Problem{where, "has no Prowlarr header, though " + ManifestName + " maps it"})
		case ok && !mapped:
			problems = append(problems, Problem{where, "has a Prowlarr header, though " + ManifestName + " maps no files to it"})
		case ok && commit != u.Commit:
			problems = append(problems, Problem{where, fmt.Sprintf("its header names commit %s, and %s names %s", commit, ManifestName, u.Commit)})
		case ok && !slices.Equal(files, want):
			problems = append(problems, Problem{where, fmt.Sprintf("its header names %s, and %s maps it to %s", strings.Join(files, ", "), ManifestName, strings.Join(want, ", "))})
		}
	}
	for _, gofile := range sortedKeys(u.Files) {
		if !seen[gofile] {
			problems = append(problems, Problem{path.Join(u.Dir, ManifestName), "maps " + gofile + ", which doesn't exist"})
		}
	}
	return problems, nil
}

// Bump moves the unit in dir to commit, dated date, rewriting its
// upstream.json and the header of every Go file it maps. A mapped file
// with no header gets one.
func Bump(root, dir, commit, date string) error {
	units, problems, err := Units(root)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(units, func(u Unit) bool { return u.Dir == dir })
	if i < 0 {
		for _, p := range problems {
			if strings.HasPrefix(p.Path, dir+"/") || p.Path == dir {
				return errors.New(p.String())
			}
		}
		return fmt.Errorf("%s has no %s", dir, ManifestName)
	}
	u := units[i]
	u.Commit, u.Date = commit, date
	if _, err := parseManifest(u.Marshal()); err != nil {
		return err
	}
	return Write(root, u)
}

// Write writes u's upstream.json and puts the header on each Go file it
// maps, replacing the one that is there.
func Write(root string, u Unit) error {
	dir := filepath.Join(root, filepath.FromSlash(u.Dir))
	if err := writeFile(filepath.Join(dir, ManifestName), u.Marshal()); err != nil {
		return err
	}
	for _, gofile := range sortedKeys(u.Files) {
		name := filepath.Join(dir, gofile)
		src, err := readFile(name)
		if err != nil {
			return err
		}
		if _, _, ok, _ := readHeader(src); ok {
			src = []byte(strings.SplitN(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n", 4)[3])
		} else {
			src = append([]byte("\n"), src...)
		}
		if err := writeFile(name, append([]byte(Header(u.Commit, u.Files[gofile])), src...)); err != nil {
			return err
		}
	}
	return nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func readFile(name string) ([]byte, error) {
	b, err := os.ReadFile(name) //nolint:gosec // G304: a file in the repository the caller names
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	return b, nil
}

func writeFile(name string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil { //nolint:gosec // G301: a source directory in the repository
		return fmt.Errorf("make %s: %w", filepath.Dir(name), err)
	}
	if err := os.WriteFile(name, b, 0o644); err != nil { //nolint:gosec // G306: a source file in the repository
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}
