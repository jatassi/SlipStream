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

	// Compute earliest of digital and physical release dates (spec §7.1)
	switch {
	case movie.ReleaseDate != nil && movie.PhysicalReleaseDate != nil:
		if movie.ReleaseDate.Before(*movie.PhysicalReleaseDate) {
			earliest = movie.ReleaseDate
		} else {
			earliest = movie.PhysicalReleaseDate
		}
	case movie.ReleaseDate != nil:
		earliest = movie.ReleaseDate
	case movie.PhysicalReleaseDate != nil:
		earliest = movie.PhysicalReleaseDate
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
