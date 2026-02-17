package downloader

import (
	"context"
	"testing"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/downloader/transmission"
	"github.com/slipstream/slipstream/internal/testutil"
)

func createTestClient(t *testing.T, queries *sqlc.Queries) *sqlc.DownloadClient {
	t.Helper()
	dc, err := queries.CreateDownloadClient(context.Background(), sqlc.CreateDownloadClientParams{
		Name:        "Test Transmission",
		Type:        "transmission",
		Host:        "localhost",
		Port:        9091,
		Enabled:     1,
		Priority:    50,
		CleanupMode: "leave",
	})
	if err != nil {
		t.Fatalf("CreateDownloadClient error = %v", err)
	}
	return dc
}

func TestClientPool_CacheHit(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	svc := NewService(tdb.Conn, &tdb.Logger)
	ctx := context.Background()

	queries := sqlc.New(tdb.Conn)
	dc := createTestClient(t, queries)

	c1, err := svc.GetClient(ctx, dc.ID)
	if err != nil {
		t.Fatalf("GetClient(1) error = %v", err)
	}

	c2, err := svc.GetClient(ctx, dc.ID)
	if err != nil {
		t.Fatalf("GetClient(2) error = %v", err)
	}

	// Both should be the same *transmission.Client pointer
	tc1, ok := c1.(*transmission.Client)
	if !ok {
		t.Fatalf("expected *transmission.Client, got %T", c1)
	}
	tc2, ok := c2.(*transmission.Client)
	if !ok {
		t.Fatalf("expected *transmission.Client, got %T", c2)
	}

	if tc1 != tc2 {
		t.Error("expected same pointer from pool, got different instances")
	}
}

func TestClientPool_InvalidateOnUpdate(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	svc := NewService(tdb.Conn, &tdb.Logger)
	ctx := context.Background()

	queries := sqlc.New(tdb.Conn)
	dc := createTestClient(t, queries)

	c1, err := svc.GetClient(ctx, dc.ID)
	if err != nil {
		t.Fatalf("GetClient(1) error = %v", err)
	}

	_, err = svc.Update(ctx, dc.ID, &UpdateClientInput{
		Name:        "Updated Transmission",
		Type:        "transmission",
		Host:        "localhost",
		Port:        9091,
		Enabled:     true,
		Priority:    50,
		CleanupMode: "leave",
	})
	if err != nil {
		t.Fatalf("Update error = %v", err)
	}

	c2, err := svc.GetClient(ctx, dc.ID)
	if err != nil {
		t.Fatalf("GetClient(2) error = %v", err)
	}

	tc1 := c1.(*transmission.Client)
	tc2 := c2.(*transmission.Client)

	if tc1 == tc2 {
		t.Error("expected different pointer after Update, got same instance")
	}
}

func TestClientPool_InvalidateOnDelete(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	svc := NewService(tdb.Conn, &tdb.Logger)
	ctx := context.Background()

	queries := sqlc.New(tdb.Conn)
	dc := createTestClient(t, queries)

	_, err := svc.GetClient(ctx, dc.ID)
	if err != nil {
		t.Fatalf("GetClient error = %v", err)
	}

	// Verify pool has the entry
	svc.clientPoolMu.RLock()
	_, pooled := svc.clientPool[dc.ID]
	svc.clientPoolMu.RUnlock()
	if !pooled {
		t.Fatal("expected client in pool after GetClient")
	}

	if err := svc.Delete(ctx, dc.ID); err != nil {
		t.Fatalf("Delete error = %v", err)
	}

	// Pool entry should be gone
	svc.clientPoolMu.RLock()
	_, pooled = svc.clientPool[dc.ID]
	svc.clientPoolMu.RUnlock()
	if pooled {
		t.Error("expected client evicted from pool after Delete")
	}

	// GetClient should now fail since the client is deleted from the DB
	_, err = svc.GetClient(ctx, dc.ID)
	if err == nil {
		t.Error("expected error from GetClient after Delete, got nil")
	}
}

func TestClientPool_ClearOnSetDB(t *testing.T) {
	tdb1 := testutil.NewTestDB(t)
	defer tdb1.Close()
	tdb2 := testutil.NewTestDB(t)
	defer tdb2.Close()

	svc := NewService(tdb1.Conn, &tdb1.Logger)
	ctx := context.Background()

	queries := sqlc.New(tdb1.Conn)
	dc := createTestClient(t, queries)

	_, err := svc.GetClient(ctx, dc.ID)
	if err != nil {
		t.Fatalf("GetClient error = %v", err)
	}

	// Verify pool is populated
	svc.clientPoolMu.RLock()
	poolLen := len(svc.clientPool)
	svc.clientPoolMu.RUnlock()
	if poolLen == 0 {
		t.Fatal("expected non-empty pool after GetClient")
	}

	svc.SetDB(tdb2.Conn)

	// Pool should be empty after SetDB
	svc.clientPoolMu.RLock()
	poolLen = len(svc.clientPool)
	svc.clientPoolMu.RUnlock()
	if poolLen != 0 {
		t.Errorf("expected empty pool after SetDB, got %d entries", poolLen)
	}

	// GetClient should fail since client doesn't exist in new DB
	_, err = svc.GetClient(ctx, dc.ID)
	if err == nil {
		t.Error("expected error from GetClient after SetDB to empty DB, got nil")
	}
}
