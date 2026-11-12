package filesapi

import (
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type summary struct {
	Status     api.Nullable[api.Enum[State]] `json:"status" doc:"The first status its monitored Items hold, in the glossary's order; null while none holds a cell"`
	Cells      api.Counts[State]             `json:"cells" doc:"Its monitored Items' cells by state"`
	Items      int                           `json:"items" minimum:"0" doc:"Its monitored Items that its source still lists, the only Items the status and counts read"`
	ItemStatus api.Counts[State]             `json:"itemStatus" doc:"Its monitored Items by status; an Item with no cell isn't counted"`
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
	Kind        string                    `json:"kind" enum:"waiting"`
	GrabID      string                    `json:"grabId" format:"uuid"`
	Code        string                    `json:"code" enum:"node_away,client_away,client_off,no_space,root_not_ready,probe_timeout,client_error" doc:"Why the Grab waits: node_away while its Node, or the Node of the client its fetch goes to, is out of step, client_away while its Torrent's client isn't responding, refused the sign-in or needs attention, client_off while the client it is pinned to is off or deleted, or its Node doesn't hold Downloading. no_space while its root has too little free space for a file, root_not_ready while its root isn't ready, probe_timeout while reading a file timed out three times, and client_error while its external client reports an error"`
	ClientError api.Nullable[clientError] `json:"clientError" doc:"The client's error, for client_error; null otherwise"`
}

type clientError struct {
	Client  string    `json:"client" doc:"The external Download Client's name"`
	Node    string    `json:"node" doc:"The client's Node's name"`
	Message string    `json:"message" doc:"The error as the client words it"`
	FailsAt time.Time `json:"failsAt" doc:"When the Grab fails unless the error clears"`
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
	files.NeedWaiting: func(n files.Need) any {
		w := waitingNeed{Kind: "waiting", GrabID: n.Grab.String(), Code: n.Code}
		if e := n.ClientError; e != nil {
			w.ClientError = api.Some(clientError{Client: e.Client, Node: e.Node, Message: e.Message, FailsAt: e.FailsAt})
		}
		return w
	},
	files.NeedNoRoot: func(files.Need) any { return noRootNeed{Kind: "no_root"} },
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
