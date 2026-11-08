package grabs

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
)

// listed is one Release a run's Indexers listed, at its position among
// the run's Releases. copies are the same Release listed by Indexers of no
// better priority, best first, which a search takes when this one's
// Indexer won't fetch it.
type listed struct {
	Release indexers.Release
	at      int
	copies  []copied
}

// listedOf is releases at their positions.
func listedOf(releases []indexers.Release) []listed {
	out := make([]listed, len(releases))
	for i, r := range releases {
		out[i] = listed{Release: r, at: i}
	}
	return out
}

// copied is a copy of a Release and its position among the run's.
type copied struct {
	Release indexers.Release
	at      int
}

// Pick is the Release a search grabs, the Edition its cells want, every
// open cell's Target it is grabbable for, in Profile order, and the notes
// its Verdicts carry there.
type Pick struct {
	Release indexers.Release
	Edition targets.Edition
	Targets []targets.TargetID
	Notes   []Reason
	at      int
	copies  []copied
}

// offers is p, then p as each of its copies.
func (p Pick) offers() []Pick {
	out := []Pick{p}
	for _, c := range p.copies {
		out = append(out, Pick{Release: c.Release, Edition: p.Edition, Targets: p.Targets, Notes: p.Notes, at: c.at})
	}
	return out
}

// Reason is one reason or note about a Release: its own, or one of a
// cell's, which Target names.
type Reason struct {
	targets.Reason
	Target targets.TargetID
}

// Refusal is a Release a search didn't grab and why.
type Refusal struct {
	Title   string
	Reasons []Reason
}

// dedupe is ls with each Release listed twice counted once: the same info
// hash, or the same title and size when either lacks one. The copy from
// the Indexer with the better priority stays, the first of a tie, and
// holds the others as its copies.
func dedupe(ls []listed) []listed {
	sorted := slices.Clone(ls)
	slices.SortStableFunc(sorted, func(a, b listed) int { return cmp.Compare(a.Release.Priority, b.Release.Priority) })
	hashes, named, unhashed := map[bencode.InfoHash]int{}, map[titleSize]int{}, map[titleSize]int{}
	var out []listed
	for _, c := range sorted {
		r := c.Release
		name := titleSize{r.Title, r.Size}
		if kept, ok := keptCopy(r, name, hashes, named, unhashed); ok {
			out[kept].copies = append(out[kept].copies, copied{Release: r, at: c.at})
			continue
		}
		named[name] = len(out)
		if r.InfoHash == nil {
			unhashed[name] = len(out)
		} else {
			hashes[*r.InfoHash] = len(out)
		}
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b listed) int { return cmp.Compare(a.at, b.at) })
	return out
}

// keptCopy is the position in dedupe's kept Releases of the Release r,
// titled and sized name, is a copy of.
func keptCopy(r indexers.Release, name titleSize, hashes map[bencode.InfoHash]int, named, unhashed map[titleSize]int) (int, bool) {
	if r.InfoHash == nil {
		i, ok := named[name]
		return i, ok
	}
	if i, ok := hashes[*r.InfoHash]; ok {
		return i, true
	}
	i, ok := unhashed[name]
	return i, ok
}

type titleSize struct {
	title string
	size  int64
}

// cells are the searched Item's open cells, and the filled ones a Release
// is judged at only as a proper or repack of their file.
type cells struct{ open, filled []files.Cell }

// at is cell i, counting open then filled.
func (c cells) at(i int) files.Cell {
	if i < len(c.open) {
		return c.open[i]
	}
	return c.filled[i-len(c.open)]
}

// properRule is Judge's upgrade rule for the same Quality at a higher
// revision, the one rule a filled cell is judged by.
const properRule = 2

// verdict is a Release's verdict at cell, counting open then filled.
// wanted is whether the cell's Edition takes the Release's cut.
type verdict struct {
	cell   int
	target targets.TargetID
	cut    release.CutKey
	wanted bool
	v      targets.Verdict
}

func (v verdict) grabbable() bool { return v.wanted && v.v.Grabbable() }

// reasons is cut_not_wanted when the cell doesn't take the cut, then the
// Target's reasons.
func (v verdict) reasons() []Reason {
	var out []Reason
	if !v.wanted {
		out = append(out, Reason{Reason: targets.Reason{Code: targets.CutNotWanted, Value: string(v.cut)}, Target: v.target})
	}
	for _, r := range v.v.Reasons() {
		out = append(out, Reason{Reason: r, Target: v.target})
	}
	return out
}

// judged is an assessed Release and its verdict at each cell it was
// judged at. first is the first of them it is grabbable at, -1 for none.
type judged struct {
	assessed
	verdicts []verdict
	first    int
}

// reasons is j's own refusals, then each verdict's reasons.
func (j judged) reasons() []Reason {
	out := slices.Clone(j.refusals)
	for _, v := range j.verdicts {
		out = append(out, v.reasons()...)
	}
	return out
}

