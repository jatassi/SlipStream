package works_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

var admin identity.Admin

type recorder[T any] struct {
	mu  sync.Mutex
	got []T
}

func (r *recorder[T]) Publish(_ context.Context, v T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = append(r.got, v)
}

func (r *recorder[T]) take() []T {
	r.mu.Lock()
	defer r.mu.Unlock()
	got := r.got
	r.got = nil
	return got
}

type harness struct {
	works   *works.Works
	lib     *storage.DB[storage.Library]
	fx      *contenttypetest.Fixture
	changed *recorder[works.Changed]
	types   *recorder[works.ContentTypesChanged]
	log     *bytes.Buffer
	art     *prefetched
}

type prefetched struct {
	mu    sync.Mutex
	wants []artwork.Want
}

func (p *prefetched) Prefetch(wants []artwork.Want) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.wants = append(p.wants, wants...)
}

func open(t testing.TB, lib *storage.DB[storage.Library], catalogs map[works.TypeID]works.Catalog, source works.Source) (*works.Works, *harness) {
	t.Helper()
	h := &harness{lib: lib, changed: &recorder[works.Changed]{}, types: &recorder[works.ContentTypesChanged]{}, log: &bytes.Buffer{}, art: &prefetched{}}
	w, err := works.Open(t.Context(), works.Config{
		Library:      lib,
		Catalogs:     catalogs,
		Source:       source,
		Artwork:      h.art,
		Timings:      works.Timings{Fetch: 5 * time.Second},
		Changed:      h.changed,
		TypesChanged: h.types,
		Log:          slog.New(slog.NewTextHandler(h.log, &slog.HandlerOptions{ReplaceAttr: dropTime})),
	})
	if err != nil {
		t.Fatal(err)
	}
	h.works = w
	return w, h
}

func dropTime(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		return slog.Attr{}
	}
	return a
}

func (h *harness) rows(t *testing.T) []works.Row {
	t.Helper()
	var rows []works.Row
	err := h.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		rows, err = h.works.RowsIn(ctx, r, works.Scope{})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	fx := contenttypetest.New()
	_, h := open(t, storagetest.Library(t, works.Schema()), map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, fx.Source())
	h.fx = fx
	return h
}

func shelf() contenttypetest.Record {
	return contenttypetest.Record{
		Tree: works.Fetched{
			Label:     "The Attic",
			SortLabel: "Attic, The",
			Children: []works.Entry{
				{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}, {Key: "c2", Label: "Gloves"}}},
				{Key: "b2", Label: "Summer", Children: []works.Entry{{Key: "c3", Label: "Hat"}}},
			},
		},
		WorkFields:  []works.FieldValue{works.TextField("origin", "Grandma")},
		LevelFields: map[works.Key][]works.FieldValue{"b1": {works.DateField("packed", works.Date{Year: 2024, Month: 11})}},
		ItemFields: map[works.Key][]works.FieldValue{
			"c1": {works.NumberField("count", 2), works.DurationField("runtime", 90*time.Second)},
			"c3": {works.NumberField("count", 1)},
		},
	}
}

