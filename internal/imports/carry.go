package imports

import (
	"context"
	"path"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/transfers"
)

// Importer is imports as grabs reaches it, for the composition root to
// register.
func (im *Imports) Importer() grabs.Importer { return starter{im} }

// Carry carries each Grab grabs 00012 found past its download into an
// Import with the Grab's id, in one Write, before any loop starts. A
// carried row going is the record that it was carried, so a second Carry
// carries nothing.
func (im *Imports) Carry(ctx context.Context) error {
	return im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		carried, err := grabs.Carried(ctx, tx)
		if err != nil {
			return err
		}
		for _, c := range carried {
			h, err := grabs.Handed(ctx, tx, c.Grab)
			if err != nil {
				return err
			}
			imp, err := im.carriedImport(ctx, tx, c, h)
			if err != nil {
				return err
			}
			if err := im.insertConverged(ctx, tx, imp); err != nil {
				return err
			}
			if err := grabs.Uncarry(ctx, tx, c.Grab); err != nil {
				return err
			}
		}
		return nil
	})
}

// carriedImport is c's Import, whose Grab's Handoff is h. A Grab carried
// at probing chose no file, and its Import reads the download as a new
// Grab's does. One past probing chose one file and its cells: that file
// goes first, sending or placing as the Grab was, and the download's
// other videos are left out.
func (im *Imports) carriedImport(ctx context.Context, tx *storage.Tx[file], c grabs.Carry, h grabs.Handoff) (Import, error) {
	id := ImportID{UUID: c.Grab.UUID}
	imp, err := im.grabImport(ctx, tx, h, id)
	if err != nil || c.Was == "probing" {
		return imp, err
	}
	i := slices.IndexFunc(h.Files, func(f engine.File) bool { return f.Index == c.FileIndex })
	tf := h.Files[i]
	chosen := File{
		Path: tf.Path, SourceIndex: new(tf.Index), Size: tf.Size, Probe: &c.Probe, ReleaseName: h.Release,
		SourceName: path.Base(strings.ReplaceAll(tf.Path, `\`, "/")), Items: h.Items, Outcome: Sending,
	}
	for _, t := range c.Fills {
		if k := slices.IndexFunc(h.Cells, func(cell grabs.Cell) bool { return cell.Key.Target == t }); k >= 0 {
			chosen.Fills = append(chosen.Fills, Fill{Cell: h.Cells[k].Key})
		}
	}
	if err := im.readName(ctx, tx, imp.ContentType, imp.Edition, &chosen); err != nil {
		return Import{}, err
	}
	holding, err := holdingOf(id, chosen)
	if c.Was == "importing" {
		chosen.Outcome = Placing
		holding, err = storing.ParseRel(files.HoldingOf(c.Grab.UUID, path.Ext(c.Path)))
	}
	if err != nil {
		return Import{}, err
	}
	if c.Was == "importing" {
		if chosen.Sent, err = im.staged(ctx, tx, id); err != nil {
			return Import{}, err
		}
	}
	chosen.Dest = &Dest{Root: c.Root, Holding: holding}
	out := []File{chosen}
	for _, f := range imp.Files {
		if *f.SourceIndex == c.FileIndex {
			continue
		}
		f.Position, f.Outcome, f.Reason, f.Items = len(out), LeftOut, &Reason{Code: SiblingBetter}, nil
		out = append(out, f)
	}
	imp.Files = out
	return imp, nil
}

// staged is the copy id's Transfer staged of its one file, nil when the
// Grab placed its file from its download on the root's Node.
func (im *Imports) staged(ctx context.Context, r storage.Reader[file], id ImportID) (*Sent, error) {
	t, ok, err := im.cfg.Transfers.Find(ctx, r, transfers.Import, id.String())
	if err != nil || !ok || t.Failure != nil {
		return nil, err
	}
	return &Sent{Transfer: t.ID}, nil
}