// pick is j as a search grabs it: at its first cell's Edition, for every
// cell's Target it is grabbable at, with their notes.
func (j judged) pick(c cells) Pick {
	p := Pick{Release: j.Release, Edition: c.at(j.first).Edition, at: j.at, copies: j.copies}
	for _, v := range j.verdicts {
		if v.grabbable() {
			p.Targets = append(p.Targets, v.target)
			p.Notes = append(p.Notes, v.reasons()...)
		}
	}
	return p
}

// Decision is decide's answer. Ranked is every Release grabbable at some
// cell, in rank order, a softly refused one included. Refused is the
// rest, in the Indexers' order.
type Decision struct {
	Ranked  []judged
	Refused []judged
}

// decide judges each assessed Release at c's cells. A hard refusal of its
// own stops judging. A soft one is recorded and the Release is judged at
// every open cell, under any cut, so a Manual search can show what it
// would fill, and at each filled cell of its cut only as a proper or
// repack of the file. Ranked is in chain order.
func decide(c cells, as []assessed) Decision {
	var d Decision
	for _, a := range as {
		j := judge(c, a)
		if j.first < 0 {
			d.Refused = append(d.Refused, j)
		} else {
			d.Ranked = append(d.Ranked, j)
		}
	}
	rank(c, d.Ranked)
	return d
}

func judge(c cells, a assessed) judged {
	j := judged{assessed: a, first: -1}
	if a.hard() {
		return j
	}
	s, cut := a.scored(), a.read.Facts.Cut
	at := func(i int, cell files.Cell) verdict {
		var held *targets.Held
		if cell.File != nil {
			held = &targets.Held{Facts: cell.File.Facts(), Score: cell.Score}
		}
		return verdict{cell: i, target: cell.Target.ID, cut: cut, wanted: cell.Edition.Takes(cut), v: targets.Judge(cell.Target, s, held)}
	}
	for i, cell := range c.open {
		j.verdicts = append(j.verdicts, at(i, cell))
	}
	for i, cell := range c.filled {
		v := at(len(c.open)+i, cell)
		if rule, _ := v.v.Upgrade(); v.wanted && rule == properRule {
			j.verdicts = append(j.verdicts, v)
		}
	}
	if i := slices.IndexFunc(j.verdicts, verdict.grabbable); i >= 0 {
		j.first = j.verdicts[i].cell
	}
	return j
}

// picks is d's Ranked an automatic search may take, which have no
// refusal of their own, in rank order.
func (d Decision) picks(c cells) []Pick {
	var out []Pick
	for _, j := range d.Ranked {
		if len(j.refusals) == 0 {
			out = append(out, j.pick(c))
		}
	}
	return out
}

// refusals is every Release of d with a refusal of its own or grabbable
// at no cell, with every reason, in the Indexers' order.
func (d Decision) refusals() []Refusal {
	refused := slices.Clone(d.Refused)
	for _, j := range d.Ranked {
		if len(j.refusals) > 0 {
			refused = append(refused, j)
		}
	}
	slices.SortFunc(refused, func(a, b judged) int { return cmp.Compare(a.at, b.at) })
	var out []Refusal
	for _, j := range refused {
		out = append(out, Refusal{Title: j.Release.Title, Reasons: j.reasons()})
	}
	return out
}

// marks is what n names, such as S01E02, in words.
func marks(n release.Name) string {
	words := make([]string, len(n.Marks))
	for i, m := range n.Marks {
		words[i] = m.Words
	}
	return strings.Join(words, " ")
}

// storedReason is a Reason as grabs_refusal and grabs_search.notes keep it.
type storedReason struct {
	Code      targets.ReasonCode `json:"code"`
	TargetID  string             `json:"targetId,omitempty"`
	Attribute targets.Attribute  `json:"attribute,omitempty"`
	Value     string             `json:"value,omitempty"`
}

func encodeReasons(rs []Reason) (string, error) {
	stored := make([]storedReason, len(rs))
	for i, r := range rs {
		stored[i] = storedReason{Code: r.Code, Attribute: r.Attribute, Value: r.Value}
		if !r.Target.IsZero() {
			stored[i].TargetID = r.Target.String()
		}
	}
	b, err := json.Marshal(stored)
	if err != nil {
		return "", fmt.Errorf("encode reasons: %w", err)
	}
	return string(b), nil
}

func decodeReasons(s string) ([]Reason, error) {
	var stored []storedReason
	if err := json.Unmarshal([]byte(s), &stored); err != nil {
		return nil, fmt.Errorf("decode reasons: %w", err)
	}
	out := make([]Reason, len(stored))
	for i, r := range stored {
		out[i] = Reason{Reason: targets.Reason{Code: r.Code, Attribute: r.Attribute, Value: r.Value}}
		if r.TargetID != "" {
			u, err := storage.ParseUUID(r.TargetID)
			if err != nil {
				return nil, fmt.Errorf("decode reasons: %w", err)
			}
			out[i].Target = targets.TargetID{UUID: u}
		}
	}
	return out, nil
}
