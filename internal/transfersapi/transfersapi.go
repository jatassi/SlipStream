// Package transfersapi is the Transfers' operations: every Transfer with
// its state, the recent speed between each two Nodes, and each Node's send
// limit. Nothing here changes a Transfer, because the job that asked for
// it decides about it.
package transfersapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers"
)

// Copies is the Node's Transfers and its roster.
type Copies interface {
	// Transfers is the record of every Transfer, false unless the Node is
	// a Controller.
	Transfers() (*transfers.Transfers, bool)
	// Nodes is the nodes service, false unless the Node is a Controller.
	Nodes() (*nodes.Service, bool)
}

func open(c Copies) (*transfers.Transfers, *nodes.Service) {
	t, ok := c.Transfers()
	roster, known := c.Nodes()
	if !ok || !known {
		panic("transfersapi: an Admin operation ran on a Node that is no Controller")
	}
	return t, roster
}

var (
	transferNotFound = api.NewCode(http.StatusNotFound, "transfer_not_found", "No Transfer has that id")
	nodeNotFound     = api.NewCode(http.StatusNotFound, "node_not_found", "No Node has this id")
)

type transfer struct {
	ID              string      `json:"id" format:"uuid"`
	Purpose         string      `json:"purpose" enum:"grab" doc:"The kind of job that asked for it"`
	Job             string      `json:"job" doc:"The id of the job that asked for it, such as its Grab's"`
	JobName         string      `json:"jobName" doc:"The job's name, such as a Work's title and year"`
	Source          end         `json:"source" doc:"The Node it copies from"`
	Destination     destination `json:"destination"`
	State           string      `json:"state" enum:"queued,waiting,needs_space,copying,failed" doc:"copying also covers files that are all verified and wait for their job to place them"`
	Reason          string      `json:"reason,omitempty" doc:"Why it isn't copying, as a sentence; absent while it copies"`
	Wait            *wait       `json:"wait,omitempty" doc:"What it waits for; absent unless its state is waiting"`
	Verified        int64       `json:"verified" minimum:"0" doc:"Bytes verified at the destination"`
	Total           int64       `json:"total" minimum:"0" doc:"Bytes"`
	Rate            int64       `json:"rate" minimum:"0" doc:"Bytes a second it copies at, else the last speed between its two Nodes; 0 while unknown"`
	EstimateSeconds int64       `json:"estimateSeconds" minimum:"0" doc:"Seconds until its files are verified; 0 while unknown"`
	Route           string      `json:"route,omitempty" enum:"receiver_dials,sender_dials,node_link" doc:"Which Node opens its lanes, or node_link when they run over the Node link; absent unless it's first in line"`
	Files           []file      `json:"files" nullable:"false" doc:"In the order they copy"`
	Details         details     `json:"details"`
}

type end struct {
	NodeID   string `json:"nodeId" format:"uuid"`
	NodeName string `json:"nodeName"`
}

type destination struct {
	Kind     string `json:"kind" enum:"root" doc:"root is a library root"`
	RootID   string `json:"rootId" format:"uuid"`
	NodeID   string `json:"nodeId" format:"uuid" doc:"The Node that holds the root now"`
	NodeName string `json:"nodeName"`
}

type wait struct {
	Kind   string `json:"kind" enum:"node_away,node_other_build,connecting,no_route,root_not_ready,storage_draining,source_checking,source_not_ready"`
	NodeID string `json:"nodeId" format:"uuid" doc:"The Node it waits for"`
}

type file struct {
	Position int    `json:"position" minimum:"0"`
	Name     string `json:"name"`
	Size     int64  `json:"size" minimum:"0" doc:"Bytes"`
	Verified bool   `json:"verified" doc:"Every byte is verified at the destination"`
	Placed   bool   `json:"placed" doc:"Its job moved it to where it belongs"`
}

type details struct {
	Lanes   int `json:"lanes" minimum:"0" doc:"Connections that carry its chunks now"`
	RTTMs   int `json:"rttMs" minimum:"0" doc:"The lanes' round trip"`
	Retried int `json:"retried" minimum:"0" doc:"Chunks asked for again since the receiving Node last started it"`
}

var routeWords = map[tv1.Route]string{
	tv1.Route_ROUTE_RECEIVER_DIALS: "receiver_dials",
	tv1.Route_ROUTE_SENDER_DIALS:   "sender_dials",
	tv1.Route_ROUTE_NODE_LINK:      "node_link",
}

