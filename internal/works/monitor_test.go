package works_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// box is one outermost level of a generated tree: its position, whether
// it is an aside or withdrawn, and its things, each written as key plus
// any of r (released), f (filled) and w (withdrawn), such as "s1e2rf".
type box struct {
	position  int
	aside     bool
	withdrawn bool
	things    []string
}

// grow builds a Work of boxes, its asides as monitorAsides says, and the
// facts its things carry.
func grow(monitorAsides bool, boxes ...box) (works.Tree, works.Facts) {
	t := works.Tree{Summary: works.Summary{ID: works.WorkID{UUID: storage.NewUUID()}, MonitorAsides: monitorAsides}}
	facts := works.Facts{}
	for _, b := range boxes {
		l := works.Level{ID: works.LevelID{UUID: storage.NewUUID()}, Depth: 1, Key: works.Key(fmt.Sprint("b", b.position)), Position: b.position, Aside: b.aside}
		if b.withdrawn {
			l.WithdrawnAt = storage.Now()
		}
		t.Levels = append(t.Levels, l)
		for i, th := range b.things {
			key := strings.TrimRight(th, "rfw")
			it := works.Item{ID: works.ItemID{UUID: storage.NewUUID()}, LevelID: l.ID, Key: works.Key(key), Position: i + 1, Aside: b.aside}
			flags := th[len(key):]
			if strings.Contains(flags, "w") {
				it.WithdrawnAt = storage.Now()
			}
			facts[it.ID] = works.ItemFacts{Released: strings.Contains(flags, "r"), Filled: strings.Contains(flags, "f")}
			t.Items = append(t.Items, it)
		}
	}
	return t, facts
}

func keysOf(t works.Tree, on map[works.ItemID]bool) string {
	var keys []string
	for _, it := range t.Items {
		if on[it.ID] {
			keys = append(keys, string(it.Key))
		}
	}
	slices.Sort(keys)
	return strings.Join(keys, " ")
}

func TestPresetTable(t *testing.T) {
	show := []box{
		{position: 0, aside: true, things: []string{"x1rf", "x2r", "x3"}},
		{position: 2, things: []string{"a1rf", "a2r", "a3rw"}},
		{position: 3, things: []string{"b1rf", "b2r", "b3"}},
		{position: 4, things: []string{"c1", "c2"}},
		{position: 5, withdrawn: true, things: []string{"d1rfw"}},
	}
	unaired := []box{
		{position: 1, things: []string{"a1", "a2"}},
		{position: 2, things: []string{"b1"}},
	}
	flat := []box{}
	for _, c := range []struct {
		name   string
		boxes  []box
		preset works.Preset
		want   string
	}{
		{"All is every live Item outside asides", show, works.PresetAll, "a1 a2 b1 b2 b3 c1 c2"},
		{"Future is what isn't out, dateless ones included", show, works.PresetFuture, "b3 c1 c2"},
		{"Missing is what has no file, plus what isn't out", show, works.PresetMissing, "a2 b2 b3 c1 c2"},
		{"Existing is what has a file", show, works.PresetExisting, "a1 b1"},
		{"First is the first main level, whatever its number", show, works.PresetFirst, "a1 a2"},
		{"Latest is the last main level with something out", show, works.PresetLatest, "b1 b2 b3"},
		{"None is nothing", show, works.PresetNone, ""},
		{"Latest with nothing out is the first main level", unaired, works.PresetLatest, "a1 a2"},
		{"Future with nothing out is everything", unaired, works.PresetFuture, "a1 a2 b1"},
		{"First of a Work with no main level is nothing", flat, works.PresetFirst, ""},
	} {
		tree, facts := grow(false, c.boxes...)
		if got := keysOf(tree, works.Monitors(tree, facts, works.Choice{Preset: c.preset})); got != c.want {
			t.Errorf("%s: monitors %q, want %q", c.name, got, c.want)
		}
		with := c.want + " x1 x2 x3"
		if len(c.boxes) > 0 && c.boxes[0].aside {
			if got := keysOf(tree, works.Monitors(tree, facts, works.Choice{Preset: c.preset, Asides: new(true)})); got != strings.TrimSpace(sorted(with)) {
				t.Errorf("%s with Include: monitors %q, want %q", c.name, got, sorted(with))
			}
		}
	}
}

func sorted(keys string) string {
	f := strings.Fields(keys)
	slices.Sort(f)
	return strings.Join(f, " ")
}

func TestAChoiceWithoutIncludeKeepsTheWorksOwnAsides(t *testing.T) {
	boxes := []box{{position: 0, aside: true, things: []string{"x1r"}}, {position: 1, things: []string{"a1r"}}}
	for _, own := range []bool{true, false} {
		tree, facts := grow(own, boxes...)
		want := map[bool]string{true: "a1 x1", false: "a1"}[own]
		if got := keysOf(tree, works.Monitors(tree, facts, works.Choice{Preset: works.PresetAll})); got != want {
			t.Errorf("All on a Work whose asides are %t monitors %q, want %q", own, got, want)
		}
		if got := keysOf(tree, works.Monitors(tree, facts, works.Choice{Preset: works.PresetAll, Asides: new(false)})); got != "a1" {
			t.Errorf("All with Include off on a Work whose asides are %t monitors %q, want a1", own, got)
		}
	}
}

func TestAWorkWithoutLevelsIsOneMainLevel(t *testing.T) {
	tree := works.Tree{Items: []works.Item{{ID: works.ItemID{UUID: storage.NewUUID()}, Key: "m1"}}}
	released := works.Facts{tree.Items[0].ID: {Released: true}}
	for p, want := range map[works.Preset]string{works.PresetFirst: "m1", works.PresetLatest: "m1", works.PresetFuture: "", works.PresetMissing: "m1"} {
		if got := keysOf(tree, works.Monitors(tree, released, works.Choice{Preset: p})); got != want {
			t.Errorf("%s on one released Item monitors %q, want %q", p, got, want)
		}
	}
}

func TestCountPresetsIsEachPresetsSizeWithoutAsides(t *testing.T) {
	tree, facts := grow(true,
		box{position: 0, aside: true, things: []string{"x1", "x2w"}},
		box{position: 1, things: []string{"a1rf", "a2r"}},
		box{position: 2, things: []string{"b1"}},
	)
	got := works.CountPresets(tree, facts)
	if got.Asides != 1 {
		t.Errorf("Include adds %d Items, want the one live aside", got.Asides)
	}
	var line []string
	for _, c := range got.Choices {
		line = append(line, fmt.Sprintf("%s=%d/%t", c.Preset, c.Monitors, c.MonitorsNew))
	}
	if want := "all=3/true future=1/true missing=2/true existing=1/false first=2/false latest=2/true none=0/false"; strings.Join(line, " ") != want {
		t.Errorf("counts are %s, want %s", strings.Join(line, " "), want)
	}
}
