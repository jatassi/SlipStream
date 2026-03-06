package tv

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/rs/zerolog"

	tvlib "github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/module"
)

var ErrNoMetadataProvider = errors.New("no metadata provider configured")

type metadataProvider struct {
	metadataSvc *metadata.Service
	tvSvc       *tvlib.Service
	logger      zerolog.Logger
}

func newMetadataProvider(metadataSvc *metadata.Service, tvSvc *tvlib.Service, logger *zerolog.Logger) *metadataProvider {
	return &metadataProvider{
		metadataSvc: metadataSvc,
		tvSvc:       tvSvc,
		logger:      logger.With().Str("component", "tv-metadata").Logger(),
	}
}

func (p *metadataProvider) Search(ctx context.Context, query string, opts module.SearchOptions) ([]module.SearchResult, error) {
	results, err := p.metadataSvc.SearchSeries(ctx, query)
	if err != nil {
		return nil, err
	}

	out := make([]module.SearchResult, len(results))
	for i := range results {
		r := &results[i]
		externalIDs := make(map[string]string)
		if r.TvdbID > 0 {
			externalIDs["tvdb"] = strconv.Itoa(r.TvdbID)
		}
		if r.TmdbID > 0 {
			externalIDs["tmdb"] = strconv.Itoa(r.TmdbID)
		}
		if r.ImdbID != "" {
			externalIDs["imdb"] = r.ImdbID
		}

		primaryID := strconv.Itoa(r.TvdbID)
		if r.TvdbID == 0 {
			primaryID = strconv.Itoa(r.TmdbID)
		}

		out[i] = module.SearchResult{
			ExternalID:  primaryID,
			Title:       r.Title,
			Year:        r.Year,
			Overview:    r.Overview,
			PosterURL:   r.PosterURL,
			BackdropURL: r.BackdropURL,
			ExternalIDs: externalIDs,
			Extra:       &r,
		}
	}
	return out, nil
}

func (p *metadataProvider) GetByID(ctx context.Context, externalID string) (*module.MediaMetadata, error) {
	id, err := strconv.Atoi(externalID)
	if err != nil {
		return nil, fmt.Errorf("invalid external ID: %w", err)
	}

	result, err := p.metadataSvc.GetSeries(ctx, id, id)
	if err != nil {
		return nil, err
	}

	externalIDs := make(map[string]string)
	if result.TvdbID > 0 {
		externalIDs["tvdb"] = strconv.Itoa(result.TvdbID)
	}
	if result.TmdbID > 0 {
		externalIDs["tmdb"] = strconv.Itoa(result.TmdbID)
	}
	if result.ImdbID != "" {
		externalIDs["imdb"] = result.ImdbID
	}

	return &module.MediaMetadata{
		ExternalID:  externalID,
		Title:       result.Title,
		Year:        result.Year,
		Overview:    result.Overview,
		PosterURL:   result.PosterURL,
		BackdropURL: result.BackdropURL,
		ExternalIDs: externalIDs,
		Extra:       result,
	}, nil
}

func (p *metadataProvider) GetExtendedInfo(ctx context.Context, externalID string) (*module.ExtendedMetadata, error) {
	tmdbID, err := strconv.Atoi(externalID)
	if err != nil {
		return nil, fmt.Errorf("invalid external ID: %w", err)
	}

	result, err := p.metadataSvc.GetExtendedSeries(ctx, tmdbID)
	if err != nil {
		return nil, err
	}

	return &module.ExtendedMetadata{
		Credits:       result.Credits,
		Ratings:       result.Ratings,
		ContentRating: result.ContentRating,
		TrailerURL:    result.TrailerURL,
		Extra:         result,
	}, nil
}

