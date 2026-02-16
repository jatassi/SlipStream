package history

import (
	"context"
	"testing"

	"github.com/slipstream/slipstream/internal/testutil"
)

// Tests for the history service status consolidation features.
// Spec: docs/status-consolidation.md - "History Integration"

func TestHistoryService_Create(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, &tdb.Logger)
	ctx := context.Background()

	entry, err := service.Create(ctx, &CreateInput{
		EventType: EventTypeGrabbed,
		MediaType: MediaTypeMovie,
		MediaID:   1,
		Source:    "test-indexer",
		Quality:   "Bluray-1080p",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if entry.ID == 0 {
		t.Error("Create() entry.ID = 0, want non-zero")
	}
	if entry.EventType != EventTypeGrabbed {
		t.Errorf("Create() EventType = %q, want %q", entry.EventType, EventTypeGrabbed)
	}
	if entry.MediaType != MediaTypeMovie {
		t.Errorf("Create() MediaType = %q, want %q", entry.MediaType, MediaTypeMovie)
	}
	if entry.MediaID != 1 {
		t.Errorf("Create() MediaID = %d, want 1", entry.MediaID)
	}
}

func TestHistoryService_LogStatusChanged(t *testing.T) {
	// Spec: "status_changed: Any media status transition not covered by existing events"
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, &tdb.Logger)
	ctx := context.Background()

	err := service.LogStatusChanged(ctx, MediaTypeMovie, 42, StatusChangedData{
		From:   "available",
		To:     "missing",
		Reason: "File disappeared from disk",
	})
	if err != nil {
		t.Fatalf("LogStatusChanged() error = %v", err)
	}

	// Verify it was stored
	resp, err := service.List(ctx, &ListOptions{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("List() returned %d items, want 1", len(resp.Items))
	}

	entry := resp.Items[0]
	if entry.EventType != EventTypeStatusChanged {
		t.Errorf("EventType = %q, want %q", entry.EventType, EventTypeStatusChanged)
	}
	if entry.MediaType != MediaTypeMovie {
		t.Errorf("MediaType = %q, want %q", entry.MediaType, MediaTypeMovie)
	}
	if entry.MediaID != 42 {
		t.Errorf("MediaID = %d, want 42", entry.MediaID)
	}
	if entry.Data == nil {
		t.Fatal("Data should not be nil")
	}
	if entry.Data["from"] != "available" {
		t.Errorf("Data.from = %v, want %q", entry.Data["from"], "available")
	}
	if entry.Data["to"] != "missing" {
		t.Errorf("Data.to = %v, want %q", entry.Data["to"], "missing")
	}
	if entry.Data["reason"] != "File disappeared from disk" {
		t.Errorf("Data.reason = %v, want %q", entry.Data["reason"], "File disappeared from disk")
	}
}

func TestHistoryService_LogStatusChanged_Episode(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, &tdb.Logger)
	ctx := context.Background()

	err := service.LogStatusChanged(ctx, MediaTypeEpisode, 100, StatusChangedData{
		From:   "failed",
		To:     "missing",
		Reason: "Manual retry",
	})
	if err != nil {
		t.Fatalf("LogStatusChanged() error = %v", err)
	}

	entries, err := service.ListByMedia(ctx, MediaTypeEpisode, 100)
	if err != nil {
		t.Fatalf("ListByMedia() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListByMedia() returned %d entries, want 1", len(entries))
	}
	if entries[0].EventType != EventTypeStatusChanged {
		t.Errorf("EventType = %q, want %q", entries[0].EventType, EventTypeStatusChanged)
	}
}

