package imports

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/transfers/source"
	"github.com/jatassi/slipstream/internal/works"
)

// purpose is imports' Transfers purpose. A file sent to another Node's
// root is its own group, keyed by its position.
func (im *Imports) purpose() transfers.Purpose {
	return transfers.Purpose{Kind: transfers.Import, Verified: im.verified, Failed: im.transferFailed, After: im.transfersCommitted, Names: transferNames}
}

// transferRequest is the Transfer imp wants: each file sending, and each
// placing from its staged copy, in position order, from imp's Node into
// their root. Empty when none is, which ends the Transfer.
func (im Import) transferRequest() transfers.Request {
	r := transfers.Request{Purpose: transfers.Import, Job: im.ID.String(), Source: im.Node}
	for _, f := range im.Files {
		if f.Outcome != Sending && (f.Outcome != Placing || f.Sent == nil) {
			continue
		}
		r.Destination = transfers.Root{ID: f.Dest.Root}
		r.Files = append(r.Files, transfers.FileRequest{Group: groupOf(f), Name: f.SourceName, Source: sentOf(im, f), Size: f.Size})
	}
	return r
}

func groupOf(f File) string { return strconv.Itoa(f.Position) }

// sentOf is what the Transfer reads for f on im's Node.
func sentOf(im Import, f File) source.Ref {
	if im.Source == FromGrab {
		return source.Torrent{Hash: *im.InfoHash, File: *f.SourceIndex}
	}
	return source.Path{Abs: f.Path}
}

// verified is the Verified hook: the sending file at group's position is
// staged whole in its root, and places from there.
func (im *Imports) verified(ctx context.Context, tx *storage.Tx[file], t transfers.Transfer, group string) error {
	position, err := strconv.Atoi(group)
	if err != nil {
		return fmt.Errorf("transfer group %q: %w", group, err)
	}
	// A group dropped and wanted again is sent again at a later position.
	order := -1
	for _, f := range t.Files {
		if f.Group == group {
			order = f.Position
		}
	}
	return im.hook(ctx, tx, t.Job, func(cur *Import) {
		if i := indexOf(cur, position); i >= 0 && cur.Files[i].Outcome == Sending {
			cur.Files[i].Outcome, cur.Files[i].Sent = Placing, &Sent{Transfer: t.ID, Order: order}
		}
	})
}

// transferFailed is the Failed hook, for the files still sending. A
// Transfer rerouted onto the source's Node places them there. A source
// gone or written off ends the Import as a removal does. Bytes that
// arrived damaged or a source that failed fail them, and Retry sends them
// again. A file placing from its staged copy finishes either way.
func (im *Imports) transferFailed(ctx context.Context, tx *storage.Tx[file], t transfers.Transfer, f transfers.Failure) error {
	return im.hook(ctx, tx, t.Job, func(cur *Import) {
		switch f.Code {
		case transfers.SourceGone:
			endImport(cur, Ending{Reason: Removed, Code: "removed"})
			return
		case transfers.SourceWrittenOff:
			endImport(cur, Ending{Reason: Removed, Code: "node_written_off"})
			return
		case transfers.Rerouted, transfers.Damaged, transfers.SourceFailed:
		}
		for i, g := range cur.Files {
			if g.Outcome != Sending {
				continue
			}
			if f.Code == transfers.Rerouted {
				cur.Files[i].Outcome = Placing
				continue
			}
			cur.Files[i].Outcome, cur.Files[i].Reason = FailedOut, &Reason{Code: ImportFail, Line: f.Sentence}
		}
	})
}

// endImport ends im outside its own steps: each file not yet final or
// placing is declined for e's reason.
func endImport(im *Import, e Ending) {
	im.Ended = &e
	for i, f := range im.Files {
		if f.Outcome.Final() || f.Outcome == Placing {
			continue
		}
		im.Files[i].Outcome, im.Files[i].Reason, im.Files[i].Hold = Declined, &Reason{Code: e.Reason}, nil
	}
}

// hook runs change over job's Import in a Transfers hook's Write and
// converges it, keeping what to publish for transfersCommitted. An Import
// that went is nothing to do.
func (im *Imports) hook(ctx context.Context, tx *storage.Tx[file], job string, change func(*Import)) error {
	id, err := storage.ParseUUID(job)
	if err != nil {
		return err
	}
	var a after
	_, err = im.writeIn(ctx, tx, ImportID{UUID: id}, &a, func(_ context.Context, _ *storage.Tx[file], cur *Import, _ *after) error {
		change(cur)
		return nil
	})
	if errors.Is(err, ErrNoImport) {
		return nil
	}
	if err != nil {
		return err
	}
	im.mu.Lock()
	defer im.mu.Unlock()
	was := im.hooked[job]
	im.hooked[job] = after{roots: append(was.roots, a.roots...), landed: append(was.landed, a.landed...), sent: true}
	return nil
}

// transfersCommitted is the After hook: once the hooks' Write commits, it
// publishes what they changed and has the reconciler place what verified.
func (im *Imports) transfersCommitted(jobs []string) {
	im.mu.Lock()
	var done []after
	for _, job := range jobs {
		if a, ok := im.hooked[job]; ok {
			done = append(done, a)
			delete(im.hooked, job)
		}
	}
	im.mu.Unlock()
	for _, a := range done {
		im.publish(context.Background(), a)
	}
	im.Touch()
}

// transferNames is the Names hook: each job's Import named by its Work's
// label, read in r.
func transferNames(ctx context.Context, r storage.Reader[file], jobs []string) (map[string]string, error) {
	out := make(map[string]string, len(jobs))
	for _, job := range jobs {
		id, err := storage.ParseUUID(job)
		if err != nil {
			return nil, err
		}
		imp, err := importOf(ctx, r, ImportID{UUID: id})
		if errors.Is(err, ErrNoImport) {
			continue
		}
		if err != nil {
			return nil, err
		}
		w, err := works.SummaryOf(ctx, r, imp.Work)
		if err != nil {
			return nil, err
		}
		out[job] = w.Label
	}
	return out, nil
}