func (p *metadataProvider) RefreshMetadata(ctx context.Context, entityID int64) (*module.RefreshResult, error) {
	series, err := p.tvSvc.GetSeries(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get series: %w", err)
	}

	if !p.metadataSvc.HasSeriesProvider() {
		return nil, ErrNoMetadataProvider
	}

	results, err := p.metadataSvc.SearchSeries(ctx, series.Title)
	if err != nil {
		return nil, fmt.Errorf("metadata search failed: %w", err)
	}

	if len(results) == 0 {
		p.logger.Warn().Str("title", series.Title).Msg("No metadata results found")
		return &module.RefreshResult{EntityID: entityID}, nil
	}

	bestMatch := p.enrichSeriesDetails(ctx, &results[0])
	fieldsChanged := p.computeSeriesFieldsChanged(series, bestMatch)

	if err := p.updateSeriesFromMetadata(ctx, series.ID, bestMatch); err != nil {
		return nil, err
	}

	childDiff := p.refreshSeasons(ctx, series, bestMatch.TmdbID, bestMatch.TvdbID)

	return &module.RefreshResult{
		EntityID:        entityID,
		Updated:         len(fieldsChanged) > 0,
		FieldsChanged:   fieldsChanged,
		ChildrenAdded:   childDiff.added,
		ChildrenUpdated: childDiff.updated,
		ChildrenRemoved: childDiff.removed,
		ArtworkURLs: module.ArtworkURLs{
			PosterURL:   bestMatch.PosterURL,
			BackdropURL: bestMatch.BackdropURL,
			LogoURL:     bestMatch.LogoURL,
		},
		Metadata: bestMatch,
	}, nil
}

type seasonEpisodeDiff struct {
	added   []module.RefreshChildEntry
	updated []module.RefreshChildEntry
	removed []module.RefreshChildEntry
}

func (p *metadataProvider) refreshSeasons(ctx context.Context, series *tvlib.Series, tmdbID, tvdbID int) seasonEpisodeDiff {
	if tmdbID == 0 && tvdbID == 0 {
		return seasonEpisodeDiff{}
	}

	seasonResults, err := p.metadataSvc.GetSeriesSeasons(ctx, tmdbID, tvdbID)
	if err != nil {
		p.logger.Warn().Err(err).Int("tmdbId", tmdbID).Int("tvdbId", tvdbID).Msg("Failed to fetch season metadata")
		return seasonEpisodeDiff{}
	}

	existingEpisodes, err := p.buildExistingEpisodeMap(ctx, series.ID)
	if err != nil {
		p.logger.Warn().Err(err).Int64("seriesId", series.ID).Msg("Failed to list existing episodes for diff")
		return seasonEpisodeDiff{}
	}

	seasonMeta := p.convertToSeasonMetadata(seasonResults)
	if err := p.tvSvc.UpdateSeasonsFromMetadata(ctx, series.ID, seasonMeta); err != nil {
		p.logger.Warn().Err(err).Int64("seriesId", series.ID).Msg("Failed to update seasons from metadata")
		return seasonEpisodeDiff{}
	}

	return p.computeEpisodeDiff(seasonMeta, existingEpisodes)
}

func (p *metadataProvider) computeEpisodeDiff(seasonMeta []tvlib.SeasonMetadata, existingEpisodes map[string]module.RefreshChildEntry) seasonEpisodeDiff {
	seenKeys := make(map[string]bool)
	var diff seasonEpisodeDiff

	for _, sm := range seasonMeta {
		for _, ep := range sm.Episodes {
			key := fmt.Sprintf("S%02dE%02d", ep.SeasonNumber, ep.EpisodeNumber)
			seenKeys[key] = true

			if _, existed := existingEpisodes[key]; existed {
				diff.updated = append(diff.updated, module.RefreshChildEntry{
					EntityType: module.EntityEpisode,
					Identifier: key,
					Title:      ep.Title,
				})
			} else {
				diff.added = append(diff.added, module.RefreshChildEntry{
					EntityType: module.EntityEpisode,
					Identifier: key,
					Title:      ep.Title,
				})
			}
		}
	}

	for key, entry := range existingEpisodes {
		if !seenKeys[key] {
			diff.removed = append(diff.removed, entry)
		}
	}

	return diff
}

