package grabs

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// ErrNotSendable is a send-to of a Grab that neither downloads nor waits
// for its client's Node: its fetch can go ahead, or its Torrent is past or
// short of downloading, or its download finished, failed or was removed.
var ErrNotSendable = errors.New("the Grab can't be sent to another client")

// SendTo sends Grab id to the client c names, in one Write that checks the
// client can take it. A downloading Grab's Torrent is retargeted, and every
// Grab downloading it points at the Torrent and Node that answers. A Grab
// that waits to fetch for its client's Node goes to c once fetched, and
// fetches at once. It tells the Nodes whose intents changed once it
// commits. It fails with ErrNoGrab, ErrNotSendable, and Usable's and
// Retarget's errors.
func (gr *Grabs) SendTo(ctx context.Context, _ identity.Admin, id GrabID, c downloadclients.Choice) error {
	var (
		sent     downloadclients.Retargeted
		changed  []db.ListDownloadingRow
		fetching *Grab
	)
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		row, err := db.New(tx).GetGrab(ctx, id)
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNoGrab
		}
		if err != nil {
			return err
		}
		rel, err := readRelated(ctx, tx, []db.GrabsGrab{row})
		if err != nil {
			return err
		}
		g, err := load(ctx, tx, row, rel)
		if err != nil {
			return err
		}
		if g.Stage == Fetching {
			fetching = &g
			return gr.sendFetching(ctx, tx, g, c)
		}
		sent, changed, err = gr.sendDownloading(ctx, tx, g, c)
		return err
	})
	if err != nil {
		return err
	}
	if fetching != nil {
		gr.cfg.Changed.Publish(ctx, Changed{GrabID: fetching.ID.String(), WorkID: fetching.WorkID.String()})
		gr.routes.change()
		return nil
	}
	gr.cfg.Downloads.Retargeted(sent)
	for _, r := range changed {
		gr.cfg.Changed.Publish(ctx, Changed{GrabID: r.ID.String(), WorkID: r.WorkID.String()})
	}
	gr.wake()
	return nil
}

// sendDownloading retargets downloading Grab g's Torrent to the client c
// names, in tx, and points every Grab downloading it at the Torrent and
// Node that answers, which it returns with those Grabs.
func (gr *Grabs) sendDownloading(ctx context.Context, tx *storage.Tx[file], g Grab, c downloadclients.Choice) (downloadclients.Retargeted, []db.ListDownloadingRow, error) {
	if g.Stage != Downloading || g.Torrent.Outcome != downloadclients.Outcome(downloadclients.Pending{}) {
		return downloadclients.Retargeted{}, nil, ErrNotSendable
	}
	to, err := downloadclients.Usable(ctx, tx, c, gr.downloading)
	if err != nil {
		return downloadclients.Retargeted{}, nil, err
	}
	now := storage.Now()
	sent, err := downloadclients.Retarget(ctx, tx, g.Torrent.ID, to, now)
	if err != nil {
		return downloadclients.Retargeted{}, nil, err
	}
	q := db.New(tx)
	changed, err := q.ListDownloading(ctx, g.Torrent.ID)
	if err != nil {
		return downloadclients.Retargeted{}, nil, err
	}
	return sent, changed, q.Repoint(ctx, db.RepointParams{ToID: sent.ID, NodeID: sent.Node, UpdatedAt: now, FromID: g.Torrent.ID})
}

// sendFetching records c as the client fetching Grab g goes to, in tx,
// while g waits for its client's Node.
func (gr *Grabs) sendFetching(ctx context.Context, tx *storage.Tx[file], g Grab, c downloadclients.Choice) error {
	waits, err := gr.waitingFor(ctx, tx, g)
	if err != nil {
		return err
	}
	if waits == nil {
		return ErrNotSendable
	}
	if _, err := downloadclients.Usable(ctx, tx, c, gr.downloading); err != nil {
		return err
	}
	p := db.SetSentParams{ID: g.ID, ClientID: c.Client, UpdatedAt: storage.Now()}
	if c.Client == nil {
		p.BuiltInOn = &c.BuiltInOn
	}
	return db.New(tx).SetSent(ctx, p)
}

// downloading reports whether node may take a Grab's download now: it
// holds Downloading and the Admin hasn't written that off.
func (gr *Grabs) downloading(node role.NodeID) bool {
	return gr.cfg.Roster.Holds(role.Downloading, node) && !gr.cfg.Roster.WrittenOff(role.Downloading, node)
}
