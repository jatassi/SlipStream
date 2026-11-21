// Package reconcilingapi is the Reconciling page: what a restored
// Controller waits on, the steps that end the wait, and the review list.
package reconcilingapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// Reconciler is Reconciling, false unless the Node is a Controller.
type Reconciler interface {
	Reconciling() (*reconciling.Service, bool)
}

func open(n Reconciler) *reconciling.Service {
	s, ok := n.Reconciling()
	if !ok {
		panic("reconcilingapi: an operation ran on a Node that is no Controller")
	}
	return s
}

var (
	notReconciling = api.NewCode(http.StatusConflict, "not_reconciling", "SlipStream isn't Reconciling")
	noRoot         = api.NewCode(http.StatusNotFound, "reconciling_root_not_found", "Reconciling doesn't wait on this library root")
	noNode         = api.NewCode(http.StatusNotFound, "reconciling_node_not_found", "Reconciling doesn't wait on this Node")
	notReviewed    = api.NewCode(http.StatusNotFound, "not_reviewed", "A torrent isn't on the review list")
)

var codes = []struct {
	err  error
	code *api.Code
}{
	{reconciling.ErrNotReconciling, notReconciling},
	{reconciling.ErrNoRoot, noRoot},
	{reconciling.ErrNoNode, noNode},
	{reconciling.ErrNotReviewed, notReviewed},
}

func answer(err error) error {
	for _, c := range codes {
		if errors.Is(err, c.err) {
			return c.code
		}
	}
	return err
}

var (
	getReconcilingOp = api.Op{
		ID: "getReconciling", Method: http.MethodGet, Path: "/reconciling",
		Summary: "Read what Reconciling waits on after a restore", Status: http.StatusOK,
	}
	continueNodeOp = api.Op{
		ID: "continueReconcilingNode", Method: http.MethodPost, Path: "/reconciling/nodes/{nodeId}/continue",
		Summary: "Stop waiting for a Node's torrents", Status: http.StatusNoContent, Errors: []*api.Code{noNode},
	}
	continueRootOp = api.Op{
		ID: "continueReconcilingRoot", Method: http.MethodPost, Path: "/reconciling/roots/{rootId}/continue",
		Summary: "Stop waiting for a library root's scan", Status: http.StatusNoContent, Errors: []*api.Code{noRoot},
	}
	finishOp = api.Op{
		ID: "finishReconciling", Method: http.MethodPost, Path: "/reconciling/finish",
		Summary: "Finish Reconciling now", Status: http.StatusNoContent, Errors: []*api.Code{notReconciling},
	}
	rehomeOp = api.Op{
		ID: "rehomeReconciling", Method: http.MethodPost, Path: "/reconciling/rehome",
		Summary: "Save the old Controller's library roots at their places on this machine", Status: http.StatusOK,
		Errors: []*api.Code{notReconciling, noRoot},
	}
	listTorrentsOp = api.Op{
		ID: "listReconcilingTorrents", Method: http.MethodGet, Path: "/reconciling/torrents",
		Summary: "List a page of the review list", Status: http.StatusOK,
	}
	readdOp = api.Op{
		ID: "readdReconcilingTorrents", Method: http.MethodPost, Path: "/reconciling/torrents/readd",
		Summary: "Re-add torrents on the review list", Status: http.StatusNoContent, Errors: []*api.Code{notReviewed},
	}
	forgetOp = api.Op{
		ID: "forgetReconcilingTorrents", Method: http.MethodPost, Path: "/reconciling/torrents/forget",
		Summary: "Forget torrents no Node holds", Status: http.StatusNoContent, Errors: []*api.Code{notReviewed},
	}
)

