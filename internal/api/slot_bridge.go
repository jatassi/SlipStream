package api

import (
	"context"

	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/tv"
)

// slotFileDeleterAdapter adapts movie and TV services to slots.FileDeleter interface.
// Req 12.2.2: Delete files when disabling a slot with delete action.
type slotFileDeleterAdapter struct {
	movieSvc *movies.Service
	tvSvc    *tv.Service
}

// DeleteFile implements slots.FileDeleter.
func (a *slotFileDeleterAdapter) DeleteFile(ctx context.Context, mediaType string, fileID int64) error {
	switch mediaType {
	case mediaTypeMovie:
		return a.movieSvc.RemoveFile(ctx, fileID)
	case mediaTypeEpisode:
		return a.tvSvc.RemoveEpisodeFile(ctx, fileID)
	default:
		return nil
	}
}
