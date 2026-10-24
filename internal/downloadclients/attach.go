package downloadclients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var ErrRemoving = errors.New("the Torrent for this info hash is being removed on that Node")

// Want is the Torrent a Grab's fetch asks for.
type Want struct {
	// Node is the Node the router picked.
	Node        role.NodeID
	Metainfo    bencode.Metainfo
	Raw         []byte
	ContentType works.TypeID
	// WrittenOff reports whether the Admin wrote off what a Node holds
	// for Downloading, so its Torrents take no new Grab.
	WrittenOff func(role.NodeID) bool
}

// Attach is the live Torrent for want's info hash on want.Node, else a
// live one on a Node not written off. Otherwise it refuses with
// ErrRemoving while the hash is Removing on want.Node, or adds a Torrent
// on want.Node, started, last in order, with the totals of the latest
// removed Torrent of the hash on any Node. It runs in the caller's tx and
// does no IO: the caller Touches the Node it answers after commit.
func Attach(ctx context.Context, tx *storage.Tx[file], want Want, now storage.Time) (ID, role.NodeID, error) {
	q := db.New(tx)
	hash := want.Metainfo.InfoHash[:]
	held, err := q.ListHeldOfHash(ctx, hash)
	if err != nil {
		return ID{}, role.NodeID{}, err
	}
	removing := false
	var elsewhere *db.ListHeldOfHashRow
	for i, t := range held {
		switch {
		case t.NodeID == want.Node && t.RemovedAt.IsZero():
			return t.ID, t.NodeID, nil
		case t.NodeID == want.Node:
			removing = true
		case t.RemovedAt.IsZero() && !want.WrittenOff(t.NodeID) && elsewhere == nil:
			elsewhere = &held[i]
		}
	}
	switch {
	case elsewhere != nil:
		return elsewhere.ID, elsewhere.NodeID, nil
	case removing:
		return ID{}, role.NodeID{}, ErrRemoving
	}
	id := ID{UUID: storage.NewUUID()}
	return id, want.Node, insert(ctx, q, id, want, now)
}

func insert(ctx context.Context, q *db.Queries, id ID, want Want, now storage.Time) error {
	m := want.Metainfo
	trackers, err := json.Marshal(hostsOf(m.Trackers()))
	if err != nil {
		return fmt.Errorf("encode the trackers: %w", err)
	}
	totals, err := q.GetInheritedTotals(ctx, m.InfoHash[:])
	if err != nil && !errors.Is(err, storage.ErrNoRows) {
		return err
	}
	order, err := q.NextOrder(ctx)
	if err != nil {
		return err
	}
	var size, private int64
	for _, f := range m.Info.Files {
		size += f.Length
	}
	if m.Info.Private {
		private = 1
	}
	return q.InsertTorrent(ctx, db.InsertTorrentParams{
		ID: id, NodeID: want.Node, InfoHash: m.InfoHash[:], Metainfo: want.Raw, Name: m.Info.Name, Size: size,
		Private: private, Trackers: string(trackers), ContentType: want.ContentType, AddedOrder: order, AddedAt: now,
		Uploaded: totals.Uploaded, Downloaded: totals.Downloaded, ActiveSeconds: totals.ActiveSeconds,
		SeedingSeconds: totals.SeedingSeconds, TotalsAt: totals.TotalsAt,
	})
}

// hostsOf is each tracker URL's host, index for index, "" for one that
// doesn't parse.
func hostsOf(urls []string) []string {
	out := make([]string, len(urls))
	for i, s := range urls {
		if u, err := url.Parse(s); err == nil {
			out[i] = u.Hostname()
		}
	}
	return out
}
