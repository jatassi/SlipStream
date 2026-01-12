package importer

import (
	"database/sql"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/import/renamer"
	"github.com/slipstream/slipstream/internal/library/scanner"
)

// parseFilename is a convenience wrapper around scanner.ParseFilename
func parseFilename(filename string) *scanner.ParsedMedia {
	return scanner.ParseFilename(filename)
}

// Handlers provides HTTP handlers for import operations.
type Handlers struct {
	service *Service
	queries *sqlc.Queries
}

// NewHandlers creates a new import handlers instance.
func NewHandlers(service *Service, db *sql.DB) *Handlers {
	return &Handlers{
		service: service,
		queries: sqlc.New(db),
	}
}

// RegisterRoutes registers import routes on an Echo group.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("/pending", h.GetPendingImports)
	g.GET("/status", h.GetImportStatus)
	g.POST("/manual", h.ManualImport)
	g.POST("/manual/preview", h.PreviewManualImport)
	g.POST("/:id/retry", h.RetryImport)
	g.POST("/scan", h.ScanDirectory)

	// Mass rename endpoints
	g.GET("/rename/preview", h.GetRenamePreview)
	g.POST("/rename/execute", h.ExecuteRename)
}

// ImportStatusResponse contains import service status.
type ImportStatusResponse struct {
	QueueLength     int `json:"queueLength"`
	ProcessingCount int `json:"processingCount"`
}

// GetImportStatus returns the current import service status.
// GET /api/v1/import/status
func (h *Handlers) GetImportStatus(c echo.Context) error {
	return c.JSON(http.StatusOK, ImportStatusResponse{
		QueueLength:     h.service.GetQueueLength(),
		ProcessingCount: h.service.GetProcessingCount(),
	})
}

// PendingImport represents a file pending import.
type PendingImport struct {
	ID          int64   `json:"id,omitempty"`
	FilePath    string  `json:"filePath"`
	FileName    string  `json:"fileName"`
	FileSize    int64   `json:"fileSize"`
	Status      string  `json:"status"`
	MediaType   *string `json:"mediaType,omitempty"`
	MediaID     *int64  `json:"mediaId,omitempty"`
	MediaTitle  *string `json:"mediaTitle,omitempty"`
	Error       *string `json:"error,omitempty"`
	Attempts    int     `json:"attempts"`
	IsProcessing bool   `json:"isProcessing"`
}

// GetPendingImports returns files pending import.
// GET /api/v1/import/pending
func (h *Handlers) GetPendingImports(c echo.Context) error {
	ctx := c.Request().Context()

	// Get pending items from queue_media table
	rows, err := h.queries.ListPendingQueueMedia(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	pending := make([]PendingImport, 0, len(rows))
	for _, row := range rows {
		item := PendingImport{
			ID:        row.ID,
			FilePath:  row.FilePath.String,
			FileName:  filepath.Base(row.FilePath.String),
			Status:    row.FileStatus,
			Attempts:  int(row.ImportAttempts),
			IsProcessing: h.service.IsProcessing(row.FilePath.String),
		}

		if row.ErrorMessage.Valid {
			item.Error = &row.ErrorMessage.String
		}

		// Add media info if available
		if row.MovieID.Valid {
			item.MediaID = &row.MovieID.Int64
			mediaType := "movie"
			item.MediaType = &mediaType
		} else if row.EpisodeID.Valid {
			item.MediaID = &row.EpisodeID.Int64
			mediaType := "episode"
			item.MediaType = &mediaType
		}

		pending = append(pending, item)
	}

	return c.JSON(http.StatusOK, pending)
}

// ManualImportRequest contains the request body for manual import.
type ManualImportRequest struct {
	Path      string `json:"path" validate:"required"`
	MediaType string `json:"mediaType" validate:"required,oneof=movie episode"`
	MediaID   int64  `json:"mediaId" validate:"required"`
	SeriesID  *int64 `json:"seriesId,omitempty"`
	SeasonNum *int   `json:"seasonNum,omitempty"`
}

// ManualImportResponse contains the response for a manual import.
type ManualImportResponse struct {
	Success         bool   `json:"success"`
	SourcePath      string `json:"sourcePath"`
	DestinationPath string `json:"destinationPath,omitempty"`
	LinkMode        string `json:"linkMode,omitempty"`
	IsUpgrade       bool   `json:"isUpgrade"`
	Error           string `json:"error,omitempty"`
}

// ManualImport imports a file manually with explicit match.
// POST /api/v1/import/manual
func (h *Handlers) ManualImport(c echo.Context) error {
	ctx := c.Request().Context()

	var req ManualImportRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Path == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "path is required")
	}

	if req.MediaType == "" || (req.MediaType != "movie" && req.MediaType != "episode") {
		return echo.NewHTTPError(http.StatusBadRequest, "mediaType must be 'movie' or 'episode'")
	}

	// Build library match
	match := &LibraryMatch{
		MediaType:  req.MediaType,
		Confidence: 1.0,
		Source:     "manual",
	}

	if req.MediaType == "movie" {
		match.MovieID = &req.MediaID
	} else {
		match.EpisodeID = &req.MediaID
		match.SeriesID = req.SeriesID
		match.SeasonNum = req.SeasonNum
	}

	// Get root folder from library item
	if err := h.service.populateRootFolder(ctx, match); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to determine root folder: "+err.Error())
	}

	// Check for existing file (upgrade)
	if err := h.service.checkForExistingFile(ctx, match); err != nil {
		// Non-fatal, just means it's not an upgrade
		_ = err
	}

	// Process the import synchronously
	result, err := h.service.ProcessManualImport(ctx, req.Path, match)

	resp := ManualImportResponse{
		SourcePath: req.Path,
	}

	if err != nil {
		resp.Success = false
		resp.Error = err.Error()
		return c.JSON(http.StatusOK, resp)
	}

	resp.Success = result.Success
	resp.DestinationPath = result.DestinationPath
	resp.LinkMode = string(result.LinkMode)
	resp.IsUpgrade = result.IsUpgrade

	if result.Error != nil {
		resp.Error = result.Error.Error()
	}

	return c.JSON(http.StatusOK, resp)
}

