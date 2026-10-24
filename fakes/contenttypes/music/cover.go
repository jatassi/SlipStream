package music

import (
	"fmt"
	"math"
	"net/url"
	"regexp"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

// CoverArtID is the music test type's cover-art host as an artwork source.
const CoverArtID artwork.SourceID = "music-coverart"

var coverKind = artwork.Kind{Key: "cover", Label: "Cover", Shape: artwork.ShapeCover, Prefetch: artwork.SizeMedium}

// thumbnails is the front image the host serves for each size class.
var thumbnails = map[artwork.Size]string{
	artwork.SizeSmall: "front-250", artwork.SizeMedium: "front-500", artwork.SizeLarge: "front-1200", artwork.SizeOriginal: "front",
}

var releaseGroup = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// covers is the cover-art host: a release group's front cover, by the
// release group's id.
type covers struct{ host endpoint.URL }

func (covers) ID() artwork.SourceID { return CoverArtID }

func (c covers) URL(shape artwork.Shape, path string, size artwork.Size) (*url.URL, error) {
	name, ok := thumbnails[size]
	if shape != artwork.ShapeCover || !ok {
		return nil, fmt.Errorf("the cover-art host serves no %s %s", size, shape)
	}
	if !releaseGroup.MatchString(path) {
		return nil, fmt.Errorf("%q is not a release group id", path)
	}
	return c.host.JoinPath("release-group", path, name), nil
}

// Rate is unbounded: the fake keeps no rate limit.
func (covers) Rate() (perSecond float64, burst int) { return math.MaxFloat64, 1 }
