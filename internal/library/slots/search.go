package slots

import (
	"context"

	"github.com/slipstream/slipstream/internal/indexer/types"
)

// SlotSearchInfo represents information about a slot needed for search operations.
// Req 7.1.3: Each slot's search is independent
type SlotSearchInfo struct {
	SlotID           int64   `json:"slotId"`
	SlotNumber       int     `json:"slotNumber"`
	SlotName         string  `json:"slotName"`
	QualityProfileID int64   `json:"qualityProfileId"`
	ProfileCutoff    int     `json:"profileCutoff"`
	HasFile          bool    `json:"hasFile"`
	FileID           *int64  `json:"fileId,omitempty"`
	CurrentQualityID *int64  `json:"currentQualityId,omitempty"`
	CurrentQuality   string  `json:"currentQuality,omitempty"`
	NeedsUpgrade     bool    `json:"needsUpgrade"`
	IsMissing        bool    `json:"isMissing"`
	Monitored        bool    `json:"monitored"`
}

// GetMovieSlotsNeedingSearch returns slots that need searching for a movie.
// Req 7.1.1: Auto-search with multiple empty monitored slots: search in parallel
// Req 8.1.3: Auto-search only runs for monitored slots
func (s *Service) GetMovieSlotsNeedingSearch(ctx context.Context, movieID int64) ([]SlotSearchInfo, error) {
	status, err := s.GetMovieStatus(ctx, movieID)
	if err != nil {
		return nil, err
	}

	result := make([]SlotSearchInfo, 0)
	for _, slotStatus := range status.SlotStatuses {
		// Req 8.1.3: Only search monitored slots
		if !slotStatus.Monitored {
			continue
		}

		// Include if missing (empty) or needs upgrade
		if slotStatus.IsMissing || slotStatus.NeedsUpgrade {
			slot, err := s.Get(ctx, slotStatus.SlotID)
			if err != nil || slot.QualityProfileID == nil {
				continue
			}

			info := SlotSearchInfo{
				SlotID:           slotStatus.SlotID,
				SlotNumber:       slotStatus.SlotNumber,
				SlotName:         slotStatus.SlotName,
				QualityProfileID: *slot.QualityProfileID,
				ProfileCutoff:    slotStatus.ProfileCutoff,
				HasFile:          slotStatus.HasFile,
				FileID:           slotStatus.FileID,
				CurrentQualityID: slotStatus.CurrentQualityID,
				CurrentQuality:   slotStatus.CurrentQuality,
				NeedsUpgrade:     slotStatus.NeedsUpgrade,
				IsMissing:        slotStatus.IsMissing,
				Monitored:        slotStatus.Monitored,
			}
			result = append(result, info)
		}
	}

	return result, nil
}

// GetEpisodeSlotsNeedingSearch returns slots that need searching for an episode.
func (s *Service) GetEpisodeSlotsNeedingSearch(ctx context.Context, episodeID int64) ([]SlotSearchInfo, error) {
	status, err := s.GetEpisodeStatus(ctx, episodeID)
	if err != nil {
		return nil, err
	}

	result := make([]SlotSearchInfo, 0)
	for _, slotStatus := range status.SlotStatuses {
		if !slotStatus.Monitored {
			continue
		}

		if slotStatus.IsMissing || slotStatus.NeedsUpgrade {
			slot, err := s.Get(ctx, slotStatus.SlotID)
			if err != nil || slot.QualityProfileID == nil {
				continue
			}

			info := SlotSearchInfo{
				SlotID:           slotStatus.SlotID,
				SlotNumber:       slotStatus.SlotNumber,
				SlotName:         slotStatus.SlotName,
				QualityProfileID: *slot.QualityProfileID,
				ProfileCutoff:    slotStatus.ProfileCutoff,
				HasFile:          slotStatus.HasFile,
				FileID:           slotStatus.FileID,
				CurrentQualityID: slotStatus.CurrentQualityID,
				CurrentQuality:   slotStatus.CurrentQuality,
				NeedsUpgrade:     slotStatus.NeedsUpgrade,
				IsMissing:        slotStatus.IsMissing,
				Monitored:        slotStatus.Monitored,
			}
			result = append(result, info)
		}
	}

	return result, nil
}

