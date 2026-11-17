package calendar

import (
	"testing"

	"github.com/jatassi/slipstream/internal/files"
)

func TestAnItemReadsByItsPlaceWithItsOwnTitle(t *testing.T) {
	s := newShelf(0, "Attic", states(3, files.Met))
	for i, label := range []string{"Hello, Ms. Cobel", "", "Thing 3"} {
		p := s.w.items[s.boxes[0][i]]
		p.Item.Label = label
		s.w.items[s.boxes[0][i]] = p
	}
	want := []string{"Thing 1 · Hello, Ms. Cobel", "Thing 2", "Thing 3"}
	for i, id := range s.boxes[0] {
		if got := itemLabel(s.w, s.w.items[id]); got != want[i] {
			t.Errorf("itemLabel of Thing %d = %q, want %q", i+1, got, want[i])
		}
	}
}