func render(tree works.Tree) string {
	names := map[works.LevelID]string{}
	for _, l := range tree.Levels {
		names[l.ID] = string(l.Key)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %q sort %q%s\n", tree.Key, tree.Label, tree.SortLabel, fields(tree.Fields))
	for _, l := range tree.Levels {
		fmt.Fprintf(&b, "  level %s depth %d under %q #%d %q%s%s\n", l.Key, l.Depth, names[l.ParentID], l.Position, l.Label, withdrawn(l.WithdrawnAt), fields(l.Fields))
	}
	for _, it := range tree.Items {
		fmt.Fprintf(&b, "  item %s in %s #%d %q%s%s\n", it.Key, names[it.LevelID], it.Position, it.Label, withdrawn(it.WithdrawnAt), fields(it.Fields))
	}
	return b.String()
}

func withdrawn(at storage.Time) string {
	if at.IsZero() {
		return ""
	}
	return " withdrawn"
}

func fields(vs []works.FieldValue) string {
	var b strings.Builder
	for _, v := range vs {
		switch v.Kind {
		case works.KindText:
			fmt.Fprintf(&b, " %s=%q", v.Key, v.Text)
		case works.KindNumber:
			fmt.Fprintf(&b, " %s=%v", v.Key, v.Number)
		case works.KindDate:
			fmt.Fprintf(&b, " %s=%s", v.Key, v.Date)
		case works.KindDuration:
			fmt.Fprintf(&b, " %s=%s", v.Key, v.Duration)
		case works.KindTexts:
			fmt.Fprintf(&b, " %s=%q", v.Key, v.Texts)
		case works.KindPeople:
			fmt.Fprintf(&b, " %s=%+v", v.Key, v.People)
		case works.KindLink:
			fmt.Fprintf(&b, " %s=%+v", v.Key, v.Link)
		}
	}
	return b.String()
}

func ids(tree works.Tree) map[string]storage.UUID {
	out := map[string]storage.UUID{"work": tree.ID.UUID}
	for _, l := range tree.Levels {
		out["level "+string(l.Key)] = l.ID.UUID
	}
	for _, it := range tree.Items {
		out["item "+string(it.Key)] = it.ID.UUID
	}
	return out
}

func sameIDs(t *testing.T, before, after works.Tree) {
	t.Helper()
	was, now := ids(before), ids(after)
	for k, id := range was {
		if now[k] != id {
			t.Errorf("%s: id %s became %s", k, id, now[k])
		}
	}
}

func TestAddRunsAlsoInItsWriteOnlyWhenItCreatesTheWork(t *testing.T) {
	h := newHarness(t)
	h.fx.Put("w1", shelf())
	refuse := errors.New("refused")
	var saw []works.Summary
	also := func(err error) works.Adding {
		return works.Adding{Monitor: true, Also: func(_ context.Context, _ *storage.Tx[storage.Library], w works.Summary) error {
			saw = append(saw, w)
			return err
		}}
	}
	if _, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", also(refuse)); !errors.Is(err, refuse) {
		t.Fatalf("an add whose Also fails = %v, want its error", err)
	}
	if rows := h.rows(t); len(rows) != 0 {
		t.Fatalf("after a failed Also the library lists %v, want no Work", rows)
	}
	tree, created, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", also(nil))
	if err != nil || !created {
		t.Fatalf("add: created %t, %v", created, err)
	}
	if len(saw) != 2 || saw[1].ID != tree.ID || saw[1].ContentType != contenttypetest.ID || saw[1].Label != tree.Label {
		t.Fatalf("Also saw %+v, want the created Work %s", saw, tree.ID)
	}
	if _, created, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", also(refuse)); err != nil || created {
		t.Fatalf("adding a Work already there: created %t, %v; want it unchanged and Also not run", created, err)
	}
	if len(saw) != 2 {
		t.Fatalf("Also ran %d times, want twice: never for a Work already there", len(saw))
	}
}

func TestAddTwiceFetchesOnce(t *testing.T) {
	h := newHarness(t)
	h.fx.Put("w1", shelf())
	tree, created, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if err != nil || !created {
		t.Fatalf("first add: created %t, %v; want a new Work", created, err)
	}
	want := `w1 "The Attic" sort "Attic, The" origin="Grandma"
  level b1 depth 1 under "" #1 "Winter" packed=2024-11
  level b2 depth 1 under "" #2 "Summer"
  item c1 in b1 #1 "Scarf" count=2 runtime=1m30s
  item c2 in b1 #2 "Gloves"
  item c3 in b2 #1 "Hat" count=1
`
	if got := render(tree); got != want {
		t.Fatalf("tree =\n%s\nwant\n%s", got, want)
	}
	if got := h.changed.take(); len(got) != 1 || !slices.Equal(got[0].WorkIDs, []string{tree.ID.String()}) {
		t.Fatalf("published %v, want one works.changed for %s", got, tree.ID)
	}

	again, created, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if err != nil || created || render(again) != want || again.ID != tree.ID {
		t.Fatalf("second add: created %t, %v, tree\n%s; want the same Work, not created", created, err, render(again))
	}
	sameIDs(t, tree, again)
	if n := h.fx.Fetches("w1"); n != 1 {
		t.Fatalf("fetches = %d, want 1", n)
	}
	if got := h.changed.take(); len(got) != 0 {
		t.Fatalf("a repeated add published %v", got)
	}
}