// GetMovieSlotSearchInfo returns search info for a specific slot for a movie.
// Used when explicitly searching for a single slot (not auto-search).
func (s *Service) GetMovieSlotSearchInfo(ctx context.Context, movieID int64, slotID int64) (*SlotSearchInfo, error) {
	status, err := s.GetMovieStatus(ctx, movieID)
	if err != nil {
		return nil, err
	}

	for _, slotStatus := range status.SlotStatuses {
		if slotStatus.SlotID != slotID {
			continue
		}

		slot, err := s.Get(ctx, slotID)
		if err != nil {
			return nil, err
		}
		if slot.QualityProfileID == nil {
			return nil, ErrSlotNoQualityProfile
		}

		return &SlotSearchInfo{
			SlotID:           slotStatus.SlotID,
			SlotNumber:       slotStatus.SlotNumber,
			SlotName:         slotStatus.SlotName,
			QualityProfileID: *slot.QualityProfileID,
			ProfileCutoff:    slotStatus.ProfileCutoff,
			HasFile:          slotStatus.HasFile,
			FileID:           slotStatus.FileID,
			CurrentQualityID: slotStatus.CurrentQualityID,
			CurrentQuality:   slotStatus.CurrentQuality,
			NeedsUpgrade:     slotStatus.NeedsUpgrade,
			IsMissing:        slotStatus.IsMissing,
			Monitored:        slotStatus.Monitored,
		}, nil
	}

	return nil, ErrSlotNotFound
}

// GetEpisodeSlotSearchInfo returns search info for a specific slot for an episode.
// Used when explicitly searching for a single slot (not auto-search).
func (s *Service) GetEpisodeSlotSearchInfo(ctx context.Context, episodeID int64, slotID int64) (*SlotSearchInfo, error) {
	status, err := s.GetEpisodeStatus(ctx, episodeID)
	if err != nil {
		return nil, err
	}

	for _, slotStatus := range status.SlotStatuses {
		if slotStatus.SlotID != slotID {
			continue
		}

		slot, err := s.Get(ctx, slotID)
		if err != nil {
			return nil, err
		}
		if slot.QualityProfileID == nil {
			return nil, ErrSlotNoQualityProfile
		}

		return &SlotSearchInfo{
			SlotID:           slotStatus.SlotID,
			SlotNumber:       slotStatus.SlotNumber,
			SlotName:         slotStatus.SlotName,
			QualityProfileID: *slot.QualityProfileID,
			ProfileCutoff:    slotStatus.ProfileCutoff,
			HasFile:          slotStatus.HasFile,
			FileID:           slotStatus.FileID,
			CurrentQualityID: slotStatus.CurrentQualityID,
			CurrentQuality:   slotStatus.CurrentQuality,
			NeedsUpgrade:     slotStatus.NeedsUpgrade,
			IsMissing:        slotStatus.IsMissing,
			Monitored:        slotStatus.Monitored,
		}, nil
	}

	return nil, ErrSlotNotFound
}

// GetTargetSlotForQuality determines which slot a release with a given quality should be assigned to.
// Returns the best matching slot for a release based on quality ID.
func (s *Service) GetTargetSlotForQuality(ctx context.Context, mediaType string, mediaID int64, qualityID int) (*SlotSearchInfo, bool, error) {
	var slots []SlotSearchInfo
	var err error

	switch mediaType {
	case "movie":
		slots, err = s.getMovieSlotCandidates(ctx, mediaID)
	case "episode":
		slots, err = s.getEpisodeSlotCandidates(ctx, mediaID)
	default:
		return nil, false, nil
	}

	if err != nil {
		return nil, false, err
	}

	if len(slots) == 0 {
		return nil, false, nil
	}

	// Find the best matching slot
	// Priority: first empty slot, then upgrade slots
	var bestSlot *SlotSearchInfo
	var isUpgrade bool

	// First, prefer empty slots
	for i := range slots {
		if slots[i].IsMissing {
			bestSlot = &slots[i]
			isUpgrade = false
			break
		}
	}

	// If no empty slot, check for upgrade opportunity
	if bestSlot == nil {
		for i := range slots {
			if slots[i].NeedsUpgrade && slots[i].CurrentQualityID != nil {
				// Check if this quality is an upgrade
				if qualityID > int(*slots[i].CurrentQualityID) {
					bestSlot = &slots[i]
					isUpgrade = true
					break
				}
			}
		}
	}

	return bestSlot, isUpgrade, nil
}

