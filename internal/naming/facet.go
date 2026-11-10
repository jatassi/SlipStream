// Package naming turns a Content Type's templates and values into the
// folder and file names its files are placed under. One character rule
// serves every OS, so a library made on Linux opens on Windows.
package naming

import (
	"context"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Facet is how a Content Type names its files: what it declares, and the
// values it reads for any set of its Works.
type Facet interface {
	// Naming is the same Declaration on every call.
	Naming() Declaration
	// NamingValues reads the values of each of ws in a few queries: the
	// Work's own, each level's and each Item's with the variant it picks.
	// A Work the type has no row for is absent.
	NamingValues(ctx context.Context, r storage.Reader[storage.Library], ws works.Among[works.WorkID]) (map[works.WorkID]Values, error)
}
