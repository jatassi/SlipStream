package movie

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/module"
)

// GetItemsInDateRange returns movie release events within the date range.
// Produces separate events for digital and physical releases (same as existing
// calendar.Service.getMovieEvents logic).
func (m *Module) GetItemsInDateRange(ctx context.Context, start, end time.Time) ([]module.CalendarItem, error) {
	rows, err := m.queries.GetMoviesInDateRange(ctx, sqlc.GetMoviesInDateRangeParams{
		FromReleaseDate:           sql.NullTime{Time: start, Valid: true},
		ToReleaseDate:             sql.NullTime{Time: end, Valid: true},
		FromPhysicalReleaseDate:   sql.NullTime{Time: start, Valid: true},
		ToPhysicalReleaseDate:     sql.NullTime{Time: end, Valid: true},
		FromTheatricalReleaseDate: sql.NullTime{Time: start, Valid: true},
		ToTheatricalReleaseDate:   sql.NullTime{Time: end, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	var items []module.CalendarItem
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	for _, row := range rows {
		status := m.resolveMovieStatus(ctx, row)

		if event, ok := m.movieReleaseItem(row, "digital", row.ReleaseDate, status, startStr, endStr); ok {
			items = append(items, event)
		}
		if event, ok := m.movieReleaseItem(row, "physical", row.PhysicalReleaseDate, status, startStr, endStr); ok {
			items = append(items, event)
		}
	}

	return items, nil
}

func (m *Module) resolveMovieStatus(ctx context.Context, row *sqlc.Movie) string {
	count, err := m.queries.CountMovieFiles(ctx, row.ID)
	if err == nil && count > 0 {
		return "available"
	}
	return row.Status
}

func (m *Module) movieReleaseItem(row *sqlc.Movie, eventType string, date sql.NullTime, status, startStr, endStr string) (module.CalendarItem, bool) {
	if !date.Valid {
		return module.CalendarItem{}, false
	}
	dateStr := date.Time.Format("2006-01-02")
	if dateStr < startStr || dateStr > endStr {
		return module.CalendarItem{}, false
	}

	externalIDs := make(map[string]string)
	if row.TmdbID.Valid {
		externalIDs["tmdb"] = fmt.Sprintf("%d", row.TmdbID.Int64)
	}
	if row.ImdbID.Valid {
		externalIDs["imdb"] = row.ImdbID.String
	}

	year := 0
	if row.Year.Valid {
		year = int(row.Year.Int64)
	}

	return module.CalendarItem{
		ID:          row.ID,
		Title:       row.Title,
		ModuleType:  module.TypeMovie,
		EntityType:  module.EntityMovie,
		EventType:   eventType,
		Date:        date.Time,
		Status:      status,
		Monitored:   row.Monitored,
		ExternalIDs: externalIDs,
		Year:        year,
	}, true
}
