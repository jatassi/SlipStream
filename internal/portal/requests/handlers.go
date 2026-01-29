package requests

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
	"github.com/slipstream/slipstream/internal/portal/users"
)

type CreateRequestInput struct {
	MediaType        string  `json:"mediaType"`
	TmdbID           *int64  `json:"tmdbId,omitempty"`
	TvdbID           *int64  `json:"tvdbId,omitempty"`
	Title            string  `json:"title"`
	Year             *int64  `json:"year,omitempty"`
	SeasonNumber     *int64  `json:"seasonNumber,omitempty"`
	EpisodeNumber    *int64  `json:"episodeNumber,omitempty"`
	MonitorType      *string `json:"monitorType,omitempty"`
	TargetSlotID     *int64  `json:"targetSlotId,omitempty"`
	PosterUrl        *string `json:"posterUrl,omitempty"`
	RequestedSeasons []int64 `json:"requestedSeasons,omitempty"`
}

type RequestUser struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type RequestWithWatchStatus struct {
	*Request
	User         *RequestUser `json:"user,omitempty"`
	IsWatching   bool         `json:"isWatching"`
	WatcherCount int64        `json:"watcherCount"`
}

type AutoApproveProcessor interface {
	ProcessAutoApprove(request *Request, user *users.User) error
}

type QueueItem struct {
	ID             string   `json:"id"`
	ClientID       int64    `json:"clientId"`
	ClientName     string   `json:"clientName"`
	Title          string   `json:"title"`
	MediaType      string   `json:"mediaType"`
	Status         string   `json:"status"`
	Progress       float64  `json:"progress"`
	Size           int64    `json:"size"`
	DownloadedSize int64    `json:"downloadedSize"`
	DownloadSpeed  int64    `json:"downloadSpeed"`
	ETA            int64    `json:"eta"`
	Season         int      `json:"season,omitempty"`
	Episode        int      `json:"episode,omitempty"`
	MovieID        *int64   `json:"movieId,omitempty"`
	SeriesID       *int64   `json:"seriesId,omitempty"`
	SeasonNumber   *int     `json:"seasonNumber,omitempty"`
	IsSeasonPack   bool     `json:"isSeasonPack,omitempty"`
}

type QueueGetter interface {
	GetQueue(ctx context.Context) ([]QueueItem, error)
}

type MediaLookup interface {
	GetMovieTmdbID(ctx context.Context, movieID int64) (*int64, error)
	GetSeriesTvdbID(ctx context.Context, seriesID int64) (*int64, error)
}

type Handlers struct {
	service         *Service
	watchersService *WatchersService
	usersService    *users.Service
	autoApprove     AutoApproveProcessor
	queueGetter     QueueGetter
	mediaLookup     MediaLookup
	logger          zerolog.Logger
}

func NewHandlers(
	service *Service,
	watchersService *WatchersService,
	usersService *users.Service,
	autoApprove AutoApproveProcessor,
	queueGetter QueueGetter,
	mediaLookup MediaLookup,
	logger zerolog.Logger,
) *Handlers {
	return &Handlers{
		service:         service,
		watchersService: watchersService,
		usersService:    usersService,
		autoApprove:     autoApprove,
		queueGetter:     queueGetter,
		mediaLookup:     mediaLookup,
		logger:          logger.With().Str("component", "portal-requests-handlers").Logger(),
	}
}

func (h *Handlers) RegisterRoutes(g *echo.Group, authMiddleware *portalmw.AuthMiddleware) {
	protected := g.Group("")
	protected.Use(authMiddleware.AnyAuth())

	protected.GET("", h.List)
	protected.POST("", h.Create)
	protected.GET("/downloads", h.Downloads)
	protected.GET("/:id", h.Get)
	protected.DELETE("/:id", h.Cancel)
	protected.POST("/:id/watch", h.Watch)
	protected.DELETE("/:id/watch", h.Unwatch)
}

