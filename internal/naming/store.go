package naming

import (
	"context"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type Config struct {
	Facets map[works.TypeID]Namer
}

// Naming owns each Content Type's naming: the scheme its files are named
// under and the values its Works name them with.
type Naming struct{ cfg Config }

func New(cfg Config) *Naming { return &Naming{cfg: cfg} }

// SchemeIn is t's scheme in r: its Declaration's templates under the
// default switches.
func (n *Naming) SchemeIn(_ context.Context, _ storage.Reader[storage.Library], t works.TypeID) (*Scheme, error) {
	return newScheme(n.cfg.Facets[t].Naming(), Settings{})
}

// ValuesIn reads ws's values through t's facet, in r.
func (n *Naming) ValuesIn(ctx context.Context, r storage.Reader[storage.Library], t works.TypeID, ws works.Among[works.WorkID]) (map[works.WorkID]Values, error) {
	return n.cfg.Facets[t].NamingValues(ctx, r, ws)
}