// PreviewImportRequest contains the request body for import preview.
type PreviewImportRequest struct {
	Path string `json:"path" validate:"required"`
}

// PreviewImportResponse contains the response for an import preview.
type PreviewImportResponse struct {
	Path           string           `json:"path"`
	FileName       string           `json:"fileName"`
	FileSize       int64            `json:"fileSize"`
	Valid          bool             `json:"valid"`
	ValidationError string          `json:"validationError,omitempty"`
	ParsedInfo     *ParsedMediaInfo `json:"parsedInfo,omitempty"`
	SuggestedMatch *SuggestedMatch  `json:"suggestedMatch,omitempty"`
}

// ParsedMediaInfo contains parsed information from a filename.
type ParsedMediaInfo struct {
	Title             string   `json:"title,omitempty"`
	Year              int      `json:"year,omitempty"`
	Season            int      `json:"season,omitempty"`
	Episode           int      `json:"episode,omitempty"`
	EndEpisode        int      `json:"endEpisode,omitempty"`
	Quality           string   `json:"quality,omitempty"`
	Source            string   `json:"source,omitempty"`
	Codec             string   `json:"codec,omitempty"`
	AudioCodecs       []string `json:"audioCodecs,omitempty"`
	AudioChannels     []string `json:"audioChannels,omitempty"`
	AudioEnhancements []string `json:"audioEnhancements,omitempty"`
	Attributes        []string `json:"attributes,omitempty"`
	IsTV              bool     `json:"isTV"`
	IsSeasonPack      bool     `json:"isSeasonPack,omitempty"`
}

// SuggestedMatch contains a suggested library match.
type SuggestedMatch struct {
	MediaType   string  `json:"mediaType"`
	MediaID     int64   `json:"mediaId"`
	MediaTitle  string  `json:"mediaTitle"`
	Confidence  float64 `json:"confidence"`
	Year        int     `json:"year,omitempty"`
	SeasonNum   *int    `json:"seasonNum,omitempty"`
	EpisodeNum  *int    `json:"episodeNum,omitempty"`
	SeriesID    *int64  `json:"seriesId,omitempty"`
	SeriesTitle *string `json:"seriesTitle,omitempty"`
}

// PreviewManualImport previews what would happen if a file was imported.
// POST /api/v1/import/manual/preview
func (h *Handlers) PreviewManualImport(c echo.Context) error {
	ctx := c.Request().Context()

	var req PreviewImportRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Path == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "path is required")
	}

	resp := PreviewImportResponse{
		Path:     req.Path,
		FileName: filepath.Base(req.Path),
	}

	// Validate the file
	validation, err := h.service.ValidateForImport(ctx, req.Path)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resp.FileSize = validation.FileSize
	resp.Valid = validation.Valid
	if !validation.Valid {
		resp.ValidationError = validation.Reason
		return c.JSON(http.StatusOK, resp)
	}

	// Parse the filename
	parsed := parseFilename(filepath.Base(req.Path))
	if parsed != nil {
		resp.ParsedInfo = &ParsedMediaInfo{
			Title:             parsed.Title,
			Year:              parsed.Year,
			Quality:           parsed.Quality,
			Source:            parsed.Source,
			Codec:             parsed.Codec,
			AudioCodecs:       parsed.AudioCodecs,
			AudioChannels:     parsed.AudioChannels,
			AudioEnhancements: parsed.AudioEnhancements,
			Attributes:        parsed.Attributes,
			IsTV:              parsed.IsTV,
			IsSeasonPack:      parsed.IsSeasonPack,
		}

		if parsed.IsTV {
			resp.ParsedInfo.Season = parsed.Season
			resp.ParsedInfo.Episode = parsed.Episode
			resp.ParsedInfo.EndEpisode = parsed.EndEpisode
		}

		// Try to find a matching library item
		match, err := h.service.matchToLibrary(ctx, req.Path, nil)
		if err == nil && match != nil {
			suggested := &SuggestedMatch{
				MediaType:  match.MediaType,
				Confidence: match.Confidence,
			}

			if match.MediaType == "movie" && match.MovieID != nil {
				suggested.MediaID = *match.MovieID
				movie, err := h.service.movies.Get(ctx, *match.MovieID)
				if err == nil {
					suggested.MediaTitle = movie.Title
					suggested.Year = movie.Year
				}
			} else if match.MediaType == "episode" && match.EpisodeID != nil {
				suggested.MediaID = *match.EpisodeID
				suggested.SeriesID = match.SeriesID
				suggested.SeasonNum = match.SeasonNum

				if match.SeriesID != nil {
					series, err := h.service.tv.GetSeries(ctx, *match.SeriesID)
					if err == nil {
						suggested.SeriesTitle = &series.Title
					}
				}

				episode, err := h.service.tv.GetEpisode(ctx, *match.EpisodeID)
				if err == nil {
					suggested.MediaTitle = episode.Title
					epNum := episode.EpisodeNumber
					suggested.EpisodeNum = &epNum
				}
			}

			resp.SuggestedMatch = suggested
		}
	}

	return c.JSON(http.StatusOK, resp)
}

