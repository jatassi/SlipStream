package filesapi

import (
	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type summary struct {
	Status     api.Nullable[api.Enum[State]] `json:"status" doc:"The first status its Items hold, in the glossary's order; null for a Work with no cell"`
	Cells      api.Counts[State]             `json:"cells" doc:"Its cells by state"`
	Items      int                           `json:"items" minimum:"0"`
	ItemStatus api.Counts[State]             `json:"itemStatus" doc:"Its Items by status; an Item with no cell isn't counted"`
}

func summaryOf(r files.Rollup) summary {
	return summary{Status: StatusOf(r.Status), Cells: CountsOf(r.Cells), Items: r.Items, ItemStatus: CountsOf(r.ByItem)}
}

// CountsOf is c on the wire, a State with none left out.
func CountsOf(c files.Counts) api.Counts[State] {
	out := api.Counts[State]{}
	for s, n := range c {
		if n > 0 {
			out[files.State(s).String()] = n
		}
	}
	return out
}

type need struct{}

func (need) Variants() (discriminator string, variants []any) {
	return "kind", []any{failedNeed{}, upgradeNeed{}, waitingNeed{}, noRootNeed{}}
}

type failedNeed struct {
	Kind     string   `json:"kind" enum:"failed"`
	ItemIDs  []string `json:"itemIds" nullable:"false" doc:"A run of Items in tree order whose entries match"`
	Edition  string   `json:"edition" doc:"any under any cut, else the key of the cut the cells want"`
	TargetID string   `json:"targetId" format:"uuid" doc:"The Target whose download failed"`
	Waiting  []string `json:"waiting" nullable:"false" doc:"The Targets of the same Items and Edition, still missing, that wait on the failed one"`
	GrabID   string   `json:"grabId" format:"uuid" doc:"The Grab that failed"`
}

type upgradeNeed struct {
	Kind     string   `json:"kind" enum:"upgrade"`
	ItemIDs  []string `json:"itemIds" nullable:"false" doc:"A run of Items in tree order whose entries match"`
	Edition  string   `json:"edition" doc:"any under any cut, else the key of the cut the cells want"`
	Upgrades []string `json:"upgrades" nullable:"false" doc:"Every upgradable Target of the Items and Edition"`
}

type waitingNeed struct {
	Kind   string `json:"kind" enum:"waiting"`
	GrabID string `json:"grabId" format:"uuid"`
	Code   string `json:"code" doc:"Why the Grab waits, such as needs_transfer"`
}

type noRootNeed struct {
	Kind string `json:"kind" enum:"no_root" doc:"The Work has no root. Its downloads wait until it has one"`
}

var needVariants = map[files.NeedKind]func(files.Need) any{
	files.NeedFailed: func(n files.Need) any {
		return failedNeed{Kind: "failed", ItemIDs: itemIDs(n.Items), Edition: string(n.Edition), TargetID: n.Target.String(), Waiting: targetIDs(n.Waiting), GrabID: n.Grab.String()}
	},
	files.NeedUpgrade: func(n files.Need) any {
		return upgradeNeed{Kind: "upgrade", ItemIDs: itemIDs(n.Items), Edition: string(n.Edition), Upgrades: targetIDs(n.Upgrades)}
	},
	files.NeedWaiting: func(n files.Need) any { return waitingNeed{Kind: "waiting", GrabID: n.Grab.String(), Code: n.Code} },
	files.NeedNoRoot:  func(files.Need) any { return noRootNeed{Kind: "no_root"} },
}

func needsOf(ns []files.Need) []api.OneOf[need] {
	out := make([]api.OneOf[need], len(ns))
	for i, n := range ns {
		out[i] = api.NewOneOf[need](needVariants[n.Kind](n))
	}
	return out
}

func itemIDs(ids []works.ItemID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

func targetIDs(ids []targets.TargetID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}
