package node

import (
	"io"
	"net/http"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/node/internal/logs"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers"
)

// Test is what the simulation tier adds to a Node: test-only Roles, such
// as roletest's Probe, beside the binary's own, and a view of each half as
// it starts. A process leaves Env.Test zero, so production links no test
// Role.
type Test struct {
	// Library is library sets beside the binary's own.
	Library []storage.Schema[storage.Library]
	// Served is control halves the Controller serves beside the binary's
	// own, and Always the Roles every Node holds beyond its nodes_role rows.
	Served []role.Served
	Always []role.Name
	// Held is the holder halves this Node runs beside the binary's own,
	// given its local.db, each time a half starts.
	Held func(local *storage.DB[storage.Local]) []role.Held
	// Started sees each half once it runs.
	Started func(Half)
	// Lockstep wraps the handler the link serves lockstep requests with.
	Lockstep func(http.Handler) http.Handler
	// Transfers sees into a Worker's Transfer source.
	Transfers transfers.Hooks
}

// Half is a started half as a test reaches it. Library, Controls and
// Cluster are the Controller's, nil on a Worker.
type Half struct {
	Link     *link.Link
	Dispatch *rpc.Dispatch
	Local    *storage.DB[storage.Local]
	Library  *storage.DB[storage.Library]
	Controls *role.Controls
	Cluster  *rpc.Cluster
}

func (t Test) held(local *storage.DB[storage.Local]) []role.Held {
	if t.Held == nil {
		return nil
	}
	return t.Held(local)
}

func (t Test) lockstep(h http.Handler) http.Handler {
	if t.Lockstep == nil {
		return h
	}
	return t.Lockstep(h)
}

func (t Test) started(h Half) {
	if t.Started != nil {
		t.Started(h)
	}
}

// OpenFileLog opens the log in dir as one file that never rotates, mirrored
// to stderr, for an Env inside a testing/synctest bubble.
func OpenFileLog(dir string, stderr io.Writer) (*logs.Log, error) {
	return logs.OpenFile(dir, stderr)
}
