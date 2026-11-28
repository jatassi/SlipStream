package conversions

import (
	"context"

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
