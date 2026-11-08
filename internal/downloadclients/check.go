package downloadclients

import (
	"bytes"
	"cmp"
	"context"
	"crypto/rand"
	"slices"
	"sync"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/health"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// healthCheck is Check now on the category, or on one client's item. It
// asks each in-step Node to poll its on clients at once, refused ones
// too, under one fresh token, then waits on Moved until each asked
// client's row carries the token. The Node writes each poll through its
// outbox, so the Answer that follows reads what the polls found. A Node
// out of step, or one whose polls don't land before ctx ends, is
// Unreached.
func (c *Clients) healthCheck(ctx context.Context, item health.Key) ([]health.Unreached, error) {
	var (
		externals []External
		names     map[role.NodeID]string
	)
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		var err error
		if externals, err = externalsIn(ctx, rd); err != nil {
			return err
		}
		names, err = namesIn(ctx, rd)
		return err
	})
	if err != nil {
		return nil, err
	}
	serving := c.cfg.Roster.Serving(role.Downloading)
	asked := map[role.NodeID][]ClientID{}
	for _, e := range externals {
		if e.On && slices.Contains(serving, e.Node) && (item == "" || ClientHealthKey(e.ID) == item) {
			asked[e.Node] = append(asked[e.Node], e.ID)
		}
	}
	token := make([]byte, 16)
	_, _ = rand.Read(token)
	var (
		mu        sync.Mutex
		unreached []health.Unreached
		landing   []ClientID
		wg        sync.WaitGroup
	)
	missed := func(words string) {
		mu.Lock()
		defer mu.Unlock()
		unreached = append(unreached, health.Unreached{Key: item, Words: words})
	}
	for node, ids := range asked {
		name := names[node]
		if !c.cfg.Roster.InStep(node) {
			missed(offlineWords(name))
			continue
		}
		wg.Go(func() {
			raw := make([][]byte, len(ids))
			for i, id := range ids {
				raw[i] = id.Bytes()
			}
			if _, err := c.cfg.Remote(node).CheckClients(ctx, &dlv1.CheckClientsRequest{Ids: raw, Check: token}); err != nil {
				missed(name + " didn't respond in time, so its Download Clients weren't checked.")
				return
			}
			mu.Lock()
			defer mu.Unlock()
			landing = append(landing, ids...)
		})
	}
	wg.Wait()
	offline, late, err := c.awaitCheck(ctx, landing, token)
	if err != nil {
		return nil, err
	}
	for _, node := range offline {
		missed(offlineWords(names[node]))
	}
	for _, node := range late {
		missed(names[node] + " didn't respond in time, so its Download Clients weren't checked.")
	}
	slices.SortFunc(unreached, func(a, b health.Unreached) int { return cmp.Compare(a.Words, b.Words) })
	return unreached, nil
}

// awaitCheck waits until each of ids carries token. It answers the Nodes
// of those that don't: offline once a Node leaves step, late once ctx
// ends. A client deleted meanwhile carries nothing and is waited on no
// more.
func (c *Clients) awaitCheck(ctx context.Context, ids []ClientID, token []byte) (offline, late []role.NodeID, err error) {
	for len(ids) > 0 {
		moved, changed := c.Moved(), c.cfg.Roster.Changed()
		unchecked, err := c.unchecked(ctx, ids, token)
		if err != nil {
			return nil, nil, err
		}
		ids, late = nil, nil
		for _, row := range unchecked {
			if !c.cfg.Roster.InStep(row.NodeID) {
				offline = withNode(offline, row.NodeID)
				continue
			}
			ids, late = append(ids, row.ID), withNode(late, row.NodeID)
		}
		if len(ids) == 0 {
			break
		}
		select {
		case <-moved:
		case <-changed:
		case <-ctx.Done():
			return offline, late, nil
		}
	}
	return offline, nil, nil
}

// unchecked is each of ids whose row doesn't carry token yet.
func (c *Clients) unchecked(ctx context.Context, ids []ClientID, token []byte) ([]db.DownloadclientsClient, error) {
	var out []db.DownloadclientsClient
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		rows, err := db.New(rd).ListClients(ctx)
		for _, row := range rows {
			if slices.Contains(ids, row.ID) && !bytes.Equal(row.PollCheck, token) {
				out = append(out, row)
			}
		}
		return err
	})
	return out, err
}

func withNode(nodes []role.NodeID, n role.NodeID) []role.NodeID {
	if slices.Contains(nodes, n) {
		return nodes
	}
	return append(nodes, n)
}

func offlineWords(node string) string {
	return node + " is offline, so its Download Clients weren't checked."
}
