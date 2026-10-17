package grabs

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// candidate is one listed release and its Facts: the title's quality and
// cut, the Indexer's advertised size, and the searched Item as Coverage.
type candidate struct {
	Item  indexers.Item
	Facts release.Facts
}

// candidateOf reads a listed release's Facts: it covers item, whose
// runtime its Content Type answers, unknown when it can't.
func candidateOf(it indexers.Item, item works.ItemID, runtime release.Fact[time.Duration]) candidate {
	f := release.Title(it.Title, release.BuiltIn())
	if it.Size > 0 {
		f.Size = release.Known(it.Size)
	}
	f.Coverage = release.Coverage{Items: []works.ItemID{item}, Runtime: runtime}
	return candidate{Item: it, Facts: f}
}

// Pick is the candidate a search grabs, every open cell's Target it is
// grabbable for, in Profile order, and the notes its Verdicts carry there.
type Pick struct {
	Item    indexers.Item
	Targets []targets.TargetID
	Notes   []Reason
}

// WrongWork is a release carrying an id that names another Work.
const WrongWork targets.ReasonCode = "wrong_work"

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

// decide picks the best candidate for open, one Item's cells a search may
// fill, in Profile order. Each candidate is judged against every open
// cell's Target and the file filling the cell; one with an id that
// differs from the same id in terms, or grabbable for none, is refused
// with every reason. The rest rank by the first open cell they are
// grabbable for, then their tier there, then seeders and size descending,
// then title, so ties replay.
func decide(open []files.Cell, terms indexers.Terms, candidates []candidate) (*Pick, []Refusal) {
	type ranked struct {
		pick       Pick
		cell, rank int
	}
	var keep []ranked
	var refused []Refusal
	for _, c := range candidates {
		if wrongWork(c.Item, terms) {
			refused = append(refused, Refusal{c.Item.Title, []Reason{{Reason: targets.Reason{Code: WrongWork}}}})
			continue
		}
		r := ranked{pick: Pick{Item: c.Item}, cell: -1}
		var reasons []Reason
		for i, cell := range open {
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
				r.cell, r.rank = i, int(rank)
			}
			r.pick.Targets = append(r.pick.Targets, cell.Target.ID)
			r.pick.Notes = append(r.pick.Notes, said...)
		}
		if r.cell < 0 {
			refused = append(refused, Refusal{c.Item.Title, reasons})
			continue
		}
		keep = append(keep, r)
	}
	if len(keep) == 0 {
		return nil, refused
	}
	best := slices.MinFunc(keep, func(a, b ranked) int {
		return cmp.Or(
			cmp.Compare(a.cell, b.cell),
			cmp.Compare(a.rank, b.rank),
			cmp.Compare(b.pick.Item.Seeders, a.pick.Item.Seeders),
			cmp.Compare(b.pick.Item.Size, a.pick.Item.Size),
			strings.Compare(a.pick.Item.Title, b.pick.Item.Title),
		)
	})
	return &best.pick, refused
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

func wrongWork(it indexers.Item, terms indexers.Terms) bool {
	for k, v := range it.IDs {
		if want, ok := terms.IDs[k]; ok && want != v {
			return true
		}
	}
	return false
}
