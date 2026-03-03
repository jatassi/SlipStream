package api

import (
	"context"

	"github.com/slipstream/slipstream/internal/domain/contracts"
	"github.com/slipstream/slipstream/internal/history"
	importer "github.com/slipstream/slipstream/internal/import"
)

// Compile-time assertion.
var _ contracts.StatusChangeLogger = (*statusChangeLoggerAdapter)(nil)

type importHistoryAdapter struct {
	svc *history.Service
}

// Create implements importer.HistoryService.
func (a *importHistoryAdapter) Create(ctx context.Context, input *importer.HistoryInput) error {
	_, err := a.svc.Create(ctx, &history.CreateInput{
		EventType: history.EventType(input.EventType),
		MediaType: history.MediaType(input.MediaType),
		MediaID:   input.MediaID,
		Source:    input.Source,
		Quality:   input.Quality,
		Data:      input.Data,
	})
	return err
}

// statusChangeLoggerAdapter adapts history.Service for status transition logging.
type statusChangeLoggerAdapter struct {
	svc *history.Service
}

func (a *statusChangeLoggerAdapter) LogStatusChanged(ctx context.Context, mediaType string, mediaID int64, from, to, reason string) error {
	return a.svc.LogStatusChanged(ctx, history.MediaType(mediaType), mediaID, history.StatusChangedData{
		From: from, To: to, Reason: reason,
	})
}
