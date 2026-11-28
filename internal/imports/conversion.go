package imports

import (
	"context"
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/conversions"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
)

// conversionStarter is the conversions.Importer imports registers: a
// checked output becomes its conversion Import in the Write that moves
// its Conversion to landing.
type conversionStarter struct{ im *Imports }

// Converter is the conversions.Importer the Controller's Conversions
// register.
func (im *Imports) Converter() conversions.Importer { return conversionStarter{im} }

// Start creates h's Conversion's Import, its one file placing from where
// the Conversion left it. A second Start of one Conversion is nothing.
func (s conversionStarter) Start(ctx context.Context, tx *storage.Tx[file], h conversions.Handoff) error {
	if _, err := db.New(tx).GetImportOfConversion(ctx, h.Conversion.UUID); !errors.Is(err, storage.ErrNoRows) {
		return err
	}
	imp, err := s.im.conversionImport(ctx, tx, h)
	if err != nil {
		return err
	}
	return s.im.insertConverged(ctx, tx, imp)
}

// End ends Conversion id's unfinished Import, as the Conversion's cancel
// does: each file not yet final is declined. Refused with
// conversions.ErrPlacing while its file places. Its Conversion hears
// nothing back.
func (s conversionStarter) End(ctx context.Context, tx *storage.Tx[file], id conversions.ID) error {
	imp, err := db.New(tx).GetImportOfConversion(ctx, id.UUID)
	if errors.Is(err, storage.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = s.im.writeIn(ctx, tx, imp, &after{}, func(_ context.Context, _ *storage.Tx[file], cur *Import, _ *after) error {
		if slices.ContainsFunc(cur.Files, func(f File) bool { return f.Outcome == Placing && f.Hold == nil }) {
			return conversions.ErrPlacing
		}
		if cur.Ended != nil {
			return nil
		}
		cur.Ended = &Ending{Reason: ConversionCanceled}
		for i, f := range cur.Files {
			if !f.Outcome.Final() {
				cur.Files[i].Outcome, cur.Files[i].Reason, cur.Files[i].Hold, cur.Files[i].PlacedBy = Declined, &Reason{Code: ConversionCanceled}, nil, ""
			}
		}
		return nil
	})
	return err
}

// Wake has the reconciler look again, once a conversions Write that
// started or ended an Import commits.
func (s conversionStarter) Wake() {
	s.im.Touch()
	s.im.cfg.Changed.Publish(context.Background(), changedOf())
}

// conversionImport is h's Import: its one file where the Conversion
// wrote it in the root's holding folder, or where its output Transfer
// staged it, bound for a holding name of its own.
func (im *Imports) conversionImport(ctx context.Context, r storage.Reader[file], h conversions.Handoff) (Import, error) {
	tree, err := im.cfg.Works.TreeIn(ctx, r, h.Work)
	if err != nil {
		return Import{}, err
	}
	imp := Import{
		ID: ImportID{UUID: storage.NewUUID()}, Source: FromConversion, Conversion: &h.Conversion, Root: h.Root,
		ContentType: tree.ContentType, Work: h.Work, Created: storage.Now(),
	}
	f := File{
		Probe: &h.Media, Title: &h.Title, Cut: h.Title.Facts.Cut, SourceName: h.SourceName, Items: h.Items, Outcome: Placing,
	}
	f.Quality, _ = h.Title.Facts.WithProbe(release.Probed{Resolution: release.Known(h.Media.Resolution())}).Quality.Get()
	f.Size, _ = h.Title.Facts.Size.Get()
	if h.Edition != targets.AnyEdition {
		f.Cut = release.CutKey(h.Edition)
		imp.Edition = &f.Cut
	}
	if h.Holding != nil {
		f.Path = h.Holding.String()
	} else {
		rel, err := holdingOf(imp.ID, File{Path: h.SourceName})
		if err != nil {
			return Import{}, err
		}
		f.Path, f.Sent = rel.String(), &Sent{Transfer: h.Sent.Transfer, Order: h.Sent.Order}
	}
	if f.Dest, err = imp.destOf(h.Root, f); err != nil {
		return Import{}, err
	}
	imp.Files = []File{f}
	return imp, nil
}

// followConversion tells imp's Conversion, in the Write that finished
// imp, how it ended: placed, declined by the Admin, or ended unplaced.
// An Import its Conversion's cancel ended tells it nothing.
func (im *Imports) followConversion(ctx context.Context, tx *storage.Tx[file], imp *Import, a *after) error {
	if imp.Conversion == nil || imp.Ended != nil && imp.Ended.Reason == ConversionCanceled {
		return nil
	}
	var r conversions.Report
	for _, f := range imp.Files {
		switch {
		case f.Outcome == Placed:
			r.Placed, r.File = true, f.FileID
		case f.Outcome == Declined && f.Reason.Code == ByAdmin:
			r.Declined = true
		}
	}
	fol, err := im.cfg.Conversions.Follow(ctx, tx, *imp.Conversion, r)
	a.converted = append(a.converted, fol)
	return err
}

// ConversionsOf reads the Conversion of each of ims that one made.
func (im *Imports) ConversionsOf(ctx context.Context, _ identity.Admin, ims []Import) (map[conversions.ID]conversions.Made, error) {
	out := map[conversions.ID]conversions.Made{}
	err := im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		for _, imp := range ims {
			if imp.Conversion == nil {
				continue
			}
			m, err := conversions.MadeFrom(ctx, r, *imp.Conversion)
			if err != nil {
				return err
			}
			out[*imp.Conversion] = m
		}
		return nil
	})
	return out, err
}
