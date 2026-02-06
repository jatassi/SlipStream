package slots

import (
	"context"

	"github.com/slipstream/slipstream/internal/indexer/types"
)

// SlotSearchInfo represents information about a slot needed for search operations.
type SlotSearchInfo struct {
	SlotID           int64  `json:"slotId"`
	SlotNumber       int    `json:"slotNumber"`
	SlotName         string `json:"slotName"`
	QualityProfileID int64  `json:"qualityProfileId"`
	ProfileCutoff    int    `json:"profileCutoff"`
	Status           string `json:"status"`
	FileID           *int64 `json:"fileId,omitempty"`
	CurrentQualityID *int64 `json:"currentQualityId,omitempty"`
	CurrentQuality   string `json:"currentQuality,omitempty"`
	Monitored        bool   `json:"monitored"`
}

// HasFile returns true if this slot has a file assigned.
func (s *SlotSearchInfo) HasFile() bool {
	return s.Status == "available" || s.Status == "upgradable"
}

// NeedsUpgrade returns true if this slot's file is below cutoff.
func (s *SlotSearchInfo) NeedsUpgrade() bool {
	return s.Status == "upgradable"
}

// IsMissing returns true if this slot is empty and needs a file.
func (s *SlotSearchInfo) IsMissing() bool {
	return s.Status == "missing"
}

func slotSearchInfoFromStatus(slotStatus SlotStatus, qualityProfileID int64) SlotSearchInfo {
	return SlotSearchInfo{
		SlotID:           slotStatus.SlotID,
		SlotNumber:       slotStatus.SlotNumber,
		SlotName:         slotStatus.SlotName,
		QualityProfileID: qualityProfileID,
		ProfileCutoff:    slotStatus.ProfileCutoff,
		Status:           slotStatus.Status,
		FileID:           slotStatus.FileID,
		CurrentQualityID: slotStatus.CurrentQualityID,
		CurrentQuality:   slotStatus.CurrentQuality,
		Monitored:        slotStatus.Monitored,
	}
}

// GetMovieSlotsNeedingSearch returns slots that need searching for a movie.
func (s *Service) GetMovieSlotsNeedingSearch(ctx context.Context, movieID int64) ([]SlotSearchInfo, error) {
	status, err := s.GetMovieStatus(ctx, movieID)
	if err != nil {
		return nil, err
	}

	result := make([]SlotSearchInfo, 0)
	for _, slotStatus := range status.SlotStatuses {
		if !slotStatus.Monitored {
			continue
		}

		if slotStatus.Status == "missing" || slotStatus.Status == "upgradable" {
			slot, err := s.Get(ctx, slotStatus.SlotID)
			if err != nil || slot.QualityProfileID == nil {
				continue
			}

			result = append(result, slotSearchInfoFromStatus(slotStatus, *slot.QualityProfileID))
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

		if slotStatus.Status == "missing" || slotStatus.Status == "upgradable" {
			slot, err := s.Get(ctx, slotStatus.SlotID)
			if err != nil || slot.QualityProfileID == nil {
				continue
			}

			result = append(result, slotSearchInfoFromStatus(slotStatus, *slot.QualityProfileID))
		}
	}

	return result, nil
}

// GetMovieSlotSearchInfo returns search info for a specific slot for a movie.
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

		info := slotSearchInfoFromStatus(slotStatus, *slot.QualityProfileID)
		return &info, nil
	}

	return nil, ErrSlotNotFound
}

// GetEpisodeSlotSearchInfo returns search info for a specific slot for an episode.
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

		info := slotSearchInfoFromStatus(slotStatus, *slot.QualityProfileID)
		return &info, nil
	}

	return nil, ErrSlotNotFound
}

// GetTargetSlotForQuality determines which slot a release with a given quality should be assigned to.
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

	var bestSlot *SlotSearchInfo
	var isUpgrade bool

	// First, prefer empty slots
	for i := range slots {
		if slots[i].IsMissing() {
			bestSlot = &slots[i]
			isUpgrade = false
			break
		}
	}

	// If no empty slot, check for upgrade opportunity
	if bestSlot == nil {
		for i := range slots {
			if slots[i].NeedsUpgrade() && slots[i].CurrentQualityID != nil {
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

		result = append(result, slotSearchInfoFromStatus(slotStatus, *slot.QualityProfileID))
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

		result = append(result, slotSearchInfoFromStatus(slotStatus, *slot.QualityProfileID))
	}

	return result, nil
}

// EnrichReleasesWithSlotInfo enriches search releases with target slot information.
func (s *Service) EnrichReleasesWithSlotInfo(ctx context.Context, releases []types.TorrentInfo, mediaType string, mediaID int64) []types.TorrentInfo {
	if !s.IsMultiVersionEnabled(ctx) {
		return releases
	}

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

	enriched := make([]types.TorrentInfo, len(releases))
	copy(enriched, releases)

	for i := range enriched {
		release := &enriched[i]

		var qualityID int
		if release.ScoreBreakdown != nil {
			qualityID = release.ScoreBreakdown.QualityID
		}

		if qualityID == 0 {
			continue
		}

		targetSlot, isUpgrade, err := s.GetTargetSlotForQuality(ctx, mediaType, mediaID, qualityID)
		if err != nil || targetSlot == nil {
			continue
		}

		release.TargetSlotID = &targetSlot.SlotID
		release.TargetSlotNumber = &targetSlot.SlotNumber
		release.TargetSlotName = targetSlot.SlotName
		release.IsSlotUpgrade = isUpgrade
		release.IsSlotNewFill = !isUpgrade && targetSlot.IsMissing()
	}

	return enriched
}
