package grabs

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// candidate is one listed release and its Facts: the title's quality and
// cut, the Indexer's advertised size, and the searched Item as Coverage.
// at is its position among the run's Releases.
type candidate struct {
	Release indexers.Release
	Facts   release.Facts
	at      int
}

// candidateOf reads a listed release's Facts with cuts, every cut its
// Work's titles can name: it covers item, whose runtime its Content Type
// answers, unknown when it can't.
func candidateOf(r indexers.Release, cuts []release.Cut, item works.ItemID, runtime release.Fact[time.Duration]) candidate {
	f := release.Title(r.Title, cuts)
	if r.Size > 0 {
		f.Size = release.Known(r.Size)
	}
	f.Coverage = release.Coverage{Items: []works.ItemID{item}, Runtime: runtime}
	return candidate{Release: r, Facts: f}
}

// Pick is the candidate a search grabs, the Edition its cells want, every
// open cell's Target it is grabbable for, in Profile order, and the notes
// its Verdicts carry there.
type Pick struct {
	Release indexers.Release
	Edition targets.Edition
	Targets []targets.TargetID
	Notes   []Reason
	at      int
}

// WrongWork is a release carrying an id that names another Work, and
// CutNotWanted one whose cut no open cell wants.
const (
	WrongWork    targets.ReasonCode = "wrong_work"
	CutNotWanted targets.ReasonCode = "cut_not_wanted"
)

// Reason is one reason or note about a candidate: its own, or one of a
// Target's Verdict, which Target names.
type Reason struct {
	targets.Reason
	Target targets.TargetID
}

// Refusal is a candidate a search didn't grab and why.
type Refusal struct {
	Title   string
	Reasons []Reason
}

// dedupe is candidates with each Release listed twice counted once: the
// same info hash, or the same title and size when either lacks one. The
// copy from the Indexer with the better priority stays, the first of a
// tie.
func dedupe(candidates []candidate) []candidate {
	sorted := slices.Clone(candidates)
	slices.SortStableFunc(sorted, func(a, b candidate) int { return cmp.Compare(a.Release.Priority, b.Release.Priority) })
	hashes, named, unhashed := map[bencode.InfoHash]bool{}, map[titleSize]bool{}, map[titleSize]bool{}
	var out []candidate
	for _, c := range sorted {
		r := c.Release
		name := titleSize{r.Title, r.Size}
		if r.InfoHash == nil && named[name] || r.InfoHash != nil && (hashes[*r.InfoHash] || unhashed[name]) {
			continue
		}
		named[name] = true
		if r.InfoHash == nil {
			unhashed[name] = true
		} else {
			hashes[*r.InfoHash] = true
		}
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b candidate) int { return cmp.Compare(a.at, b.at) })
	return out
}

type titleSize struct {
	title string
	size  int64
}

// decide ranks the candidates grabbable for open, one Item's cells a
// search may fill, in Edition then Profile order. Each candidate is judged
// against the Target of every open cell its cut can fill, any cell under
// any cut, and the file filling the cell; one with an id that differs from
// the same id in terms, whose cut no open cell wants, or grabbable for
// none, is refused with every reason. The rest rank by the first open cell
// they are grabbable for, then their tier there, then their Indexer's
// priority, then seeders and size descending, then title, so ties replay.
func decide(open []files.Cell, terms indexers.Terms, candidates []candidate) ([]Pick, []Refusal) {
	var keep []ranked
	var refused []Refusal
	for _, c := range candidates {
		if wrongWork(c.Release, terms) {
			refused = append(refused, Refusal{c.Release.Title, []Reason{{Reason: targets.Reason{Code: WrongWork}}}})
			continue
		}
		if !slices.ContainsFunc(open, func(cell files.Cell) bool { return cell.Edition.Takes(c.Facts.Cut) }) {
			refused = append(refused, Refusal{c.Release.Title, []Reason{{Reason: targets.Reason{Code: CutNotWanted, Value: string(c.Facts.Cut)}}}})
			continue
		}
		r, reasons := judged(open, c)
		if r.cell < 0 {
			refused = append(refused, Refusal{c.Release.Title, reasons})
			continue
		}
		keep = append(keep, r)
	}
	slices.SortStableFunc(keep, func(a, b ranked) int {
		return cmp.Or(
			cmp.Compare(a.cell, b.cell),
			cmp.Compare(a.rank, b.rank),
			cmp.Compare(a.pick.Release.Priority, b.pick.Release.Priority),
			cmp.Compare(b.pick.Release.Seeders, a.pick.Release.Seeders),
			cmp.Compare(b.pick.Release.Size, a.pick.Release.Size),
			strings.Compare(a.pick.Release.Title, b.pick.Release.Title),
		)
	})
	picks := make([]Pick, len(keep))
	for i, r := range keep {
		picks[i] = r.pick
	}
	return picks, refused
}

// ranked is a candidate as decide ranks it: the pick it would make, the
// first open cell it is grabbable for, -1 for none, and its tier there.
type ranked struct {
	pick       Pick
	cell, rank int
}

// judged judges c against the Target of every open cell its cut can fill,
// and answers every reason given.
func judged(open []files.Cell, c candidate) (ranked, []Reason) {
	r := ranked{pick: Pick{Release: c.Release, at: c.at}, cell: -1}
	var reasons []Reason
	for i, cell := range open {
		if !cell.Edition.Takes(c.Facts.Cut) {
			continue
		}
		var held *release.Facts
		if cell.File != nil {
			held = new(cell.File.Facts())
		}
		v := targets.Judge(cell.Target, c.Facts, held)
		var said []Reason
		for _, reason := range v.Reasons() {
			said = append(said, Reason{Reason: reason, Target: cell.Target.ID})
		}
		reasons = append(reasons, said...)
		if !v.Grabbable() {
			continue
		}
		if r.cell < 0 {
			rank, _ := v.Rank()
			r.cell, r.rank, r.pick.Edition = i, int(rank), cell.Edition
		}
		r.pick.Targets = append(r.pick.Targets, cell.Target.ID)
		r.pick.Notes = append(r.pick.Notes, said...)
	}
	return r, reasons
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

func wrongWork(r indexers.Release, terms indexers.Terms) bool {
	for k, v := range r.IDs {
		if want, ok := terms.IDs[k]; ok && want != v {
			return true
		}
	}
	return false
}