// RetryImport retries a failed import.
// POST /api/v1/import/:id/retry
func (h *Handlers) RetryImport(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	// Get the queue_media record
	qm, err := h.queries.GetQueueMedia(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "import record not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if qm.FileStatus != "failed" {
		return echo.NewHTTPError(http.StatusBadRequest, "can only retry failed imports")
	}

	if !qm.FilePath.Valid || qm.FilePath.String == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "no file path for this import")
	}

	// Reset the status and queue for reimport
	_, err = h.queries.UpdateQueueMediaStatus(ctx, sqlc.UpdateQueueMediaStatusParams{
		ID:         id,
		FileStatus: "pending",
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Build the job
	job := ImportJob{
		SourcePath: qm.FilePath.String,
		Manual:     false,
		QueueMedia: &QueueMedia{
			ID:                qm.ID,
			DownloadMappingID: qm.DownloadMappingID,
			FilePath:          qm.FilePath.String,
			FileStatus:        "pending",
			ImportAttempts:    int(qm.ImportAttempts),
		},
	}

	if qm.MovieID.Valid {
		job.QueueMedia.MovieID = &qm.MovieID.Int64
	}
	if qm.EpisodeID.Valid {
		job.QueueMedia.EpisodeID = &qm.EpisodeID.Int64
	}

	// Queue the import
	if err := h.service.QueueImport(job); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "import queued for retry",
	})
}

// ScanDirectoryRequest contains the request body for directory scanning.
type ScanDirectoryRequest struct {
	Path string `json:"path" validate:"required"`
}

// ScannedFile represents a video file found during scanning.
type ScannedFile struct {
	Path           string           `json:"path"`
	FileName       string           `json:"fileName"`
	FileSize       int64            `json:"fileSize"`
	Valid          bool             `json:"valid"`
	ValidationError string          `json:"validationError,omitempty"`
	ParsedInfo     *ParsedMediaInfo `json:"parsedInfo,omitempty"`
	SuggestedMatch *SuggestedMatch  `json:"suggestedMatch,omitempty"`
}

// ScanDirectoryResponse contains the response for a directory scan.
type ScanDirectoryResponse struct {
	Path   string        `json:"path"`
	Files  []ScannedFile `json:"files"`
	Total  int           `json:"total"`
	Valid  int           `json:"valid"`
}

