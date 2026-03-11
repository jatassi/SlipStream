package tv

import (
	"context"
	"time"
)

func (m *Module) ComputeAvailabilityDate(ctx context.Context, entityID int64) (*time.Time, error) {
	episode, err := m.tvService.GetEpisode(ctx, entityID)
	if err != nil {
		return nil, err
	}
	return episode.AirDate, nil
}

func (m *Module) CheckReleaseDateTransitions(ctx context.Context) (int, error) {
	result1, err := m.queries.UpdateUnreleasedEpisodesToMissing(ctx)
	if err != nil {
		return 0, err
	}
	rows1, _ := result1.RowsAffected()

	result2, err := m.queries.UpdateUnreleasedEpisodesToMissingDateOnly(ctx)
	if err != nil {
		return int(rows1), err
	}
	rows2, _ := result2.RowsAffected()

	return int(rows1 + rows2), nil
}
