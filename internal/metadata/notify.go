package metadata

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
)

// Sources are the image hosts every type's source names.
func (c HolderConfig) Sources() artwork.Sources {
	lists := make([][]artwork.Source, 0, len(c.Types))
	for _, s := range c.Types {
		lists = append(lists, s.Sources())
	}
	return artwork.NewSources(lists...)
}

// Image is ref at size: the stored one, or else one fetched through a
// holder, as Artwork does for a token.
func (m *Metadata) Image(ctx context.Context, ref artwork.Ref, size artwork.Size) (artwork.Image, error) {
	return m.image(ctx, artwork.Want{Ref: ref, Size: size})
}

// PublicURL is ref's URL on its source's own public host, for a provider
// that takes an image only by URL. Fetching it tells that host which title
// it is.
func (m *Metadata) PublicURL(ref artwork.Ref, size artwork.Size) (string, error) {
	src, ok := m.cfg.Sources[ref.Source]
	if !ok {
		return "", fmt.Errorf("this Controller has no artwork source %q", ref.Source)
	}
	u, err := src.URL(ref.Shape, ref.Path, size)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
