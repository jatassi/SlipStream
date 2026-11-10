package grabs

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/storage"
)

var ErrGrabFinished = errors.New("the Grab is imported or failed")

// GrabBusyError is a cancel of a Grab placing its file. Words says what it
// waits for, "" when it waits for nothing.
type GrabBusyError struct {
	Stage Stage
	Words string
}

func (e *GrabBusyError) Error() string { return "the Grab is " + string(e.Stage) }

// Cancel stops Grab id: at fetching it fails; at downloading its Torrent
// goes with reason canceled and it fails; at transferring it fails and
// the Transfer is canceled. With blocklist and an info hash known, its
// Release gets a by_hand entry. Canceling a canceled Grab changes nothing.
// It fails with ErrNoGrab, ErrGrabFinished, *GrabBusyError and
// ErrReconciling.
func (gr *Grabs) Cancel(ctx context.Context, by identity.Admin, id GrabID, blocklist bool) error {
	var c canceledGrab
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
		c, err = gr.cancelIn(ctx, tx, g, actorOf(by), blocklist)
		return err
	})
	if err != nil {
		return err
	}
	c.publish(ctx, gr)
	return nil
}

// canceledGrab is what a cancel publishes once its Write commits: nothing
// when the Grab was canceled already.
type canceledGrab struct {
	grab    *Grab
	touched *downloadclients.Touched
	blocked bool
}

func (c canceledGrab) publish(ctx context.Context, gr *Grabs) {
	if c.grab == nil {
		return
	}
	gr.entered(ctx, *c.grab, Failed)
	if c.touched != nil {
		gr.cfg.Downloads.Touched(*c.touched)
	}
	if c.blocked {
		gr.cfg.BlocklistChanged.Publish(ctx, BlocklistChanged{})
	}
	gr.wake()
}

// cancelIn is the one cancel rule, in tx, which a grab that replaces a
// downloading Grab calls too. It compares on the stage g was read at, so a
// fetch that commits after it finds g gone from fetching and commits
// nothing, and one that committed first makes g downloading here.
func (gr *Grabs) cancelIn(ctx context.Context, tx *storage.Tx[file], g Grab, by Actor, blocklist bool) (canceledGrab, error) {
	var c canceledGrab
	switch g.Stage {
	case Failed:
		if g.Failed.Code == Canceled {
			return canceledGrab{}, nil
		}
		return canceledGrab{}, ErrGrabFinished
	case Imported:
		return canceledGrab{}, ErrGrabFinished
	case Probing, Importing:
		words, err := gr.waitWords(ctx, tx, g)
		if err != nil {
			return canceledGrab{}, err
		}
		return canceledGrab{}, &GrabBusyError{Stage: g.Stage, Words: words}
	case Downloading:
		t, err := gr.removeCanceled(ctx, tx, g)
		if err != nil {
			return canceledGrab{}, err
		}
		c.touched = &t
	case Fetching, Transferring:
	}
	if h := g.infoHash(); blocklist && h != nil {
		if err := gr.blockByHand(ctx, tx, g, *h, by); err != nil {
			return canceledGrab{}, err
		}
		c.blocked = true
	}
	o := fail{Canceled, canceledWords(by)}
	if _, err := gr.move(ctx, tx, g, o); err != nil {
		return canceledGrab{}, err
	}
	if g.Stage == Fetching {
		if _, _, err := gr.left(ctx, tx, g, o, nil); err != nil {
			return canceledGrab{}, err
		}
	}
	c.grab = &g
	return c, nil
}

// removeCanceled removes downloading Grab g's Torrent with reason
// canceled, with its download copy unless SlipStream only attached it.
// The Torrent's other downloading Grabs then read it gone and fail
// canceled. A Torrent already removed needs no removal. A restored
// Controller removes nothing while Reconciling.
func (gr *Grabs) removeCanceled(ctx context.Context, tx *storage.Tx[file], g Grab) (downloadclients.Touched, error) {
	reconciling, err := gr.cfg.Reconciling(ctx, tx)
	if err != nil {
		return downloadclients.Touched{}, err
	}
	if reconciling {
		return downloadclients.Touched{}, ErrReconciling
	}
	results, t, err := gr.cfg.Downloads.RemoveIn(ctx, tx, []downloadclients.ID{g.Torrent.ID}, downloadclients.Canceled, true)
	if err != nil {
		return downloadclients.Touched{}, err
	}
	if r := results[0].Refusal; r != nil && r.Code == downloadclients.RefusedImporting {
		return downloadclients.Touched{}, &GrabBusyError{Stage: Importing, Words: "Another Grab is placing its file from this Grab's torrent. Cancel this Grab once that finishes."}
	}
	return t, nil
}

// waitWords is what probing or importing Grab g waits for, read in r: the
// Node its step runs on while that Node is out of step, else "".
func (gr *Grabs) waitWords(ctx context.Context, r storage.Reader[file], g Grab) (string, error) {
	node := g.Node
	if g.Stage == Importing {
		node = g.Plan.Root.Node
	}
	if gr.cfg.Roster.InStep(node) {
		return "", nil
	}
	named, err := nodes.Names(ctx, r)
	if err != nil {
		return "", err
	}
	for _, n := range named {
		if n.ID == node {
			return "This Grab is waiting for " + n.Name + " to come back. It can't be canceled until it places its file.", nil
		}
	}
	return "", nil
}

// infoHash is the info hash of g's Release: its Torrent's once fetched,
// else the Indexer's or its magnet link's, nil when none is known.
func (g Grab) infoHash() *bencode.InfoHash {
	if g.Torrent != nil {
		return &g.Torrent.InfoHash
	}
	if g.Release.InfoHash != nil {
		return g.Release.InfoHash
	}
	if l, err := magnet.Parse(g.Release.magnet); err == nil {
		return &l.InfoHash
	}
	return nil
}

// cancellable reports whether a cancel stops g now.
func (g Grab) cancellable() bool {
	return g.Stage == Fetching || g.Stage == Downloading || g.Stage == Transferring
}

// blockByHand puts g's Release, by its info hash h, on the Blocklist as
// by blocklisted it when canceling g.
func (gr *Grabs) blockByHand(ctx context.Context, tx *storage.Tx[file], g Grab, h bencode.InfoHash, by Actor) error {
	e := BlockEntry{
		InfoHash: h, Title: g.Release.Title, Size: g.Release.Size, Indexer: g.Ticket.Indexer, Tracker: g.Ticket.Tracker,
		Work: g.WorkID, Grab: g.ID, Cause: CauseByHand, By: &by,
	}
	if !g.Ticket.Indexer.IsZero() {
		idx, err := indexers.Get(ctx, tx, g.Ticket.Indexer)
		if err != nil {
			return err
		}
		e.IndexerName = idx.Name
	}
	return BlocklistOn{Reconciling: gr.cfg.Reconciling}.Block(ctx, tx, e)
}

// canceledWords is a canceled Grab's failure message.
func canceledWords(by Actor) string {
	if by.Name == "" {
		return "This Grab was canceled through the API key."
	}
	return by.Name + " canceled this Grab."
}