func transferOf(v transfers.View, jobName string, names map[role.NodeID]string) transfer {
	out := transfer{
		ID:              v.ID.String(),
		Purpose:         string(v.Purpose),
		Job:             v.Job,
		JobName:         jobName,
		Source:          end{NodeID: v.Source.String(), NodeName: names[v.Source]},
		State:           v.State.String(),
		Reason:          v.Reason(names),
		Verified:        v.Verified,
		Total:           v.Total,
		Rate:            v.Rate,
		EstimateSeconds: int64(v.Estimate / time.Second),
		Route:           routeWords[v.Route],
		Files:           make([]file, len(v.Files)),
		Details:         details{Lanes: v.Details.Lanes, RTTMs: v.Details.RTTMillis, Retried: v.Details.Retried},
	}
	switch d := v.Destination.(type) {
	case transfers.Root:
		out.Destination = destination{Kind: "root", RootID: d.ID.String(), NodeID: v.DestNode.String(), NodeName: names[v.DestNode]}
	default:
		panic("transfersapi: a destination of no known kind")
	}
	if v.Failure != nil {
		out.Reason = v.Failure.Sentence
	}
	if v.Wait != nil {
		out.Wait = &wait{Kind: v.Wait.Kind.String(), NodeID: v.Wait.Node.String()}
	}
	for i, f := range v.Files {
		out.Files[i] = file{Position: f.Position, Name: f.Name, Size: f.Size, Verified: f.Verified, Placed: f.Placed}
	}
	return out
}

type listInput struct {
	NodeID   string `query:"nodeId" format:"uuid" doc:"Only Transfers this Node sends or receives"`
	InfoHash string `query:"infoHash" pattern:"^[0-9a-f]{40}$" doc:"Only Transfers with a file from this torrent, by its v1 info hash in hex"`
}

type transferList struct {
	Transfers []transfer `json:"transfers" nullable:"false" doc:"Oldest first"`
}

type transferListOutput struct{ Body transferList }

type transferInput struct {
	ID string `path:"id" format:"uuid"`
}

type transferOutput struct{ Body transfer }

type speed struct {
	SourceNodeID      string     `json:"sourceNodeId" format:"uuid"`
	DestinationNodeID string     `json:"destinationNodeId" format:"uuid"`
	Rate              int64      `json:"rate" minimum:"0" doc:"Bytes a second, the speed of the last Transfer from the source to the destination"`
	RateAt            time.Time  `json:"rateAt"`
	Route             string     `json:"route,omitempty" enum:"receiver_dials,sender_dials,node_link" doc:"The last route that moved bytes; absent while none has"`
	RouteAt           *time.Time `json:"routeAt,omitempty"`
}

type speedList struct {
	Speeds []speed `json:"speeds" nullable:"false" doc:"One for each source and destination that a Transfer moved bytes between"`
}

type speedListOutput struct{ Body speedList }

type settings struct {
	SendLimit setting.RateField `json:"sendLimit"`
}

type settingsPatch struct {
	SendLimit api.Patch[int64] `json:"sendLimit,omitempty" doc:"Bytes a second, 0 for no limit; null resets it to its default"`
}

type nodeInput struct {
	NodeID string `path:"nodeId" format:"uuid"`
}

type settingsInput struct {
	NodeID string `path:"nodeId" format:"uuid"`
	Body   settingsPatch
}

type settingsOutput struct{ Body settings }

func settingsOf(o transfers.NodeOverrides) *settingsOutput {
	return &settingsOutput{Body: settings{SendLimit: transfers.SendLimit.Field(o.SendLimit)}}
}

// nodeNames is every Node's name, tombstones included, so a Transfer
// still names a Node removed under it.
func nodeNames(ctx context.Context, roster *nodes.Service) (map[role.NodeID]string, error) {
	named, err := roster.Names(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[role.NodeID]string, len(named))
	for id, n := range named {
		out[id] = n.Name
	}
	return out, nil
}

// node is the Node id names, nodeNotFound unless the roster has it.
func node(ctx context.Context, roster *nodes.Service, id string) (role.NodeID, error) {
	u, err := storage.ParseUUID(id)
	if err != nil {
		return role.NodeID{}, nodeNotFound
	}
	n := nodes.ID{UUID: u}
	if _, err := roster.Get(ctx, n); errors.Is(err, nodes.ErrNotFound) {
		return role.NodeID{}, nodeNotFound
	} else if err != nil {
		return role.NodeID{}, err
	}
	return n, nil
}

type handlers struct{ copies Copies }

func (h handlers) listTransfers(ctx context.Context, _ identity.Admin, in *listInput) (*transferListOutput, error) {
	t, roster := open(h.copies)
	var f transfers.Filter
	if in.NodeID != "" {
		u, err := storage.ParseUUID(in.NodeID)
		if err != nil {
			return nil, err
		}
		f.Node = role.NodeID{UUID: u}
	}
	if in.InfoHash != "" {
		hash, err := bencode.ParseInfoHash(in.InfoHash)
		if err != nil {
			return nil, err
		}
		f.Source = &hash
	}
	vs, err := t.List(ctx, f)
	if err != nil {
		return nil, err
	}
	out, err := present(ctx, t, roster, vs)
	if err != nil {
		return nil, err
	}
	return &transferListOutput{Body: transferList{Transfers: out}}, nil
}

