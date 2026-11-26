package playback

import (
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Siblings are the files a play could have started on instead: every file
// of each played Item that isn't gone, and each of their Works' goals.
type Siblings struct {
	Files map[works.ItemID][]Sibling
	Goals map[works.WorkID]Goal
}

// WithListed marks each sibling the server's index names, with its row's
// item key.
func (s Siblings) WithListed(itemKeys map[files.ID]string) Siblings {
	out := Siblings{Files: make(map[works.ItemID][]Sibling, len(s.Files)), Goals: s.Goals}
	for item, sibs := range s.Files {
		sibs = slices.Clone(sibs)
		for i := range sibs {
			sibs[i].ItemKey, sibs[i].Listed = itemKeys[sibs[i].File]
		}
		out.Files[item] = sibs
	}
	return out
}

// Of is every file that holds item and isn't gone.
func (s Siblings) Of(item works.ItemID) []Sibling { return s.Files[item] }

type Sibling struct {
	File  files.ID
	Work  works.WorkID
	Label string // the file's own Label, plus " (extra N)" when Kept
	Kept  bool
	// Listed is whether the server's index names it, and ItemKey is its
	// index row's item key.
	Listed   bool
	ItemKey  string
	Met      []TargetRef
	Media    media.Record
	Sidecars []Sidecar
}

// Sidecar is a subtitle file beside a library file.
type Sidecar struct {
	Language string
	Codec    release.SubtitleCodec
}

// Goal is what one Work wants: its Profile and that Profile's Targets in
// Profile order.
type Goal struct {
	Profile TargetRef
	Targets []targets.TargetID
}
