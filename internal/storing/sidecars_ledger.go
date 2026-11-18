package storing

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing/internal/db"
)

// The ledger is every hash SlipStream wrote as a Sidecar under a root. A
// write records its hash before its bytes land and nothing removes it, so
// a companion whose hash the ledger holds is SlipStream's own even after
// the Controller restores an older Backup.

func (s *Sidecars) putLedger(ctx context.Context, root storage.UUID, sha [32]byte, at Rel) error {
	err := s.local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return db.New(tx).PutLedger(ctx, db.PutLedgerParams{RootID: root, Sha256: sha[:], Path: at.String(), WrittenAt: storage.Now()})
	})
	if err != nil {
		return fmt.Errorf("record the Sidecar in the ledger: %w", err)
	}
	return nil
}

// markLedger sets each companion's ledger flag from the ledger of root.
func (s *Sidecars) markLedger(ctx context.Context, root storage.UUID, cs []*storagev1.SidecarCompanion) error {
	if len(cs) == 0 {
		return nil
	}
	hashes := make([][]byte, len(cs))
	for i, c := range cs {
		hashes[i] = c.GetSha256()
	}
	var held [][]byte
	err := s.local.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Local]) error {
		var err error
		held, err = db.New(r).LedgerHolds(ctx, db.LedgerHoldsParams{RootID: root, Hashes: hashes})
		return err
	})
	if err != nil {
		return fmt.Errorf("read the ledger: %w", err)
	}
	for _, c := range cs {
		c.Ledger = slices.ContainsFunc(held, func(h []byte) bool { return bytes.Equal(h, c.GetSha256()) })
	}
	return nil
}