// List returns requests for the authenticated user
// GET /api/v1/requests
func (h *Handlers) List(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	filters := ListFilters{
		// Always filter by the authenticated user - portal users should only see their own requests
		UserID: &claims.UserID,
	}

	if status := c.QueryParam("status"); status != "" {
		filters.Status = &status
	}
	if mediaType := c.QueryParam("mediaType"); mediaType != "" {
		filters.MediaType = &mediaType
	}

	requests, err := h.service.List(c.Request().Context(), filters)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Collect unique user IDs and fetch users
	userIDs := make(map[int64]bool)
	for _, req := range requests {
		userIDs[req.UserID] = true
	}

	userMap := make(map[int64]*RequestUser)
	for userID := range userIDs {
		if user, err := h.usersService.Get(c.Request().Context(), userID); err == nil && user != nil {
			userMap[userID] = &RequestUser{
				ID:          user.ID,
				Username:    user.Username,
				DisplayName: user.DisplayName,
			}
		}
	}

	results := make([]*RequestWithWatchStatus, len(requests))
	for i, req := range requests {
		isWatching, _ := h.watchersService.IsWatching(c.Request().Context(), req.ID, claims.UserID)
		watcherCount, _ := h.watchersService.CountWatchers(c.Request().Context(), req.ID)
		results[i] = &RequestWithWatchStatus{
			Request:      req,
			User:         userMap[req.UserID],
			IsWatching:   isWatching,
			WatcherCount: watcherCount,
		}
	}

	return c.JSON(http.StatusOK, results)
}

// Create creates a new request
// POST /api/v1/requests
func (h *Handlers) Create(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var input CreateRequestInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if input.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if input.MediaType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "mediaType is required")
	}

	request, err := h.service.Create(c.Request().Context(), claims.UserID, CreateInput{
		MediaType:        input.MediaType,
		TmdbID:           input.TmdbID,
		TvdbID:           input.TvdbID,
		Title:            input.Title,
		Year:             input.Year,
		SeasonNumber:     input.SeasonNumber,
		EpisodeNumber:    input.EpisodeNumber,
		MonitorType:      input.MonitorType,
		TargetSlotID:     input.TargetSlotID,
		PosterUrl:        input.PosterUrl,
		RequestedSeasons: input.RequestedSeasons,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrAlreadyRequested):
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		case errors.Is(err, ErrInvalidMediaType):
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	if h.autoApprove != nil {
		user, err := h.usersService.Get(c.Request().Context(), claims.UserID)
		if err != nil {
			h.logger.Warn().Err(err).Int64("userID", claims.UserID).Msg("failed to get user for auto-approve")
		} else if user != nil {
			if err := h.autoApprove.ProcessAutoApprove(request, user); err != nil {
				h.logger.Warn().Err(err).Int64("requestID", request.ID).Msg("auto-approve processing failed")
			} else {
				// Re-fetch request to get updated status after auto-approve
				if updated, err := h.service.Get(c.Request().Context(), request.ID); err == nil {
					request = updated
				}
			}
		}
	}

	return c.JSON(http.StatusCreated, request)
}

