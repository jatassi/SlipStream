package scanning

import (
	"path"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// Group is one Work folder's found and listed files: the first folder
// under the root, or a file loose at the root's top on its own.
type Group struct {
	Folder string // root-relative; a loose file's own path
	Loose  bool
	Found  []storing.Walked
	Listed []storing.Walked
	// Entry is the pending entry already listing the folder, nil for none.
	Entry *Entry
}

// Name is what the group's Hint reads: the folder's name, or a loose
// file's without its extension.
func (g Group) Name() string {
	if g.Loose {
		return strings.TrimSuffix(g.Folder, path.Ext(g.Folder))
	}
	return g.Folder
}

type Entry struct {
	ID      storage.UUID
	Ignored bool
	Hint    Hint
}

// Groups puts p's Found and Listed by Work folder, in folder order.
// entries are the root's pending entries by folder. A group whose entry is
// ignored and whose files are all Listed is left out: ignored folders are
// neither probed nor searched again until their files change.
func Groups(p Plan, entries map[string]Entry) []Group {
	by := map[string]*Group{}
	of := func(w storing.Walked) *Group {
		folder, loose := workFolder(w.Path.String())
		g := by[folder]
		if g == nil {
			g = &Group{Folder: folder, Loose: loose}
			if e, ok := entries[folder]; ok {
				g.Entry = &e
			}
			by[folder] = g
		}
		return g
	}
	for _, w := range p.Found {
		g := of(w)
		g.Found = append(g.Found, w)
	}
	for _, w := range p.Listed {
		g := of(w)
		g.Listed = append(g.Listed, w)
	}
	out := make([]Group, 0, len(by))
	for _, g := range by {
		if g.Entry != nil && g.Entry.Ignored && len(g.Found) == 0 {
			continue
		}
		out = append(out, *g)
	}
	slices.SortFunc(out, func(a, b Group) int { return strings.Compare(a.Folder, b.Folder) })
	return out
}

// workFolder is the first folder of p, or p itself when it lies
// loose at the root's top.
func workFolder(p string) (folder string, loose bool) {
	if folder, ok := firstFolder(p); ok {
		return folder, false
	}
	return p, true
}

// Shelved is one library Work of the root's type that a group may join.
type Shelved struct {
	Work   works.WorkID
	Root   roots.ID // its home, zero while it has none
	Label  string
	Titles []string // every stored title beyond the label
	Claim  Claim
}

// Shelf indexes the root type's library Works for the three library rules,
// once per unit.
type Shelf struct {
	root    roots.ID
	homes   map[works.WorkID]roots.ID
	years   map[works.WorkID]int
	folders map[string][]works.WorkID
	ids     map[ID][]works.WorkID
	titles  map[string][]works.WorkID
}

// NewShelf indexes ws, and the folder of each of the root's recorded
// files, gone ones included.
func NewShelf(root roots.ID, ws []Shelved, recorded []Recorded) *Shelf {
	s := &Shelf{root: root, homes: map[works.WorkID]roots.ID{}, years: map[works.WorkID]int{}, folders: map[string][]works.WorkID{}, ids: map[ID][]works.WorkID{}, titles: map[string][]works.WorkID{}}
	for _, r := range recorded {
		folder, _ := workFolder(r.Path.String())
		index(s.folders, norm.NFC.String(folder), r.Work)
	}
	for _, w := range ws {
		s.Add(w)
	}
	return s
}

// Add takes a Work the unit just added, so a later group of the same unit
// sees it.
func (s *Shelf) Add(w Shelved) {
	s.homes[w.Work], s.years[w.Work] = w.Root, w.Claim.Year
	for _, id := range w.Claim.IDs {
		index(s.ids, id, w.Work)
	}
	for _, t := range append([]string{w.Label}, w.Titles...) {
		if f := Fold(t); f != "" {
			index(s.titles, f, w.Work)
		}
	}
}

func index[K comparable](m map[K][]works.WorkID, k K, w works.WorkID) {
	if !slices.Contains(m[k], w) {
		m[k] = append(m[k], w)
	}
}

// Matched is the first of the three library rules that names exactly one
// Work. Zero Work when none does.
type Matched struct {
	Work works.WorkID
	Rule Rule
	// Elsewhere is the Work's home when it isn't the scanned root: the
	// group waits with Elsewhere and offers only Ignore.
	Elsewhere *roots.ID
}

type Rule uint8

const (
	ByFolder Rule = iota + 1 // a Work has a recorded file in the folder
	ByTag                    // an id tag one Work claims
	ByTitle                  // a folded title and year one Work holds
)

// Match applies the library rules in order: a recorded file in folder, an
// id tag, then the title and year. A hint with no year matches by title
// only when exactly one Work has that title.
func (s *Shelf) Match(folder string, h Hint) Matched {
	if w, ok := one(s.folders[norm.NFC.String(folder)]); ok {
		return s.matched(w, ByFolder)
	}
	var tagged []works.WorkID
	for _, id := range h.IDs {
		for _, w := range s.ids[id] {
			if !slices.Contains(tagged, w) {
				tagged = append(tagged, w)
			}
		}
	}
	if w, ok := one(tagged); ok {
		return s.matched(w, ByTag)
	}
	titled := s.titles[Fold(h.Title)]
	if h.Year != 0 {
		titled = slices.DeleteFunc(slices.Clone(titled), func(w works.WorkID) bool { return s.years[w] != h.Year })
	}
	if w, ok := one(titled); ok {
		return s.matched(w, ByTitle)
	}
	return Matched{}
}

func (s *Shelf) matched(w works.WorkID, r Rule) Matched {
	m := Matched{Work: w, Rule: r}
	if home := s.homes[w]; !home.IsZero() && home != s.root {
		m.Elsewhere = &home
	}
	return m
}

func one(ws []works.WorkID) (works.WorkID, bool) {
	if len(ws) != 1 {
		return works.WorkID{}, false
	}
	return ws[0], true
}

// Candidate is a source search result as the confidence rule reads it.
type Candidate struct {
	Key    works.Key
	Titles []string // label and original label
	Year   int      // Facet.YearOf
	// WorkID is set when the candidate is already in the library.
	WorkID works.WorkID
}

// Confident is the index of the one candidate a hint names with
// confidence, -1 for none: the one whose key an id tag names (Hint.Key),
// or, when the hint has a year, the one candidate with a folded title and
// the year equal to the hint's. A hint with no year and no key is never
// confident.
func Confident(h Hint, cs []Candidate) int {
	if h.Key != "" {
		if i := slices.IndexFunc(cs, func(c Candidate) bool { return c.Key == h.Key }); i >= 0 {
			return i
		}
	}
	if h.Year != 0 {
		title := Fold(h.Title)
		match := -1
		for i, c := range cs {
			if c.Year != h.Year || !slices.ContainsFunc(c.Titles, func(t string) bool { return Fold(t) == title }) {
				continue
			}
			if match >= 0 {
				return -1
			}
			match = i
		}
		if match >= 0 {
			return match
		}
	}
	return -1
}

// Fold is a title as the rules compare it: letters and digits only, lower
// case, NFC. Accents stay.
func Fold(title string) string {
	var b strings.Builder
	for _, r := range norm.NFC.String(title) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}
