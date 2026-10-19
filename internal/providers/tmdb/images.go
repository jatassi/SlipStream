package tmdb

import (
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
)

// ImagesID is the TMDB image host as an artwork source.
const ImagesID artwork.SourceID = "tmdb-images"

// sizes is TMDB's size name for each size class of each shape it serves.
var sizes = map[artwork.Shape]map[artwork.Size]string{
	artwork.ShapePoster:   {artwork.SizeSmall: "w185", artwork.SizeMedium: "w342", artwork.SizeLarge: "w780", artwork.SizeOriginal: "original"},
	artwork.ShapeBackdrop: {artwork.SizeSmall: "w300", artwork.SizeMedium: "w780", artwork.SizeLarge: "w1280", artwork.SizeOriginal: "original"},
	artwork.ShapeLogo:     {artwork.SizeSmall: "w185", artwork.SizeMedium: "w300", artwork.SizeLarge: "w500", artwork.SizeOriginal: "original"},
	artwork.ShapePhoto:    {artwork.SizeSmall: "w45", artwork.SizeMedium: "w185", artwork.SizeLarge: "h632", artwork.SizeOriginal: "original"},
}

// fileName is one TMDB image file, as a record's *_path names it.
var fileName = regexp.MustCompile(`^/[A-Za-z0-9_-]+\.(jpg|png|svg|webp)$`)

type images struct {
	host    Endpoints
	timings Timings
}

// Images is the TMDB image host as an artwork source. It shares TMDB's
// rate, which TMDB documents for its API alone.
func (c *Client) Images() artwork.Source {
	return images{host: c.endpoints, timings: c.timings}
}

func (images) ID() artwork.SourceID { return ImagesID }

func (i images) URL(shape artwork.Shape, path string, size artwork.Size) (*url.URL, error) {
	name, ok := sizes[shape][size]
	if !ok {
		return nil, fmt.Errorf("TMDB serves no %s %s", size, shape)
	}
	if !fileName.MatchString(path) {
		return nil, fmt.Errorf("%q is not a TMDB image file", path)
	}
	return i.host.Images.JoinPath(name, path), nil
}

func (i images) Rate() (perSecond float64, burst int) {
	return float64(time.Second) / float64(i.timings.Interval), i.timings.Burst
}