// ScanDirectory scans a directory for importable video files.
// POST /api/v1/import/scan
func (h *Handlers) ScanDirectory(c echo.Context) error {
	ctx := c.Request().Context()

	var req ScanDirectoryRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Path == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "path is required")
	}

	// Find video files
	files, err := h.service.findVideoFiles(req.Path)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resp := ScanDirectoryResponse{
		Path:  req.Path,
		Files: make([]ScannedFile, 0, len(files)),
		Total: len(files),
	}

	for _, path := range files {
		file := ScannedFile{
			Path:     path,
			FileName: filepath.Base(path),
		}

		// Validate the file
		validation, err := h.service.ValidateForImport(ctx, path)
		if err != nil {
			file.Valid = false
			file.ValidationError = err.Error()
		} else {
			file.FileSize = validation.FileSize
			file.Valid = validation.Valid
			if !validation.Valid {
				file.ValidationError = validation.Reason
			}
		}

		if file.Valid {
			resp.Valid++
		}

		// Always parse filename and attempt matching (even for invalid files)
		// This lets users see potential matches before having valid files
		parsed := parseFilename(filepath.Base(path))
		if parsed != nil {
			file.ParsedInfo = &ParsedMediaInfo{
				Title:             parsed.Title,
				Year:              parsed.Year,
				Quality:           parsed.Quality,
				Source:            parsed.Source,
				Codec:             parsed.Codec,
				AudioCodecs:       parsed.AudioCodecs,
				AudioChannels:     parsed.AudioChannels,
				AudioEnhancements: parsed.AudioEnhancements,
				Attributes:        parsed.Attributes,
				IsTV:              parsed.IsTV,
				IsSeasonPack:      parsed.IsSeasonPack,
			}

			if parsed.IsTV {
				file.ParsedInfo.Season = parsed.Season
				file.ParsedInfo.Episode = parsed.Episode
				file.ParsedInfo.EndEpisode = parsed.EndEpisode
			}

			// Try to find a matching library item
			match, err := h.service.matchToLibrary(ctx, path, nil)
			if err == nil && match != nil {
				suggested := &SuggestedMatch{
					MediaType:  match.MediaType,
					Confidence: match.Confidence,
				}

				if match.MediaType == "movie" && match.MovieID != nil {
					suggested.MediaID = *match.MovieID
					movie, err := h.service.movies.Get(ctx, *match.MovieID)
					if err == nil {
						suggested.MediaTitle = movie.Title
						suggested.Year = movie.Year
					}
				} else if match.MediaType == "episode" && match.EpisodeID != nil {
					suggested.MediaID = *match.EpisodeID
					suggested.SeriesID = match.SeriesID
					suggested.SeasonNum = match.SeasonNum

					if match.SeriesID != nil {
						series, err := h.service.tv.GetSeries(ctx, *match.SeriesID)
						if err == nil {
							suggested.SeriesTitle = &series.Title
						}
					}

					episode, err := h.service.tv.GetEpisode(ctx, *match.EpisodeID)
					if err == nil {
						suggested.MediaTitle = episode.Title
						epNum := episode.EpisodeNumber
						suggested.EpisodeNum = &epNum
					}
				}

				file.SuggestedMatch = suggested
			}
		}

		resp.Files = append(resp.Files, file)
	}

	return c.JSON(http.StatusOK, resp)
}

// RenamePreviewRequest contains query parameters for rename preview.
type RenamePreviewRequest struct {
	MediaType string `query:"type"`    // "series" or "movie"
	MediaID   *int64 `query:"mediaId"` // Optional specific series/movie ID
}

// GetRenamePreview returns a preview of files that would be renamed.
// GET /api/v1/import/rename/preview?type=series&mediaId=123
func (h *Handlers) GetRenamePreview(c echo.Context) error {
	ctx := c.Request().Context()

	mediaType := c.QueryParam("type")
	if mediaType == "" {
		mediaType = "series" // Default to series
	}

	var mediaID *int64
	if idStr := c.QueryParam("mediaId"); idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid mediaId")
		}
		mediaID = &id
	}

	var previews []RenamePreview
	var err error

	switch mediaType {
	case "series", "episode":
		previews, err = h.service.GetRenamePreviewSeries(ctx, mediaID)
	case "movie":
		previews, err = h.service.GetRenamePreviewMovies(ctx, mediaID)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "type must be 'series' or 'movie'")
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Filter to only items that need renaming if requested
	needsRenameOnly := c.QueryParam("needsRename") == "true"
	if needsRenameOnly {
		filtered := make([]RenamePreview, 0)
		for _, p := range previews {
			if p.NeedsRename {
				filtered = append(filtered, p)
			}
		}
		previews = filtered
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"total":    len(previews),
		"previews": previews,
	})
}

// ExecuteRenameRequest contains the request body for executing renames.
type ExecuteRenameRequest struct {
	MediaType string  `json:"mediaType" validate:"required,oneof=series movie episode"`
	FileIDs   []int64 `json:"fileIds" validate:"required"`
}

// ExecuteRename performs mass rename operations.
// POST /api/v1/import/rename/execute
func (h *Handlers) ExecuteRename(c echo.Context) error {
	ctx := c.Request().Context()

	var req ExecuteRenameRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.MediaType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "mediaType is required")
	}

	if len(req.FileIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "fileIds is required and cannot be empty")
	}

	result, err := h.service.ExecuteMassRename(ctx, req.MediaType, req.FileIDs)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Broadcast rename progress via WebSocket
	if h.service.hub != nil {
		h.service.hub.Broadcast("rename:completed", map[string]interface{}{
			"total":     result.Total,
			"succeeded": result.Succeeded,
			"failed":    result.Failed,
			"skipped":   result.Skipped,
		})
	}

	return c.JSON(http.StatusOK, result)
}

// SettingsHandlers provides HTTP handlers for import settings.
type SettingsHandlers struct {
	queries *sqlc.Queries
	service *Service
}

// NewSettingsHandlers creates a new settings handlers instance.
func NewSettingsHandlers(db *sql.DB, service *Service) *SettingsHandlers {
	return &SettingsHandlers{
		queries: sqlc.New(db),
		service: service,
	}
}

// RegisterSettingsRoutes registers import settings routes on an Echo group.
func (h *SettingsHandlers) RegisterSettingsRoutes(g *echo.Group) {
	g.GET("/import", h.GetSettings)
	g.PUT("/import", h.UpdateSettings)
	g.POST("/import/naming/preview", h.PreviewNamingPattern)
	g.POST("/import/naming/validate", h.ValidateNamingPattern)
	g.POST("/import/naming/parse", h.ParseFilename)
}