// Get returns a single request
// GET /api/v1/requests/:id
func (h *Handlers) Get(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	request, err := h.service.Get(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrRequestNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	isWatching, _ := h.watchersService.IsWatching(c.Request().Context(), request.ID, claims.UserID)
	watcherCount, _ := h.watchersService.CountWatchers(c.Request().Context(), request.ID)

	var reqUser *RequestUser
	if user, err := h.usersService.Get(c.Request().Context(), request.UserID); err == nil && user != nil {
		reqUser = &RequestUser{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
		}
	}

	return c.JSON(http.StatusOK, &RequestWithWatchStatus{
		Request:      request,
		User:         reqUser,
		IsWatching:   isWatching,
		WatcherCount: watcherCount,
	})
}

// Cancel cancels a pending request
// DELETE /api/v1/requests/:id
func (h *Handlers) Cancel(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	err = h.service.Cancel(c.Request().Context(), id, claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, ErrRequestNotFound):
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		case errors.Is(err, ErrNotOwner):
			return echo.NewHTTPError(http.StatusForbidden, err.Error())
		case errors.Is(err, ErrCannotCancel):
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// Watch adds user as watcher to a request
// POST /api/v1/requests/:id/watch
func (h *Handlers) Watch(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	// Verify user exists in database (handles dev/prod database mismatch after mode switch)
	if _, err := h.usersService.Get(c.Request().Context(), claims.UserID); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not found - please log in again")
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	_, err = h.service.Get(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrRequestNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	watcher, err := h.watchersService.Watch(c.Request().Context(), id, claims.UserID)
	if err != nil {
		if errors.Is(err, ErrAlreadyWatching) {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, watcher)
}

// Unwatch removes user as watcher from a request
// DELETE /api/v1/requests/:id/watch
func (h *Handlers) Unwatch(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	// Verify user exists in database (handles dev/prod database mismatch after mode switch)
	if _, err := h.usersService.Get(c.Request().Context(), claims.UserID); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not found - please log in again")
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	err = h.watchersService.Unwatch(c.Request().Context(), id, claims.UserID)
	if err != nil {
		if errors.Is(err, ErrNotWatching) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// PortalDownload represents a download item with request context
type PortalDownload struct {
	QueueItem
	RequestID      int64  `json:"requestId"`
	RequestTitle   string `json:"requestTitle"`
	RequestMediaID *int64 `json:"requestMediaId,omitempty"`
}

// Downloads returns active downloads for the user's requests
// GET /api/v1/requests/downloads
func (h *Handlers) Downloads(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	if h.queueGetter == nil {
		h.logger.Warn().Msg("queueGetter is nil")
		return c.JSON(http.StatusOK, []PortalDownload{})
	}

	ctx := c.Request().Context()

	userRequests, err := h.service.ListByUser(ctx, claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	type requestInfo struct {
		id        int64
		title     string
		mediaID   *int64
		mediaType string
	}

	// Build maps for requests with MediaID (internal library ID)
	requestsByMovieID := make(map[int64]requestInfo)
	requestsBySeriesID := make(map[int64]requestInfo)
	// Build maps for requests by external IDs (tmdbID/tvdbID) for requests without MediaID
	requestsByTmdbID := make(map[int64]requestInfo)
	requestsByTvdbID := make(map[int64]requestInfo)

	for _, req := range userRequests {
		info := requestInfo{
			id:        req.ID,
			title:     req.Title,
			mediaID:   req.MediaID,
			mediaType: req.MediaType,
		}

		if req.MediaID != nil {
			if req.MediaType == MediaTypeMovie {
				requestsByMovieID[*req.MediaID] = info
			} else if req.MediaType == MediaTypeSeries || req.MediaType == MediaTypeSeason {
				requestsBySeriesID[*req.MediaID] = info
			}
		} else {
			if req.MediaType == MediaTypeMovie && req.TmdbID != nil {
				requestsByTmdbID[*req.TmdbID] = info
			} else if (req.MediaType == MediaTypeSeries || req.MediaType == MediaTypeSeason) && req.TvdbID != nil {
				requestsByTvdbID[*req.TvdbID] = info
			}
		}
	}

	queue, err := h.queueGetter.GetQueue(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var downloads []PortalDownload
	for _, item := range queue {
		var reqInfo requestInfo
		var found bool

		// First, try to match by internal MediaID
		if item.MovieID != nil {
			reqInfo, found = requestsByMovieID[*item.MovieID]
		} else if item.SeriesID != nil {
			reqInfo, found = requestsBySeriesID[*item.SeriesID]
		}

		// If not found and mediaLookup is available, try matching by external IDs
		if !found && h.mediaLookup != nil {
			if item.MovieID != nil {
				if tmdbID, err := h.mediaLookup.GetMovieTmdbID(ctx, *item.MovieID); err == nil && tmdbID != nil {
					reqInfo, found = requestsByTmdbID[*tmdbID]
				}
			} else if item.SeriesID != nil {
				if tvdbID, err := h.mediaLookup.GetSeriesTvdbID(ctx, *item.SeriesID); err == nil && tvdbID != nil {
					reqInfo, found = requestsByTvdbID[*tvdbID]
				}
			}
		}

		if found {
			downloads = append(downloads, PortalDownload{
				QueueItem:      item,
				RequestID:      reqInfo.id,
				RequestTitle:   reqInfo.title,
				RequestMediaID: reqInfo.mediaID,
			})
		}
	}

	if downloads == nil {
		downloads = []PortalDownload{}
	}

	h.logger.Debug().Int("requestCount", len(userRequests)).Int("queueSize", len(queue)).Int("matched", len(downloads)).Msg("downloads matched")

	return c.JSON(http.StatusOK, downloads)
}
