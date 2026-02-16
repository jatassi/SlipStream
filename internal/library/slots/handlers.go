package slots

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/scanner"
)

// Handlers provides HTTP handlers for slot operations.
type Handlers struct {
	service *Service
}

// NewHandlers creates new slot handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers the slot routes.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	// Multi-version settings
	g.GET("/settings", h.GetSettings)
	g.PUT("/settings", h.UpdateSettings)

	// Version slots
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.PUT("/:id", h.Update)
	g.PUT("/:id/enabled", h.SetEnabled)
	g.PUT("/:id/profile", h.SetProfile)
	g.GET("/:id/disable-check", h.CheckDisableSlot)
	g.POST("/:id/disable-with-action", h.DisableSlotWithAction)

	// Validation
	g.POST("/validate", h.ValidateConfiguration)
	g.POST("/validate-naming", h.ValidateNaming)

	// Req 18.2.2: Endpoint for auto-detecting slot assignment
	g.POST("/evaluate", h.EvaluateReleaseSlot)

	// Review queue for files without slot assignment
	g.GET("/review-queue", h.GetReviewQueue)
	g.GET("/review-queue/stats", h.GetReviewQueueStats)
	g.GET("/review-queue/:mediaType/:fileId", h.GetReviewQueueItemDetails)
	g.POST("/review-queue/:mediaType/:fileId/resolve", h.ResolveReviewQueueItem)

	// Migration (Req 14.1.1-14.2.3)
	g.POST("/migration/preview", h.GenerateMigrationPreview)
	g.POST("/migration/execute", h.ExecuteMigration)

	// Profile change with action (Req 15.1.1-15.1.2)
	g.GET("/:id/profile-change-check", h.CheckProfileChange)
	g.POST("/:id/profile-with-action", h.ChangeSlotProfile)

	// Slot assignments for movies
	g.GET("/movies/:movieId/assignments", h.GetMovieSlotAssignments)
	g.GET("/movies/:movieId/status", h.GetMovieStatus)
	g.POST("/movies/:movieId/slots/:slotId/assign", h.AssignMovieFile)
	g.POST("/movies/:movieId/slots/:slotId/unassign", h.UnassignMovieSlot)
	g.PUT("/movies/:movieId/slots/:slotId/monitored", h.SetMovieSlotMonitored)

	// Slot assignments for episodes
	g.GET("/episodes/:episodeId/assignments", h.GetEpisodeSlotAssignments)
	g.GET("/episodes/:episodeId/status", h.GetEpisodeStatus)
	g.POST("/episodes/:episodeId/slots/:slotId/assign", h.AssignEpisodeFile)
	g.POST("/episodes/:episodeId/slots/:slotId/unassign", h.UnassignEpisodeSlot)
	g.PUT("/episodes/:episodeId/slots/:slotId/monitored", h.SetEpisodeSlotMonitored)
}

// GetSettings returns the multi-version settings.
// GET /api/v1/slots/settings
func (h *Handlers) GetSettings(c echo.Context) error {
	settings, err := h.service.GetSettings(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, settings)
}

// UpdateSettings updates the multi-version settings.
// PUT /api/v1/slots/settings
func (h *Handlers) UpdateSettings(c echo.Context) error {
	var input UpdateMultiVersionSettingsInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	settings, err := h.service.UpdateSettings(c.Request().Context(), input)
	if err != nil {
		if errors.Is(err, ErrDryRunRequired) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if errors.Is(err, ErrProfilesNotExclusive) || errors.Is(err, ErrMissingProfileForSlot) {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, settings)
}

// List returns all slots.
// GET /api/v1/slots
func (h *Handlers) List(c echo.Context) error {
	slots, err := h.service.List(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, slots)
}

// Get returns a single slot.
// GET /api/v1/slots/:id
func (h *Handlers) Get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	slot, err := h.service.Get(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrSlotNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, slot)
}

// Update updates a slot.
// PUT /api/v1/slots/:id
func (h *Handlers) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input UpdateSlotInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	slot, err := h.service.Update(c.Request().Context(), id, input)
	if err != nil {
		if errors.Is(err, ErrSlotNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		if errors.Is(err, ErrInvalidSlot) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, slot)
}