// ImportSettingsResponse represents the import settings API response.
type ImportSettingsResponse struct {
	// Validation settings
	ValidationLevel    string   `json:"validationLevel"`
	MinimumFileSizeMB  int64    `json:"minimumFileSizeMB"`
	VideoExtensions    []string `json:"videoExtensions"`

	// Matching settings
	MatchConflictBehavior string `json:"matchConflictBehavior"`
	UnknownMediaBehavior  string `json:"unknownMediaBehavior"`

	// TV naming settings
	RenameEpisodes           bool   `json:"renameEpisodes"`
	ReplaceIllegalCharacters bool   `json:"replaceIllegalCharacters"`
	ColonReplacement         string `json:"colonReplacement"`
	CustomColonReplacement   string `json:"customColonReplacement,omitempty"`

	// Episode format patterns
	StandardEpisodeFormat string `json:"standardEpisodeFormat"`
	DailyEpisodeFormat    string `json:"dailyEpisodeFormat"`
	AnimeEpisodeFormat    string `json:"animeEpisodeFormat"`

	// Folder patterns
	SeriesFolderFormat   string `json:"seriesFolderFormat"`
	SeasonFolderFormat   string `json:"seasonFolderFormat"`
	SpecialsFolderFormat string `json:"specialsFolderFormat"`

	// Multi-episode
	MultiEpisodeStyle string `json:"multiEpisodeStyle"`

	// Movie naming settings
	RenameMovies      bool   `json:"renameMovies"`
	MovieFolderFormat string `json:"movieFolderFormat"`
	MovieFileFormat   string `json:"movieFileFormat"`
}

