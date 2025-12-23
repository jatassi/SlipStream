package librarymanager

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
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
