package transfers_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"slices"
	"testing"
	"testing/synctest"

	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// receives is the ids of the Transfers e's intent receives, and keeps.
func (e *nodeEnd) receives() (active, kept []string) {
	in, _ := e.now()
	for _, a := range in.GetActive() {
		if a.GetSide() == tv1.Side_SIDE_RECEIVE {
			active = append(active, hex.EncodeToString(a.GetTransferId()))
		}
	}
	for _, k := range in.GetKeep() {
		kept = append(kept, hex.EncodeToString(k.GetTransferId()))
	}
	return active, kept
}

// TestANormalTransferPreemptsARunningConversionCopyOnItsPair: a Conversion
// copy into home's scratch folder runs alone on its pair, steps behind an
// Import queued there even while its receiver runs it, keeps its staging,
// and runs again once the Import ends.
func TestANormalTransferPreemptsARunningConversionCopyOnItsPair(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		var conv transfers.ID
		r.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			var err error
			conv, err = r.tr.Want(ctx, tx, transfers.Request{
				Purpose: transfers.Conversion, Job: "src/home/7", Source: r.seedbox, Priority: transfers.Low,
				Destination: transfers.Scratch{Node: r.home, Name: "in/s-7.mkv"},
				Files:       []transfers.FileRequest{{Name: "Heat (1995).mkv", Source: source.Library{Root: [16]byte{7}, Rel: "Heat (1995)/Heat (1995).mkv"}, Size: 40 << 30}},
			})
			return err
		})
		in, _ := r.homeEnd.now()
		if a := in.GetActive(); len(a) != 1 || !bytes.Equal(a[0].GetTransferId(), conv.Bytes()) || a[0].GetDestination().GetScratch() != "in/s-7.mkv" {
			t.Fatalf("home receives %v, want the copy into its scratch folder's in/s-7.mkv", a)
		}
		r.homeEnd.facts(&tv1.Facts{Receiving: []*tv1.Receiving{{TransferId: conv.Bytes(), Route: tv1.Route_ROUTE_RECEIVER_DIALS, Active: true, Moving: true, VerifiedBytes: 3 << 30}}})

		imp := r.start(t, "heat", "")
		active, kept := r.homeEnd.receives()
		if !slices.Equal(active, []string{hex.EncodeToString(imp.Bytes())}) {
			t.Errorf("home receives %v with an Import queued, want only the Import %x", active, imp.Bytes())
		}
		if len(kept) != 2 || !slices.Contains(kept, hex.EncodeToString(conv.Bytes())) {
			t.Errorf("home keeps %v, want the copy's staging kept beside the Import's", kept)
		}
		if v := r.view(t, conv); v.State != transfers.Queued || v.Ahead != 1 {
			t.Errorf("the copy is %v ahead %d, want queued behind the Import", v.State, v.Ahead)
		}

		r.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error { return r.end(ctx, tx, "heat") })
		if active, _ := r.homeEnd.receives(); !slices.Equal(active, []string{hex.EncodeToString(conv.Bytes())}) {
			t.Errorf("home receives %v once the Import ended, want the copy %x again", active, conv.Bytes())
		}
	})
}