// SetEnabledInput is the request body for setting slot enabled status.
type SetEnabledInput struct {
	Enabled bool `json:"enabled"`
}

// SetEnabled enables or disables a slot.
// PUT /api/v1/slots/:id/enabled
func (h *Handlers) SetEnabled(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input SetEnabledInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	slot, err := h.service.SetEnabled(c.Request().Context(), id, input.Enabled)
	if err != nil {
		if errors.Is(err, ErrSlotNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		if errors.Is(err, ErrSlotHasFiles) {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, slot)
}

// SetProfileInput is the request body for setting slot profile.
type SetProfileInput struct {
	QualityProfileID *int64 `json:"qualityProfileId"`
}

// SetProfile sets the quality profile for a slot.
// PUT /api/v1/slots/:id/profile
func (h *Handlers) SetProfile(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input SetProfileInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	slot, err := h.service.SetProfile(c.Request().Context(), id, input.QualityProfileID)
	if err != nil {
		if errors.Is(err, ErrSlotNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, slot)
}

// AttributeIssueResponse represents a specific attribute overlap in the API response.
type AttributeIssueResponse struct {
	Attribute string `json:"attribute"`
	Message   string `json:"message"`
}

// SlotConflictResponse represents a conflict between two slots in the API response.
type SlotConflictResponse struct {
	SlotAName string                   `json:"slotAName"`
	SlotBName string                   `json:"slotBName"`
	Issues    []AttributeIssueResponse `json:"issues"`
}

// ValidateConfigurationResponse is the response for slot configuration validation.
type ValidateConfigurationResponse struct {
	Valid     bool                   `json:"valid"`
	Errors    []string               `json:"errors,omitempty"`
	Conflicts []SlotConflictResponse `json:"conflicts,omitempty"`
}

// ValidateConfiguration validates the current slot configuration.
// POST /api/v1/slots/validate
func (h *Handlers) ValidateConfiguration(c echo.Context) error {
	result := h.service.ValidateSlotConfigurationFull(c.Request().Context())

	// Convert service conflicts to response format
	var conflicts []SlotConflictResponse
	for _, c := range result.Conflicts {
		var issues []AttributeIssueResponse
		for _, i := range c.Issues {
			issues = append(issues, AttributeIssueResponse(i))
		}
		conflicts = append(conflicts, SlotConflictResponse{
			SlotAName: c.SlotAName,
			SlotBName: c.SlotBName,
			Issues:    issues,
		})
	}

	return c.JSON(http.StatusOK, ValidateConfigurationResponse{
		Valid:     result.Valid,
		Errors:    result.Errors,
		Conflicts: conflicts,
	})
}

// ValidateNamingInput is the request body for validating filename formats.
// Req 4.1.4: Validation occurs when saving slot configuration
type ValidateNamingInput struct {
	MovieFileFormat   string `json:"movieFileFormat"`
	EpisodeFileFormat string `json:"episodeFileFormat"`
}

// ValidateNaming validates that filename formats include required differentiator tokens.
// POST /api/v1/slots/validate-naming
// Req 4.1.1-4.1.5: File naming validation
func (h *Handlers) ValidateNaming(c echo.Context) error {
	var input ValidateNamingInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Get enabled slots with their profiles
	slots, err := h.service.ListEnabled(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Build slot configs for validation
	slotConfigs := make([]quality.SlotConfig, 0, len(slots))
	for _, slot := range slots {
		if slot.QualityProfileID == nil {
			continue
		}
		profile, err := h.service.qualityService.Get(c.Request().Context(), *slot.QualityProfileID)
		if err != nil {
			continue
		}
		slotConfigs = append(slotConfigs, quality.SlotConfig{
			SlotNumber: slot.SlotNumber,
			SlotName:   slot.Name,
			Enabled:    true,
			Profile:    profile,
		})
	}

	// Validate naming formats
	validation := ValidateSlotNaming(slotConfigs, input.MovieFileFormat, input.EpisodeFileFormat)
	warnings := BuildNamingValidationWarnings(&validation)

	response := struct {
		SlotNamingValidation
		Warnings []string `json:"warnings"`
	}{
		SlotNamingValidation: validation,
		Warnings:             warnings,
	}

	return c.JSON(http.StatusOK, response)
}

// EvaluateReleaseSlotInput is the request body for evaluating release slot assignment.
type EvaluateReleaseSlotInput struct {
	ReleaseTitle string `json:"releaseTitle"`
	MediaType    string `json:"mediaType"` // "movie" or "episode"
	MediaID      int64  `json:"mediaId"`
}

// EvaluateReleaseSlot evaluates which slot a release would be assigned to.
// POST /api/v1/slots/evaluate
// Req 18.2.2: Auto-detect best slot for a release
func (h *Handlers) EvaluateReleaseSlot(c echo.Context) error {
	var input EvaluateReleaseSlotInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if input.ReleaseTitle == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "releaseTitle is required")
	}
	if input.MediaType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "mediaType is required")
	}
	if input.MediaID == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "mediaId is required")
	}

	// Parse the release title
	parsed := scanner.ParseFilename(input.ReleaseTitle)
	if parsed == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse release title")
	}

	// Evaluate against slots
	eval, err := h.service.EvaluateRelease(c.Request().Context(), parsed, input.MediaType, input.MediaID)
	if err != nil {
		if errors.Is(err, ErrNoMatchingSlot) {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"hasMatch":    false,
				"assignments": []interface{}{},
			})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"hasMatch":          len(eval.Assignments) > 0,
		"recommendedSlotId": eval.RecommendedSlotID,
		"requiresSelection": eval.RequiresSelection,
		"matchingCount":     eval.MatchingCount,
		"assignments":       eval.Assignments,
	})
}