func TestHistoryService_ListFiltered(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, &tdb.Logger)
	ctx := context.Background()

	// Create entries of different types
	_, _ = service.Create(ctx, &CreateInput{
		EventType: EventTypeGrabbed,
		MediaType: MediaTypeMovie,
		MediaID:   1,
	})
	_, _ = service.Create(ctx, &CreateInput{
		EventType: EventTypeStatusChanged,
		MediaType: MediaTypeMovie,
		MediaID:   2,
	})
	_, _ = service.Create(ctx, &CreateInput{
		EventType: EventTypeStatusChanged,
		MediaType: MediaTypeEpisode,
		MediaID:   3,
	})

	// Filter by event type
	resp, err := service.List(ctx, &ListOptions{
		EventType: string(EventTypeStatusChanged),
		Page:      1,
		PageSize:  50,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(resp.Items) != 2 {
		t.Errorf("List(event=status_changed) returned %d items, want 2", len(resp.Items))
	}

	// Filter by media type
	resp2, err := service.List(ctx, &ListOptions{
		MediaType: string(MediaTypeEpisode),
		Page:      1,
		PageSize:  50,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(resp2.Items) != 1 {
		t.Errorf("List(media=episode) returned %d items, want 1", len(resp2.Items))
	}
}

func TestHistoryService_EventTypes(t *testing.T) {
	// Verify all spec-required event types exist
	expectedTypes := []EventType{
		EventTypeGrabbed,
		EventTypeImported,
		EventTypeDeleted,
		EventTypeFailed,
		EventTypeRenamed,
		EventTypeAutoSearchDownload,
		EventTypeAutoSearchFailed,
		EventTypeImportFailed,
		EventTypeSlotAssigned,
		EventTypeSlotReassigned,
		EventTypeSlotUnassigned,
		EventTypeStatusChanged,
	}

	for _, et := range expectedTypes {
		if string(et) == "" {
			t.Errorf("EventType %v should not be empty", et)
		}
	}
}

func TestHistoryService_StatusChangedData(t *testing.T) {
	// Verify StatusChangedData serialization
	data := StatusChangedData{
		From:   "missing",
		To:     "downloading",
		Reason: "Auto-search grab",
	}

	m, err := ToJSON(data)
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	if m["from"] != "missing" {
		t.Errorf("from = %v, want %q", m["from"], "missing")
	}
	if m["to"] != "downloading" {
		t.Errorf("to = %v, want %q", m["to"], "downloading")
	}
	if m["reason"] != "Auto-search grab" {
		t.Errorf("reason = %v, want %q", m["reason"], "Auto-search grab")
	}
}

func TestHistoryService_LogAutoSearchDownload(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, &tdb.Logger)
	ctx := context.Background()

	err := service.LogAutoSearchDownload(ctx, MediaTypeMovie, 1, "Bluray-1080p", &AutoSearchDownloadData{
		ReleaseName: "Test.Movie.2024.1080p.BluRay",
		Indexer:     "test-indexer",
		ClientName:  "test-client",
		DownloadID:  "dl-123",
		Source:      "scheduled",
	})
	if err != nil {
		t.Fatalf("LogAutoSearchDownload() error = %v", err)
	}

	entries, _ := service.ListByMedia(ctx, MediaTypeMovie, 1)
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
	if entries[0].EventType != EventTypeAutoSearchDownload {
		t.Errorf("EventType = %q, want %q", entries[0].EventType, EventTypeAutoSearchDownload)
	}
	if entries[0].Quality != "Bluray-1080p" {
		t.Errorf("Quality = %q, want %q", entries[0].Quality, "Bluray-1080p")
	}
}

func TestHistoryService_DeleteAll(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, &tdb.Logger)
	ctx := context.Background()

	_, _ = service.Create(ctx, &CreateInput{
		EventType: EventTypeGrabbed,
		MediaType: MediaTypeMovie,
		MediaID:   1,
	})
	_, _ = service.Create(ctx, &CreateInput{
		EventType: EventTypeStatusChanged,
		MediaType: MediaTypeEpisode,
		MediaID:   2,
	})

	err := service.DeleteAll(ctx)
	if err != nil {
		t.Fatalf("DeleteAll() error = %v", err)
	}

	resp, _ := service.List(ctx, &ListOptions{Page: 1, PageSize: 50})
	if len(resp.Items) != 0 {
		t.Errorf("After DeleteAll: %d items remain", len(resp.Items))
	}
}

func TestHistoryService_Pagination(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, &tdb.Logger)
	ctx := context.Background()

	// Create 5 entries
	for i := 1; i <= 5; i++ {
		_, _ = service.Create(ctx, &CreateInput{
			EventType: EventTypeGrabbed,
			MediaType: MediaTypeMovie,
			MediaID:   int64(i),
		})
	}

	resp, err := service.List(ctx, &ListOptions{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(resp.Items) != 2 {
		t.Errorf("Page 1 items = %d, want 2", len(resp.Items))
	}
	if resp.TotalCount != 5 {
		t.Errorf("TotalCount = %d, want 5", resp.TotalCount)
	}
	if resp.TotalPages != 3 {
		t.Errorf("TotalPages = %d, want 3", resp.TotalPages)
	}
}