func TestRacingAddsMakeOneWork(t *testing.T) {
	h := newHarness(t)
	h.fx.Put("w1", shelf())
	release := h.fx.Hold(2)
	type result struct {
		tree    works.Tree
		created bool
		err     error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			tree, created, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
			results <- result{tree, created, err}
		}()
	}
	release()
	a, b := <-results, <-results
	if a.err != nil || b.err != nil || a.created == b.created || a.tree.ID != b.tree.ID {
		t.Fatalf("racing adds = (%s %t %v) and (%s %t %v); want one Work, created once", a.tree.ID, a.created, a.err, b.tree.ID, b.created, b.err)
	}
	sameIDs(t, a.tree, b.tree)
	if rows := h.rows(t); len(rows) != 1 {
		t.Fatalf("works = %v, want one", rows)
	}
}

func TestRefreshKeepsEveryID(t *testing.T) {
	h := newHarness(t)
	h.fx.Put("w1", shelf())
	first, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	h.changed.take()

	moved := shelf()
	moved.Tree.Label = "The Loft"
	moved.Tree.Children = []works.Entry{
		{Key: "b2", Label: "Summer", Children: []works.Entry{{Key: "c2", Label: "Gloves"}}},
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Wool scarf"}}},
	}
	h.fx.Put("w1", moved)
	second, err := h.works.Refresh(t.Context(), admin, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := `w1 "The Loft" sort "Attic, The" origin="Grandma"
  level b2 depth 1 under "" #1 "Summer"
  level b1 depth 1 under "" #2 "Winter" packed=2024-11
  item c2 in b2 #1 "Gloves"
  item c3 in b2 #1 "Hat" withdrawn count=1
  item c1 in b1 #1 "Wool scarf" count=2 runtime=1m30s
`
	if got := render(second); got != want {
		t.Fatalf("refreshed tree =\n%s\nwant\n%s", got, want)
	}
	sameIDs(t, first, second)
	if got := h.changed.take(); len(got) != 1 || !slices.Equal(got[0].WorkIDs, []string{first.ID.String()}) {
		t.Fatalf("published %v, want one works.changed for %s", got, first.ID)
	}

	h.fx.Put("w1", shelf())
	third, err := h.works.Refresh(t.Context(), admin, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	want = `w1 "The Attic" sort "Attic, The" origin="Grandma"
  level b1 depth 1 under "" #1 "Winter" packed=2024-11
  level b2 depth 1 under "" #2 "Summer"
  item c1 in b1 #1 "Scarf" count=2 runtime=1m30s
  item c2 in b1 #2 "Gloves"
  item c3 in b2 #1 "Hat" count=1
`
	if got := render(third); got != want {
		t.Fatalf("tree after the return =\n%s\nwant\n%s", got, want)
	}
	sameIDs(t, first, third)

	fourth, err := h.works.Refresh(t.Context(), admin, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	third.RefreshedAt, fourth.RefreshedAt = storage.Time{}, storage.Time{}
	if a, b := encode(t, third), encode(t, fourth); a != b {
		t.Fatalf("a second refresh of one record changed more than refreshedAt:\n%s\n%s", a, b)
	}
}

func encode(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestGoneUpstreamMarksAndChangesNothingElse(t *testing.T) {
	h := newHarness(t)
	h.fx.Put("w1", shelf())
	tree, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	h.changed.take()
	h.fx.Fail("w1", fmt.Errorf("lookup w1: %w", works.ErrNotFound))
	if _, err := h.works.Refresh(t.Context(), admin, tree.ID); !errors.Is(err, works.ErrGoneUpstream) {
		t.Fatalf("refresh of a Work the source lost = %v, want ErrGoneUpstream", err)
	}
	gone, err := h.works.Get(t.Context(), admin, tree.ID)
	if err != nil || gone.GoneUpstreamAt.IsZero() {
		t.Fatalf("after the source lost it: gone upstream at %v, %v; want it marked", gone.GoneUpstreamAt, err)
	}
	unmarked := gone
	unmarked.GoneUpstreamAt = storage.Time{}
	if encode(t, unmarked) != encode(t, tree) {
		t.Fatalf("marking it gone changed more than the mark:\n%s\n%s", encode(t, unmarked), encode(t, tree))
	}
	if got := h.changed.take(); len(got) != 1 || !slices.Equal(got[0].WorkIDs, []string{tree.ID.String()}) {
		t.Fatalf("marking it gone published %v, want one works.changed for %s", got, tree.ID)
	}
	if _, err := h.works.Refresh(t.Context(), admin, tree.ID); !errors.Is(err, works.ErrGoneUpstream) {
		t.Fatalf("a second refresh = %v, want ErrGoneUpstream", err)
	}
	if got := h.changed.take(); len(got) != 0 {
		t.Fatalf("a Work already marked gone published %v", got)
	}
	h.fx.Fail("w1", fmt.Errorf("lookup w1: %w", works.ErrUnavailable))
	if _, err := h.works.Refresh(t.Context(), admin, tree.ID); !errors.Is(err, works.ErrUnavailable) {
		t.Fatalf("refresh with the source down = %v, want ErrUnavailable", err)
	}
	still, err := h.works.Get(t.Context(), admin, tree.ID)
	if err != nil || still.GoneUpstreamAt.IsZero() || len(h.changed.take()) != 0 {
		t.Fatalf("an unanswered refresh moved the mark or published: %v, %v", still.GoneUpstreamAt, err)
	}
	if _, err := h.works.Refresh(t.Context(), admin, works.WorkID{UUID: storage.NewUUID()}); !errors.Is(err, works.ErrNoWork) {
		t.Fatalf("refresh of an unknown id = %v, want ErrNoWork", err)
	}
}

func TestAGoodFetchClearsGoneUpstream(t *testing.T) {
	h := newHarness(t)
	h.fx.Put("w1", shelf())
	tree, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	h.fx.Fail("w1", works.ErrNotFound)
	if _, err := h.works.Refresh(t.Context(), admin, tree.ID); !errors.Is(err, works.ErrGoneUpstream) {
		t.Fatal(err)
	}
	h.changed.take()
	h.fx.Fail("w1", nil)
	back, err := h.works.Refresh(t.Context(), admin, tree.ID)
	if err != nil || !back.GoneUpstreamAt.IsZero() {
		t.Fatalf("a good fetch: gone upstream at %v, %v; want it cleared", back.GoneUpstreamAt, err)
	}
	if got := h.changed.take(); len(got) != 1 {
		t.Fatalf("clearing the mark published %v, want one works.changed", got)
	}
}

func TestAnInvalidTreeIsRefusedWhole(t *testing.T) {
	h := newHarness(t)
	r := shelf()
	r.Tree.Children[1].Children = append(r.Tree.Children[1].Children, works.Entry{Key: "c1", Label: "Scarf again"})
	h.fx.Put("w1", r)
	_, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if want := "the metadata source sent a record SlipStream can't use: box b2: two Items have key c1"; err == nil || err.Error() != want || !errors.Is(err, works.ErrInvalid) {
		t.Fatalf("add = %v, want %q", err, want)
	}
	if want := `level=WARN msg="metadata invalid" contentType=fixture key=w1 err="the metadata source sent a record SlipStream can't use: box b2: two Items have key c1"`; strings.TrimSpace(h.log.String()) != want {
		t.Fatalf("log = %q, want %q", h.log.String(), want)
	}
	if rows := h.rows(t); len(rows) != 0 {
		t.Fatalf("works = %v, want none", rows)
	}
}

func TestASourceErrorOutsideTheTwoIsABugOfTheType(t *testing.T) {
	h := newHarness(t)
	h.fx.Fail("w1", errors.New("disk on fire"))
	_, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if want := "content type fixture: fetch w1: disk on fire"; err == nil || err.Error() != want {
		t.Fatalf("add = %v, want %q", err, want)
	}
	h.fx.Fail("w1", nil)
	if _, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{}); !errors.Is(err, works.ErrNotFound) {
		t.Fatalf("add of a key the source lacks = %v, want ErrNotFound", err)
	}
}

func TestAFieldOfTheWrongKindIsABug(t *testing.T) {
	h := newHarness(t)
	r := shelf()
	r.ItemFields["c3"] = []works.FieldValue{works.TextField("count", "one")}
	h.fx.Put("w1", r)
	_, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if err == nil || !strings.HasSuffix(err.Error(), ": item c3: field count is text; its spec says number") {
		t.Fatalf("add = %v, want the mismatched field named", err)
	}
}

func TestSearchMarksWorksAlreadyAdded(t *testing.T) {
	h := newHarness(t)
	h.fx.Put("w1", shelf())
	other := shelf()
	other.Tree.Label = "The Attic Annex"
	h.fx.Put("w2", other)
	tree, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	found, err := h.works.Search(t.Context(), admin, contenttypetest.ID, "attic")
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(found))
	for i, c := range found {
		got[i] = fmt.Sprintf("%s %q%s added=%t", c.Key, c.Label, fields(c.Fields), c.WorkID == tree.ID)
	}
	if want := []string{`w1 "The Attic" origin="Grandma" added=true`, `w2 "The Attic Annex" origin="Grandma" added=false`}; strings.Join(got, "\n") != strings.Join(want, "\n") || !found[1].WorkID.IsZero() {
		t.Fatalf("candidates = %q, want %q", got, want)
	}
	h.fx.FailSearch(fmt.Errorf("search: %w", works.ErrUnavailable))
	if _, err := h.works.Search(t.Context(), admin, contenttypetest.ID, "attic"); !errors.Is(err, works.ErrUnavailable) {
		t.Fatalf("search = %v, want ErrUnavailable", err)
	}
}

func TestATypeTurnsOffOnlyWhileItHoldsNoWorks(t *testing.T) {
	h := newHarness(t)
	h.fx.Put("w1", shelf())
	off, err := h.works.SetEnabled(t.Context(), admin, contenttypetest.ID, false)
	if err != nil || off.Enabled || off.ID != contenttypetest.ID || off.Name != "Fixture" {
		t.Fatalf("turn off = %+v, %v; want fixture off", off, err)
	}
	if _, err := h.works.SetEnabled(t.Context(), admin, contenttypetest.ID, false); err != nil {
		t.Fatal(err)
	}
	if got := h.types.take(); len(got) != 1 || got[0].ContentType != "fixture" {
		t.Fatalf("published %v, want one works.content-types-changed for fixture", got)
	}
	types, err := h.works.Types(t.Context())
	if err != nil || len(types) != 1 || types[0].Enabled {
		t.Fatalf("types = %+v, %v; want fixture off", types, err)
	}
	if _, err := h.works.Search(t.Context(), admin, contenttypetest.ID, "attic"); !errors.Is(err, works.ErrTypeOff) {
		t.Fatalf("search of an off type = %v, want ErrTypeOff", err)
	}
	if _, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{}); !errors.Is(err, works.ErrTypeOff) {
		t.Fatalf("add to an off type = %v, want ErrTypeOff", err)
	}
	on, err := h.works.SetEnabled(t.Context(), admin, contenttypetest.ID, true)
	if err != nil || !on.Enabled {
		t.Fatalf("turn on = %+v, %v; want fixture on", on, err)
	}
	if got := h.types.take(); len(got) != 1 {
		t.Fatalf("published %v, want one works.content-types-changed", got)
	}
	if again, err := h.works.SetEnabled(t.Context(), admin, contenttypetest.ID, true); err != nil || !reflect.DeepEqual(again, on) {
		t.Fatalf("turn on again = %+v, %v; want %+v", again, err, on)
	}
	if got := h.types.take(); len(got) != 0 {
		t.Fatalf("turning on a type already on published %v", got)
	}
	if enabled, err := works.Enabled(t.Context(), h.lib, contenttypetest.ID); err != nil || !enabled {
		t.Fatalf("Enabled = %t, %v; want true", enabled, err)
	}

	if _, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.works.SetEnabled(t.Context(), admin, contenttypetest.ID, false); !errors.Is(err, works.ErrTypeInUse) {
		t.Fatalf("turning off a type holding a Work = %v, want ErrTypeInUse", err)
	}
	if got := h.types.take(); len(got) != 0 {
		t.Fatalf("a refused turn-off published %v", got)
	}
}

func TestAnUnregisteredTypeIsNotFound(t *testing.T) {
	h := newHarness(t)
	ghost := works.TypeID("ghost")
	if _, _, err := h.works.Add(t.Context(), admin, ghost, "w1", works.Adding{}); !errors.Is(err, works.ErrNoType) {
		t.Errorf("add = %v, want ErrNoType", err)
	}
	if _, err := h.works.Search(t.Context(), admin, ghost, "x"); !errors.Is(err, works.ErrNoType) {
		t.Errorf("search = %v, want ErrNoType", err)
	}
	err := h.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		_, err := h.works.RowsIn(ctx, r, works.Scope{Type: &ghost})
		return err
	})
	if !errors.Is(err, works.ErrNoType) {
		t.Errorf("rows = %v, want ErrNoType", err)
	}
	if _, err := h.works.SetEnabled(t.Context(), admin, ghost, false); !errors.Is(err, works.ErrNoType) {
		t.Errorf("set enabled = %v, want ErrNoType", err)
	}
}

