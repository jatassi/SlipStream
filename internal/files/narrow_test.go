package files_test

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// matching is format f matching pattern instead.
func matching(f formats.Format, pattern string) formats.Format {
	value, err := json.Marshal(map[string]string{"value": pattern})
	if err != nil {
		panic(err)
	}
	f.Conditions = []formats.Condition{{Name: f.Name, Kind: formats.ReleaseTitle, Fields: value}}
	return f
}

// byGroup is format f matching the release group group instead.
func byGroup(f formats.Format, group string) formats.Format {
	value, err := json.Marshal(map[string]string{"value": "^" + group + "$"})
	if err != nil {
		panic(err)
	}
	f.Conditions = []formats.Condition{{Name: f.Name, Kind: formats.ReleaseGroup, Fields: value}}
	return f
}

// saved saves c and answers its preview's Impact and the Works the save
// published.
func (l *library) saved(t *testing.T, svc *files.Files, c targets.Change) (impact files.Impact, sent []string) {
	t.Helper()
	impact = l.change(t, svc, c, files.Mode{Unwanted: files.Keep, Preview: true})
	l.change(t, svc, c, files.Mode{Unwanted: files.Keep})
	sent = l.published.events[len(l.published.events)-1].WorkIDs
	slices.Sort(sent)
	return impact, sent
}

func ids(ws ...works.Tree) []string {
	out := make([]string, len(ws))
	for i, w := range ws {
		out[i] = w.ID.String()
	}
	slices.Sort(out)
	return out
}

func TestAFormatEditsImpactIsTheWorksHoldingAFileWhoseScoreItMoves(t *testing.T) {
	l := open(t)
	svc := l.files(files.NoGrabs{})
	hdr, remux := titled("HDR"), titled("Remux")
	l.change(t, svc, targets.AddFormats{Dialect: formats.SlipStream, Add: []formats.Format{hdr, remux}}, files.Mode{Unwanted: files.Keep})
	web := []release.Quality{release.WEBDL1080p, release.WEBRip1080p}
	scored, plain := l.target(t, "Scored", web), l.target(t, "Plain", web)
	spec := targets.Spec{Name: "Scored", Tiers: [][]release.Quality{web}, Cutoff: release.WEBDL1080p, Scores: targets.Scores{Formats: map[formats.ID]int{hdr.ID: 10}, Increment: 1}}
	l.change(t, svc, targets.UpdateTarget{ID: scored.ID, Spec: spec}, files.Mode{Unwanted: files.Keep})
	scoring, unscored := l.profile(t, "Scoring", scored), l.profile(t, "Plain", plain)
	withHDR, withDV := l.work(t, "w1", true, &scoring), l.work(t, "w2", true, &scoring)
	unscoredDV := l.work(t, "w3", true, &unscored)
	l.grabbed(t, svc, withHDR, 0, release.WEBDL1080p, "Scarf.1080p.WEB-DL.HDR")
	l.grabbed(t, svc, withDV, 0, release.WEBDL1080p, "Scarf.1080p.WEB-DL.DV")
	l.grabbed(t, svc, unscoredDV, 0, release.WEBDL1080p, "Scarf.1080p.WEB-DL.DV")

	for _, c := range []struct {
		name   string
		change targets.Change
		works  []string
		scores map[works.WorkID]string
	}{
		{"an edit to a format no Target scores", targets.UpdateFormat{ID: remux.ID, Format: matching(remux, `\bDV\b`)}, nil, nil},
		{"replacing a format no Target scores", targets.AddFormats{Dialect: formats.SlipStream, Replace: map[formats.ID]formats.Format{remux.ID: matching(remux, `\bHDR\b`)}}, nil, nil},
		{"an edit to a scored format no held file's match of flips", targets.UpdateFormat{ID: hdr.ID, Format: matching(hdr, `\bHDR\b|\bHDR10\b`)}, nil, nil},
		{"an edit to a scored format that a held file now matches", targets.UpdateFormat{ID: hdr.ID, Format: matching(hdr, `\b(HDR|DV)\b`)}, ids(withDV), map[works.WorkID]string{
			withHDR.ID: "Scarf.1080p.WEB-DL.HDR at 10 met wanted=false", withDV.ID: "Scarf.1080p.WEB-DL.DV at 10 met wanted=false",
		}},
		{"a score change", targets.UpdateTarget{ID: scored.ID, Spec: withScore(spec, hdr.ID, 4)}, ids(withHDR, withDV), map[works.WorkID]string{
			withHDR.ID: "Scarf.1080p.WEB-DL.HDR at 4 met wanted=false", withDV.ID: "Scarf.1080p.WEB-DL.DV at 4 met wanted=false",
		}},
		{"replacing a scored format", targets.AddFormats{Dialect: formats.SlipStream, Replace: map[formats.ID]formats.Format{hdr.ID: matching(hdr, `\bHDR\b`)}}, ids(withDV), map[works.WorkID]string{
			withHDR.ID: "Scarf.1080p.WEB-DL.HDR at 4 met wanted=false", withDV.ID: "Scarf.1080p.WEB-DL.DV at 0 met wanted=false",
		}},
		{"deleting a scored format", targets.DeleteFormat{ID: hdr.ID}, ids(withHDR), map[works.WorkID]string{
			withHDR.ID: "Scarf.1080p.WEB-DL.HDR at 0 met wanted=false", withDV.ID: "Scarf.1080p.WEB-DL.DV at 0 met wanted=false",
		}},
	} {
		impact, sent := l.saved(t, svc, c.change)
		if impact.Works != len(c.works) || !slices.Equal(sent, c.works) {
			t.Errorf("%s: the Impact counts %d Works and the save names %v, want %v", c.name, impact.Works, sent, c.works)
		}
		for w, want := range c.scores {
			if got := held(t, l, svc, w); got[0] != want {
				t.Errorf("%s: %v's first cell is %q, want %q", c.name, w, got[0], want)
			}
		}
	}
}