func Routes(r *api.Router, n Reconciler) {
	o := r.Owner("reconciling", api.Admin)
	api.Register(o, api.Admin, getReconcilingOp, func(ctx context.Context, _ identity.Admin, _ *struct{}) (*pageOutput, error) {
		p, err := open(n).Page(ctx)
		if err != nil {
			return nil, err
		}
		return &pageOutput{Body: pageOf(p)}, nil
	})
	api.Register(o, api.Admin, continueNodeOp, func(ctx context.Context, _ identity.Admin, in *nodeInput) (*struct{}, error) {
		id, err := parse(in.NodeID, noNode)
		if err != nil {
			return nil, err
		}
		return nil, answer(open(n).ContinueNode(ctx, nodes.ID{UUID: id}))
	})
	api.Register(o, api.Admin, continueRootOp, func(ctx context.Context, _ identity.Admin, in *rootInput) (*struct{}, error) {
		id, err := parse(in.RootID, noRoot)
		if err != nil {
			return nil, err
		}
		return nil, answer(open(n).ContinueRoot(ctx, roots.ID{UUID: id}))
	})
	api.Register(o, api.Reconfirmed, finishOp, func(ctx context.Context, _ identity.Admin, _ *struct{}) (*struct{}, error) {
		return nil, answer(open(n).Finish(ctx))
	})
	api.Register(o, api.Admin, rehomeOp, func(ctx context.Context, by identity.Admin, in *rehomeInput) (*rehomeOutput, error) {
		return rehome(ctx, open(n), by, in)
	})
	api.Register(o, api.Admin, listTorrentsOp, func(ctx context.Context, _ identity.Admin, in *listInput) (*listOutput, error) {
		return list(ctx, open(n), in)
	})
	api.Register(o, api.Admin, readdOp, func(ctx context.Context, _ identity.Admin, in *readdInput) (*struct{}, error) {
		return nil, readd(ctx, open(n), in)
	})
	api.Register(o, api.Admin, forgetOp, func(ctx context.Context, _ identity.Admin, in *forgetInput) (*struct{}, error) {
		return nil, forget(ctx, open(n), in)
	})
}

func parse(s string, missing *api.Code) (storage.UUID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return storage.UUID{}, missing
	}
	return u, nil
}

type nodeInput struct {
	NodeID string `path:"nodeId" format:"uuid"`
}

type rootInput struct {
	RootID string `path:"rootId" format:"uuid"`
}

type pageOutput struct{ Body page }

type page struct {
	Restore api.Nullable[restore] `json:"restore" doc:"The restore Reconciling runs for; null outside one, when the page lists only the Nodes that keep their torrents"`
	Nodes   []nodeRow             `json:"nodes" nullable:"false" doc:"Each Node Reconciling waits on, or whose adoption gave it the Downloading Role"`
	Roots   []rootRow             `json:"roots" nullable:"false" doc:"Each library root Reconciling waits on"`
	Unknown []unknown             `json:"unknown" nullable:"false" doc:"Nodes not in this Backup that tried to connect lately, newest last; empty outside a restore"`
	Review  []group               `json:"review" nullable:"false" doc:"The review list's groups, with how many torrents each holds"`
}

type restore struct {
	Backup     string    `json:"backup" doc:"The Backup's name"`
	TakenAt    time.Time `json:"takenAt" doc:"When the Backup was taken"`
	StartedAt  time.Time `json:"startedAt" doc:"When Reconciling began"`
	NewMachine bool      `json:"newMachine" doc:"Whether the Backup was restored on a machine other than the one that took it"`
	Rehome     bool      `json:"rehome" doc:"Whether the old Controller's folders still need finding"`
}

