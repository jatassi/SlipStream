package tvdb

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
)

// ArtworkID is the source id of TVDB's image host, apart from the API's
// Line.
const ArtworkID artwork.SourceID = "tvdb-artwork"

// artworkFile is one image on TVDB's host, as the path of the URL a record
// names it by.
var artworkFile = regexp.MustCompile(`^/banners/(?:[A-Za-z0-9_-]+/)*[A-Za-z0-9_-]+\.(?:jpg|jpeg|png|webp)$`)

// artworkPath is the path of the artwork URL raw, empty when raw names
// nothing on TVDB's host. A record names its artwork by absolute URL; a
// stored reference keeps the path, so a changed endpoint never invalidates
// it.
func artworkPath(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || !artworkFile.MatchString(u.Path) {
		return ""
	}
	return u.Path
}

type artworks struct {
	host    Endpoints
	timings Timings
}

// Artwork is the image host TVDB's artwork paths live on, rebased onto
// Endpoints.Artwork: originals, and the _t thumbnail TVDB keeps beside
// each for the small and medium sizes. It shares TVDB's rate.
func (c *Client) Artwork() artwork.Source {
	return artworks{host: c.endpoints, timings: c.timings}
}

func (artworks) ID() artwork.SourceID { return ArtworkID }

func (a artworks) URL(shape artwork.Shape, path string, size artwork.Size) (*url.URL, error) {
	if !artworkFile.MatchString(path) {
		return nil, fmt.Errorf("%q is not a TVDB artwork path", path)
	}
	switch size {
	case artwork.SizeSmall, artwork.SizeMedium:
		dot := strings.LastIndexByte(path, '.')
		path = path[:dot] + "_t" + path[dot:]
	case artwork.SizeLarge, artwork.SizeOriginal:
	default:
		return nil, fmt.Errorf("TVDB serves no %s %s", size, shape)
	}
	return a.host.Artwork.JoinPath(path), nil
}

func (a artworks) Rate() (perSecond float64, burst int) {
	return float64(time.Second) / float64(a.timings.Interval), a.timings.Burst
}
