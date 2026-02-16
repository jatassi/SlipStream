package librarymanager

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
)

// Handlers provides HTTP handlers for library management.
type Handlers struct {
	service *Service
}

// NewHandlers creates new library manager handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// ScanRootFolder handles POST /api/v1/rootfolders/:id/scan
// Triggers an async scan of the specified root folder.
func (h *Handlers) ScanRootFolder(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid root folder ID")
	}

	// Check if already scanning
	if h.service.IsScanActive(id) {
		return c.JSON(http.StatusConflict, map[string]interface{}{
			"message":      "Scan already in progress",
			"rootFolderId": id,
		})
	}

	// Run scan asynchronously
	go func() {
		ctx := context.Background()
		_, err := h.service.ScanRootFolder(ctx, id)
		if err != nil {
			h.service.logger.Error().Err(err).Int64("rootFolderId", id).Msg("Async scan failed")
		}
	}()

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"message":      "Scan started",
		"rootFolderId": id,
	})
}

// GetScanStatus handles GET /api/v1/rootfolders/:id/scan
// Returns the current scan status for a root folder.
func (h *Handlers) GetScanStatus(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid root folder ID")
	}

	activityID := h.service.GetActiveScanActivity(id)
	if activityID == "" {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"rootFolderId": id,
			"active":       false,
		})
	}

	// Get activity details from progress manager
	if h.service.progress != nil {
		activity := h.service.progress.GetActivity(activityID)
		if activity != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"rootFolderId": id,
				"active":       true,
				"activity":     activity,
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"rootFolderId": id,
		"active":       true,
		"activityId":   activityID,
	})
}

// CancelScan handles DELETE /api/v1/rootfolders/:id/scan
// Cancels an active scan for a root folder.
func (h *Handlers) CancelScan(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid root folder ID")
	}

	if h.service.CancelScan(id) {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message":      "Scan cancelled",
			"rootFolderId": id,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":      "No active scan to cancel",
		"rootFolderId": id,
	})
}

// GetAllScanStatuses handles GET /api/v1/scans
// Returns all active scan statuses.
func (h *Handlers) GetAllScanStatuses(c echo.Context) error {
	if h.service.progress == nil {
		return c.JSON(http.StatusOK, []interface{}{})
	}

	// Get all scan activities from progress manager
	activities := h.service.progress.GetActivitiesByType("scan")
	return c.JSON(http.StatusOK, activities)
}

// ScanAllRootFolders handles POST /api/v1/scans
// Triggers scans for all root folders.
func (h *Handlers) ScanAllRootFolders(c echo.Context) error {
	ctx := context.Background()

	// Get all root folders
	folders, err := h.service.rootfolders.List(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list root folders")
	}

	// Start scan for each folder
	startedScans := make([]int64, 0)
	for _, folder := range folders {
		if !h.service.IsScanActive(folder.ID) {
			go func(id int64) {
				scanCtx := context.Background()
				_, err := h.service.ScanRootFolder(scanCtx, id)
				if err != nil {
					h.service.logger.Error().Err(err).Int64("rootFolderId", id).Msg("Async scan failed")
				}
			}(folder.ID)
			startedScans = append(startedScans, folder.ID)
		}
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"message":       "Scans started",
		"rootFolderIds": startedScans,
	})
}

// RefreshMovie handles POST /api/v1/movies/:id/refresh
// Refreshes metadata for a single movie.
func (h *Handlers) RefreshMovie(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid movie ID")
	}

	movie, err := h.service.RefreshMovieMetadata(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrNoMetadataProvider) {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "no metadata provider configured")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, movie)
}

// RefreshSeries handles POST /api/v1/series/:id/refresh
// Refreshes metadata for a single series.
func (h *Handlers) RefreshSeries(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid series ID")
	}

	series, err := h.service.RefreshSeriesMetadata(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrNoMetadataProvider) {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "no metadata provider configured")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, series)
}

// RefreshAllMovies handles POST /api/v1/movies/refresh
// Scans all movie root folders and refreshes metadata for all movies.
func (h *Handlers) RefreshAllMovies(c echo.Context) error {
	go func() {
		ctx := context.Background()
		if err := h.service.RefreshAllMovies(ctx); err != nil {
			h.service.logger.Error().Err(err).Msg("Refresh all movies failed")
		}
	}()

	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "Refresh started for all movies",
	})
}

// RefreshAllSeries handles POST /api/v1/series/refresh
// Scans all TV root folders and refreshes metadata for all series.
func (h *Handlers) RefreshAllSeries(c echo.Context) error {
	go func() {
		ctx := context.Background()
		if err := h.service.RefreshAllSeries(ctx); err != nil {
			h.service.logger.Error().Err(err).Msg("Refresh all series failed")
		}
	}()

	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "Refresh started for all series",
	})
}

// AddMovie handles POST /api/v1/library/movies
// Creates a new movie and downloads artwork in the background.
func (h *Handlers) AddMovie(c echo.Context) error {
	var input AddMovieInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if claims := portalmw.GetPortalUser(c); claims != nil {
		input.AddedBy = &claims.UserID
	}

	movie, err := h.service.AddMovie(c.Request().Context(), &input)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, movie)
}

// AddSeries handles POST /api/v1/library/series
// Creates a new series and downloads artwork in the background.
func (h *Handlers) AddSeries(c echo.Context) error {
	var input AddSeriesInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if claims := portalmw.GetPortalUser(c); claims != nil {
		input.AddedBy = &claims.UserID
	}

	series, err := h.service.AddSeries(c.Request().Context(), &input)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, series)
}
