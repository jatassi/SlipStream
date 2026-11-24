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

// The ledger is every hash of a provider's subtitle SlipStream wrote as a
// Sidecar under a root. A write marked Ledger records its hash before its
// bytes land, and nothing removes it. A companion whose hash the ledger
// holds is SlipStream's own even after the Controller restores an older
// Backup, so an Admin's bytes the Keeper moves or retimes stay out of it.

func (s *Sidecars) putLedger(ctx context.Context, root storage.UUID, record bool, sha [32]byte, at Rel) error {
	if !record {
		return nil
	}
	return putLedger(ctx, s.local, root, sha, at)
}

func putLedger(ctx context.Context, local *storage.DB[storage.Local], root storage.UUID, sha [32]byte, at Rel) error {
	err := local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
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
	held, err := ledgerHolds(ctx, s.local, root, hashes)
	if err != nil {
		return err
	}
	for _, c := range cs {
		c.Ledger = containsHash(held, c.GetSha256())
	}
	return nil
}

func containsHash(held [][]byte, h []byte) bool {
	return slices.ContainsFunc(held, func(x []byte) bool { return bytes.Equal(x, h) })
}

// ledgerHolds is which of hashes the ledger of root holds.
func ledgerHolds(ctx context.Context, local *storage.DB[storage.Local], root storage.UUID, hashes [][]byte) ([][]byte, error) {
	var held [][]byte
	err := local.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Local]) error {
		var err error
		held, err = db.New(r).LedgerHolds(ctx, db.LedgerHoldsParams{RootID: root, Hashes: hashes})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("read the ledger: %w", err)
	}
	return held, nil
}