// GetSettings returns the current import settings.
// GET /api/v1/settings/import
func (h *SettingsHandlers) GetSettings(c echo.Context) error {
	ctx := c.Request().Context()

	// Ensure settings exist
	if err := h.queries.EnsureImportSettingsExist(ctx); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	row, err := h.queries.GetImportSettings(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resp := ImportSettingsResponse{
		ValidationLevel:          row.ValidationLevel,
		MinimumFileSizeMB:        row.MinimumFileSizeMb,
		VideoExtensions:          strings.Split(row.VideoExtensions, ","),
		MatchConflictBehavior:    row.MatchConflictBehavior,
		UnknownMediaBehavior:     row.UnknownMediaBehavior,
		RenameEpisodes:           row.RenameEpisodes,
		ReplaceIllegalCharacters: row.ReplaceIllegalCharacters,
		ColonReplacement:         row.ColonReplacement,
		StandardEpisodeFormat:    row.StandardEpisodeFormat,
		DailyEpisodeFormat:       row.DailyEpisodeFormat,
		AnimeEpisodeFormat:       row.AnimeEpisodeFormat,
		SeriesFolderFormat:       row.SeriesFolderFormat,
		SeasonFolderFormat:       row.SeasonFolderFormat,
		SpecialsFolderFormat:     row.SpecialsFolderFormat,
		MultiEpisodeStyle:        row.MultiEpisodeStyle,
		RenameMovies:             row.RenameMovies,
		MovieFolderFormat:        row.MovieFolderFormat,
		MovieFileFormat:          row.MovieFileFormat,
	}

	if row.CustomColonReplacement.Valid {
		resp.CustomColonReplacement = row.CustomColonReplacement.String
	}

	return c.JSON(http.StatusOK, resp)
}

// UpdateSettingsRequest contains fields to update.
type UpdateSettingsRequest struct {
	// Validation settings
	ValidationLevel   *string  `json:"validationLevel,omitempty"`
	MinimumFileSizeMB *int64   `json:"minimumFileSizeMB,omitempty"`
	VideoExtensions   []string `json:"videoExtensions,omitempty"`

	// Matching settings
	MatchConflictBehavior *string `json:"matchConflictBehavior,omitempty"`
	UnknownMediaBehavior  *string `json:"unknownMediaBehavior,omitempty"`

	// TV naming settings
	RenameEpisodes           *bool   `json:"renameEpisodes,omitempty"`
	ReplaceIllegalCharacters *bool   `json:"replaceIllegalCharacters,omitempty"`
	ColonReplacement         *string `json:"colonReplacement,omitempty"`
	CustomColonReplacement   *string `json:"customColonReplacement,omitempty"`

	// Episode format patterns
	StandardEpisodeFormat *string `json:"standardEpisodeFormat,omitempty"`
	DailyEpisodeFormat    *string `json:"dailyEpisodeFormat,omitempty"`
	AnimeEpisodeFormat    *string `json:"animeEpisodeFormat,omitempty"`

	// Folder patterns
	SeriesFolderFormat   *string `json:"seriesFolderFormat,omitempty"`
	SeasonFolderFormat   *string `json:"seasonFolderFormat,omitempty"`
	SpecialsFolderFormat *string `json:"specialsFolderFormat,omitempty"`

	// Multi-episode
	MultiEpisodeStyle *string `json:"multiEpisodeStyle,omitempty"`

	// Movie naming settings
	RenameMovies      *bool   `json:"renameMovies,omitempty"`
	MovieFolderFormat *string `json:"movieFolderFormat,omitempty"`
	MovieFileFormat   *string `json:"movieFileFormat,omitempty"`
}

// UpdateSettings updates import settings.
// PUT /api/v1/settings/import
func (h *SettingsHandlers) UpdateSettings(c echo.Context) error {
	ctx := c.Request().Context()

	var req UpdateSettingsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Ensure settings exist
	if err := h.queries.EnsureImportSettingsExist(ctx); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Get current settings
	current, err := h.queries.GetImportSettings(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Build update params merging current with requested changes
	params := sqlc.UpdateImportSettingsParams{
		ValidationLevel:          current.ValidationLevel,
		MinimumFileSizeMb:        current.MinimumFileSizeMb,
		VideoExtensions:          current.VideoExtensions,
		MatchConflictBehavior:    current.MatchConflictBehavior,
		UnknownMediaBehavior:     current.UnknownMediaBehavior,
		RenameEpisodes:           current.RenameEpisodes,
		ReplaceIllegalCharacters: current.ReplaceIllegalCharacters,
		ColonReplacement:         current.ColonReplacement,
		CustomColonReplacement:   current.CustomColonReplacement,
		StandardEpisodeFormat:    current.StandardEpisodeFormat,
		DailyEpisodeFormat:       current.DailyEpisodeFormat,
		AnimeEpisodeFormat:       current.AnimeEpisodeFormat,
		SeriesFolderFormat:       current.SeriesFolderFormat,
		SeasonFolderFormat:       current.SeasonFolderFormat,
		SpecialsFolderFormat:     current.SpecialsFolderFormat,
		MultiEpisodeStyle:        current.MultiEpisodeStyle,
		RenameMovies:             current.RenameMovies,
		MovieFolderFormat:        current.MovieFolderFormat,
		MovieFileFormat:          current.MovieFileFormat,
	}

	// Apply updates
	if req.ValidationLevel != nil {
		params.ValidationLevel = *req.ValidationLevel
	}
	if req.MinimumFileSizeMB != nil {
		params.MinimumFileSizeMb = *req.MinimumFileSizeMB
	}
	if req.VideoExtensions != nil {
		params.VideoExtensions = strings.Join(req.VideoExtensions, ",")
	}
	if req.MatchConflictBehavior != nil {
		params.MatchConflictBehavior = *req.MatchConflictBehavior
	}
	if req.UnknownMediaBehavior != nil {
		params.UnknownMediaBehavior = *req.UnknownMediaBehavior
	}
	if req.RenameEpisodes != nil {
		params.RenameEpisodes = *req.RenameEpisodes
	}
	if req.ReplaceIllegalCharacters != nil {
		params.ReplaceIllegalCharacters = *req.ReplaceIllegalCharacters
	}
	if req.ColonReplacement != nil {
		params.ColonReplacement = *req.ColonReplacement
	}
	if req.CustomColonReplacement != nil {
		params.CustomColonReplacement = sql.NullString{String: *req.CustomColonReplacement, Valid: true}
	}
	if req.StandardEpisodeFormat != nil {
		params.StandardEpisodeFormat = *req.StandardEpisodeFormat
	}
	if req.DailyEpisodeFormat != nil {
		params.DailyEpisodeFormat = *req.DailyEpisodeFormat
	}
	if req.AnimeEpisodeFormat != nil {
		params.AnimeEpisodeFormat = *req.AnimeEpisodeFormat
	}
	if req.SeriesFolderFormat != nil {
		params.SeriesFolderFormat = *req.SeriesFolderFormat
	}
	if req.SeasonFolderFormat != nil {
		params.SeasonFolderFormat = *req.SeasonFolderFormat
	}
	if req.SpecialsFolderFormat != nil {
		params.SpecialsFolderFormat = *req.SpecialsFolderFormat
	}
	if req.MultiEpisodeStyle != nil {
		params.MultiEpisodeStyle = *req.MultiEpisodeStyle
	}
	if req.RenameMovies != nil {
		params.RenameMovies = *req.RenameMovies
	}
	if req.MovieFolderFormat != nil {
		params.MovieFolderFormat = *req.MovieFolderFormat
	}
	if req.MovieFileFormat != nil {
		params.MovieFileFormat = *req.MovieFileFormat
	}

	// Update in database
	updated, err := h.queries.UpdateImportSettings(ctx, params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Update the renamer in the import service
	if h.service != nil {
		h.service.UpdateRenamerSettings(renamer.Settings{
			RenameEpisodes:           updated.RenameEpisodes,
			ReplaceIllegalCharacters: updated.ReplaceIllegalCharacters,
			ColonReplacement:         renamer.ColonReplacement(updated.ColonReplacement),
			CustomColonReplacement:   updated.CustomColonReplacement.String,
			StandardEpisodeFormat:    updated.StandardEpisodeFormat,
			DailyEpisodeFormat:       updated.DailyEpisodeFormat,
			AnimeEpisodeFormat:       updated.AnimeEpisodeFormat,
			SeriesFolderFormat:       updated.SeriesFolderFormat,
			SeasonFolderFormat:       updated.SeasonFolderFormat,
			SpecialsFolderFormat:     updated.SpecialsFolderFormat,
			MultiEpisodeStyle:        renamer.MultiEpisodeStyle(updated.MultiEpisodeStyle),
			RenameMovies:             updated.RenameMovies,
			MovieFolderFormat:        updated.MovieFolderFormat,
			MovieFileFormat:          updated.MovieFileFormat,
		})
	}

	// Return updated settings
	resp := ImportSettingsResponse{
		ValidationLevel:          updated.ValidationLevel,
		MinimumFileSizeMB:        updated.MinimumFileSizeMb,
		VideoExtensions:          strings.Split(updated.VideoExtensions, ","),
		MatchConflictBehavior:    updated.MatchConflictBehavior,
		UnknownMediaBehavior:     updated.UnknownMediaBehavior,
		RenameEpisodes:           updated.RenameEpisodes,
		ReplaceIllegalCharacters: updated.ReplaceIllegalCharacters,
		ColonReplacement:         updated.ColonReplacement,
		StandardEpisodeFormat:    updated.StandardEpisodeFormat,
		DailyEpisodeFormat:       updated.DailyEpisodeFormat,
		AnimeEpisodeFormat:       updated.AnimeEpisodeFormat,
		SeriesFolderFormat:       updated.SeriesFolderFormat,
		SeasonFolderFormat:       updated.SeasonFolderFormat,
		SpecialsFolderFormat:     updated.SpecialsFolderFormat,
		MultiEpisodeStyle:        updated.MultiEpisodeStyle,
		RenameMovies:             updated.RenameMovies,
		MovieFolderFormat:        updated.MovieFolderFormat,
		MovieFileFormat:          updated.MovieFileFormat,
	}

	if updated.CustomColonReplacement.Valid {
		resp.CustomColonReplacement = updated.CustomColonReplacement.String
	}

	return c.JSON(http.StatusOK, resp)
}

// PatternPreviewRequest contains the request body for pattern preview.
type PatternPreviewRequest struct {
	Pattern   string `json:"pattern" validate:"required"`
	MediaType string `json:"mediaType" validate:"oneof=episode movie folder"`
}

// PatternPreviewResponse contains the response for pattern preview.
type PatternPreviewResponse struct {
	Pattern  string             `json:"pattern"`
	Preview  string             `json:"preview"`
	Valid    bool               `json:"valid"`
	Error    string             `json:"error,omitempty"`
	Tokens   []TokenBreakdown   `json:"tokens,omitempty"`
}

// TokenBreakdown provides detailed token info for debugging.
type TokenBreakdown struct {
	Token    string `json:"token"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	Empty    bool   `json:"empty"`
	Modified bool   `json:"modified"`
}

// PreviewNamingPattern previews a naming pattern with sample data.
// POST /api/v1/settings/import/naming/preview
func (h *SettingsHandlers) PreviewNamingPattern(c echo.Context) error {
	var req PatternPreviewRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Pattern == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "pattern is required")
	}

	resolver := renamer.NewResolver(renamer.DefaultSettings())
	sampleCtx := renamer.GetSampleContext()

	resp := PatternPreviewResponse{
		Pattern: req.Pattern,
	}

	// Validate first
	if err := resolver.ValidatePattern(req.Pattern); err != nil {
		resp.Valid = false
		resp.Error = err.Error()
		return c.JSON(http.StatusOK, resp)
	}

	// Generate preview
	preview, err := resolver.PreviewPattern(req.Pattern, sampleCtx)
	if err != nil {
		resp.Valid = false
		resp.Error = err.Error()
		return c.JSON(http.StatusOK, resp)
	}

	resp.Valid = true
	resp.Preview = preview

	// Get token breakdown for debugging
	breakdown := resolver.GetTokenBreakdown(req.Pattern, sampleCtx)
	resp.Tokens = make([]TokenBreakdown, len(breakdown))
	for i, b := range breakdown {
		resp.Tokens[i] = TokenBreakdown{
			Token:    b.Token,
			Name:     b.Name,
			Value:    b.Value,
			Empty:    b.Empty,
			Modified: b.Modified,
		}
	}

	return c.JSON(http.StatusOK, resp)
}

// PatternValidateRequest contains the request body for pattern validation.
type PatternValidateRequest struct {
	Pattern string `json:"pattern" validate:"required"`
}

// PatternValidateResponse contains the response for pattern validation.
type PatternValidateResponse struct {
	Pattern string   `json:"pattern"`
	Valid   bool     `json:"valid"`
	Error   string   `json:"error,omitempty"`
	Tokens  []string `json:"tokens,omitempty"`
}

// ValidateNamingPattern validates a naming pattern syntax.
// POST /api/v1/settings/import/naming/validate
func (h *SettingsHandlers) ValidateNamingPattern(c echo.Context) error {
	var req PatternValidateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Pattern == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "pattern is required")
	}

	resolver := renamer.NewResolver(renamer.DefaultSettings())

	resp := PatternValidateResponse{
		Pattern: req.Pattern,
	}

	if err := resolver.ValidatePattern(req.Pattern); err != nil {
		resp.Valid = false
		resp.Error = err.Error()
	} else {
		resp.Valid = true
		// Extract token names for reference
		tokens := renamer.ParseTokens(req.Pattern)
		resp.Tokens = make([]string, len(tokens))
		for i, t := range tokens {
			resp.Tokens[i] = t.Name
		}
	}

	return c.JSON(http.StatusOK, resp)
}

// ParseFilenameRequest contains the request body for filename parsing.
type ParseFilenameRequest struct {
	Filename string `json:"filename" validate:"required"`
}

// ParseFilenameResponse contains the response for filename parsing.
type ParseFilenameResponse struct {
	Filename   string              `json:"filename"`
	ParsedInfo *ParsedMediaInfo    `json:"parsedInfo,omitempty"`
	Tokens     []ParsedTokenDetail `json:"tokens"`
}

// ParsedTokenDetail provides detailed breakdown of a parsed token.
type ParsedTokenDetail struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Raw   string `json:"raw,omitempty"`
}

// ParseFilename parses a filename and returns extracted metadata tokens.
// POST /api/v1/settings/import/naming/parse
func (h *SettingsHandlers) ParseFilename(c echo.Context) error {
	var req ParseFilenameRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Filename == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "filename is required")
	}

	resp := ParseFilenameResponse{
		Filename: req.Filename,
		Tokens:   []ParsedTokenDetail{},
	}

	parsed := parseFilename(req.Filename)
	if parsed == nil {
		return c.JSON(http.StatusOK, resp)
	}

	resp.ParsedInfo = &ParsedMediaInfo{
		Title:             parsed.Title,
		Year:              parsed.Year,
		Quality:           parsed.Quality,
		Source:            parsed.Source,
		Codec:             parsed.Codec,
		AudioCodecs:       parsed.AudioCodecs,
		AudioChannels:     parsed.AudioChannels,
		AudioEnhancements: parsed.AudioEnhancements,
		Attributes:        parsed.Attributes,
		IsTV:              parsed.IsTV,
		IsSeasonPack:      parsed.IsSeasonPack,
	}

	if parsed.IsTV {
		resp.ParsedInfo.Season = parsed.Season
		resp.ParsedInfo.Episode = parsed.Episode
		resp.ParsedInfo.EndEpisode = parsed.EndEpisode
	}

	// Build token breakdown for display
	if parsed.Title != "" {
		resp.Tokens = append(resp.Tokens, ParsedTokenDetail{
			Name:  "Title",
			Value: parsed.Title,
		})
	}

	if parsed.Year > 0 {
		resp.Tokens = append(resp.Tokens, ParsedTokenDetail{
			Name:  "Year",
			Value: strconv.Itoa(parsed.Year),
		})
	}

	if parsed.IsTV {
		if parsed.Season > 0 {
			resp.Tokens = append(resp.Tokens, ParsedTokenDetail{
				Name:  "Season",
				Value: strconv.Itoa(parsed.Season),
			})
		}
		if parsed.Episode > 0 {
			epValue := strconv.Itoa(parsed.Episode)
			if parsed.EndEpisode > 0 && parsed.EndEpisode != parsed.Episode {
				epValue += "-" + strconv.Itoa(parsed.EndEpisode)
			}
			resp.Tokens = append(resp.Tokens, ParsedTokenDetail{
				Name:  "Episode",
				Value: epValue,
			})
		}
		if parsed.IsSeasonPack {
			resp.Tokens = append(resp.Tokens, ParsedTokenDetail{
				Name:  "Type",
				Value: "Season Pack",
			})
		}
	}

	if parsed.Quality != "" {
		resp.Tokens = append(resp.Tokens, ParsedTokenDetail{
			Name:  "Quality",
			Value: parsed.Quality,
		})
	}

	if parsed.Source != "" {
		resp.Tokens = append(resp.Tokens, ParsedTokenDetail{
			Name:  "Source",
			Value: parsed.Source,
		})
	}

	if parsed.Codec != "" {
		resp.Tokens = append(resp.Tokens, ParsedTokenDetail{
			Name:  "Video Codec",
			Value: parsed.Codec,
		})
	}

	for _, attr := range parsed.Attributes {
		resp.Tokens = append(resp.Tokens, ParsedTokenDetail{
			Name:  "Video Attribute",
			Value: attr,
		})
	}

	for _, codec := range parsed.AudioCodecs {
		resp.Tokens = append(resp.Tokens, ParsedTokenDetail{
			Name:  "Audio Codec",
			Value: codec,
		})
	}

	for _, channels := range parsed.AudioChannels {
		resp.Tokens = append(resp.Tokens, ParsedTokenDetail{
			Name:  "Audio Channels",
			Value: channels,
		})
	}

	for _, enhancement := range parsed.AudioEnhancements {
		resp.Tokens = append(resp.Tokens, ParsedTokenDetail{
			Name:  "Audio Enhancement",
			Value: enhancement,
		})
	}

	return c.JSON(http.StatusOK, resp)
}
