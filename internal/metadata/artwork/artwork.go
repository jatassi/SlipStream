// Package artwork is the vocabulary of images a Content Type declares and a
// provider serves: references, shapes, size classes and the sources that
// turn a reference into a URL. It does no network IO.
package artwork

import (
	"fmt"
	"net/url"
)

// Shape is an image's aspect, which decides the sizes a source offers.
type Shape string

const (
	ShapePoster   Shape = "poster"
	ShapeBackdrop Shape = "backdrop"
	ShapeLogo     Shape = "logo"
	ShapePhoto    Shape = "photo"
	ShapeCover    Shape = "cover"
)

func ParseShape(s string) (Shape, error) {
	switch v := Shape(s); v {
	case ShapePoster, ShapeBackdrop, ShapeLogo, ShapePhoto, ShapeCover:
		return v, nil
	}
	return "", fmt.Errorf("%q is not an artwork shape", s)
}

// Size is a size class; each source maps it to one of its own sizes per
// shape.
type Size string

const (
	SizeSmall    Size = "small"
	SizeMedium   Size = "medium"
	SizeLarge    Size = "large"
	SizeOriginal Size = "original"
)

func ParseSize(s string) (Size, error) {
	switch v := Size(s); v {
	case SizeSmall, SizeMedium, SizeLarge, SizeOriginal:
		return v, nil
	}
	return "", fmt.Errorf("%q is not an artwork size", s)
}

type SourceID string

// Ref names an image at its source. It never holds a URL, so a changed
// endpoint never invalidates a stored reference.
type Ref struct {
	Source SourceID
	Shape  Shape
	Path   string
}

func (r Ref) IsZero() bool { return r == Ref{} }

// Want is one image at one size, as a prefetch asks for it.
type Want struct {
	Ref  Ref
	Size Size
}

// Kind is one artwork kind a Content Type declares.
type Kind struct {
	Key   string
	Label string
	// Item is true for a kind each Item has, false for one per Work.
	Item  bool
	Shape Shape
	// Prefetch is the size fetched when a Work is added, empty for none.
	Prefetch Size
}

// Source turns a path and a size into a URL on its provider's Endpoints
// entry, refusing a path that isn't one of its file names. Provider
// packages build them; a type's provider half lists the ones it uses.
type Source interface {
	ID() SourceID
	URL(shape Shape, path string, size Size) (*url.URL, error)
	// Rate is the image host's token bucket per holder.
	Rate() (perSecond float64, burst int)
}

type Sources map[SourceID]Source

// NewSources is the union of lists by ID. Two different Sources claiming
// one ID is a wiring bug, so it panics.
func NewSources(lists ...[]Source) Sources {
	s := Sources{}
	for _, l := range lists {
		for _, src := range l {
			if had, ok := s[src.ID()]; ok && had != src {
				panic(fmt.Sprintf("artwork: two sources claim the id %q", src.ID()))
			}
			s[src.ID()] = src
		}
	}
	return s
}
