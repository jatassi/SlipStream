package scanning

import (
	"errors"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// Recorded is one files_file row of a root, as the compare reads it.
type Recorded struct {
	ID   storage.UUID
	Work works.WorkID
	Path storing.Rel // where its bytes belong
	Sits storing.Rel // where they are now: Path unless a move is pending
	Next storing.Rel // the disk queue's next hop, zero for none
	Size int64
	Seen *storing.Seen // nil until a scan or an import recorded it
	Gone bool
	// Stale is a media record older than the parser's.
	Stale bool
}

// ErrUnfinished is a walk whose last page isn't Done.
var ErrUnfinished = errors.New("the walk stopped before its last page")

// Listing is a whole walk. Only Finish makes one, and only from pages that
// end Done, so an unfinished walk can never reach Compare and mark
// anything gone.
type Listing struct {
	entries  []storing.Walked
	unread   []storing.Unread
	caseFold bool
}

func Finish(pages []storing.Page) (Listing, error) {
	if len(pages) == 0 || !pages[len(pages)-1].Done {
		return Listing{}, ErrUnfinished
	}
	var l Listing
	for _, p := range pages {
		l.entries = append(l.entries, p.Entries...)
		l.unread = append(l.unread, p.Unread...)
	}
	l.caseFold = pages[len(pages)-1].CaseFold
	return l, nil
}

// Len is how many files the walk listed.
func (l Listing) Len() int { return len(l.entries) }

// Known is what the Controller records of a root besides its files.
type Known struct {
	// Held are paths whose bytes the disk queue still moves or removes:
	// files_move.sits, files_doomed.path and .sits. A walked path here is
	// neither found nor a move target.
	Held []storing.Rel
	// Listed are the files Pending imports list, with what the walk read
	// then. A walked path listed with the same size and modification time
	// is Listed, never Found. A zero modification time is a listing made
	// before the scan recorded one, and matches any.
	Listed []storing.Walked
}

// Guard is the stop rule. Share is library.scan.stopShare, 1 to 100: a
// comparison that would mark more than Share percent of the root's live
// files gone marks none, and 100 never stops on the share. Every Work
// folder absent stops at any Share. Confirmed is Mark them gone: the count
// the Admin confirmed, so a comparison with at most that many gone files
// passes, and one that finds more stops again with the new count instead
// of marking a vanish the Admin never saw. Zero when nothing is confirmed.
type Guard struct {
	Share     int
	Confirmed int
}

// Plan is every transition of one root. Each recorded file is in at most
// one list, and each walked entry in at most one.
type Plan struct {
	// Same is a recorded file at its path, or where its bytes sit or go
	// next, with its size. Returned when it was gone.
	Same []Same
	// Changed is a recorded file at its path with another size: probed
	// again, its quality read again, its Grab link dropped.
	Changed []Same
	// Moved is a recorded file absent from its path and found elsewhere
	// by its identity, or on a volume without ids by its size and base
	// name: its path changes, nothing else. Includes a case-only or
	// NFD/NFC-only respelling.
	Moved []Moved
	Gone  Gone
	// Held are live recorded files under a folder the walk couldn't read:
	// never gone this time.
	Held []Recorded
	// Found are walked files no record, held path or listing names.
	Found []storing.Walked
	// Listed are walked files Pending imports already list unchanged.
	Listed []storing.Walked
	// Vanished are listed paths the walk didn't find.
	Vanished []storing.Rel
	Unread   []storing.Unread
}

// Same is a recorded file and where the walk found it.
type Same struct {
	Rec      Recorded
	At       storing.Walked
	Reprobe  Reprobe
	Returned bool
}

// Reprobe is why a file's media record is read again.
type Reprobe uint8

const (
	Keep    Reprobe = iota // nothing to read again
	Touched                // its modification time changed
	Stale                  // its media record is older than the parser's
	Resized                // its size changed
)

type Moved struct {
	Rec      Recorded
	To       storing.Walked
	Returned bool
}

// Gone is what a comparison does with the live recorded files it didn't
// find: Marked or Stopped.
type Gone interface{ gone() }

// Marked are the gone candidates: the unit confirms each with VerifyIn,
// and files.Scan re-checks each in its Write.
type Marked struct{ Files []Recorded }

// Stopped is the guard holding every gone candidate back: Gone of the
// root's Of live files.
type Stopped struct{ Gone, Of int }

func (Marked) gone()  {}
func (Stopped) gone() {}

// Compare decides every transition of one root. Paths compare in their
// NFC form, and case-folded too when the walk read the root
// case-insensitive. Moves are matched before gone and found.
func Compare(l Listing, recorded []Recorded, k Known, g Guard) Plan {
	c := newComparison(l)
	p := Plan{Unread: l.unread}
	var absent []Recorded
	for _, r := range recorded {
		e, matched, ok := c.find(r)
		switch {
		case !ok:
			absent = append(absent, r)
		case e.Size != r.Size:
			p.Changed = append(p.Changed, Same{Rec: r, At: e, Reprobe: Resized, Returned: r.Gone})
		case e.Path != matched:
			p.Moved = append(p.Moved, Moved{Rec: r, To: e, Returned: r.Gone})
		default:
			p.Same = append(p.Same, Same{Rec: r, At: e, Reprobe: reprobeOf(r, e), Returned: r.Gone})
		}
	}
	moved, absent := c.moves(absent)
	p.Moved = append(p.Moved, moved...)
	var gone []Recorded
	for _, r := range absent {
		if r.Gone {
			continue
		}
		if c.unreadHolds(r) {
			p.Held = append(p.Held, r)
		} else {
			gone = append(gone, r)
		}
	}
	p.Found, p.Listed, p.Vanished = c.unclaimed(k)
	p.Gone = g.judge(gone, recorded, c)
	return p
}

func reprobeOf(r Recorded, e storing.Walked) Reprobe {
	switch {
	case r.Seen != nil && !r.Seen.Modified.Equal(e.Seen.Modified):
		return Touched
	case r.Stale:
		return Stale
	}
	return Keep
}

type comparison struct {
	l       Listing
	byKey   map[string]int
	claimed []bool
}

func newComparison(l Listing) *comparison {
	c := &comparison{l: l, byKey: make(map[string]int, len(l.entries)), claimed: make([]bool, len(l.entries))}
	for i, e := range l.entries {
		k := c.key(e.Path.String())
		if _, dup := c.byKey[k]; !dup {
			c.byKey[k] = i
		}
	}
	return c
}

func (c *comparison) key(path string) string {
	if c.l.caseFold {
		return storing.Fold(path)
	}
	return norm.NFC.String(path)
}

// find is the unclaimed entry where r's bytes are, looked for where they
// sit, then where they go next, then where they belong, and the path it
// was found by.
func (c *comparison) find(r Recorded) (storing.Walked, storing.Rel, bool) {
	for _, at := range []storing.Rel{r.Sits, r.Next, r.Path} {
		if at == (storing.Rel{}) {
			continue
		}
		if i, ok := c.byKey[c.key(at.String())]; ok && !c.claimed[i] {
			c.claimed[i] = true
			return c.l.entries[i], at, true
		}
	}
	return storing.Walked{}, storing.Rel{}, false
}

// moves matches absent records to unclaimed entries: by file id and size,
// then, where either side has no id, by size and folded base name when
// exactly one record and one entry share them. It answers the moves and
// the records still absent.
func (c *comparison) moves(absent []Recorded) ([]Moved, []Recorded) {
	byID, rest := c.movesByID(absent)
	bySize, still := c.movesBySizeAndName(rest)
	return append(byID, bySize...), still
}

func (c *comparison) take(r Recorded, i int) Moved {
	c.claimed[i] = true
	return Moved{Rec: r, To: c.l.entries[i], Returned: r.Gone}
}

func (c *comparison) movesByID(absent []Recorded) ([]Moved, []Recorded) {
	byID := map[storing.FileID][]int{}
	for i, e := range c.l.entries {
		if !c.claimed[i] && !e.Seen.ID.IsZero() {
			byID[e.Seen.ID] = append(byID[e.Seen.ID], i)
		}
	}
	var moved []Moved
	var rest []Recorded
	for _, r := range absent {
		if r.Seen != nil && !r.Seen.ID.IsZero() {
			if at := byID[r.Seen.ID]; len(at) == 1 && c.l.entries[at[0]].Size == r.Size {
				moved = append(moved, c.take(r, at[0]))
				continue
			}
		}
		rest = append(rest, r)
	}
	return moved, rest
}

type sizeName struct {
	size int64
	name string
}

func (c *comparison) movesBySizeAndName(absent []Recorded) ([]Moved, []Recorded) {
	recs, ents := map[sizeName][]int{}, map[sizeName][]int{}
	for i, r := range absent {
		k := sizeName{r.Size, c.baseKey(r.Path)}
		recs[k] = append(recs[k], i)
	}
	for i, e := range c.l.entries {
		if !c.claimed[i] {
			k := sizeName{e.Size, c.baseKey(e.Path)}
			ents[k] = append(ents[k], i)
		}
	}
	var moved []Moved
	matched := make([]bool, len(absent))
	for k, ri := range recs {
		ei := ents[k]
		if len(ri) != 1 || len(ei) != 1 {
			continue
		}
		r, e := absent[ri[0]], c.l.entries[ei[0]]
		if r.Seen == nil || r.Seen.ID.IsZero() || e.Seen.ID.IsZero() {
			moved = append(moved, c.take(r, ei[0]))
			matched[ri[0]] = true
		}
	}
	var still []Recorded
	for i, r := range absent {
		if !matched[i] {
			still = append(still, r)
		}
	}
	return moved, still
}

func (c *comparison) baseKey(p storing.Rel) string {
	s := p.String()
	return storing.Fold(s[strings.LastIndexByte(s, '/')+1:])
}

// unreadHolds reports whether r's bytes may lie under a folder the walk
// couldn't read.
func (c *comparison) unreadHolds(r Recorded) bool {
	for _, u := range c.l.unread {
		folder := c.key(u.Path)
		for _, at := range []storing.Rel{r.Sits, r.Next, r.Path} {
			k := c.key(at.String())
			if at != (storing.Rel{}) && (k == folder || strings.HasPrefix(k, folder+"/")) {
				return true
			}
		}
	}
	return false
}

// unclaimed sorts the entries no record claimed into found and listed, and
// names the listed paths the walk didn't find.
func (c *comparison) unclaimed(k Known) (found, listed []storing.Walked, vanished []storing.Rel) {
	held := map[string]bool{}
	for _, h := range k.Held {
		held[c.key(h.String())] = true
	}
	pending := make(map[string]storing.Walked, len(k.Listed))
	for _, w := range k.Listed {
		key := c.key(w.Path.String())
		pending[key] = w
		if _, ok := c.byKey[key]; !ok {
			vanished = append(vanished, w.Path)
		}
	}
	for i, e := range c.l.entries {
		key := c.key(e.Path.String())
		if c.claimed[i] || held[key] {
			continue
		}
		if w, ok := pending[key]; ok && w.Size == e.Size && (w.Seen.Modified.IsZero() || w.Seen.Modified.Equal(e.Seen.Modified)) {
			listed = append(listed, e)
			continue
		}
		found = append(found, e)
	}
	return found, listed, vanished
}

func (g Guard) judge(gone, recorded []Recorded, c *comparison) Gone {
	if len(gone) <= g.Confirmed {
		return Marked{Files: gone}
	}
	live := 0
	for _, r := range recorded {
		if !r.Gone {
			live++
		}
	}
	if (g.Share < 100 && len(gone)*100 > g.Share*live) || c.everyFolderAbsent(recorded) {
		return Stopped{Gone: len(gone), Of: live}
	}
	return Marked{Files: gone}
}

// everyFolderAbsent reports whether no walked entry lies in any Work
// folder a live record names: the first folder of its path. A root whose
// live files all lie loose at its top has no Work folder.
func (c *comparison) everyFolderAbsent(recorded []Recorded) bool {
	folders := map[string]bool{}
	for _, r := range recorded {
		if folder, ok := firstFolder(c.key(r.Path.String())); ok && !r.Gone {
			folders[folder] = true
		}
	}
	if len(folders) == 0 {
		return false
	}
	for k := range c.byKey {
		if folder, ok := firstFolder(k); ok && folders[folder] {
			return false
		}
	}
	return true
}

func firstFolder(path string) (string, bool) {
	folder, _, ok := strings.Cut(path, "/")
	return folder, ok
}

// Share is one Work's part of a plan's recorded transitions, with the gone
// candidates VerifyIn confirmed.
type Share struct {
	Same    []Same
	Changed []Same
	Moved   []Moved
	Gone    []Recorded
}

// ByWork splits p's recorded transitions by Work, keeping only the gone
// candidates in confirmed. A Same with nothing to write, Keep with its
// stamps already recorded, is left out, and so is a Work left with none.
func (p Plan) ByWork(confirmed []Recorded) map[works.WorkID]Share {
	out := map[works.WorkID]Share{}
	add := func(w works.WorkID, f func(*Share)) {
		s := out[w]
		f(&s)
		out[w] = s
	}
	for _, s := range p.Same {
		if s.Reprobe != Keep || s.Returned || s.Rec.Seen == nil || !s.Rec.Seen.Same(s.At.Seen) {
			add(s.Rec.Work, func(sh *Share) { sh.Same = append(sh.Same, s) })
		}
	}
	for _, s := range p.Changed {
		add(s.Rec.Work, func(sh *Share) { sh.Changed = append(sh.Changed, s) })
	}
	for _, m := range p.Moved {
		add(m.Rec.Work, func(sh *Share) { sh.Moved = append(sh.Moved, m) })
	}
	ok := make(map[storage.UUID]bool, len(confirmed))
	for _, r := range confirmed {
		ok[r.ID] = true
	}
	marked, _ := p.Gone.(Marked)
	for _, r := range marked.Files {
		if ok[r.ID] {
			add(r.Work, func(sh *Share) { sh.Gone = append(sh.Gone, r) })
		}
	}
	return out
}

// Reprobes are the files whose media record is read again: Changed, and
// Same with Reprobe Touched or Stale.
func (s Share) Reprobes() []storing.Walked {
	var out []storing.Walked
	for _, c := range s.Changed {
		out = append(out, c.At)
	}
	for _, same := range s.Same {
		if same.Reprobe != Keep {
			out = append(out, same.At)
		}
	}
	return out
}
