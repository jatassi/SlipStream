package grabs

import (
	"maps"
	"math/rand/v2"
	"slices"
	"strconv"
	"testing"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
)

// standing is how an Indexer answers Take: deleted, stopped, or with
// grabs left before its grab limit.
type standing struct {
	removed, stopped bool
	grabs            int
}

// refuses is the NotTakenError Take answers r with while r's Indexer stands
// at s, nil when it takes r.
func (s standing) refuses(r indexers.Release) error {
	switch {
	case s.removed:
		return &indexers.NotTakenError{Outcome: indexers.Outcome{Kind: indexers.Removed}}
	case s.stopped:
		return &indexers.NotTakenError{Outcome: indexers.Outcome{Kind: indexers.IsStopped}}
	case s.grabs == 0 && !r.Offered().IsMagnet():
		return &indexers.NotTakenError{Outcome: indexers.Outcome{Kind: indexers.LimitReached}}
	}
	return nil
}

// reserving takes as Indexers.Take does in a Write: a Release it takes
// spends one of its Indexer's grabs, unless it is a magnet link.
func reserving(sts map[indexers.ID]standing) taker {
	return func(r indexers.Release) (indexers.Ticket, error) {
		s := sts[r.Indexer]
		if err := s.refuses(r); err != nil {
			return indexers.Ticket{}, err
		}
		if !r.Offered().IsMagnet() {
			s.grabs--
			sts[r.Indexer] = s
		}
		return indexers.Ticket{Indexer: r.Indexer}, nil
	}
}

// reading answers as reserving does from the same standings and spends
// nothing.
func reading(sts map[indexers.ID]standing) taker {
	return func(r indexers.Release) (indexers.Ticket, error) {
		if err := sts[r.Indexer].refuses(r); err != nil {
			return indexers.Ticket{}, err
		}
		return indexers.Ticket{Indexer: r.Indexer}, nil
	}
}

type poolTitle struct {
	title string
	ids   map[string]string
}

// pool is the titles a generated search lists, each with the ids its
// Indexer tags it with: grabbable at 1080p or Small, refused by Quality,
// another Work's, and another title's.
var pool = []poolTitle{
	{"Orchard.1080p.BluRay.A", nil},
	{"Orchard.1080p.WEB-DL.B", nil},
	{"Orchard.1080p.WEBRip.C", nil},
	{"Orchard.720p.WEB-DL.D", nil},
	{"Orchard.576p.DVD.E", nil},
	{"Orchard.1080p.TS.F", nil},
	{"Orchard.1080p.WEB-DL.Other", map[string]string{"tmdbid": "9000002"}},
	{"Orchards.1080p.WEB-DL.G", nil},
}

// generated is one generated search: the open cells, every pick its
// decision makes over the Releases its Indexers listed, the positions
// gone and each Indexer's standing.
type generated struct {
	c     cells
	picks []Pick
	gone  map[int]bool
	sts   map[indexers.ID]standing
}

func generate(rng *rand.Rand, reads []release.Read, ids []indexers.ID) generated {
	g := generated{gone: map[int]bool{}, sts: map[indexers.ID]standing{}}
	priority := map[indexers.ID]int{}
	for _, id := range ids {
		priority[id] = 10 * rng.IntN(3)
		g.sts[id] = standing{removed: rng.IntN(8) == 0, stopped: rng.IntN(6) == 0, grabs: rng.IntN(3)}
	}
	for _, c := range []files.Cell{cell(hd, files.Missing), cell(small, files.Missing)} {
		if rng.IntN(3) > 0 {
			g.c.open = append(g.c.open, c)
		}
	}
	hashes := []string{"", "a", "b", "c"}
	ls := make([]listed, 1+rng.IntN(8))
	for i := range ls {
		k := rng.IntN(len(pool))
		id := ids[rng.IntN(len(ids))]
		r := indexers.Release{Indexer: id, Priority: priority[id], Release: searching.Release{
			Title: pool[k].title, IDs: pool[k].ids, Seeders: rng.IntN(4), Size: int64(10 * (1 + rng.IntN(2))),
			Link: searching.LinkOf("https://site.test/dl/" + strconv.Itoa(i)),
		}}
		if h := hashes[rng.IntN(len(hashes))]; h != "" {
			hash := bencode.InfoHash([]byte(h + "aaaaaaaaaaaaaaaaaaa"))
			r.InfoHash = &hash
			if rng.IntN(3) == 0 {
				r.Link, r.Magnet = searching.Link{}, searching.LinkOf("magnet:?xt=urn:btih:"+hash.String())
			}
		}
		ls[i] = listed{Release: r, at: i}
		if rng.IntN(5) == 0 {
			g.gone[i] = true
		}
	}
	var as []assessed
	for _, l := range dedupe(ls) {
		k := slices.IndexFunc(pool, func(p poolTitle) bool { return p.title == l.Release.Title })
		as = append(as, assessRead(orchard, l, reads[k]))
	}
	g.picks = decide(g.c, as).picks(g.c)
	return g
}

