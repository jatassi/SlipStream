package importer

import (
	"context"
	"fmt"

	"github.com/slipstream/slipstream/internal/module"
)

// matchToLibraryViaModule uses the module's ImportHandler to match a download to a
// library entity. The result is stored in both the legacy LibraryMatch fields and
// the new ModuleEntity field for downstream module-aware processing.
func (s *Service) matchToLibraryViaModule(ctx context.Context, _ string, mapping *DownloadMapping) (*LibraryMatch, error) {
	cd := mappingToModuleDownload(mapping)

	mod := s.registry.Get(cd.ModuleType)
	if mod == nil {
		return nil, fmt.Errorf("module %s not found in registry", cd.ModuleType)
	}

	importHandler, ok := mod.(module.ImportHandler)
	if !ok {
		return nil, fmt.Errorf("module %s does not implement ImportHandler", cd.ModuleType)
	}

	entities, err := importHandler.MatchDownload(ctx, &cd)
	if err != nil {
		return nil, fmt.Errorf("module match failed: %w", err)
	}
	if len(entities) == 0 {
		return nil, ErrNoMatch
	}

	// Use the first matched entity (for single-file downloads)
	entity := &entities[0]
	match := moduleEntityToLibraryMatch(entity)

	return match, nil
}

// mappingToModuleDownload converts a DownloadMapping to the module-system CompletedDownload.
func mappingToModuleDownload(mapping *DownloadMapping) module.CompletedDownload {
	cd := module.CompletedDownload{
		DownloadID: mapping.DownloadID,
		ClientID:   mapping.DownloadClientID,
		MappingID:  mapping.ID,
		Source:     mapping.Source,
	}

	cd.ModuleType = module.Type(mapping.ModuleType)
	cd.EntityType = module.EntityType(mapping.EntityType)
	cd.EntityID = mapping.EntityID

	if mapping.IsSeasonPack || mapping.IsCompleteSeries {
		cd.IsGroupDownload = true
		if mapping.IsCompleteSeries {
			cd.GroupEntityType = module.EntitySeries
		} else {
			cd.GroupEntityType = module.EntitySeason
		}
		if mapping.SeriesID != nil {
			cd.GroupEntityID = *mapping.SeriesID
		}
	}

	if mapping.SeasonNumber != nil {
		cd.SeasonNumber = mapping.SeasonNumber
	}
	cd.TargetSlotID = mapping.TargetSlotID

	return cd
}

// moduleEntityToLibraryMatch converts a module.MatchedEntity to a LibraryMatch,
// populating both the legacy fields and the ModuleEntity field.
func moduleEntityToLibraryMatch(entity *module.MatchedEntity) *LibraryMatch {
	match := &LibraryMatch{
		Source:             entity.Source,
		Confidence:         entity.Confidence,
		RootFolder:         entity.RootFolder,
		IsUpgrade:          entity.IsUpgrade,
		ExistingFileID:     entity.ExistingFileID,
		ExistingFile:       entity.ExistingFilePath,
		CandidateQualityID: entity.CandidateQualityID,
		ExistingQualityID:  entity.ExistingQualityID,
		QualityProfileID:   entity.QualityProfileID,
		ModuleEntity:       entity,
	}

	// Populate legacy fields based on module/entity type
	switch entity.ModuleType {
	case module.TypeMovie:
		match.MediaType = mediaTypeMovie
		id := entity.EntityID
		match.MovieID = &id
	case module.TypeTV:
		switch entity.EntityType {
		case module.EntityEpisode:
			match.MediaType = mediaTypeEpisode
			id := entity.EntityID
			match.EpisodeID = &id
			if len(entity.EntityIDs) > 1 {
				match.EpisodeIDs = entity.EntityIDs
			}
			// Extract season/series from TokenData
			if seriesID, ok := entity.TokenData["SeriesID"].(int64); ok {
				match.SeriesID = &seriesID
			}
			if seasonNum, ok := entity.TokenData["SeasonNumber"].(int); ok {
				match.SeasonNum = &seasonNum
			}
		case module.EntitySeries:
			match.MediaType = mediaTypeEpisode
			id := entity.EntityID
			match.SeriesID = &id
		}
	}

	return match
}
