package mstest

import (
	"context"
	"fmt"
	"testing"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Apply applies one of node's reports, as the Role's receive Write does,
// and follows its commit as the Role does.
func (w *World) Apply(ctx context.Context, node role.NodeID, r *msv1.Report) error {
	err := w.Lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return w.ms.Apply(ctx, tx, node, r)
	})
	if err != nil {
		return err
	}
	w.ms.Applied(node, string(role.MediaServerLink))
	return nil
}

// Plays is every play of server, by start, each as "<path> <decision>
// <reasons> by <user> on <player>, open" or "..., ended <n>s in" for one
// that ended that long after it started.
func (w *World) Plays(t *testing.T, server link.ServerID) []string {
	t.Helper()
	var out []string
	err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err := db.New(r).ListPlays(ctx, server)
		for _, p := range rows {
			end := "open"
			if !p.EndedAt.IsZero() {
				end = fmt.Sprintf("ended %ds in", int(p.EndedAt.Time().Sub(p.StartedAt.Time()).Seconds()))
			}
			out = append(out, fmt.Sprintf("%s %s %s by %s on %s, %s", p.ServerPath, p.Decision, p.Reasons, p.UserName, p.Player, end))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// SessionsChanged is every media-servers.sessions-changed published so
// far.
func (w *World) SessionsChanged() []mediaservers.SessionsChanged {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]mediaservers.SessionsChanged(nil), w.sessions...)
}

type sessionsEvents struct{ w *World }

func (e sessionsEvents) Publish(_ context.Context, v mediaservers.SessionsChanged) {
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	e.w.sessions = append(e.w.sessions, v)
}
