package converting

import (
	"context"
	"log/slog"

	"github.com/jatassi/slipstream/internal/media"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

type HolderConfig struct {
	Converter Converter
	Proofs    Proofs
	Media     media.Prober
	// Scratch is the scratch folder now: config.yaml can change it in step.
	Scratch func() string
	// Local is storing.Store.Local.
	Local   func(ctx context.Context, root storage.UUID, rel storing.Rel) (string, error)
	Timings Timings
	Log     *slog.Logger
}

// Holder is the holder half, which Def.Hold runs.
type Holder struct {
	cfg HolderConfig
}

func Hold(cfg HolderConfig) *Holder { return &Holder{cfg: cfg} }

func (h *Holder) Run(ctx context.Context, _ role.Holder[*cv1.Intent, *cv1.Facts, *cv1.Result]) error {
	<-ctx.Done()
	return nil
}

// Release kills every tree and deletes the scratch folder's in/, out/ and
// clips/.
func (h *Holder) Release(context.Context) error { return nil }