// TestPlanRowOneIsChoose holds Manual search's plan to automatic search's
// take. Over generated searches, choosing with a taker that only reads the
// Indexers' standings offers the very Release that Take's reservations
// take, and reserves nothing. That Release is the first offer whose
// Indexer would fetch it, each pick in rank order offered before its
// copies, a position gone never offered.
func TestPlanRowOneIsChoose(t *testing.T) {
	reads := make([]release.Read, len(pool))
	for i, p := range pool {
		reads[i] = release.Title(p.title, orchard.names, orchard.cuts)
	}
	ids := []indexers.ID{{UUID: storage.NewUUID()}, {UUID: storage.NewUUID()}, {UUID: storage.NewUUID()}}
	rng := rand.New(rand.NewPCG(32, 157)) //nolint:gosec // seeded on purpose so a failure replays
	var chosen, none int
	for n := range 2000 {
		if choseOne(t, n, generate(rng, reads, ids)) {
			chosen++
		} else {
			none++
		}
	}
	if chosen < 200 || none < 200 {
		t.Fatalf("the generator chose in %d cases and none in %d, want both often", chosen, none)
	}
}

// choseOne checks choose over case n, g, and reports whether it took a
// Release.
func choseOne(t *testing.T, n int, g generated) bool {
	t.Helper()
	before := maps.Clone(g.sts)
	read, readTicket, readOK, err := choose(g.picks, g.gone, reading(g.sts))
	if err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(g.sts, before) {
		t.Fatalf("case %d: reading the standings spent a grab: %v, was %v", n, g.sts, before)
	}
	took, ticket, ok, err := choose(g.picks, g.gone, reserving(g.sts))
	if err != nil {
		t.Fatal(err)
	}
	if ok != readOK || took.Release.Title != read.Release.Title || took.at != read.at || ticket != readTicket {
		t.Fatalf("case %d: reading the standings chose %q at %d (%t), Take took %q at %d (%t)",
			n, read.Release.Title, read.at, readOK, took.Release.Title, took.at, ok)
	}
	want, wantOK := firstTakeable(g, before)
	if ok != wantOK || took.at != want.at {
		t.Fatalf("case %d: choose took %q at %d (%t), want %q at %d (%t): the first offer not gone whose Indexer takes it",
			n, took.Release.Title, took.at, ok, want.Release.Title, want.at, wantOK)
	}
	spent := 0
	for id, s := range g.sts {
		spent += before[id].grabs - s.grabs
	}
	if wantSpent := map[bool]int{true: 1, false: 0}[ok && !took.Release.Offered().IsMagnet()]; spent != wantSpent {
		t.Fatalf("case %d: Take spent %d grabs taking %q, want %d", n, spent, took.Release.Title, wantSpent)
	}
	return ok
}

// firstTakeable is the first of g's picks, or of its copies, not gone
// whose Indexer at sts takes it.
func firstTakeable(g generated, sts map[indexers.ID]standing) (Pick, bool) {
	for _, pick := range g.picks {
		for _, p := range pick.offers() {
			if !g.gone[p.at] && sts[p.Release.Indexer].refuses(p.Release) == nil {
				return p, true
			}
		}
	}
	return Pick{}, false
}
