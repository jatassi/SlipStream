// Package golden runs the [TestCase] rows cmd/arr-sync copies from Sonarr
// and Radarr against SlipStream's reading of each name. Every upstream test
// method has one Go check, and each fixture file has two hand-kept
// siblings: <File>.pending.json lists the rows that don't pass yet, and
// <File>.overrides.json the rows where SlipStream reads a name differently
// from upstream on purpose, each with its reason.
package golden

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var updatePending = flag.Bool("update-pending", false, "rewrite each fixture file's pending list to the rows that fail")

// Check is one upstream test method's assertion, made on what SlipStream
// reads. It returns nil when the row passes.
type Check func(Row) error

// Fixture is one fixture file and a check for each of its methods.
type Fixture struct {
	// Path is the fixture JSON, relative to the test's package.
	Path   string
	Checks map[string]Check
	// Unported names, with the reason, each method whose assertion is on
	// something SlipStream doesn't read from a name.
	Unported map[string]string
}

type fixtureFile struct {
	Methods []method `json:"methods"`
}

type method struct {
	Name   string  `json:"name"`
	Params []param `json:"params"`
	Cases  []struct {
		Args []json.RawMessage `json:"args"`
	} `json:"cases"`
}

type param struct {
	Name    string          `json:"name"`
	Type    string          `json:"type"`
	Default json.RawMessage `json:"default"`
}

// row names one [TestCase] in the pending and overrides lists.
type row struct {
	Method string            `json:"method"`
	Args   []json.RawMessage `json:"args"`
}

type override struct {
	row
	// Ours are the arguments SlipStream's reading passes with.
	Ours []json.RawMessage `json:"ours"`
	// Reads is what SlipStream reads where the upstream method takes no
	// expected value to replace, such as a group upstream wants unknown.
	Reads  json.RawMessage `json:"reads,omitempty"`
	Reason string          `json:"reason"`
}

func (r row) key() string {
	b, err := json.Marshal(r.Args)
	if err != nil {
		panic(err)
	}
	return r.Method + " " + string(b)
}

// Run checks every row of f. A row fails the test when it fails and isn't
// pending, or passes and is. With -update-pending, it writes the pending
// list instead.
func Run(t testing.TB, f Fixture) {
	t.Helper()
	var file fixtureFile
	readJSON(t, f.Path, &file)
	r := run{t: t, f: f, pending: map[string]int{}, ours: map[string]override{}, used: map[string]bool{}}
	pendingPath, overridesPath := sibling(f.Path, "pending"), sibling(f.Path, "overrides")
	if !*updatePending {
		var rows []row
		readJSON(t, pendingPath, &rows)
		for _, p := range rows {
			r.pending[compact(t, p).key()]++
		}
	}
	var overrides []override
	readJSON(t, overridesPath, &overrides)
	for _, o := range overrides {
		o.row = compact(t, o.row)
		r.ours[o.key()] = o
	}
	methods := map[string]bool{}
	for _, m := range file.Methods {
		methods[m.Name] = true
		r.method(m)
	}
	r.unmatched(methods, pendingPath, overridesPath)
	if *updatePending {
		writePending(t, pendingPath, r.failing)
	}
	t.Logf("%s: %d rows pass, %d pending, %d in unported methods", filepath.Base(f.Path), r.passed, len(r.failing), r.unported)
}

type run struct {
	t       testing.TB
	f       Fixture
	pending map[string]int
	ours    map[string]override
	used    map[string]bool

	failing          []row
	passed, unported int
}

func (r *run) method(m method) {
	check, ok := r.f.Checks[m.Name]
	if _, skip := r.f.Unported[m.Name]; skip {
		if ok {
			r.t.Errorf("%s has a check and is listed as unported. Keep one.", m.Name)
		}
		r.unported += len(m.Cases)
		return
	}
	if !ok {
		r.t.Errorf("Upstream method %s in %s has no Go check. Add one to the harness.", m.Name, r.f.Path)
		return
	}
	for _, c := range m.Cases {
		upstream := compact(r.t, row{Method: m.Name, Args: c.Args})
		key := upstream.key()
		row := Row{method: m.Name, params: m.Params, args: upstream.Args}
		if o, ok := r.ours[key]; ok {
			row.args, row.reads = o.Ours, o.Reads
			r.used[key] = true
		}
		err := check(row)
		wasPending := r.pending[key] > 0
		r.pending[key]--
		switch {
		case err != nil:
			r.failing = append(r.failing, upstream)
			if !wasPending && !*updatePending {
				r.t.Errorf("%s: %v", key, err)
			}
		case wasPending:
			r.t.Errorf("%s passes now. Run make golden-pending.", key)
		default:
			r.passed++
		}
	}
}

