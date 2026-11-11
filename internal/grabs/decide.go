package grabs

import (
	"cmp"
	"encoding/json"
	"errors"
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
func (c cells) at(i int) *files.Cell {
	if i < len(c.open) {
		return &c.open[i]
	}
	return &c.filled[i-len(c.open)]
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
	*assessed
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
	js := make([]judged, len(as))
	for i := range as {
		js[i] = judge(c, &as[i])
	}
	return decided(c, js)
}

// decided is decide's answer over js, each judged at c's cells.
func decided(c cells, js []judged) Decision {
	var d Decision
	for _, j := range js {
		if j.first < 0 {
			d.Refused = append(d.Refused, j)
		} else {
			d.Ranked = append(d.Ranked, j)
		}
	}
	rank(c, d.Ranked)
	return d
}

func judge(c cells, a *assessed) judged {
	j := judged{assessed: a, first: -1}
	if a.hard() {
		return j
	}
	cut := a.read.Facts.Cut
	at := func(i int, cell files.Cell) verdict {
		var held *targets.Held
		if cell.File != nil {
			held = &targets.Held{Facts: cell.File.Facts(), Score: cell.Score}
		}
		return verdict{cell: i, target: cell.Target.ID, cut: cut, wanted: cell.Edition.Takes(cut), v: targets.Judge(cell.Target, a.scored, held)}
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
	j.first = firstGrabbable(j.verdicts)
	return j
}

// only is j, judged at open cells only, as judge judges it at the cells
// ks names, in the order judged: its verdicts there, each cell counted by
// its place in ks.
func (j judged) only(ks []int) judged {
	out := judged{assessed: j.assessed}
	for _, v := range j.verdicts {
		if i := slices.Index(ks, v.cell); i >= 0 {
			v.cell = i
			out.verdicts = append(out.verdicts, v)
		}
	}
	out.first = firstGrabbable(out.verdicts)
	return out
}

// firstGrabbable is the cell of the first of vs that is grabbable, -1 for
// none.
func firstGrabbable(vs []verdict) int {
	if i := slices.IndexFunc(vs, verdict.grabbable); i >= 0 {
		return vs[i].cell
	}
	return -1
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

// taker readies r's fetch: its Ticket, or *indexers.NotTakenError while
// r's Indexer wouldn't fetch it now. Automatic search takes with
// Indexers.Take in its Write, which reserves a grab; Manual search's plan
// takes by reading the Indexers' standings, which reserves nothing.
type taker func(r indexers.Release) (indexers.Ticket, error)

// choose is the first of picks, or of its copies, that take takes,
// skipping the positions gone, and the Ticket take readied it with. ok is
// false when take takes none. Automatic search and Manual search's plan
// both choose with it, so the two never walk the offers apart.
func choose(picks []Pick, gone map[int]bool, take taker) (Pick, indexers.Ticket, bool, error) {
	for _, pick := range picks {
		for _, p := range pick.offers() {
			if gone[p.at] {
				continue
			}
			t, err := take(p.Release)
			if errors.As(err, new(*indexers.NotTakenError)) {
				continue
			}
			return p, t, err == nil, err
		}
	}
	return Pick{}, indexers.Ticket{}, false, nil
}

// marks is what n names, such as S01E02, in words.
func marks(n release.Name) string {
	words := make([]string, len(n.Marks))
	for i, m := range n.Marks {
		words[i] = m.Words
	}
	return strings.Join(words, " ")
}

// storedReason is a Reason as grabs_release.reasons and grabs_search.notes keep it.
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