// GetMovieSlotAssignments returns slot assignments for a movie.
// GET /api/v1/slots/movies/:movieId/assignments
func (h *Handlers) GetMovieSlotAssignments(c echo.Context) error {
	movieID, err := strconv.ParseInt(c.Param("movieId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid movie id")
	}

	assignments, err := h.service.GetMovieSlotAssignments(c.Request().Context(), movieID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, assignments)
}

// AssignFileInput is the request body for assigning a file to a slot.
type AssignFileInput struct {
	FileID int64 `json:"fileId"`
}

// AssignMovieFile assigns a movie file to a slot.
// POST /api/v1/slots/movies/:movieId/slots/:slotId/assign
func (h *Handlers) AssignMovieFile(c echo.Context) error {
	movieID, err := strconv.ParseInt(c.Param("movieId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid movie id")
	}

	slotID, err := strconv.ParseInt(c.Param("slotId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid slot id")
	}

	var input AssignFileInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.service.AssignFileToSlot(c.Request().Context(), "movie", movieID, slotID, input.FileID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// UnassignMovieSlot removes a file from a movie slot.
// POST /api/v1/slots/movies/:movieId/slots/:slotId/unassign
func (h *Handlers) UnassignMovieSlot(c echo.Context) error {
	movieID, err := strconv.ParseInt(c.Param("movieId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid movie id")
	}

	slotID, err := strconv.ParseInt(c.Param("slotId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid slot id")
	}

	if err := h.service.UnassignFileFromSlot(c.Request().Context(), "movie", movieID, slotID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// GetEpisodeSlotAssignments returns slot assignments for an episode.
// GET /api/v1/slots/episodes/:episodeId/assignments
func (h *Handlers) GetEpisodeSlotAssignments(c echo.Context) error {
	episodeID, err := strconv.ParseInt(c.Param("episodeId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid episode id")
	}

	assignments, err := h.service.GetEpisodeSlotAssignments(c.Request().Context(), episodeID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, assignments)
}

// AssignEpisodeFile assigns an episode file to a slot.
// POST /api/v1/slots/episodes/:episodeId/slots/:slotId/assign
func (h *Handlers) AssignEpisodeFile(c echo.Context) error {
	episodeID, err := strconv.ParseInt(c.Param("episodeId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid episode id")
	}

	slotID, err := strconv.ParseInt(c.Param("slotId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid slot id")
	}

	var input AssignFileInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.service.AssignFileToSlot(c.Request().Context(), "episode", episodeID, slotID, input.FileID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// UnassignEpisodeSlot removes a file from an episode slot.
// POST /api/v1/slots/episodes/:episodeId/slots/:slotId/unassign
func (h *Handlers) UnassignEpisodeSlot(c echo.Context) error {
	episodeID, err := strconv.ParseInt(c.Param("episodeId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid episode id")
	}

	slotID, err := strconv.ParseInt(c.Param("slotId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid slot id")
	}

	if err := h.service.UnassignFileFromSlot(c.Request().Context(), "episode", episodeID, slotID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// GetMovieStatus returns the slot status for a movie.
// GET /api/v1/slots/movies/:movieId/status
func (h *Handlers) GetMovieStatus(c echo.Context) error {
	movieID, err := strconv.ParseInt(c.Param("movieId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid movie id")
	}

	status, err := h.service.GetMovieStatus(c.Request().Context(), movieID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, status)
}

// GetEpisodeStatus returns the slot status for an episode.
// GET /api/v1/slots/episodes/:episodeId/status
func (h *Handlers) GetEpisodeStatus(c echo.Context) error {
	episodeID, err := strconv.ParseInt(c.Param("episodeId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid episode id")
	}

	status, err := h.service.GetEpisodeStatus(c.Request().Context(), episodeID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, status)
}

// SetMonitoredInput is the request body for setting monitored status.
type SetMonitoredInput struct {
	Monitored bool `json:"monitored"`
}

// SetMovieSlotMonitored sets the monitored status for a movie slot.
// PUT /api/v1/slots/movies/:movieId/slots/:slotId/monitored
func (h *Handlers) SetMovieSlotMonitored(c echo.Context) error {
	movieID, err := strconv.ParseInt(c.Param("movieId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid movie id")
	}

	slotID, err := strconv.ParseInt(c.Param("slotId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid slot id")
	}

	var input SetMonitoredInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.service.SetSlotMonitored(c.Request().Context(), "movie", movieID, slotID, input.Monitored); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// SetEpisodeSlotMonitored sets the monitored status for an episode slot.
// PUT /api/v1/slots/episodes/:episodeId/slots/:slotId/monitored
func (h *Handlers) SetEpisodeSlotMonitored(c echo.Context) error {
	episodeID, err := strconv.ParseInt(c.Param("episodeId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid episode id")
	}

	slotID, err := strconv.ParseInt(c.Param("slotId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid slot id")
	}

	var input SetMonitoredInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.service.SetSlotMonitored(c.Request().Context(), "episode", episodeID, slotID, input.Monitored); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// CheckDisableSlot checks if a slot can be disabled and returns file info.
// GET /api/v1/slots/:id/disable-check
// Req 12.2.1: When user disables slot with files, prompt for action
func (h *Handlers) CheckDisableSlot(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	result, err := h.service.CheckDisableSlot(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}

// DisableSlotWithActionInput is the request body for disabling a slot with action.
type DisableSlotWithActionInput struct {
	Action string `json:"action"` // "delete", "keep", or "cancel"
}

// DisableSlotWithAction disables a slot with the specified action for existing files.
// POST /api/v1/slots/:id/disable-with-action
// Req 12.2.2: Options: delete files, keep unassigned, or cancel
func (h *Handlers) DisableSlotWithAction(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input DisableSlotWithActionInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Validate action
	action := DisableSlotAction(input.Action)
	if action != DisableActionDelete && action != DisableActionKeep && action != DisableActionCancel {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid action: must be 'delete', 'keep', or 'cancel'")
	}

	if action == DisableActionCancel {
		return c.JSON(http.StatusOK, map[string]string{"status": "cancelled"})
	}

	err = h.service.DisableSlotWithAction(c.Request().Context(), DisableSlotRequest{
		SlotID: id,
		Action: action,
	})
	if err != nil {
		if errors.Is(err, ErrSlotNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Fetch the updated slot
	slot, err := h.service.Get(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, slot)
}

// GetReviewQueue returns files that are in the review queue (no slot assigned).
// GET /api/v1/slots/review-queue
// Req 13.1.3: Extra files (more than slot count) queued for user review
func (h *Handlers) GetReviewQueue(c echo.Context) error {
	items, err := h.service.ListReviewQueueItems(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, items)
}

// GetReviewQueueStats returns statistics about the review queue.
// GET /api/v1/slots/review-queue/stats
func (h *Handlers) GetReviewQueueStats(c echo.Context) error {
	stats, err := h.service.GetReviewQueueStats(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, stats)
}

// GetReviewQueueItemDetails returns detailed information about a review queue item.
// GET /api/v1/slots/review-queue/:mediaType/:fileId
// Req 14.3.2: Show file details, detected quality, and available slot options
func (h *Handlers) GetReviewQueueItemDetails(c echo.Context) error {
	mediaType := c.Param("mediaType")
	fileID, err := strconv.ParseInt(c.Param("fileId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
	}

	details, err := h.service.GetReviewQueueItemDetails(c.Request().Context(), mediaType, fileID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, details)
}

// ResolveReviewQueueItemInput is the request body for resolving a review queue item.
type ResolveReviewQueueItemInput struct {
	Action       string `json:"action"`       // "assign", "delete", or "skip"
	TargetSlotID *int64 `json:"targetSlotId"` // Required for "assign"
}

// ResolveReviewQueueItem resolves a review queue item.
// POST /api/v1/slots/review-queue/:mediaType/:fileId/resolve
// Req 14.3.3: Allow assignment to specific slot or deletion
func (h *Handlers) ResolveReviewQueueItem(c echo.Context) error {
	mediaType := c.Param("mediaType")
	fileID, err := strconv.ParseInt(c.Param("fileId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
	}

	var input ResolveReviewQueueItemInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	action := ReviewItemAction(input.Action)
	if action != ReviewActionAssign && action != ReviewActionDelete && action != ReviewActionSkip {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid action: must be 'assign', 'delete', or 'skip'")
	}

	err = h.service.ResolveReviewQueueItem(c.Request().Context(), ResolveReviewItemRequest{
		MediaType:    mediaType,
		FileID:       fileID,
		Action:       action,
		TargetSlotID: input.TargetSlotID,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// GenerateMigrationPreview generates a preview of the migration.
// POST /api/v1/slots/migration/preview
// Req 14.1.1: Dry run preview is required before enabling multi-version
func (h *Handlers) GenerateMigrationPreview(c echo.Context) error {
	preview, err := h.service.GenerateMigrationPreview(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, preview)
}

// ExecuteMigration executes the migration.
// POST /api/v1/slots/migration/execute
// Req 14.2.1-14.2.3: Execute migration and assign files to slots
func (h *Handlers) ExecuteMigration(c echo.Context) error {
	var input ExecuteMigrationInput
	// Bind is optional - if no body provided, overrides will be empty
	_ = c.Bind(&input)

	result, err := h.service.ExecuteMigration(c.Request().Context(), input.Overrides)
	if err != nil {
		if errors.Is(err, ErrMissingProfileForSlot) || errors.Is(err, ErrProfilesNotExclusive) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}

// CheckProfileChangeInput is the request body for checking profile change impact.
type CheckProfileChangeInput struct {
	NewProfileID int64 `json:"newProfileId"`
}

// CheckProfileChange checks if changing a slot's profile requires user action.
// GET /api/v1/slots/:id/profile-change-check
// Req 15.1.1: When user changes a slot's quality profile after files are assigned, prompt for action
func (h *Handlers) CheckProfileChange(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input CheckProfileChangeInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	result, err := h.service.CheckProfileChange(c.Request().Context(), id, input.NewProfileID)
	if err != nil {
		if errors.Is(err, ErrSlotNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}

// ChangeSlotProfileInput is the request body for changing a slot's profile with action.
type ChangeSlotProfileInput struct {
	NewProfileID int64  `json:"newProfileId"`
	Action       string `json:"action"` // "keep", "reevaluate", or "cancel"
}

// ChangeSlotProfile changes a slot's profile with the specified action.
// POST /api/v1/slots/:id/profile-with-action
// Req 15.1.2: Options: keep current assignments, re-evaluate (may queue non-matches), or cancel
func (h *Handlers) ChangeSlotProfile(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input ChangeSlotProfileInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	action := ProfileChangeAction(input.Action)
	if action != ProfileChangeKeep && action != ProfileChangeReevaluate && action != ProfileChangeCancel {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid action: must be 'keep', 'reevaluate', or 'cancel'")
	}

	slot, err := h.service.ChangeSlotProfile(c.Request().Context(), ProfileChangeRequest{
		SlotID:       id,
		NewProfileID: input.NewProfileID,
		Action:       action,
	})
	if err != nil {
		if errors.Is(err, ErrSlotNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, slot)
}
