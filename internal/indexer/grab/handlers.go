package grab

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/slipstream/slipstream/internal/indexer/types"
)

// Handlers provides HTTP handlers for grab operations.
type Handlers struct {
	service *Service
}

// NewHandlers creates new grab handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{
		service: service,
	}
}

// RegisterRoutes registers the grab routes.
// These are typically registered under /api/v1/search/grab
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.POST("/grab", h.Grab)
	g.POST("/grab/bulk", h.GrabBulk)
	g.GET("/grab/history", h.GetHistory)
}

// GrabRequestDTO is the API request format for grabbing a release.
type GrabRequestDTO struct {
	Release      ReleaseDTO `json:"release"`
	ClientID     int64      `json:"clientId,omitempty"`
	MediaType    string     `json:"mediaType,omitempty"`
	MediaID      int64      `json:"mediaId,omitempty"`
	SeriesID     int64      `json:"seriesId,omitempty"`
	SeasonNumber int        `json:"seasonNumber,omitempty"`
	IsSeasonPack bool       `json:"isSeasonPack,omitempty"`
	// Req 18.2.1: API grab requests accept optional target_slot parameter
	TargetSlotID *int64 `json:"targetSlotId,omitempty"`
}

// ReleaseDTO is the API format for a release.
type ReleaseDTO struct {
	GUID        string `json:"guid"`
	Title       string `json:"title"`
	DownloadURL string `json:"downloadUrl"`
	IndexerID   int64  `json:"indexerId"`
	IndexerName string `json:"indexer,omitempty"`
	Protocol    string `json:"protocol"`
	Size        int64  `json:"size,omitempty"`
	ImdbID      int    `json:"imdbId,omitempty"`
	TmdbID      int    `json:"tmdbId,omitempty"`
	TvdbID      int    `json:"tvdbId,omitempty"`
}

// BulkGrabRequestDTO is the API request format for grabbing multiple releases.
type BulkGrabRequestDTO struct {
	Releases     []ReleaseDTO `json:"releases"`
	ClientID     int64        `json:"clientId,omitempty"`
	MediaType    string       `json:"mediaType,omitempty"`
	MediaID      int64        `json:"mediaId,omitempty"`
	SeriesID     int64        `json:"seriesId,omitempty"`
	SeasonNumber int          `json:"seasonNumber,omitempty"`
	IsSeasonPack bool         `json:"isSeasonPack,omitempty"`
	// Req 18.2.1: API grab requests accept optional target_slot parameter
	TargetSlotID *int64 `json:"targetSlotId,omitempty"`
}

// Grab handles POST /grab - grab a single release.
func (h *Handlers) Grab(c echo.Context) error {
	var req GrabRequestDTO
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Release.DownloadURL == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "downloadUrl is required",
		})
	}
	if req.Release.IndexerID == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "indexerId is required",
		})
	}

	// Convert DTO to internal type
	release := h.dtoToRelease(req.Release)

	// Req 18.2.1, 18.2.2: API grab requests accept optional target_slot parameter
	// If omitted, auto-detect happens upstream in autosearch or caller provides it
	result, err := h.service.Grab(c.Request().Context(), GrabRequest{
		Release:      release,
		ClientID:     req.ClientID,
		MediaType:    req.MediaType,
		MediaID:      req.MediaID,
		SeriesID:     req.SeriesID,
		SeasonNumber: req.SeasonNumber,
		IsSeasonPack: req.IsSeasonPack,
		TargetSlotID: req.TargetSlotID,
	})

	if err != nil {
		// Return the result even on error - it contains error details
		return c.JSON(http.StatusOK, result)
	}

	return c.JSON(http.StatusOK, result)
}

// GrabBulk handles POST /grab/bulk - grab multiple releases.
func (h *Handlers) GrabBulk(c echo.Context) error {
	var req BulkGrabRequestDTO
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if len(req.Releases) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "At least one release is required",
		})
	}

	// Convert DTOs to internal types
	releases := make([]*types.ReleaseInfo, 0, len(req.Releases))
	for _, dto := range req.Releases {
		releases = append(releases, h.dtoToRelease(dto))
	}

	// Req 18.2.1: Bulk grab also accepts optional target_slot parameter
	result, err := h.service.GrabBulk(c.Request().Context(), BulkGrabRequest{
		Releases:     releases,
		ClientID:     req.ClientID,
		MediaType:    req.MediaType,
		MediaID:      req.MediaID,
		SeriesID:     req.SeriesID,
		SeasonNumber: req.SeasonNumber,
		IsSeasonPack: req.IsSeasonPack,
		TargetSlotID: req.TargetSlotID,
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// GetHistory handles GET /grab/history - get grab history.
func (h *Handlers) GetHistory(c echo.Context) error {
	limit := 50
	offset := 0

	if l := c.QueryParam("limit"); l != "" {
		if err := echo.QueryParamsBinder(c).Int("limit", &limit).BindError(); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid limit parameter",
			})
		}
	}

	if o := c.QueryParam("offset"); o != "" {
		if err := echo.QueryParamsBinder(c).Int("offset", &offset).BindError(); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid offset parameter",
			})
		}
	}

	history, err := h.service.GetGrabHistory(c.Request().Context(), limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, history)
}

// dtoToRelease converts a ReleaseDTO to a types.ReleaseInfo.
func (h *Handlers) dtoToRelease(dto ReleaseDTO) *types.ReleaseInfo {
	return &types.ReleaseInfo{
		GUID:        dto.GUID,
		Title:       dto.Title,
		DownloadURL: dto.DownloadURL,
		IndexerID:   dto.IndexerID,
		IndexerName: dto.IndexerName,
		Protocol:    types.Protocol(dto.Protocol),
		Size:        dto.Size,
		ImdbID:      dto.ImdbID,
		TmdbID:      dto.TmdbID,
		TvdbID:      dto.TvdbID,
	}
}