func (h handlers) getTransfer(ctx context.Context, _ identity.Admin, in *transferInput) (*transferOutput, error) {
	t, roster := open(h.copies)
	u, err := storage.ParseUUID(in.ID)
	if err != nil {
		return nil, transferNotFound
	}
	v, err := t.Get(ctx, transfers.ID{UUID: u})
	if errors.Is(err, transfers.ErrNotFound) {
		return nil, transferNotFound
	} else if err != nil {
		return nil, err
	}
	out, err := present(ctx, t, roster, []transfers.View{v})
	if err != nil {
		return nil, err
	}
	return &transferOutput{Body: out[0]}, nil
}

// present is vs on the wire, with their jobs' and Nodes' names.
func present(ctx context.Context, t *transfers.Transfers, roster *nodes.Service, vs []transfers.View) ([]transfer, error) {
	jobs, err := t.JobNames(ctx, vs)
	if err != nil {
		return nil, err
	}
	names, err := nodeNames(ctx, roster)
	if err != nil {
		return nil, err
	}
	out := make([]transfer, len(vs))
	for i, v := range vs {
		out[i] = transferOf(v, jobs[v.ID], names)
	}
	return out, nil
}

func (h handlers) listSpeeds(ctx context.Context, _ identity.Admin, _ *struct{}) (*speedListOutput, error) {
	t, _ := open(h.copies)
	ps, err := t.Speeds(ctx)
	if err != nil {
		return nil, err
	}
	out := speedList{Speeds: make([]speed, len(ps))}
	for i, p := range ps {
		out.Speeds[i] = speed{SourceNodeID: p.Source.String(), DestinationNodeID: p.Destination.String(), Rate: p.Rate, RateAt: p.RateAt, Route: routeWords[p.Route]}
		if !p.RouteAt.IsZero() {
			out.Speeds[i].RouteAt = &p.RouteAt
		}
	}
	return &speedListOutput{Body: out}, nil
}

func (h handlers) getSettings(ctx context.Context, _ identity.Admin, in *nodeInput) (*settingsOutput, error) {
	t, roster := open(h.copies)
	id, err := node(ctx, roster, in.NodeID)
	if err != nil {
		return nil, err
	}
	o, err := t.NodeSettings(ctx, id)
	if err != nil {
		return nil, err
	}
	return settingsOf(o), nil
}

func (h handlers) setSettings(ctx context.Context, _ identity.Admin, in *settingsInput) (*settingsOutput, error) {
	t, roster := open(h.copies)
	id, err := node(ctx, roster, in.NodeID)
	if err != nil {
		return nil, err
	}
	var ch transfers.NodeChange
	if in.Body.SendLimit.Sent() {
		v, err := transfers.SendLimit.Parse(in.Body.SendLimit.Value())
		if err != nil {
			return nil, settingapi.Refused(err)
		}
		ch.SendLimit = &v
	}
	o, err := t.SetNodeSettings(ctx, id, ch)
	if err != nil {
		return nil, err
	}
	return settingsOf(o), nil
}

func Routes(r *api.Router, copies Copies) {
	o := r.Owner("transfers", api.Admin)
	h := handlers{copies}

	api.Register(o, api.Admin, api.Op{
		ID:      "listTransfers",
		Method:  http.MethodGet,
		Path:    "/transfers",
		Summary: "List the Transfers between Nodes, with each one's state, progress and speed",
		Status:  http.StatusOK,
	}, h.listTransfers)

	api.Register(o, api.Admin, api.Op{
		ID:      "listTransferSpeeds",
		Method:  http.MethodGet,
		Path:    "/transfers/speeds",
		Summary: "List the recent speed and route of each pair of Nodes a Transfer moved bytes between",
		Status:  http.StatusOK,
	}, h.listSpeeds)

	api.Register(o, api.Admin, api.Op{
		ID:      "getTransfer",
		Method:  http.MethodGet,
		Path:    "/transfers/{id}",
		Summary: "Get a Transfer with its state, progress, speed and files",
		Status:  http.StatusOK,
		Errors:  []*api.Code{transferNotFound},
	}, h.getTransfer)

	api.Register(o, api.Admin, api.Op{
		ID:      "getTransferSettings",
		Method:  http.MethodGet,
		Path:    "/transfers/nodes/{nodeId}/settings",
		Summary: "Get a Node's Transfer settings",
		Status:  http.StatusOK,
		Errors:  []*api.Code{nodeNotFound},
	}, h.getSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "setTransferSettings",
		Method:  http.MethodPatch,
		Path:    "/transfers/nodes/{nodeId}/settings",
		Summary: "Change a Node's Transfer settings, which the Node applies at once",
		Status:  http.StatusOK,
		Errors:  []*api.Code{nodeNotFound, settingapi.ErrOutOfRange},
	}, h.setSettings)
}
