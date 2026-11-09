package files_test

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// attic adds Works of two Items, Scarf and Mittens, each
// wanting Web.
type attic struct {
	l   *library
	web targets.Target
	p   targets.ProfileID
}

func newAttic(t *testing.T) *attic {
	l := open(t)
	web := l.target(t, "Web", []release.Quality{release.WEBDL1080p})
	return &attic{l: l, web: web, p: l.profile(t, "Web", web)}
}

func (s *attic) work(t *testing.T, key works.Key, label string) works.Tree {
	t.Helper()
	s.l.fx.Put(key, contenttypetest.Record{Tree: works.Fetched{Label: label, Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}, {Key: "c2", Label: "Mittens"}}},
	}}})
	tree, _, err := s.l.works.Add(t.Context(), admin, contenttypetest.ID, key, works.Adding{
		Monitor: true,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			if err := targets.Assign(ctx, tx, w, &s.p); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, w, nil)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

// found records a file of tree's item filling Web at p, holding p, as a
// library scan finds one, and answers its id.
func (s *attic) found(t *testing.T, tree works.Tree, item int, p string) files.ID {
	t.Helper()
	s.l.place(t, tree.Items[item].ID, tree.ID, p, release.WEBDL1080p, s.web.ID)
	s.l.put(t, p, p)
	return s.l.rows(t)[p]
}

// relocate moves the file at from to to, on disk and in its row.
func (s *attic) relocate(t *testing.T, from, to string) {
	t.Helper()
	at := func(p string) string { return filepath.Join(s.l.root.Path, filepath.FromSlash(p)) }
	if err := os.MkdirAll(filepath.Dir(at(to)), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(at(from), at(to)); err != nil {
		t.Fatal(err)
	}
	err := s.l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `UPDATE files_file SET path = ? WHERE path = ?`, to, from)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// confirm is each of ps confirmed as shown, leaving leave out.
func confirm(ps []files.WorkPlan, leave ...files.ID) []files.Confirm {
	out := make([]files.Confirm, len(ps))
	for i, p := range ps {
		out[i] = files.Confirm{Work: p.Work, Digest: p.Digest, Leave: leave}
	}
	return out
}

type route struct {
	from, to string
	blocked  files.Blocked
}

// routes is ps's rows by Work, without ids, groups or companions.
func routes(ps []files.WorkPlan) map[works.WorkID][]route {
	out := map[works.WorkID][]route{}
	for _, p := range ps {
		for _, r := range p.Rows {
			out[p.Work] = append(out[p.Work], route{r.From, r.To, r.Blocked})
		}
	}
	return out
}

func TestAConfirmedPlanRenamesItsFilesWithCompanionsAndLeavesTakenPaths(t *testing.T) {
	s := newAttic(t)
	svc := s.l.files(files.NoGrabs{})
	w1, w2 := s.work(t, "w1", "Shelf w1"), s.work(t, "w2", "Shelf w2")
	s.found(t, w1, 0, "Loft/a.mkv")
	s.l.put(t, "Loft/a.en.srt", "a's subtitles")
	s.found(t, w2, 0, "Loft/b.mkv")
	s.l.put(t, "Shelf w2/Shelf w2 - Scarf - Web.mkv", "a file of my own")

	ps, err := svc.Plan(t.Context(), admin, files.TypeScope(contenttypetest.ID), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[works.WorkID][]route{
		w1.ID: {{"Loft/a.mkv", "Shelf w1/Shelf w1 - Scarf - Web.mkv", files.Blocked{}}},
		w2.ID: {{"Loft/b.mkv", "Shelf w2/Shelf w2 - Scarf - Web.mkv", files.Blocked{Reason: files.Occupied}}},
	}
	if got := routes(ps); !maps.EqualFunc(got, want, func(a, b []route) bool { return len(a) == 1 && a[0] == b[0] }) {
		t.Fatalf("plan = %v, want w1's file moved and w2's blocked by the file at its new path: %v", got, want)
	}
	one, err := svc.PlanOf(t.Context(), admin, w1.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c := one.Rows[0].Companions; one.Digest != ps[0].Digest || len(c) != 1 || c[0] != "Loft/a.en.srt" {
		t.Fatalf("w1's own plan = %+v, want the library plan's digest %s and its subtitles beside it", one, ps[0].Digest)
	}

	done, err := svc.Rename(t.Context(), admin, confirm(ps))
	if err != nil {
		t.Fatal(err)
	}
	if done.Queued != 1 || len(done.Changed) != 0 {
		t.Fatalf("confirm = %+v, want w1's one file queued and no Work changed", done)
	}
	if n, err := svc.Moving(t.Context(), files.LibraryScope()); err != nil || n != 1 {
		t.Fatalf("moving in the library = %d, %v; want 1", n, err)
	}
	s.l.settle(t, svc)
	if got, want := s.l.onDisk(t), map[string]string{
		"Shelf w1/Shelf w1 - Scarf - Web.mkv":    "Loft/a.mkv",
		"Shelf w1/Shelf w1 - Scarf - Web.en.srt": "a's subtitles",
		"Loft/b.mkv":                             "Loft/b.mkv",
		"Shelf w2/Shelf w2 - Scarf - Web.mkv":    "a file of my own",
	}; !maps.Equal(got, want) {
		t.Fatalf("the root holds %v, want %v", got, want)
	}
}

func TestTwoWorksRenderingOnePathBlockEachOther(t *testing.T) {
	s := newAttic(t)
	svc := s.l.files(files.NoGrabs{})
	w1, w2, w3 := s.work(t, "w1", "Attic"), s.work(t, "w2", "attic"), s.work(t, "w3", "Den")
	s.found(t, w1, 0, "Loft/a.mkv")
	s.found(t, w2, 0, "Loft/b.mkv")
	s.found(t, w3, 0, "Loft/c.mkv")

	ps, err := svc.Plan(t.Context(), admin, files.LibraryScope(), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := routes(ps)
	if r := got[w1.ID]; len(r) != 1 || r[0].blocked != (files.Blocked{Reason: files.Clash, Other: w2.ID}) {
		t.Errorf("w1's rows = %v, want a clash naming w2", r)
	}
	if r := got[w2.ID]; len(r) != 1 || r[0].blocked != (files.Blocked{Reason: files.Clash, Other: w1.ID}) {
		t.Errorf("w2's rows = %v, want a clash naming w1", r)
	}
	if r := got[w3.ID]; len(r) != 1 || r[0] != (route{"Loft/c.mkv", "Den/Den - Scarf - Web.mkv", files.Blocked{}}) {
		t.Errorf("w3's rows = %v, want its file moved to Den", r)
	}
}

func TestAWorkThatChangedSinceItsPlanIsLeft(t *testing.T) {
	s := newAttic(t)
	svc := s.l.files(files.NoGrabs{})
	w1 := s.work(t, "w1", "Shelf w1")
	s.found(t, w1, 0, "Loft/a.mkv")
	ps, err := svc.Plan(t.Context(), admin, files.WorkScope(w1.ID), nil)
	if err != nil {
		t.Fatal(err)
	}
	s.relocate(t, "Loft/a.mkv", "Den/a.mkv")

	done, err := svc.Rename(t.Context(), admin, confirm(ps))
	if err != nil {
		t.Fatal(err)
	}
	if done.Queued != 0 || len(done.Changed) != 1 || done.Changed[0] != w1.ID {
		t.Fatalf("confirm = %+v, want nothing queued and w1 changed", done)
	}
	if got := s.l.onDisk(t); !maps.Equal(got, map[string]string{"Den/a.mkv": "Loft/a.mkv"}) {
		t.Fatalf("the root holds %v, want the file where it was moved to", got)
	}
}

func TestLeavingOneFileOfASwapLeavesBoth(t *testing.T) {
	s := newAttic(t)
	svc := s.l.files(files.NoGrabs{})
	w1 := s.work(t, "w1", "Shelf w1")
	scarf := s.found(t, w1, 0, "Shelf w1/Shelf w1 - Mittens - Web.mkv")
	s.found(t, w1, 1, "Shelf w1/Shelf w1 - Scarf - Web.mkv")
	ps, err := svc.Plan(t.Context(), admin, files.WorkScope(w1.ID), nil)
	if err != nil {
		t.Fatal(err)
	}
	if rows := ps[0].Rows; len(rows) != 2 || rows[0].Group == 0 || rows[0].Group != rows[1].Group {
		t.Fatalf("rows = %+v, want two rows in one group", rows)
	}

	done, err := svc.Rename(t.Context(), admin, confirm(ps, scarf))
	if err != nil || done.Queued != 0 || len(done.Changed) != 0 {
		t.Fatalf("confirm leaving the Scarf's file = %+v, %v; want nothing queued", done, err)
	}
	done, err = svc.Rename(t.Context(), admin, confirm(ps))
	if err != nil || done.Queued != 2 {
		t.Fatalf("confirm of both = %+v, %v; want both queued", done, err)
	}
	s.l.settle(t, svc)
	if got, want := s.l.onDisk(t), map[string]string{
		"Shelf w1/Shelf w1 - Scarf - Web.mkv":   "Shelf w1/Shelf w1 - Mittens - Web.mkv",
		"Shelf w1/Shelf w1 - Mittens - Web.mkv": "Shelf w1/Shelf w1 - Scarf - Web.mkv",
	}; !maps.Equal(got, want) {
		t.Fatalf("the root holds %v, want the two files traded", got)
	}
}

func TestAPlanOfADraftNamesFilesUnderTheDraft(t *testing.T) {
	s := newAttic(t)
	svc := s.l.files(files.NoGrabs{})
	w1 := s.work(t, "w1", "Shelf w1")
	s.found(t, w1, 0, "Loft/scarf, as found.mkv")
	draft := func(key, value string) *files.Draft {
		return &files.Draft{Type: contenttypetest.ID, Patch: setting.Patch{string(contenttypetest.ID) + ".naming." + key: json.RawMessage(value)}}
	}

	ps, err := svc.Plan(t.Context(), admin, files.LibraryScope(), draft("renameFiles", "false"))
	if err != nil {
		t.Fatal(err)
	}
	if got := routes(ps)[w1.ID]; len(got) != 1 || got[0] != (route{"Loft/scarf, as found.mkv", "Shelf w1/scarf, as found.mkv", files.Blocked{}}) {
		t.Fatalf("rows with renaming off = %v, want the file moved to its folder under its own name", got)
	}
	_, err = svc.Plan(t.Context(), admin, files.LibraryScope(), draft("file.standard", `"{Shelf} - {Thing}"`))
	if refused, ok := errors.AsType[*naming.RefusedError](err); !ok || len(refused.Problems) == 0 {
		t.Fatalf("plan of a template without {Target} = %v, want the save check's refusal", err)
	}
	if _, err := svc.Plan(t.Context(), admin, files.TypeScope("other"), nil); !errors.Is(err, works.ErrNoType) {
		t.Fatalf("plan of a type the library hasn't = %v, want ErrNoType", err)
	}
	unmatched, err := svc.Unmatched(t.Context(), admin)
	if err != nil || !maps.Equal(unmatched, map[works.TypeID]int{contenttypetest.ID: 1}) {
		t.Fatalf("unmatched = %v, %v; want the one found file", unmatched, err)
	}
}
