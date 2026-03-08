package module

import (
	"context"

	"github.com/rs/zerolog"
)

// FileRemovalTransitionParams provides the callbacks needed for the framework
// to transition an entity to "missing" after its last file is removed.
type FileRemovalTransitionParams struct {
	ModuleType Type
	EntityType EntityType
	EntityID   int64
	Logger     *zerolog.Logger

	// GetCurrentStatus returns the entity's current status. Returns "" if entity not found.
	GetCurrentStatus func(ctx context.Context, entityID int64) (string, error)
	// SetMissingAndUnmonitor sets the entity's status to "missing" and unmonitors it.
	SetMissingAndUnmonitor func(ctx context.Context, entityID int64) error
	// LogStatusChange records the status transition in history (optional, may be nil).
	LogStatusChange func(ctx context.Context, entityType string, entityID int64, oldStatus, newStatus, reason string) error
	// BroadcastUpdate sends a WebSocket update for the affected entity (optional, may be nil).
	BroadcastUpdate func()
}

// TransitionToMissingAfterFileRemoval handles the common pattern of transitioning
// an entity to "missing" status and unmonitoring it when its last file is removed.
// Call this after confirming the entity has zero remaining files.
func TransitionToMissingAfterFileRemoval(ctx context.Context, p *FileRemovalTransitionParams) {
	oldStatus, _ := p.GetCurrentStatus(ctx, p.EntityID)

	if err := p.SetMissingAndUnmonitor(ctx, p.EntityID); err != nil {
		p.Logger.Error().Err(err).
			Str("moduleType", string(p.ModuleType)).
			Str("entityType", string(p.EntityType)).
			Int64("entityId", p.EntityID).
			Msg("Failed to transition entity to missing after file removal")
		return
	}

	if p.LogStatusChange != nil && oldStatus != "" && oldStatus != StatusMissing {
		_ = p.LogStatusChange(ctx, string(p.EntityType), p.EntityID, oldStatus, StatusMissing, "File removed")
	}

	if p.BroadcastUpdate != nil {
		p.BroadcastUpdate()
	}
}
