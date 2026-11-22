package downloadclients

import (
	"context"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/role"
)

// NodeTorrents is one Downloading Node's stored Torrents counted as the
// Torrents status sentence counts them: Downloading and Seeding by state,
// Failures and Warnings by severity, and Torrents every one.
type NodeTorrents struct {
	Node                                               role.NodeID
	Torrents, Downloading, Seeding, Failures, Warnings int
}

// NodeCounts counts each Downloading Node's live and Removing Torrents, in
// Node order, a Node that holds none at zeros. It decodes each report and
// builds no row.
func (c *Clients) NodeCounts(ctx context.Context) ([]NodeTorrents, error) {
	rows, err := db.New(c.cfg.Library).ListCountable(ctx)
	if err != nil {
		return nil, err
	}
	nodes := c.cfg.Roster.Serving(role.Downloading)
	out := make([]NodeTorrents, len(nodes))
	at := make(map[role.NodeID]*NodeTorrents, len(nodes))
	for i, n := range nodes {
		out[i] = NodeTorrents{Node: n}
		at[n] = &out[i]
	}
	for _, row := range rows {
		n, ok := at[row.NodeID]
		if !ok {
			continue
		}
		rep, err := reportOf(row.Report)
		if err != nil {
			return nil, err
		}
		f := Facts{Removing: !row.RemovedAt.IsZero(), Wanted: Wanted(row.Wanted), Report: rep}
		s := StateOf(f)
		n.Torrents++
		if s == StateDownloading {
			n.Downloading++
		}
		if s == StateSeeding {
			n.Seeding++
		}
		sev := SeverityOf(f, s)
		if sev == SeverityFailure {
			n.Failures++
		}
		if sev == SeverityWarning {
			n.Warnings++
		}
	}
	return out, nil
}