func openAt(t *testing.T, path string) *storage.DB[storage.Library] {
	t.Helper()
	lib, err := storage.OpenLibrary(t.Context(), path, []storage.Schema[storage.Library]{works.Schema()},
		func(context.Context, *storage.Unmigrated, storage.Plan) error { return nil },
		storage.Timings{BusyTimeout: 5 * time.Second}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	return lib
}

func TestARestartReadsTheSameTree(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	fx := contenttypetest.New()
	fx.Put("w1", shelf())
	lib := openAt(t, path)
	w, _ := open(t, lib, map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, fx.Source())
	tree, _, err := w.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	if err := lib.Close(); err != nil {
		t.Fatal(err)
	}
	lib = openAt(t, path)
	t.Cleanup(func() { _ = lib.Close() })
	w, _ = open(t, lib, map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, fx.Source())
	again, err := w.Get(t.Context(), admin, tree.ID)
	if err != nil || encode(t, again) != encode(t, tree) {
		t.Fatalf("after a restart = %s, %v; want %s", encode(t, again), err, encode(t, tree))
	}
}

func TestOpenRefusesWorksOfAnUnregisteredType(t *testing.T) {
	lib := storagetest.Library(t, works.Schema())
	fx := contenttypetest.New()
	fx.Put("w1", shelf())
	ghost, _ := open(t, lib, map[works.TypeID]works.Catalog{"ghost": fx}, fx.SourceAs("ghost"))
	if _, _, err := ghost.Add(t.Context(), admin, "ghost", "w1", works.Adding{}); err != nil {
		t.Fatal(err)
	}
	_, err := works.Open(t.Context(), works.Config{Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}})
	var refused *storage.RefusedError
	if want := "library.db holds Works of Content Type ghost, which this binary doesn't register"; !errors.As(err, &refused) || err.Error() != want {
		t.Fatalf("open = %v, want a *storage.RefusedError %q", err, want)
	}
}

