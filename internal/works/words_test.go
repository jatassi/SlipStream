package works_test

import (
	"maps"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestNumbersWordsFoldsRunsOfThreeOrMore(t *testing.T) {
	episode := works.Noun{One: "Episode", Many: "Episodes"}
	for want, ns := range map[string][]int{
		"Episode 4":                    {4},
		"Episodes 1 and 2":             {1, 2},
		"Episodes 1 to 10":             {1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		"Episodes 1, 2 and 5":          {1, 2, 5},
		"Episodes 1 to 3 and 7 to 9":   {1, 2, 3, 7, 8, 9},
		"Episodes 2, 4 to 6 and 9":     {2, 4, 5, 6, 9},
		"Episodes 1 to 4, 6, 7 and 12": {1, 2, 3, 4, 6, 7, 12},
	} {
		if got := works.NumbersWords(episode, ns); got != want {
			t.Errorf("NumbersWords(%v) = %q, want %q", ns, got, want)
		}
	}
}

func TestPlacesCountEachLevelInTreeOrderWithdrawnIncluded(t *testing.T) {
	one, two := works.LevelID{UUID: storage.NewUUID()}, works.LevelID{UUID: storage.NewUUID()}
	id := func() works.ItemID { return works.ItemID{UUID: storage.NewUUID()} }
	a, b, gone, c, d := id(), id(), id(), id(), id()
	got := works.Places([]works.Item{
		{ID: a, LevelID: one}, {ID: b, LevelID: one}, {ID: c, LevelID: two},
		{ID: gone, LevelID: one, WithdrawnAt: storage.Now()}, {ID: d, LevelID: two},
	})
	if want := map[works.ItemID]int{a: 1, b: 2, gone: 3, c: 1, d: 2}; !maps.Equal(got, want) {
		t.Fatalf("Places = %v, want %v", got, want)
	}
}