// unmatched fails each check, unported method, pending row and override
// that names nothing in the fixture.
func (r *run) unmatched(methods map[string]bool, pendingPath, overridesPath string) {
	for name := range r.f.Checks {
		if !methods[name] {
			r.t.Errorf("%s has a check, and %s has no such method. Delete the check.", name, r.f.Path)
		}
	}
	for name := range r.f.Unported {
		if !methods[name] {
			r.t.Errorf("%s is listed as unported, and %s has no such method. Delete it from the list.", name, r.f.Path)
		}
	}
	for key, n := range r.pending {
		if n > 0 {
			r.t.Errorf("%s is pending in %s, and the fixture has no such row. Run make golden-pending.", key, pendingPath)
		}
	}
	for key := range r.ours {
		if !r.used[key] {
			r.t.Errorf("%s is overridden in %s, and the fixture has no such row. Delete the override.", key, overridesPath)
		}
	}
}

func sibling(path, kind string) string {
	return strings.TrimSuffix(path, ".json") + "." + kind + ".json"
}

func readJSON(t testing.TB, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path) //nolint:gosec // a fixture file the test names
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
}

func compact(t testing.TB, r row) row {
	t.Helper()
	args := make([]json.RawMessage, len(r.Args))
	for i, a := range r.Args {
		var b bytes.Buffer
		if err := json.Compact(&b, a); err != nil {
			t.Fatalf("%s: %v", r.Method, err)
		}
		args[i] = b.Bytes()
	}
	return row{Method: r.Method, Args: args}
}

// writePending writes one row per line, in fixture order, so a change to
// the list diffs by row.
func writePending(t testing.TB, path string, rows []row) {
	t.Helper()
	var b bytes.Buffer
	b.WriteString("[")
	for i, r := range rows {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("\n  ")
		enc := json.NewEncoder(&b)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(r); err != nil {
			t.Fatal(err)
		}
		b.Truncate(b.Len() - 1)
	}
	b.WriteString("\n]\n")
	if err := os.WriteFile(path, b.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

// Row is one [TestCase]'s arguments, read by parameter name. A parameter
// the row leaves out reads as its default.
type Row struct {
	method string
	params []param
	args   []json.RawMessage
	reads  json.RawMessage
}

// Reads decodes into v what an override says SlipStream reads instead of
// the method's fixed expectation, and is false when no override says.
func (r Row) Reads(v any) bool {
	if r.reads == nil {
		return false
	}
	if err := json.Unmarshal(r.reads, v); err != nil {
		panic(fmt.Sprintf("%s's override reads: %v", r.method, err))
	}
	return true
}

func (r Row) arg(name string) json.RawMessage {
	for i, p := range r.params {
		switch {
		case p.Name != name:
			continue
		case strings.HasPrefix(p.Type, "params "):
			rest, err := json.Marshal(r.args[i:])
			if err != nil {
				panic(err)
			}
			return rest
		case i < len(r.args):
			return r.args[i]
		}
		return p.Default
	}
	panic(fmt.Sprintf("%s has no parameter %s", r.method, name))
}

func (r Row) decode(name string, v any) {
	if err := json.Unmarshal(r.arg(name), v); err != nil {
		panic(fmt.Sprintf("%s's %s: %v", r.method, name, err))
	}
}

// String is a string argument, "" for null. A path that upstream passes
// through AsOsAgnostic reads as written: release.File reads both
// separators on every OS.
func (r Row) String(name string) string {
	var call struct {
		Call string `json:"call"`
		On   string `json:"on"`
	}
	if json.Unmarshal(r.arg(name), &call) == nil && call.Call == "AsOsAgnostic" {
		return call.On
	}
	var s *string
	r.decode(name, &s)
	if s == nil {
		return ""
	}
	return *s
}

func (r Row) Int(name string) int {
	var v int
	r.decode(name, &v)
	return v
}

func (r Row) Float(name string) float64 {
	var v float64
	r.decode(name, &v)
	return v
}

func (r Row) Bool(name string) bool {
	var v bool
	r.decode(name, &v)
	return v
}

func (r Row) Strings(name string) []string {
	var v []string
	r.decode(name, &v)
	return v
}

func (r Row) Ints(name string) []int {
	var v []int
	r.decode(name, &v)
	return v
}

// Ref is an argument naming an upstream enum value, such as
// "Quality.WEBDL720p".
func (r Row) Ref(name string) string {
	var v struct {
		Ref string `json:"ref"`
	}
	r.decode(name, &v)
	return v.Ref
}

// Same is nil when got equals want, and otherwise names what, with both
// values as JSON.
func Same[T any](what string, got, want T) error {
	if reflect.DeepEqual(got, want) {
		return nil
	}
	return fmt.Errorf("%s reads as %s, want %s", what, show(got), show(want))
}

func show(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
