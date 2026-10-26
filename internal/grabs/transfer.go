package grabs

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// startTransfer starts g's Transfer in tx: its planned file, from the Node
// that downloaded it into its root. A Transfer that failed goes and a new
// one starts, as when a re-home takes the root off the Node a Transfer
// was rerouted to.
func (gr *Grabs) startTransfer(ctx context.Context, tx *storage.Tx[file], g Grab) error {
	t, ok, err := gr.cfg.Transfers.Find(ctx, tx, transfers.Grab, g.ID.String())
	if err != nil {
		return err
	}
	if ok && t.Failure != nil {
		if err := gr.cfg.Transfers.Cancel(ctx, tx, t.ID); err != nil {
			return err
		}
	}
	i := slices.IndexFunc(g.Torrent.Files, func(f engine.File) bool { return f.Index == g.Plan.FileIndex })
	if i < 0 {
		return fmt.Errorf("grab %s: its torrent has no file %d", g.ID, g.Plan.FileIndex)
	}
	video := g.Torrent.Files[i]
	_, err = gr.cfg.Transfers.Start(ctx, tx, transfers.Request{
		Purpose:     transfers.Grab,
		Job:         g.ID.String(),
		Source:      g.Node,
		Destination: transfers.Root{ID: g.Plan.Root.ID},
		Files: []transfers.FileRequest{{
			Group:  g.ItemID.String(),
			Name:   path.Base(video.Path),
			Source: source.Torrent{Hash: g.Torrent.InfoHash, File: video.Index},
			Size:   video.Size,
		}},
	})
	return err
}

// cancelTransfer cancels Grab id's Transfer in tx, if it has one.
func (gr *Grabs) cancelTransfer(ctx context.Context, tx *storage.Tx[file], id GrabID) error {
	t, ok, err := gr.cfg.Transfers.Find(ctx, tx, transfers.Grab, id.String())
	if err != nil || !ok {
		return err
	}
	return gr.cfg.Transfers.Cancel(ctx, tx, t.ID)
}

// transferVerified is the Verified hook: a transferring Grab whose file
// is verified moves to importing.
func (gr *Grabs) transferVerified(ctx context.Context, tx *storage.Tx[file], t transfers.Transfer, _ string) error {
	return gr.moveJob(ctx, tx, t.Job, advance{to: Importing})
}

// transferFailed is the Failed hook. A Transfer rerouted onto the Node
// that downloaded the file sends the Grab to importing, which links it
// there. A source written off fails the Grab node_written_off, as the
// write-off pass does. Any other failure fails it transfer_failed with the
// Transfer's sentence.
func (gr *Grabs) transferFailed(ctx context.Context, tx *storage.Tx[file], t transfers.Transfer, f transfers.Failure) error {
	var o outcome = fail{TransferFailed, f.Sentence}
	switch f.Code {
	case transfers.Rerouted:
		o = advance{to: Importing}
	case transfers.SourceWrittenOff:
		o = fail{NodeWrittenOff, writtenOffWords}
	case transfers.SourceGone, transfers.SourceFailed, transfers.Damaged:
	}
	return gr.moveJob(ctx, tx, t.Job, o)
}

// moveJob moves the Grab job names by o in tx, when it still transfers.
// A job no Grab has, or a Grab that left transferring, is nothing to do.
func (gr *Grabs) moveJob(ctx context.Context, tx *storage.Tx[file], job string, o outcome) error {
	id, err := storage.ParseUUID(job)
	if err != nil {
		return nil //nolint:nilerr // a job this purpose never wrote is nothing to do
	}
	row, err := db.New(tx).GetGrab(ctx, GrabID{UUID: id})
	if errors.Is(err, storage.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if row.Stage != Transferring {
		return nil
	}
	rel, err := readRelated(ctx, tx, []db.GrabsGrab{row})
	if err != nil {
		return err
	}
	g, err := load(ctx, tx, row, rel)
	if err != nil {
		return err
	}
	if _, err := gr.move(ctx, tx, g, o); err != nil {
		return err
	}
	gr.mu.Lock()
	gr.moved[job] = Changed{GrabID: job, WorkID: g.WorkID.String()}
	gr.mu.Unlock()
	return nil
}

// transfersCommitted is the After hook: once the Write that moved jobs
// commits, it tells Admins and runs their next step.
func (gr *Grabs) transfersCommitted(jobs []string) {
	gr.mu.Lock()
	var moved []Changed
	for _, job := range jobs {
		if c, ok := gr.moved[job]; ok {
			moved = append(moved, c)
			delete(gr.moved, job)
		}
	}
	gr.mu.Unlock()
	for _, c := range moved {
		gr.cfg.Changed.Publish(context.Background(), c)
	}
	gr.wake()
}
