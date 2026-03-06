package movie

import (
	"context"
	"time"
)

func (m *Module) ComputeAvailabilityDate(ctx context.Context, entityID int64) (*time.Time, error) {
	movie, err := m.movieService.Get(ctx, entityID)
	if err != nil {
		return nil, err
	}

	var earliest *time.Time

	if movie.ReleaseDate != nil {
		earliest = movie.ReleaseDate
	}
	if movie.PhysicalReleaseDate != nil {
		if earliest == nil || movie.PhysicalReleaseDate.Before(*earliest) {
			earliest = movie.PhysicalReleaseDate
		}
	}

	return earliest, nil
}

func (m *Module) CheckReleaseDateTransitions(ctx context.Context) (int, error) {
	result, err := m.queries.UpdateUnreleasedMoviesToMissing(ctx)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}