func (p *metadataProvider) buildExistingEpisodeMap(ctx context.Context, seriesID int64) (map[string]module.RefreshChildEntry, error) {
	episodes, err := p.tvSvc.ListEpisodes(ctx, seriesID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list episodes: %w", err)
	}

	existing := make(map[string]module.RefreshChildEntry, len(episodes))
	for _, ep := range episodes {
		key := fmt.Sprintf("S%02dE%02d", ep.SeasonNumber, ep.EpisodeNumber)
		existing[key] = module.RefreshChildEntry{
			EntityType: module.EntityEpisode,
			Identifier: key,
			EntityID:   ep.ID,
			Title:      ep.Title,
		}
	}
	return existing, nil
}

func (p *metadataProvider) convertToSeasonMetadata(seasonResults []metadata.SeasonResult) []tvlib.SeasonMetadata {
	seasonMeta := make([]tvlib.SeasonMetadata, len(seasonResults))
	for i, sr := range seasonResults {
		episodes := make([]tvlib.EpisodeMetadata, len(sr.Episodes))
		for j, ep := range sr.Episodes {
			episodes[j] = tvlib.EpisodeMetadata{
				EpisodeNumber: ep.EpisodeNumber,
				SeasonNumber:  ep.SeasonNumber,
				Title:         ep.Title,
				Overview:      ep.Overview,
				AirDate:       ep.AirDate,
				Runtime:       ep.Runtime,
			}
		}
		seasonMeta[i] = tvlib.SeasonMetadata{
			SeasonNumber: sr.SeasonNumber,
			Name:         sr.Name,
			Overview:     sr.Overview,
			PosterURL:    sr.PosterURL,
			AirDate:      sr.AirDate,
			Episodes:     episodes,
		}
	}
	return seasonMeta
}

func (p *metadataProvider) enrichSeriesDetails(ctx context.Context, match *metadata.SeriesResult) *metadata.SeriesResult {
	if match.TmdbID > 0 {
		if detail, err := p.metadataSvc.GetSeriesByTMDB(ctx, match.TmdbID); err == nil {
			match = detail
		}
	}

	if match.TmdbID > 0 {
		if logoURL, err := p.metadataSvc.GetSeriesLogoURL(ctx, match.TmdbID); err == nil && logoURL != "" {
			match.LogoURL = logoURL
		}
	}

	return match
}

func (p *metadataProvider) computeSeriesFieldsChanged(series *tvlib.Series, match *metadata.SeriesResult) []string {
	var changed []string
	if series.Title != match.Title {
		changed = append(changed, "title")
	}
	if series.Year != match.Year {
		changed = append(changed, "year")
	}
	if series.TvdbID != match.TvdbID {
		changed = append(changed, "tvdbId")
	}
	if series.TmdbID != match.TmdbID {
		changed = append(changed, "tmdbId")
	}
	if series.ImdbID != match.ImdbID {
		changed = append(changed, "imdbId")
	}
	if series.Overview != match.Overview {
		changed = append(changed, "overview")
	}
	return changed
}

func (p *metadataProvider) updateSeriesFromMetadata(ctx context.Context, seriesID int64, match *metadata.SeriesResult) error {
	title := match.Title
	year := match.Year
	tvdbID := match.TvdbID
	tmdbID := match.TmdbID
	imdbID := match.ImdbID
	overview := match.Overview
	runtime := match.Runtime
	status := match.Status
	network := match.Network
	networkLogoURL := match.NetworkLogoURL

	_, err := p.tvSvc.UpdateSeries(ctx, seriesID, &tvlib.UpdateSeriesInput{
		Title:            &title,
		Year:             &year,
		TvdbID:           &tvdbID,
		TmdbID:           &tmdbID,
		ImdbID:           &imdbID,
		Overview:         &overview,
		Runtime:          &runtime,
		ProductionStatus: &status,
		Network:          &network,
		NetworkLogoURL:   &networkLogoURL,
	})
	return err
}
