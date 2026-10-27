package grabs_test

import (
	"context"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/transfers/source"
	"github.com/jatassi/slipstream/internal/works"
)

func TestADeletedWorksGrabLosesItsTransfer(t *testing.T) {
	p := newPipeline(t)
	id := p.grab("https://site.test/attic.torrent", nil)
	run(t, p.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := p.copies.Start(ctx, tx, transfers.Request{
			Purpose: transfers.Grab, Job: id.String(), Source: p.node, Destination: transfers.Root{ID: p.root},
			Files: []transfers.FileRequest{{Name: "The.Attic.1080p.WEB.mkv", Source: source.Torrent{File: 0}, Size: 4096}},
		})
		return err
	})

	run(t, p.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return p.g.CancelTransfersIn(ctx, tx, []works.WorkID{p.work})
	})

	err := p.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		if _, ok, err := p.copies.Find(ctx, r, transfers.Grab, id.String()); err != nil || ok {
			t.Fatalf("the Grab's Transfer is still recorded (%t, %v), want it cancelled", ok, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
