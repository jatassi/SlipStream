package tv

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/module"
)

// streamingServicesWithEarlyRelease lists networks that release content the night before.
var streamingServicesWithEarlyRelease = map[string]bool{
	"Apple TV+": true,
	"Apple TV":  true,
}

// GetItemsInDateRange returns episode air date events within the date range.
// Groups 3+ same-season same-day episodes into a single season release event.
func (m *Module) GetItemsInDateRange(ctx context.Context, start, end time.Time) ([]module.CalendarItem, error) {
	rows, err := m.queries.GetEpisodesInDateRange(ctx, sqlc.GetEpisodesInDateRangeParams{
		AirDate:   sql.NullTime{Time: start, Valid: true},
		AirDate_2: sql.NullTime{Time: end, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	type seasonKey struct {
		SeriesID     int64
		SeasonNumber int64
		Date         string
	}
	grouped := make(map[seasonKey][]*sqlc.GetEpisodesInDateRangeRow)
	for _, row := range rows {
		key := seasonKey{
			SeriesID:     row.SeriesID,
			SeasonNumber: row.SeasonNumber,
			Date:         row.AirDate.Time.Format("2006-01-02"),
		}
		grouped[key] = append(grouped[key], row)
	}

	var items []module.CalendarItem
	for key, episodes := range grouped {
		if len(episodes) >= 3 {
			items = append(items, m.seasonReleaseItem(ctx, key.SeriesID, key.SeasonNumber, key.Date, episodes))
		} else {
			for _, row := range episodes {
				items = append(items, m.episodeItem(ctx, row))
			}
		}
	}

	return items, nil
}

func (m *Module) seasonReleaseItem(ctx context.Context, seriesID, seasonNumber int64, date string, episodes []*sqlc.GetEpisodesInDateRangeRow) module.CalendarItem {
	first := episodes[0]

	availableCount := 0
	for _, ep := range episodes {
		files, err := m.queries.ListEpisodeFilesByEpisode(ctx, ep.ID)
		if err == nil && len(files) > 0 {
			availableCount++
		}
	}

	status := "missing"
	if availableCount == len(episodes) {
		status = "available"
	} else if availableCount > 0 {
		status = "downloading"
	}

	network := ""
	if first.Network.Valid {
		network = first.Network.String
	}

	externalIDs := make(map[string]string)
	if first.SeriesTmdbID.Valid {
		externalIDs["tmdb"] = fmt.Sprintf("%d", first.SeriesTmdbID.Int64)
	}

	dateTime, _ := time.Parse("2006-01-02", date)

	return module.CalendarItem{
		ID:          first.ID,
		Title:       fmt.Sprintf("Season %d", seasonNumber),
		ModuleType:  module.TypeTV,
		EntityType:  module.EntityEpisode,
		EventType:   "airDate",
		Date:        dateTime,
		Status:      status,
		Monitored:   first.Monitored,
		ExternalIDs: externalIDs,
		ParentID:    seriesID,
		ParentTitle: first.SeriesTitle,
		Extra: map[string]any{
			"seasonNumber":  int(seasonNumber),
			"episodeNumber": len(episodes),
			"network":       network,
			"earlyAccess":   streamingServicesWithEarlyRelease[network],
		},
	}
}

func (m *Module) episodeItem(ctx context.Context, row *sqlc.GetEpisodesInDateRangeRow) module.CalendarItem {
	hasFile := false
	files, err := m.queries.ListEpisodeFilesByEpisode(ctx, row.ID)
	if err == nil && len(files) > 0 {
		hasFile = true
	}

	status := "missing"
	if hasFile {
		status = "available"
	}

	network := ""
	if row.Network.Valid {
		network = row.Network.String
	}

	title := row.Title.String
	if title == "" {
		title = fmt.Sprintf("Episode %d", row.EpisodeNumber)
	}

	externalIDs := make(map[string]string)
	if row.SeriesTmdbID.Valid {
		externalIDs["tmdb"] = fmt.Sprintf("%d", row.SeriesTmdbID.Int64)
	}

	return module.CalendarItem{
		ID:          row.ID,
		Title:       title,
		ModuleType:  module.TypeTV,
		EntityType:  module.EntityEpisode,
		EventType:   "airDate",
		Date:        row.AirDate.Time,
		Status:      status,
		Monitored:   row.Monitored,
		ExternalIDs: externalIDs,
		ParentID:    row.SeriesID,
		ParentTitle: row.SeriesTitle,
		Extra: map[string]any{
			"seasonNumber":  int(row.SeasonNumber),
			"episodeNumber": int(row.EpisodeNumber),
			"network":       network,
			"earlyAccess":   streamingServicesWithEarlyRelease[network],
		},
	}
}
