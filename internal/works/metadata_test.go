package works_test

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/works"
)

func votes(n int64) *int64 { return &n }

func photo(path string) artwork.Ref {
	return artwork.Ref{Source: "fixture", Shape: artwork.ShapePoster, Path: path}
}

func snap(path string) artwork.Ref {
	return artwork.Ref{Source: "fixture", Shape: artwork.ShapeBackdrop, Path: path}
}

func rated() contenttypetest.Record {
	r := shelf()
	r.Tree.Ratings = []works.Rating{
		{Source: "tmdb", Value: 7.5, Votes: votes(1200)},
		{Source: "imdb", Value: 8.1, Votes: votes(90000)},
		{Item: "c1", Source: "tmdb", Value: 6},
	}
	r.Tree.RatingsRead = true
	r.Tree.Artwork = []works.Art{
		{Kind: "photo", Ref: photo("/attic.jpg")},
		{Item: "c1", Kind: "snap", Ref: snap("/scarf.jpg")},
	}
	return r
}

// scores renders a tree's ratings and artwork, the Work's then each Item's.
func scores(tree works.Tree) string {
	var b strings.Builder
	line := func(owner string, rs []works.StoredRating, art map[string]artwork.Ref) {
		for _, r := range rs {
			fmt.Fprintf(&b, "%s %s=%v", owner, r.Source, r.Value)
			if r.Votes != nil {
				fmt.Fprintf(&b, " votes=%d", *r.Votes)
			}
			b.WriteString("\n")
		}
		for _, kind := range slices.Sorted(maps.Keys(art)) {
			fmt.Fprintf(&b, "%s %s=%s:%s:%s\n", owner, kind, art[kind].Source, art[kind].Shape, art[kind].Path)
		}
	}
	line(string(tree.Key), tree.Ratings, tree.Artwork)
	for _, it := range tree.Items {
		line(string(it.Key), it.Ratings, it.Artwork)
	}
	return b.String()
}

func TestARefreshThatChangesNothingPublishesNothing(t *testing.T) {
	h := newHarness(t)
	h.fx.Put("w1", rated())
	tree, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	h.changed.take()
	refresh := func(what string, want int) {
		t.Helper()
		if _, err := h.works.Refresh(t.Context(), admin, tree.ID); err != nil {
			t.Fatal(err)
		}
		if got := h.changed.take(); len(got) != want {
			t.Fatalf("a refresh that %s published %v, want %d works.changed", what, got, want)
		}
	}
	refresh("changes nothing", 0)
	for _, c := range []struct {
		what string
		edit func(*contenttypetest.Record)
	}{
		{"moves a rating", func(r *contenttypetest.Record) { r.Tree.Ratings[1].Value = 8.2 }},
		{"moves a vote count", func(r *contenttypetest.Record) { r.Tree.Ratings[0].Votes = votes(1300) }},
		{"adds a rating source", func(r *contenttypetest.Record) {
			r.Tree.Ratings = append(r.Tree.Ratings, works.Rating{Source: "metacritic", Value: 70})
		}},
		{"moves an artwork reference", func(r *contenttypetest.Record) { r.Tree.Artwork[0].Ref.Path = "/attic-2.jpg" }},
		{"changes the type's own rows", func(r *contenttypetest.Record) {
			r.WorkFields = []works.FieldValue{works.TextField("origin", "Grandpa")}
		}},
		{"renames an Item", func(r *contenttypetest.Record) { r.Tree.Children[0].Children[1].Label = "Mittens" }},
		{"renames the Work", func(r *contenttypetest.Record) { r.Tree.Label = "The Loft" }},
	} {
		r := rated()
		c.edit(&r)
		h.fx.Put("w1", r)
		refresh(c.what, 1)
		refresh(c.what+" again", 0)
		h.fx.Put("w1", rated())
		h.changed.take()
		if _, err := h.works.Refresh(t.Context(), admin, tree.ID); err != nil {
			t.Fatal(err)
		}
		h.changed.take()
	}
}