// getMovieSlotCandidates returns all enabled slots with their current state for a movie.
func (s *Service) getMovieSlotCandidates(ctx context.Context, movieID int64) ([]SlotSearchInfo, error) {
	status, err := s.GetMovieStatus(ctx, movieID)
	if err != nil {
		return nil, err
	}

	result := make([]SlotSearchInfo, 0, len(status.SlotStatuses))
	for _, slotStatus := range status.SlotStatuses {
		slot, err := s.Get(ctx, slotStatus.SlotID)
		if err != nil || slot.QualityProfileID == nil {
			continue
		}

		info := SlotSearchInfo{
			SlotID:           slotStatus.SlotID,
			SlotNumber:       slotStatus.SlotNumber,
			SlotName:         slotStatus.SlotName,
			QualityProfileID: *slot.QualityProfileID,
			ProfileCutoff:    slotStatus.ProfileCutoff,
			HasFile:          slotStatus.HasFile,
			FileID:           slotStatus.FileID,
			CurrentQualityID: slotStatus.CurrentQualityID,
			CurrentQuality:   slotStatus.CurrentQuality,
			NeedsUpgrade:     slotStatus.NeedsUpgrade,
			IsMissing:        slotStatus.IsMissing,
			Monitored:        slotStatus.Monitored,
		}
		result = append(result, info)
	}

	return result, nil
}

// getEpisodeSlotCandidates returns all enabled slots with their current state for an episode.
func (s *Service) getEpisodeSlotCandidates(ctx context.Context, episodeID int64) ([]SlotSearchInfo, error) {
	status, err := s.GetEpisodeStatus(ctx, episodeID)
	if err != nil {
		return nil, err
	}

	result := make([]SlotSearchInfo, 0, len(status.SlotStatuses))
	for _, slotStatus := range status.SlotStatuses {
		slot, err := s.Get(ctx, slotStatus.SlotID)
		if err != nil || slot.QualityProfileID == nil {
			continue
		}

		info := SlotSearchInfo{
			SlotID:           slotStatus.SlotID,
			SlotNumber:       slotStatus.SlotNumber,
			SlotName:         slotStatus.SlotName,
			QualityProfileID: *slot.QualityProfileID,
			ProfileCutoff:    slotStatus.ProfileCutoff,
			HasFile:          slotStatus.HasFile,
			FileID:           slotStatus.FileID,
			CurrentQualityID: slotStatus.CurrentQualityID,
			CurrentQuality:   slotStatus.CurrentQuality,
			NeedsUpgrade:     slotStatus.NeedsUpgrade,
			IsMissing:        slotStatus.IsMissing,
			Monitored:        slotStatus.Monitored,
		}
		result = append(result, info)
	}

	return result, nil
}

// EnrichReleasesWithSlotInfo enriches search releases with target slot information.
// Req 11.1.1: Search results indicate which slot each release would fill
// Req 11.1.2: Show whether grab would be upgrade vs new fill
func (s *Service) EnrichReleasesWithSlotInfo(ctx context.Context, releases []types.TorrentInfo, mediaType string, mediaID int64) []types.TorrentInfo {
	if !s.IsMultiVersionEnabled(ctx) {
		return releases
	}

	// Get all slot candidates for this media
	var candidates []SlotSearchInfo
	var err error

	switch mediaType {
	case "movie":
		candidates, err = s.getMovieSlotCandidates(ctx, mediaID)
	case "episode":
		candidates, err = s.getEpisodeSlotCandidates(ctx, mediaID)
	default:
		return releases
	}

	if err != nil || len(candidates) == 0 {
		return releases
	}

	// Enrich each release with slot info
	enriched := make([]types.TorrentInfo, len(releases))
	copy(enriched, releases)

	for i := range enriched {
		release := &enriched[i]

		// Get quality ID from score breakdown
		var qualityID int
		if release.ScoreBreakdown != nil {
			qualityID = release.ScoreBreakdown.QualityID
		}

		if qualityID == 0 {
			continue
		}

		// Find target slot
		targetSlot, isUpgrade, err := s.GetTargetSlotForQuality(ctx, mediaType, mediaID, qualityID)
		if err != nil || targetSlot == nil {
			continue
		}

		// Enrich the release
		release.TargetSlotID = &targetSlot.SlotID
		release.TargetSlotNumber = &targetSlot.SlotNumber
		release.TargetSlotName = targetSlot.SlotName
		release.IsSlotUpgrade = isUpgrade
		release.IsSlotNewFill = !isUpgrade && targetSlot.IsMissing
	}

	return enriched
}
