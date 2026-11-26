package mediaservers

import (
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/mediaservers/playback"
	"github.com/jatassi/slipstream/internal/works"
)

// PlayedFile is a library file as a play's snapshot keeps it.
type PlayedFile struct {
	Work works.WorkID
	// Item is its first Item in tree order.
	Item  works.ItemID
	Label string // the file's own Label, plus " (extra N)" for a Kept file
	Met   []playback.TargetRef
	Media media.Record
}
