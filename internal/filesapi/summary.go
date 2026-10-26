package filesapi

import (
	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/works"
)

type summary struct {
	Status     api.Nullable[api.Enum[State]] `json:"status" doc:"The first status its Items hold, in the glossary's order; null for a Work with no cell"`
	Cells      api.Counts[State]             `json:"cells" doc:"Its cells by state"`
	Items      int                           `json:"items" minimum:"0"`
	ItemStatus api.Counts[State]             `json:"itemStatus" doc:"Its Items by status; an Item with no cell isn't counted"`
}

func summaryOf(items []works.Item, cells []files.Cell) summary {
	out := summary{Cells: api.Counts[State]{}, Items: len(items), ItemStatus: api.Counts[State]{}}
	if s, ok := files.Status(cells); ok {
		out.Status = StatusOf(s)
	}
	of := map[works.ItemID][]files.Cell{}
	for _, c := range cells {
		out.Cells[c.State.String()]++
		of[c.Item.ID] = append(of[c.Item.ID], c)
	}
	for _, it := range items {
		if s, ok := files.Status(of[it.ID]); ok {
			out.ItemStatus[s.String()]++
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
