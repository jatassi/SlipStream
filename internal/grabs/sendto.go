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

// ErrNotDownloading is a send-to of a Grab whose Torrent isn't downloading:
// it is past or short of downloading, or its download finished, failed or
// was removed.
var ErrNotDownloading = errors.New("the Grab isn't downloading")

// SendTo sends downloading Grab id's Torrent to the client c names, in one
// Write that checks the client can take it, retargets the Torrent and
// points every Grab downloading it at the Torrent and Node that answers.
// It tells the Nodes whose intents changed once it commits. It fails with
// ErrNoGrab, ErrNotDownloading, and Usable's and Retarget's errors.
func (gr *Grabs) SendTo(ctx context.Context, _ identity.Admin, id GrabID, c downloadclients.Choice) error {
	var (
		sent    downloadclients.Retargeted
		changed []db.ListDownloadingRow
	)
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		row, err := q.GetGrab(ctx, id)
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
		if g.Stage != Downloading || g.Torrent.Outcome != downloadclients.Outcome(downloadclients.Pending{}) {
			return ErrNotDownloading
		}
		to, err := downloadclients.Usable(ctx, tx, c, gr.downloading)
		if err != nil {
			return err
		}
		now := storage.Now()
		if sent, err = downloadclients.Retarget(ctx, tx, g.Torrent.ID, to, now); err != nil {
			return err
		}
		if changed, err = q.ListDownloading(ctx, g.Torrent.ID); err != nil {
			return err
		}
		return q.Repoint(ctx, db.RepointParams{ToID: sent.ID, NodeID: sent.Node, UpdatedAt: now, FromID: g.Torrent.ID})
	})
	if err != nil {
		return err
	}
	gr.cfg.Downloads.Retargeted(sent)
	for _, r := range changed {
		gr.cfg.Changed.Publish(ctx, Changed{GrabID: r.ID.String(), WorkID: r.WorkID.String()})
	}
	gr.wake()
	return nil
}

// downloading reports whether node may take a Grab's download now: it
// holds Downloading and the Admin hasn't written that off.
func (gr *Grabs) downloading(node role.NodeID) bool {
	return gr.cfg.Roster.Holds(role.Downloading, node) && !gr.cfg.Roster.WrittenOff(role.Downloading, node)
}