type described struct {
	*contenttypetest.Fixture
	d works.Description
}

func (d described) Describe() works.Description { return d.d }

func TestOpenPanicsOnADescriptionThatBreaksARule(t *testing.T) {
	bad := func(edit func(*works.Description)) works.Description {
		d := contenttypetest.Description
		d.Fields = append([]works.FieldSpec(nil), d.Fields...)
		edit(&d)
		return d
	}
	for want, d := range map[string]works.Description{
		"works: content type fixture: field key Year is not ^[a-z][a-z0-9_]{0,31}$": bad(func(d *works.Description) {
			d.Fields[0].Key = "Year"
		}),
		"works: content type fixture: field packed sits at level 2, which the type does not declare": bad(func(d *works.Description) {
			d.Fields[1].Tier = works.TierLevel(2)
		}),
		"works: content type fixture: field count is declared twice on the Items": bad(func(d *works.Description) {
			d.Fields[3].Key = "count"
		}),
		"works: content type fixture: field origin has no kind": bad(func(d *works.Description) {
			d.Fields[0].Kind = 0
		}),
		"works: content type fixture: it has no icon": bad(func(d *works.Description) {
			d.Icon = 0
		}),
		"works: content type fixture: noun {One:Box Many:} needs both One and Many": bad(func(d *works.Description) {
			d.Levels = []works.Noun{{One: "Box"}}
		}),
		"works: content type fixture: it names no source": bad(func(d *works.Description) {
			d.Source = ""
		}),
		"works: content type fixture: it has no Items shape": bad(func(d *works.Description) {
			d.Items = 0
		}),
		"works: content type fixture: it holds one Item and declares levels": bad(func(d *works.Description) {
			d.Items = works.ItemsOne
		}),
		"works: content type fixture: field origin is in no place": bad(func(d *works.Description) {
			d.Fields[0].In = nil
		}),
		"works: content type fixture: field origin is in place 9, which doesn't exist": bad(func(d *works.Description) {
			d.Fields[0].In = []works.Place{9}
		}),
		"works: content type fixture: field origin is in about twice": bad(func(d *works.Description) {
			d.Fields[0].In = []works.Place{works.PlaceAbout, works.PlaceAbout}
		}),
		"works: content type fixture: field tags is texts, and tagline takes text only": bad(func(d *works.Description) {
			d.Fields[4].In = []works.Place{works.PlaceTagline}
		}),
		"works: content type fixture: fields origin and blurb are both in overview, which takes one": bad(func(d *works.Description) {
			d.Fields[0].In = []works.Place{works.PlaceOverview}
			d.Fields = append(d.Fields, works.FieldSpec{Key: "blurb", Label: "Blurb", Kind: works.KindText, In: []works.Place{works.PlaceOverview}})
		}),
		"works: content type fixture: field origin is text, and credits takes people only": bad(func(d *works.Description) {
			d.Fields[0].In = []works.Place{works.PlaceCredits}
		}),
		"works: content type fixture: field makers is people, which facts doesn't take": bad(func(d *works.Description) {
			d.Fields[5].In = []works.Place{works.PlaceFacts}
		}),
		"works: content type fixture: field origin has list flags 1000, which aren't all flags": bad(func(d *works.Description) {
			d.Fields[0].List = 8
		}),
		"works: content type fixture: field count sits at the Items and is listed, and only Work-tier fields are": bad(func(d *works.Description) {
			d.Fields[2].List = works.ListColumn
		}),
		"works: content type fixture: field makers is people, which the Library can't list": bad(func(d *works.Description) {
			d.Fields[5].List = works.ListColumn
		}),
		"works: content type fixture: field page is link, which the Library can't list": bad(func(d *works.Description) {
			d.Fields[6].List = works.ListFilter
		}),
		"works: content type fixture: field tags is texts, which the Library can't sort by": bad(func(d *works.Description) {
			d.Fields[4].List = works.ListSort
		}),
		"works: content type fixture: field origin is shown with nowhere, which is no other text field of the Work": bad(func(d *works.Description) {
			d.Fields[0].With = "nowhere"
		}),
		"works: content type fixture: field origin is shown with tags, which is no other text field of the Work": bad(func(d *works.Description) {
			d.Fields[0].With = "tags"
		}),
		"works: content type fixture: field origin is shown with origin, which is no other text field of the Work": bad(func(d *works.Description) {
			d.Fields[0].With = "origin"
		}),
		"works: content type fixture: field origin is shown with packed, which is no other text field of the Work": bad(func(d *works.Description) {
			d.Fields[1].Kind = works.KindText
			d.Fields[0].With = "packed"
		}),
		"works: content type fixture: field origin shows art nope, which is no Work-tier artwork kind": bad(func(d *works.Description) {
			d.Fields[0].Art = "nope"
		}),
		"works: content type fixture: field origin shows art snap, which is no Work-tier artwork kind": bad(func(d *works.Description) {
			d.Fields[0].Art = "snap"
		}),
		"works: content type fixture: its poster art nope is no Work-tier artwork kind of shape poster": bad(func(d *works.Description) {
			d.Art.Poster = "nope"
		}),
		"works: content type fixture: its backdrop art photo is no Work-tier artwork kind of shape backdrop": bad(func(d *works.Description) {
			d.Art.Backdrop = "photo"
		}),
		"works: content type fixture: its title art snap is no Work-tier artwork kind of shape logo": bad(func(d *works.Description) {
			d.Art.Title = "snap"
		}),
		"works: content type fixture: it shows ratings from letterboxd, which RatingSources doesn't list": bad(func(d *works.Description) {
			d.Ratings = []works.RatingSource{"letterboxd"}
		}),
		"works: content type fixture: it shows ratings from imdb twice": bad(func(d *works.Description) {
			d.Ratings = []works.RatingSource{"imdb", "tmdb", "imdb"}
		}),
	} {
		func() {
			defer func() {
				if got := recover(); got != want {
					t.Errorf("panic = %v, want %q", got, want)
				}
			}()
			_, _ = works.Open(t.Context(), works.Config{
				Library:  storagetest.Library(t, works.Schema()),
				Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: described{contenttypetest.New(), d}},
			})
		}()
	}
}