type nodeRow struct {
	ID        string               `json:"id" format:"uuid"`
	Name      string               `json:"name"`
	Since     time.Time            `json:"since" doc:"When Reconciling began waiting on it"`
	Held      int                  `json:"held" minimum:"0" doc:"How many torrents its list held"`
	Reported  bool                 `json:"reported" doc:"Whether every page of its list of torrents is in"`
	Adopted   bool                 `json:"adopted" doc:"Whether its torrents were matched with the library"`
	Continued bool                 `json:"continued" doc:"Whether Reconciling stopped waiting on it. It still keeps every torrent until SlipStream has matched its list of torrents with the library"`
	Granted   api.Nullable[string] `json:"granted" doc:"The line naming the Downloading Role its adoption gave it, such as vps-relay now has the Downloading Role for the 1,240 torrents it reported; null when it got none"`
	Refused   api.Nullable[string] `json:"refused" doc:"Why SlipStream refused its list of torrents; null when it didn't. It keeps every torrent until a list is taken"`
	Link      string               `json:"link" enum:"in-step,reconnecting,offline,lapsed,mismatched" doc:"How it stands with this Controller. A Node on a new machine that stays offline may still be paired with the old Controller"`
	LinkSince time.Time            `json:"linkSince" doc:"Since when it stands so"`
}

type rootRow struct {
	ID        string `json:"id" format:"uuid"`
	Label     string `json:"label" doc:"The root as a person reads it, such as Movies library root on Media server"`
	Saved     bool   `json:"saved" doc:"Whether the re-home step saved it. A root not saved waits for its folder, and no scan reads it"`
	Scanned   bool   `json:"scanned" doc:"Whether a scan finished since it was saved"`
	Continued bool   `json:"continued" doc:"Whether Reconciling stopped waiting on its scan"`
}

type unknown struct {
	Fingerprint string    `json:"fingerprint" doc:"Its node key's fingerprint"`
	Address     string    `json:"address" doc:"The address it connected from"`
	OS          string    `json:"os"`
	Arch        string    `json:"arch"`
	At          time.Time `json:"at" doc:"When it last tried to connect"`
}

type group struct {
	Reason string `json:"reason" enum:"not_held,conflict" doc:"not_held: recorded, and no Node reported it after the restore. conflict: two Nodes hold it and both keep it"`
	Found  string `json:"found" enum:"not_looked,no_files,files" doc:"What its Node found of its files"`
	Count  int    `json:"count" minimum:"0"`
}

var foundWords = map[reconciling.Found]string{reconciling.NotLooked: "not_looked", reconciling.NoFiles: "no_files", reconciling.Files: "files"}

func pageOf(p reconciling.Page) page {
	out := page{Nodes: []nodeRow{}, Roots: []rootRow{}, Unknown: []unknown{}, Review: []group{}}
	if r := p.Restore; r != nil {
		out.Restore = api.Some(restore{Backup: r.Backup, TakenAt: r.TakenAt, StartedAt: r.StartedAt, NewMachine: r.NewMachine, Rehome: r.Rehome})
	}
	for _, k := range p.Nodes {
		st := p.Standing[k.Node]
		row := nodeRow{
			ID: k.Node.String(), Name: p.Names[k.Node], Since: k.Since, Held: k.Held, Reported: k.Reported, Adopted: k.Adopted,
			Continued: k.Continued, Link: st.Standing.String(), LinkSince: st.Since,
		}
		if k.Granted {
			row.Granted = api.Some(reconciling.GrantWords(p.Names[k.Node], k.Held))
		}
		if k.Refused != "" {
			row.Refused = api.Some(k.Refused)
		}
		out.Nodes = append(out.Nodes, row)
	}
	for _, r := range p.Roots {
		out.Roots = append(out.Roots, rootRow{ID: r.Root.String(), Label: p.Words[r.Root], Saved: r.Saved, Scanned: r.Scanned, Continued: r.Continued})
	}
	for _, u := range p.Unknown {
		out.Unknown = append(out.Unknown, unknown{Fingerprint: u.Key.Fingerprint(), Address: u.Address.Addr().String(), OS: u.OS, Arch: u.Arch, At: u.At})
	}
	for _, g := range p.Groups {
		out.Review = append(out.Review, group{Reason: string(g.Reason), Found: foundWords[g.Found], Count: g.Count})
	}
	return out
}
