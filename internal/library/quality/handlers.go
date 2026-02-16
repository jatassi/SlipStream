package quality

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// Handlers provides HTTP handlers for quality profile operations.
type Handlers struct {
	service *Service
}

// NewHandlers creates new quality profile handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers the quality profile routes.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET("/qualities", h.ListQualities)
	g.GET("/attributes", h.ListAttributes)
	g.POST("/check-exclusivity", h.CheckExclusivity)
	g.GET("/:id", h.Get)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

// List returns all quality profiles.
// GET /api/v1/qualityprofiles
func (h *Handlers) List(c echo.Context) error {
	profiles, err := h.service.List(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, profiles)
}

// Get returns a single quality profile.
// GET /api/v1/qualityprofiles/:id
func (h *Handlers) Get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	profile, err := h.service.Get(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, profile)
}

// Create creates a new quality profile.
// POST /api/v1/qualityprofiles
func (h *Handlers) Create(c echo.Context) error {
	var input CreateProfileInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	profile, err := h.service.Create(c.Request().Context(), &input)
	if err != nil {
		if errors.Is(err, ErrInvalidProfile) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, profile)
}

// Update updates an existing quality profile.
// PUT /api/v1/qualityprofiles/:id
func (h *Handlers) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input UpdateProfileInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	profile, err := h.service.Update(c.Request().Context(), id, &input)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		if errors.Is(err, ErrInvalidProfile) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Recalculate status for all media using this profile (cutoff/upgrades may have changed)
	if _, err := h.service.RecalculateStatusForProfile(c.Request().Context(), id); err != nil {
		h.service.logger.Warn().Err(err).Int64("profileId", id).Msg("Failed to recalculate status after profile update")
	}

	return c.JSON(http.StatusOK, profile)
}

// Delete deletes a quality profile.
// DELETE /api/v1/qualityprofiles/:id
func (h *Handlers) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	if err := h.service.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		if errors.Is(err, ErrProfileInUse) {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// ListQualities returns the predefined quality definitions.
// GET /api/v1/qualityprofiles/qualities
func (h *Handlers) ListQualities(c echo.Context) error {
	return c.JSON(http.StatusOK, h.service.GetQualities())
}

// ListAttributes returns the supported attribute values for quality profiles.
// GET /api/v1/qualityprofiles/attributes
func (h *Handlers) ListAttributes(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"hdrFormats":    HDRFormats,
		"videoCodecs":   VideoCodecs,
		"audioCodecs":   AudioCodecs,
		"audioChannels": AudioChannels,
		"modes":         []AttributeMode{AttributeModeAcceptable, AttributeModePreferred, AttributeModeRequired},
	})
}

// CheckExclusivityInput is the request body for exclusivity checking.
type CheckExclusivityInput struct {
	ProfileIDs []int64 `json:"profileIds"`
}

// CheckExclusivityResponse is the response for exclusivity checking.
type CheckExclusivityResponse struct {
	Valid   bool                   `json:"valid"`
	Errors  []SlotExclusivityError `json:"errors,omitempty"`
	Details []ExclusivityDetail    `json:"details,omitempty"`
}

// ExclusivityDetail provides detail about a specific profile pair check.
type ExclusivityDetail struct {
	ProfileAID   int64    `json:"profileAId"`
	ProfileAName string   `json:"profileAName"`
	ProfileBID   int64    `json:"profileBId"`
	ProfileBName string   `json:"profileBName"`
	AreExclusive bool     `json:"areExclusive"`
	Conflicts    []string `json:"conflicts,omitempty"`
	Overlaps     []string `json:"overlaps,omitempty"`
	Hints        []string `json:"hints,omitempty"`
}

// CheckExclusivity checks if the given profiles are mutually exclusive.
// POST /api/v1/qualityprofiles/check-exclusivity
func (h *Handlers) CheckExclusivity(c echo.Context) error {
	var input CheckExclusivityInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if len(input.ProfileIDs) < 2 {
		return echo.NewHTTPError(http.StatusBadRequest, "at least 2 profile IDs are required")
	}

	ctx := c.Request().Context()
	profiles, err := h.loadProfiles(ctx, input.ProfileIDs)
	if err != nil {
		return err
	}

	details, allExclusive := compareProfilePairs(profiles)

	response := CheckExclusivityResponse{
		Valid:   allExclusive,
		Details: details,
	}

	if !allExclusive {
		response.Errors = h.buildExclusivityErrors(profiles)
	}

	return c.JSON(http.StatusOK, response)
}

func (h *Handlers) loadProfiles(ctx context.Context, ids []int64) ([]*Profile, error) {
	profiles := make([]*Profile, 0, len(ids))
	for _, id := range ids {
		profile, err := h.service.Get(ctx, id)
		if err != nil {
			if errors.Is(err, ErrProfileNotFound) {
				return nil, echo.NewHTTPError(http.StatusNotFound, "profile not found: "+strconv.FormatInt(id, 10))
			}
			return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

func compareProfilePairs(profiles []*Profile) ([]ExclusivityDetail, bool) {
	var details []ExclusivityDetail
	allExclusive := true

	for i := 0; i < len(profiles); i++ {
		for j := i + 1; j < len(profiles); j++ {
			result := CheckMutualExclusivity(profiles[i], profiles[j])
			hints := GetProfileExclusivityHints(profiles[i], profiles[j])

			details = append(details, ExclusivityDetail{
				ProfileAID:   profiles[i].ID,
				ProfileAName: profiles[i].Name,
				ProfileBID:   profiles[j].ID,
				ProfileBName: profiles[j].Name,
				AreExclusive: result.AreExclusive,
				Conflicts:    result.ConflictingAttrs,
				Overlaps:     result.OverlappingAttrs,
				Hints:        hints,
			})

			if !result.AreExclusive {
				allExclusive = false
			}
		}
	}

	return details, allExclusive
}

func (h *Handlers) buildExclusivityErrors(profiles []*Profile) []SlotExclusivityError {
	slots := make([]SlotConfig, len(profiles))
	for i, p := range profiles {
		slots[i] = SlotConfig{
			SlotNumber: i + 1,
			SlotName:   "Slot " + strconv.Itoa(i+1),
			Enabled:    true,
			Profile:    p,
		}
	}
	errs, _ := ValidateSlotExclusivity(slots)
	return errs
}
