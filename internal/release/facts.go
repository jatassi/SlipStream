// Package release is what SlipStream knows about a release or a file: the
// fixed quality table, the vocabularies hard filters name, the cuts, and
// Facts, the one typed record the title parser and the probe fill.
package release

import (
	"time"

	"github.com/jatassi/slipstream/internal/works"
)

// Coverage is the Items a release or file holds and their runtime,
// unknown when any covered Item's is.
type Coverage struct {
	Items   []works.ItemID
	Runtime Fact[time.Duration]
	// Original is the Work's original language, unknown until a Content
	// Type facet answers it.
	Original Fact[Language]
}

// Facts is everything known about one release or file. The release parser
// grows here without touching what judges it.
type Facts struct {
	Quality   Fact[Quality]
	Cut       CutKey // Standard when nothing names one
	HDR       Fact[[]HDR]
	Video     Fact[VideoCodec]
	Audio     Fact[[]AudioCodec]
	Channels  Fact[[]Channels]
	Languages Fact[[]Language]
	Size      Fact[int64] // bytes
	Coverage  Coverage
}