func TestATargetEditBeyondItsScoresTouchesEveryWorkWantingIt(t *testing.T) {
	l := open(t)
	svc := l.files(files.NoGrabs{})
	hdr := titled("HDR")
	l.change(t, svc, targets.AddFormats{Dialect: formats.SlipStream, Add: []formats.Format{hdr}}, files.Mode{Unwanted: files.Keep})
	web := []release.Quality{release.WEBDL1080p, release.WEBRip1080p}
	scored := l.target(t, "Scored", web)
	scoring := l.profile(t, "Scoring", scored)
	withHDR, plain := l.work(t, "w1", true, &scoring), l.work(t, "w2", true, &scoring)
	l.grabbed(t, svc, withHDR, 0, release.WEBDL1080p, "Scarf.1080p.WEB-DL.HDR")
	l.grabbed(t, svc, plain, 0, release.WEBDL1080p, "Scarf.1080p.WEB-DL")
	spec := targets.Spec{Name: "Renamed", Tiers: [][]release.Quality{web}, Cutoff: release.WEBDL1080p, Scores: targets.Scores{Formats: map[formats.ID]int{hdr.ID: 10}, Increment: 1}}
	if _, sent := l.saved(t, svc, targets.UpdateTarget{ID: scored.ID, Spec: spec}); !slices.Equal(sent, ids(withHDR, plain)) {
		t.Errorf("renaming a Target as it scores a format: the save names %v, want both Works", sent)
	}
	if impact, sent := l.saved(t, svc, targets.UpdateTarget{ID: scored.ID, Spec: spec}); impact.Works != 0 || len(sent) != 0 {
		t.Errorf("saving a Target unchanged: the Impact counts %d Works and the save names %v, want none", impact.Works, sent)
	}
}

func withScore(s targets.Spec, id formats.ID, score int) targets.Spec {
	s.Scores.Formats = map[formats.ID]int{id: score}
	return s
}

// narrowed is four Works of a library with a format for each of words:
// two want from two Targets, one from one of them, and one from the
// type's default Profile.
type narrowed struct {
	*library
	svc     *files.Files
	trees   []works.Tree
	specs   []targets.Spec
	targets []targets.Target
	formats []formats.Format
}

func newNarrowed(t *testing.T) *narrowed {
	t.Helper()
	n := &narrowed{library: open(t), specs: []targets.Spec{
		{Scores: targets.Scores{Increment: 1}, Name: "Disc", Tiers: [][]release.Quality{{release.Bluray1080p}, {release.WEBDL1080p, release.WEBRip1080p}}, Cutoff: release.Bluray1080p},
		{Scores: targets.Scores{Increment: 1}, Name: "Web", Tiers: [][]release.Quality{{release.WEBDL1080p, release.WEBRip1080p}}, Cutoff: release.WEBDL1080p},
	}}
	for _, spec := range n.specs {
		n.targets = append(n.targets, n.target(t, spec.Name, spec.Tiers...))
	}
	both, web := n.profile(t, "Both", n.targets...), n.profile(t, "Web", n.targets[1])
	n.svc = n.files(files.NoGrabs{})
	n.trees = []works.Tree{n.work(t, "w1", true, &both), n.work(t, "w2", true, &both), n.work(t, "w3", true, &web), n.work(t, "w4", true, nil)}
	for _, w := range words {
		n.formats = append(n.formats, titled(w))
	}
	n.change(t, n.svc, targets.AddFormats{Dialect: formats.SlipStream, Add: n.formats}, files.Mode{Unwanted: files.Keep})
	return n
}

