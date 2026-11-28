package conversions

import (
	"context"
	"path"
	"strings"

	"github.com/jatassi/slipstream/internal/encode"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

// Importer is imports, which places a made file as a conversion Import.
type Importer interface {
	// Start creates the Import. A second Start of one Conversion is
	// nothing.
	Start(ctx context.Context, tx *storage.Tx[storage.Library], h Handoff) error
	// End ends its Import, refused with ErrPlacing while its file places.
	End(ctx context.Context, tx *storage.Tx[storage.Library], id ID) error
	Wake()
}

// Handoff is a checked output, for its Import. Exactly one of Holding and
// Sent is set.
type Handoff struct {
	Conversion ID
	Work       works.WorkID
	Items      []works.ItemID
	Edition    targets.Edition
	Root       roots.ID
	// Holding is set when ffmpeg wrote the file into the root's holding
	// folder.
	Holding *storing.Rel
	// Sent is set when an output Transfer brought it.
	Sent       *Sent
	Media      media.Record
	Title      release.Read
	SourceName string
	Derived    files.Derived
}

type Sent struct {
	Transfer transfers.ID
	Order    int
}

// handoff hands a checked output to imports in the Write that checked it.
// The Conversion stays landing until imports reports through Follow.
func handoff(ctx context.Context, m *move) error {
	if m.row.Stage != StageLanding {
		return nil
	}
	src, err := m.source(ctx)
	if err != nil {
		return err
	}
	t, err := m.target(ctx)
	if err != nil {
		return err
	}
	h := Handoff{
		Conversion: m.row.ID, Work: m.row.Work, Items: m.row.Items, Edition: m.row.Edition, Root: *m.row.Root,
		Media: *m.ev.Probe, Title: encode.Rewrite(src.Read, *m.ev.Probe, encode.CopiesVideo(src.Media, t.Settings)),
		SourceName: keptName(m.row.Snapshot, t.Settings.Container),
		Derived:    files.Derived{From: src.ID, Target: *m.row.Target, Version: m.row.Version},
	}
	if src.Node == *m.row.Node {
		rel := holdingRel(m.row.ID, m.row.Attempt, t.Settings.Container)
		h.Holding = &rel
	} else {
		id, err := m.outputTransfer(ctx, 0)
		if err != nil {
			return err
		}
		h.Sent = &Sent{Transfer: id}
	}
	return m.c.importer.Start(ctx, m.tx, h)
}

// keptName is the made file's name when renaming is off: the source's
// stem, then the Target, as "Heat.1995.2160p.UHD.BluRay - 720p mobile.mkv".
func keptName(s Snapshot, c encode.Container) string {
	return strings.TrimSuffix(s.SourceName, path.Ext(s.SourceName)) + " - " + s.Target + "." + string(c)
}
