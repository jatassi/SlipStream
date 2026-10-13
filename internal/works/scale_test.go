package works_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/works"
)

const (
	longSeasons  = 40
	longEpisodes = 30
)

func longShow(seasons int, renamed bool) contenttypetest.Record {
	r := contenttypetest.Record{
		Tree:        works.Fetched{Label: "Evening Tides"},
		LevelFields: map[works.Key][]works.FieldValue{},
		ItemFields:  map[works.Key][]works.FieldValue{},
	}
	for s := 1; s <= seasons; s++ {
		key := works.Key(fmt.Sprintf("sn%d", s))
		season := works.Entry{Key: key, Label: fmt.Sprintf("Season %d", s)}
		r.LevelFields[key] = []works.FieldValue{works.NumberField("number", float64(s))}
		for e := 1; e <= longEpisodes; e++ {
			ep := works.Key(fmt.Sprintf("x%02d%02d", s, e))
			label := fmt.Sprintf("Episode %d", e)
			if renamed && e%10 == 0 {
				label += " (Part 2)"
			}
			season.Children = append(season.Children, works.Entry{Key: ep, Label: label})
			r.ItemFields[ep] = []works.FieldValue{works.DateField("aired", works.Date{Year: 1980 + s, Month: 1 + e%12, Day: 1 + e%28})}
		}
		r.Tree.Children = append(r.Tree.Children, season)
	}
	return r
}

func TestALongRunningShowRefreshes(t *testing.T) {
	h := shaped(t, "seasonal", seasonalShape)
	h.fx.Put("s1", longShow(longSeasons, false))
	start := time.Now()
	first, _, err := h.works.Add(t.Context(), admin, "seasonal", "s1")
	added := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Levels) != longSeasons || len(first.Items) != longSeasons*longEpisodes {
		t.Fatalf("added %d seasons and %d episodes, want %d and %d", len(first.Levels), len(first.Items), longSeasons, longSeasons*longEpisodes)
	}

	h.fx.Put("s1", longShow(longSeasons+1, true))
	start = time.Now()
	second, err := h.works.Refresh(t.Context(), admin, first.ID)
	refreshed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	start = time.Now()
	got, err := h.works.Get(t.Context(), admin, first.ID)
	read := time.Since(start)
	if err != nil || encode(t, got) != encode(t, second) {
		t.Fatalf("get = %v; want the refreshed tree", err)
	}
	t.Logf("add %v, refresh %v, get %v", added, refreshed, read)

	sameIDs(t, first, second)
	if len(second.Levels) != longSeasons+1 || len(second.Items) != (longSeasons+1)*longEpisodes {
		t.Fatalf("refreshed into %d seasons and %d episodes, want %d and %d", len(second.Levels), len(second.Items), longSeasons+1, (longSeasons+1)*longEpisodes)
	}
	last := second.Levels[longSeasons]
	if last.Key != "sn41" || last.Position != 41 || !last.WithdrawnAt.IsZero() {
		t.Fatalf("last season = %s #%d withdrawn %v, want sn41 #41 live", last.Key, last.Position, last.WithdrawnAt)
	}
	was := map[works.Key]string{}
	for _, it := range first.Items {
		was[it.Key] = it.Label
	}
	renamed, added := 0, 0
	for _, it := range second.Items {
		label, ok := was[it.Key]
		switch {
		case !ok:
			added++
		case label != it.Label:
			renamed++
		}
	}
	if renamed != 120 || added != 30 {
		t.Fatalf("%d episodes renamed and %d added, want 120 and 30", renamed, added)
	}
	tenth := second.Items[9]
	if tenth.Key != "x0110" || tenth.Label != "Episode 10 (Part 2)" || tenth.LevelID != second.Levels[0].ID || tenth.Position != 10 {
		t.Fatalf("episode 10 of season 1 = %+v", tenth)
	}
}

func BenchmarkLongShow(b *testing.B) {
	original, revised := longShow(longSeasons, false), longShow(longSeasons+1, true)
	b.Run("Add", func(b *testing.B) {
		h := shaped(b, "seasonal", seasonalShape)
		n := 0
		for b.Loop() {
			n++
			key := works.Key(fmt.Sprintf("s%d", n))
			h.fx.Put(key, original)
			if _, _, err := h.works.Add(b.Context(), admin, "seasonal", key); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("Refresh", func(b *testing.B) {
		h := shaped(b, "seasonal", seasonalShape)
		h.fx.Put("s1", original)
		tree, _, err := h.works.Add(b.Context(), admin, "seasonal", "s1")
		if err != nil {
			b.Fatal(err)
		}
		records := []contenttypetest.Record{revised, original}
		n := 0
		for b.Loop() {
			h.fx.Put("s1", records[n%2])
			n++
			if _, err := h.works.Refresh(b.Context(), admin, tree.ID); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("Get", func(b *testing.B) {
		h := shaped(b, "seasonal", seasonalShape)
		h.fx.Put("s1", revised)
		tree, _, err := h.works.Add(b.Context(), admin, "seasonal", "s1")
		if err != nil {
			b.Fatal(err)
		}
		for b.Loop() {
			if _, err := h.works.Get(b.Context(), admin, tree.ID); err != nil {
				b.Fatal(err)
			}
		}
	})
}
