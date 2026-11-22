package node

import (
	"sync"
	"testing"
)

type uint64er interface{ Uint64() uint64 }

func firstThree(r uint64er) [3]uint64 { return [3]uint64{r.Uint64(), r.Uint64(), r.Uint64()} }

func TestStreamsAreKeyedBySeedAndName(t *testing.T) {
	s := NewStreams("fixed")
	torrent := firstThree(s.Stream("torrent"))
	if want := [3]uint64{17256744228552245810, 11623976731303118876, 6640235039819700663}; torrent != want {
		t.Fatalf("torrent draws = %v, want %v", torrent, want)
	}
	if link := firstThree(s.Stream("link")); link != [3]uint64{4298339582837690137, 5079177034435995883, 3674502440458912974} {
		t.Fatalf("link draws = %v", link)
	}
	if again := firstThree(NewStreams("fixed").Stream("torrent")); again != torrent {
		t.Fatalf("a second Node with seed fixed drew %v, want %v", again, torrent)
	}
}

func TestStreamNameIsTakenOnce(t *testing.T) {
	s := NewStreams("fixed")
	s.Stream("torrent")
	defer func() {
		if r := recover(); r != "node: random stream torrent requested twice" {
			t.Fatalf("recovered %v", r)
		}
	}()
	s.Stream("torrent")
}

func TestStreamIsSafeForConcurrentDraws(t *testing.T) {
	r := NewStreams("fixed").Stream("torrent")
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 1000 {
				r.Uint64()
			}
		})
	}
	wg.Wait()
	if got, want := r.Uint64(), firstAfter(8000); got != want {
		t.Fatalf("draw after 8000 concurrent draws = %d, want %d", got, want)
	}
}

func firstAfter(n int) uint64 {
	r := NewStreams("fixed").Stream("torrent")
	for range n {
		r.Uint64()
	}
	return r.Uint64()
}