func TestAMissingRatingSourceKeepsItsRow(t *testing.T) {
	h := newHarness(t)
	h.fx.Put("w1", rated())
	tree, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	if tree.RatingsReadAt.IsZero() {
		t.Fatal("an add whose ratings providers answered left the ratings read time zero")
	}
	first := tree.Ratings[0].FetchedAt

	r := rated()
	r.Tree.Ratings = []works.Rating{{Source: "tmdb", Value: 7.7, Votes: votes(1250)}}
	r.Tree.RatingsRead = false
	h.fx.Put("w1", r)
	time.Sleep(time.Millisecond)
	again, err := h.works.Refresh(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := "w1 imdb=8.1 votes=90000\nw1 tmdb=7.7 votes=1250\nw1 photo=fixture:poster:/attic.jpg\nc1 tmdb=6\nc1 snap=fixture:backdrop:/scarf.jpg\n"
	if got := scores(again); got != want {
		t.Fatalf("after a fetch without IMDb or the Item's rating =\n%s\nwant\n%s", got, want)
	}
	if again.RatingsReadAt != tree.RatingsReadAt {
		t.Fatalf("a fetch whose ratings providers didn't answer moved the ratings read time from %v to %v", tree.RatingsReadAt, again.RatingsReadAt)
	}
	byWork := map[works.RatingSource]works.StoredRating{}
	for _, s := range again.Ratings {
		byWork[s.Source] = s
	}
	if byWork["imdb"].FetchedAt != first || !byWork["tmdb"].FetchedAt.Time().After(first.Time()) {
		t.Fatalf("fetched at: imdb %v, tmdb %v; want imdb kept at %v and tmdb moved", byWork["imdb"].FetchedAt, byWork["tmdb"].FetchedAt, first)
	}
}

func TestAMissingArtworkKindKeepsItsRow(t *testing.T) {
	h := newHarness(t)
	h.fx.Put("w1", rated())
	tree, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	r := rated()
	r.Tree.Artwork = []works.Art{{Item: "c1", Kind: "snap", Ref: snap("/scarf-2.jpg")}}
	h.fx.Put("w1", r)
	again, err := h.works.Refresh(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := "w1 imdb=8.1 votes=90000\nw1 tmdb=7.5 votes=1200\nw1 photo=fixture:poster:/attic.jpg\nc1 tmdb=6\nc1 snap=fixture:backdrop:/scarf-2.jpg\n"
	if got := scores(again); got != want {
		t.Fatalf("after a fetch without the Work's photo =\n%s\nwant\n%s", got, want)
	}
}

func TestARatingOrArtworkTheTypeCantHaveIsInvalid(t *testing.T) {
	for want, edit := range map[string]func(*contenttypetest.Record){
		`shelf w1: it has a rating from "letterboxd", which is no ratings source`: func(r *contenttypetest.Record) {
			r.Tree.Ratings[0].Source = "letterboxd"
		},
		"shelf w1: its IMDb rating 11 is outside 0 to 10": func(r *contenttypetest.Record) { r.Tree.Ratings[1].Value = 11 },
		"shelf w1: its TMDB rating has -1 votes":          func(r *contenttypetest.Record) { r.Tree.Ratings[0].Votes = votes(-1) },
		"thing c1: it has two TMDB ratings": func(r *contenttypetest.Record) {
			r.Tree.Ratings = append(r.Tree.Ratings, works.Rating{Item: "c1", Source: "tmdb", Value: 5})
		},
		"shelf w1: it names thing c9, which it doesn't list": func(r *contenttypetest.Record) { r.Tree.Ratings[2].Item = "c9" },
		`shelf w1: it has "snap" artwork, a kind it can't have`: func(r *contenttypetest.Record) {
			r.Tree.Artwork[0].Kind = "snap"
		},
		"shelf w1: its photo artwork is a backdrop, not a poster": func(r *contenttypetest.Record) {
			r.Tree.Artwork[0].Ref.Shape = artwork.ShapeBackdrop
		},
		"thing c1: its snap artwork names no source or no path": func(r *contenttypetest.Record) { r.Tree.Artwork[1].Ref.Path = "" },
	} {
		h := newHarness(t)
		r := rated()
		edit(&r)
		h.fx.Put("w1", r)
		_, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
		if !errors.Is(err, works.ErrInvalid) || !strings.HasSuffix(err.Error(), ": "+want) {
			t.Errorf("add = %v, want ErrInvalid ending %q", err, want)
		}
	}
}

func TestSetMonitoredPublishesChanged(t *testing.T) {
	h := newHarness(t)
	h.fx.Put("w1", shelf())
	h.fx.Put("w2", shelf())
	a, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w2", works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	h.changed.take()
	monitored := func(tree works.Tree) string { return monitoredKeys(t, h, tree.ID) }
	published := func() []string { return publishedIDs(h) }
	both := []string{a.ID.String(), b.ID.String()}
	slices.Sort(both)

	items := []works.ItemID{a.Items[0].ID, a.Items[1].ID, b.Items[2].ID}
	if err := h.works.SetMonitored(t.Context(), admin, items, true); err != nil {
		t.Fatal(err)
	}
	if got := published(); !reflect.DeepEqual(got, both) {
		t.Fatalf("monitoring Items of two Works published %v, want one works.changed each for %v", got, both)
	}
	if ga, gb := monitored(a), monitored(b); ga != "c1 c2" || gb != "c3" {
		t.Fatalf("monitored = %q and %q, want c1 c2 and c3", ga, gb)
	}
	if err := h.works.SetMonitored(t.Context(), admin, items, true); err != nil {
		t.Fatal(err)
	}
	if got := published(); len(got) != 0 {
		t.Fatalf("monitoring Items already monitored published %v", got)
	}
	if err := h.works.SetMonitored(t.Context(), admin, []works.ItemID{b.Items[2].ID, b.Items[0].ID}, false); err != nil {
		t.Fatal(err)
	}
	if got := published(); !reflect.DeepEqual(got, []string{b.ID.String()}) {
		t.Fatalf("unmonitoring one Work's Item published %v, want one works.changed for %s", got, b.ID)
	}
	if ga, gb := monitored(a), monitored(b); ga != "c1 c2" || gb != "" {
		t.Fatalf("monitored = %q and %q, want c1 c2 and none", ga, gb)
	}
}

func monitoredKeys(t *testing.T, h *harness, id works.WorkID) string {
	t.Helper()
	got, err := h.works.Get(t.Context(), admin, id)
	if err != nil {
		t.Fatal(err)
	}
	var on []string
	for _, it := range got.Items {
		if it.Monitored {
			on = append(on, string(it.Key))
		}
	}
	return strings.Join(on, " ")
}

func publishedIDs(h *harness) []string {
	var ids []string
	for _, c := range h.changed.take() {
		ids = append(ids, c.WorkID)
	}
	slices.Sort(ids)
	return ids
}

func (p *prefetched) take() []artwork.Want {
	p.mu.Lock()
	defer p.mu.Unlock()
	got := p.wants
	p.wants = nil
	return got
}

func TestAddPrefetchesTheKindsThatAskOnce(t *testing.T) {
	h := newHarness(t)
	h.fx.Put("w1", rated())
	if _, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{}); err != nil {
		t.Fatal(err)
	}
	if got, want := h.art.take(), []artwork.Want{{Ref: photo("/attic.jpg"), Size: artwork.SizeMedium}}; !slices.Equal(got, want) {
		t.Fatalf("adding prefetched %v, want only the photo, the one kind that asks, at medium %v", got, want)
	}
	if _, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{}); err != nil {
		t.Fatal(err)
	}
	h.fx.Put("w2", shelf())
	if _, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w2", works.Adding{}); err != nil {
		t.Fatal(err)
	}
	if got := h.art.take(); len(got) != 0 {
		t.Fatalf("adding a Work already there, then one with no artwork, prefetched %v, want nothing", got)
	}
}

func TestListCarriesEachWorksOwnArtwork(t *testing.T) {
	h := newHarness(t)
	h.fx.Put("w1", rated())
	h.fx.Put("w2", shelf())
	for _, k := range []works.Key{"w1", "w2"} {
		if _, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, k, works.Adding{}); err != nil {
			t.Fatal(err)
		}
	}
	listed, err := h.works.List(t.Context(), admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[works.Key]map[string]artwork.Ref{}
	for _, l := range listed {
		got[l.Key] = l.Artwork
	}
	if want := map[works.Key]map[string]artwork.Ref{"w1": {"photo": photo("/attic.jpg")}, "w2": {}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("List's artwork = %v, want each Work's own and no Item's: %v", got, want)
	}
}