// step imports a file of a random name, or makes a random Change that
// moves scores, and says which it did with the Impact its preview gave.
func (n *narrowed) step(t *testing.T, r *rand.Rand) (string, *files.Impact) {
	t.Helper()
	at := r.IntN(len(n.targets))
	spec := &n.specs[at]
	f := &n.formats[r.IntN(len(n.formats))]
	var (
		c   targets.Change
		did string
	)
	switch r.IntN(6) {
	case 0:
		name := marking(r, "Scarf.1080p", append(slices.Clone(words), "PROPER")) + "-" + pick(r, groups)
		q := []release.Quality{release.WEBDL1080p, release.WEBRip1080p, release.Bluray1080p}[r.IntN(3)]
		tree := n.trees[r.IntN(len(n.trees))]
		n.grabbed(t, n.svc, tree, r.IntN(len(tree.Items)), q, name)
		return "imported " + name + " as " + q.String() + " into " + tree.Label, nil
	case 1:
		spec.Scores.Formats = map[formats.ID]int{}
		for _, f := range n.formats {
			if v := r.IntN(21) - 10; v != 0 && r.IntN(2) == 0 {
				spec.Scores.Formats[f.ID] = v
			}
		}
		c, did = targets.UpdateTarget{ID: n.targets[at].ID, Spec: *spec}, fmt.Sprintf("scored %s %v", spec.Name, spec.Scores.Formats)
	case 2:
		spec.Scores.UpgradeUntil, spec.Scores.Increment = r.IntN(10), 1+r.IntN(3)
		spec.Propers = []targets.Propers{targets.PreferAndUpgrade, targets.PreferNoUpgrade, targets.DontPrefer}[r.IntN(3)]
		c, did = targets.UpdateTarget{ID: n.targets[at].ID, Spec: *spec}, fmt.Sprintf("set %s's rules to %+v", spec.Name, spec.Scores)
	case 3:
		word := words[r.IntN(len(words))]
		c, did = targets.UpdateFormat{ID: f.ID, Format: matching(*f, `\b`+word+`\b`)}, "pointed "+f.Name+" at "+word
		if r.IntN(2) == 0 {
			group := pick(r, groups)
			c, did = targets.UpdateFormat{ID: f.ID, Format: byGroup(*f, group)}, "pointed "+f.Name+" at the group "+group
		}
	case 4:
		word := words[r.IntN(len(words))]
		c, did = targets.AddFormats{Dialect: formats.SlipStream, Replace: map[formats.ID]formats.Format{f.ID: matching(*f, `\b`+word+`\b`)}}, "replaced "+f.Name+" with one matching "+word
	default:
		gone := f.ID
		for i := range n.specs {
			delete(n.specs[i].Scores.Formats, gone)
		}
		impact := n.change(t, n.svc, targets.DeleteFormat{ID: gone}, files.Mode{Unwanted: files.Keep, Preview: true})
		n.change(t, n.svc, targets.DeleteFormat{ID: gone}, files.Mode{Unwanted: files.Keep})
		*f = titled(f.Name)
		n.change(t, n.svc, targets.AddFormats{Dialect: formats.SlipStream, Add: []formats.Format{*f}}, files.Mode{Unwanted: files.Keep})
		return "deleted " + f.Name + " and added it again", &impact
	}
	impact := n.change(t, n.svc, c, files.Mode{Unwanted: files.Keep, Preview: true})
	n.change(t, n.svc, c, files.Mode{Unwanted: files.Keep})
	return did, &impact
}

// cells is each cell of w as its file's path, its score and its state.
func (n *narrowed) cells(t *testing.T, w works.WorkID) []string {
	t.Helper()
	var out []string
	err := n.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		cells, err := n.svc.Cells(ctx, r, w)
		for _, c := range cells {
			path := "none"
			if c.File != nil {
				path = c.File.Path
			}
			out = append(out, fmt.Sprintf("%s at %d %s", path, c.Score, c.State))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

var groups = []string{"TEAM", "CREW"}

// scoredFromScratch fails when a Work's stored fill scores aren't those
// a refill from scratch gives.
func (n *narrowed) scoredFromScratch(t *testing.T, when string) {
	t.Helper()
	for _, w := range n.trees {
		if stored, scratch := n.scores(t, n.svc, w.ID); !maps.Equal(stored, scratch) {
			t.Fatalf("%s: %s's stored fill scores %v, a refill from scratch gives %v", when, w.Label, stored, scratch)
		}
	}
}

// everyCell is each Work's cells.
func (n *narrowed) everyCell(t *testing.T) [][]string {
	t.Helper()
	out := make([][]string, len(n.trees))
	for i, w := range n.trees {
		out[i] = n.cells(t, w.ID)
	}
	return out
}

func TestANarrowedChangeMissesNoWorkWhoseCellsItChanges(t *testing.T) {
	r := rand.New(rand.NewPCG(48, 2)) //nolint:gosec // seeded on purpose so failures replay
	for run := range 4 {
		n := newNarrowed(t)
		for step := range 40 {
			before := n.everyCell(t)
			did, impact := n.step(t, r)
			changed := 0
			for i, after := range n.everyCell(t) {
				if !slices.Equal(before[i], after) {
					changed++
				}
			}
			n.scoredFromScratch(t, fmt.Sprintf("run %d, step %d, after it %s", run, step, did))
			if impact != nil && impact.Works != changed {
				t.Fatalf("run %d, step %d, after it %s: the Impact counts %d Works, and %d Works' cells changed", run, step, did, impact.Works, changed)
			}
		}
	}
}
