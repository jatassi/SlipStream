package filesapi

import (
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type summary struct {
	Status     api.Nullable[api.Enum[State]] `json:"status" doc:"Its monitored Items' first status in the order failed, missing, downloading, converting, upgradable, unreleased, met; null while none has a Target"`
	Cells      api.Counts[State]             `json:"cells" doc:"Its monitored Items at each of their Targets, by state"`
	Items      int                           `json:"items" minimum:"0" doc:"Its monitored Items that its source still lists, the only Items the status and counts read"`
	ItemStatus api.Counts[State]             `json:"itemStatus" doc:"Its monitored Items by status; an Item with no Target isn't counted"`
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
	return "kind", []any{failedNeed{}, conversionFailedNeed{}, upgradeNeed{}, notFoundNeed{}, WaitingNeed{}, noRootNeed{}}
}

type failedNeed struct {
	Kind     string   `json:"kind" enum:"failed"`
	ItemIDs  []string `json:"itemIds" nullable:"false" doc:"A run of Items in tree order whose entries match"`
	Edition  string   `json:"edition" doc:"any under any cut, else the key of the cut these Items need at this Target"`
	TargetID string   `json:"targetId" format:"uuid" doc:"The Target whose download failed"`
	Waiting  []string `json:"waiting" nullable:"false" doc:"The Targets of the same Items and Edition, still missing, that wait on the failed one"`
	GrabID   string   `json:"grabId" format:"uuid" doc:"The Grab that failed"`
}

type conversionFailedNeed struct {
	Kind         string   `json:"kind" enum:"conversion_failed"`
	ConversionID string   `json:"conversionId" format:"uuid"`
	ItemIDs      []string `json:"itemIds" nullable:"false" doc:"A run of Items in tree order whose entries match"`
	Edition      string   `json:"edition" doc:"any under any cut, else the key of the cut these Items need at this Target"`
	TargetID     string   `json:"targetId" format:"uuid" doc:"The Target whose made file failed"`
	Code         string   `json:"code" doc:"The fault, such as encoder_failed"`
	Class        string   `json:"class" enum:"run,source" doc:"run: it failed on every Node it may try; source: its source file can't be converted"`
}

type upgradeNeed struct {
	Kind     string   `json:"kind" enum:"upgrade"`
	ItemIDs  []string `json:"itemIds" nullable:"false" doc:"A run of Items in tree order whose entries match"`
	Edition  string   `json:"edition" doc:"any under any cut, else the key of the cut these Items need"`
	Upgrades []string `json:"upgrades" nullable:"false" doc:"Every upgradable Target of the Items and Edition"`
}

type notFoundNeed struct {
	Kind     string   `json:"kind" enum:"not_found" doc:"Missing Items whose latest search found no Release, or none acceptable. It shows on the Work page only, and the Library's Needs you count leaves it out"`
	ItemIDs  []string `json:"itemIds" nullable:"false" doc:"A run of Items in tree order whose entries match"`
	Edition  string   `json:"edition" doc:"any under any cut, else the key of the cut these Items need at this Target"`
	GroupKey string   `json:"groupKey" doc:"The key of the search group whose latest search found them nothing. Its state in the Work's searching says when it searched and when it searches next"`
}

// WaitingNeed is a Grab that waits on an Admin, as the Work page words it.
type WaitingNeed struct {
	Kind        string                    `json:"kind" enum:"waiting"`
	GrabID      string                    `json:"grabId" format:"uuid"`
	Code        string                    `json:"code" enum:"node_away,client_away,client_off,no_space,root_not_ready,probe_timeout,probe_error,client_error" doc:"Why the Grab waits: node_away while its Node, or the Node of the client its fetch goes to, is out of sync with the Controller, client_away while its Torrent's client isn't responding, refused the sign-in or needs attention, client_off while the client it is pinned to is off or deleted, or its Node doesn't have the Downloading Role. no_space while its root has too little free space for a file, root_not_ready while its root isn't ready, probe_timeout while reading a file timed out three times, probe_error while reading a file failed once and SlipStream tries once more, and client_error while its external client reports an error"`
	ClientError api.Nullable[clientError] `json:"clientError" doc:"The client's error, for client_error; null otherwise"`
	ImportWait  api.Nullable[importWait]  `json:"importWait" doc:"The root the Import waits on, for no_space and root_not_ready; null otherwise"`
}

type importWait struct {
	RootID string              `json:"rootId" format:"uuid"`
	Space  api.Nullable[space] `json:"space" doc:"For no_space, the bytes free and the bytes the file needs; null when a clone found the volume full or for root_not_ready"`
}

type space struct {
	Free  int64 `json:"free" minimum:"0" doc:"Bytes free on the root's volume above its Storage error level"`
	Needs int64 `json:"needs" minimum:"1" doc:"Bytes the file needs"`
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
	files.NeedConversionFailed: func(n files.Need) any {
		return conversionFailedNeed{
			Kind: "conversion_failed", ConversionID: n.Conversion.String(), ItemIDs: itemIDs(n.Items), Edition: string(n.Edition),
			TargetID: n.Target.String(), Code: n.Code, Class: n.Class,
		}
	},
	files.NeedUpgrade: func(n files.Need) any {
		return upgradeNeed{Kind: "upgrade", ItemIDs: itemIDs(n.Items), Edition: string(n.Edition), Upgrades: targetIDs(n.Upgrades)}
	},
	files.NeedNotFound: func(n files.Need) any {
		return notFoundNeed{Kind: "not_found", ItemIDs: itemIDs(n.Items), Edition: string(n.Edition), GroupKey: n.Group}
	},
	files.NeedWaiting: func(n files.Need) any { return WaitingOf(n) },
	files.NeedNoRoot:  func(files.Need) any { return noRootNeed{Kind: "no_root"} },
}

// WaitingOf is waiting Need n on the wire.
func WaitingOf(n files.Need) WaitingNeed {
	w := WaitingNeed{Kind: "waiting", GrabID: n.Grab.String(), Code: n.Code}
	if e := n.ClientError; e != nil {
		w.ClientError = api.Some(clientError{Client: e.Client, Node: e.Node, Message: e.Message, FailsAt: e.FailsAt})
	}
	if h := n.Hold; h != nil {
		iw := importWait{RootID: h.Root.String()}
		if h.Needs > 0 {
			iw.Space = api.Some(space{Free: h.Free, Needs: h.Needs})
		}
		w.ImportWait = api.Some(iw)
	}
	return w
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
