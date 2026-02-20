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
	PosterURL        *string `json:"posterUrl,omitempty"`
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
	ID             string  `json:"id"`
	ClientID       int64   `json:"clientId"`
	ClientName     string  `json:"clientName"`
	Title          string  `json:"title"`
	MediaType      string  `json:"mediaType"`
	Status         string  `json:"status"`
	Progress       float64 `json:"progress"`
	Size           int64   `json:"size"`
	DownloadedSize int64   `json:"downloadedSize"`
	DownloadSpeed  int64   `json:"downloadSpeed"`
	ETA            int64   `json:"eta"`
	Season         int     `json:"season"`
	Episode        int     `json:"episode"`
	MovieID        *int64  `json:"movieId,omitempty"`
	SeriesID       *int64  `json:"seriesId,omitempty"`
	SeasonNumber   *int    `json:"seasonNumber,omitempty"`
	IsSeasonPack   bool    `json:"isSeasonPack"`
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
	logger          *zerolog.Logger
}

func NewHandlers(
	service *Service,
	watchersService *WatchersService,
	usersService *users.Service,
	autoApprove AutoApproveProcessor,
	queueGetter QueueGetter,
	mediaLookup MediaLookup,
	logger *zerolog.Logger,
) *Handlers {
	subLogger := logger.With().Str("component", "portal-requests-handlers").Logger()
	return &Handlers{
		service:         service,
		watchersService: watchersService,
		usersService:    usersService,
		autoApprove:     autoApprove,
		queueGetter:     queueGetter,
		mediaLookup:     mediaLookup,
		logger:          &subLogger,
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

	filters := h.buildListFilters(c, claims.UserID)

	requests, err := h.service.List(c.Request().Context(), filters)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	userMap := h.fetchRequestUsers(c.Request().Context(), requests)
	results := h.enrichRequestsWithWatchStatus(c.Request().Context(), requests, userMap, claims.UserID)

	return c.JSON(http.StatusOK, results)
}

func (h *Handlers) buildListFilters(c echo.Context, userID int64) ListFilters {
	filters := ListFilters{}

	scope := c.QueryParam("scope")
	if scope != "all" {
		filters.UserID = &userID
	}

	if status := c.QueryParam("status"); status != "" {
		filters.Status = &status
	}
	if mediaType := c.QueryParam("mediaType"); mediaType != "" {
		filters.MediaType = &mediaType
	}

	return filters
}

func (h *Handlers) fetchRequestUsers(ctx context.Context, requests []*Request) map[int64]*RequestUser {
	userIDs := make(map[int64]bool)
	for _, req := range requests {
		userIDs[req.UserID] = true
	}

	userMap := make(map[int64]*RequestUser)
	for userID := range userIDs {
		if user, err := h.usersService.Get(ctx, userID); err == nil && user != nil {
			userMap[userID] = &RequestUser{
				ID:          user.ID,
				Username:    user.Username,
				DisplayName: user.DisplayName,
			}
		}
	}

	return userMap
}

func (h *Handlers) enrichRequestsWithWatchStatus(ctx context.Context, requests []*Request, userMap map[int64]*RequestUser, currentUserID int64) []*RequestWithWatchStatus {
	results := make([]*RequestWithWatchStatus, len(requests))
	for i, req := range requests {
		isWatching, _ := h.watchersService.IsWatching(ctx, req.ID, currentUserID)
		watcherCount, _ := h.watchersService.CountWatchers(ctx, req.ID)
		results[i] = &RequestWithWatchStatus{
			Request:      req,
			User:         userMap[req.UserID],
			IsWatching:   isWatching,
			WatcherCount: watcherCount,
		}
	}
	return results
}

// Create creates a new request
// POST /api/v1/requests
func (h *Handlers) Create(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	input, err := h.parseAndValidateCreateInput(c)
	if err != nil {
		return err
	}

	request, err := h.createRequest(c.Request().Context(), claims.UserID, input)
	if err != nil {
		return err
	}

	h.processAutoApprove(c.Request().Context(), claims.UserID, request)

	return c.JSON(http.StatusCreated, request)
}

func (h *Handlers) parseAndValidateCreateInput(c echo.Context) (*CreateInput, error) {
	var input CreateRequestInput
	if err := c.Bind(&input); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if input.Title == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if input.MediaType == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "mediaType is required")
	}

	createInput := CreateInput(input)
	return &createInput, nil
}

func (h *Handlers) createRequest(ctx context.Context, userID int64, input *CreateInput) (*Request, error) {
	request, err := h.service.Create(ctx, userID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrAlreadyRequested):
			return nil, echo.NewHTTPError(http.StatusConflict, err.Error())
		case errors.Is(err, ErrInvalidMediaType):
			return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
		default:
			return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}
	return request, nil
}

func (h *Handlers) processAutoApprove(ctx context.Context, userID int64, request *Request) {
	if h.autoApprove == nil {
		return
	}

	user, err := h.usersService.Get(ctx, userID)
	if err != nil {
		h.logger.Warn().Err(err).Int64("userID", userID).Msg("failed to get user for auto-approve")
		return
	}
	if user == nil {
		return
	}

	if err := h.autoApprove.ProcessAutoApprove(request, user); err != nil {
		h.logger.Warn().Err(err).Int64("requestID", request.ID).Msg("auto-approve processing failed")
		return
	}

	if updated, err := h.service.Get(ctx, request.ID); err == nil {
		*request = *updated
	}
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

	requestMaps := h.buildRequestMaps(userRequests)

	queue, err := h.queueGetter.GetQueue(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	downloads := h.matchDownloadsToRequests(ctx, queue, requestMaps)

	h.logger.Debug().Int("requestCount", len(userRequests)).Int("queueSize", len(queue)).Int("matched", len(downloads)).Msg("downloads matched")

	return c.JSON(http.StatusOK, downloads)
}

type requestInfo struct {
	id        int64
	title     string
	mediaID   *int64
	mediaType string
}

type requestMaps struct {
	byMovieID  map[int64]requestInfo
	bySeriesID map[int64]requestInfo
	byTmdbID   map[int64]requestInfo
	byTvdbID   map[int64]requestInfo
}

func (h *Handlers) buildRequestMaps(userRequests []*Request) *requestMaps {
	maps := &requestMaps{
		byMovieID:  make(map[int64]requestInfo),
		bySeriesID: make(map[int64]requestInfo),
		byTmdbID:   make(map[int64]requestInfo),
		byTvdbID:   make(map[int64]requestInfo),
	}

	for _, req := range userRequests {
		info := requestInfo{
			id:        req.ID,
			title:     req.Title,
			mediaID:   req.MediaID,
			mediaType: req.MediaType,
		}

		if req.MediaID != nil {
			h.indexRequestByMediaID(maps, req.MediaType, *req.MediaID, info)
		} else {
			h.indexRequestByExternalID(maps, req, info)
		}
	}

	return maps
}

func (h *Handlers) indexRequestByMediaID(maps *requestMaps, mediaType string, mediaID int64, info requestInfo) {
	switch mediaType {
	case MediaTypeMovie:
		maps.byMovieID[mediaID] = info
	case MediaTypeSeries, MediaTypeSeason:
		maps.bySeriesID[mediaID] = info
	}
}

func (h *Handlers) indexRequestByExternalID(maps *requestMaps, req *Request, info requestInfo) {
	if req.MediaType == MediaTypeMovie && req.TmdbID != nil {
		maps.byTmdbID[*req.TmdbID] = info
	} else if (req.MediaType == MediaTypeSeries || req.MediaType == MediaTypeSeason) && req.TvdbID != nil {
		maps.byTvdbID[*req.TvdbID] = info
	}
}

func (h *Handlers) matchDownloadsToRequests(ctx context.Context, queue []QueueItem, maps *requestMaps) []PortalDownload {
	var downloads []PortalDownload
	for i := range queue {
		item := &queue[i]
		reqInfo, found := h.findMatchingRequest(ctx, item, maps)
		if found {
			downloads = append(downloads, PortalDownload{
				QueueItem:      *item,
				RequestID:      reqInfo.id,
				RequestTitle:   reqInfo.title,
				RequestMediaID: reqInfo.mediaID,
			})
		}
	}

	if downloads == nil {
		downloads = []PortalDownload{}
	}

	return downloads
}

func (h *Handlers) findMatchingRequest(ctx context.Context, item *QueueItem, maps *requestMaps) (requestInfo, bool) {
	reqInfo, found := h.matchByInternalID(item, maps)
	if found {
		return reqInfo, true
	}

	return h.matchByExternalID(ctx, item, maps)
}

func (h *Handlers) matchByInternalID(item *QueueItem, maps *requestMaps) (requestInfo, bool) {
	if item.MovieID != nil {
		if info, ok := maps.byMovieID[*item.MovieID]; ok {
			return info, true
		}
	}
	if item.SeriesID != nil {
		if info, ok := maps.bySeriesID[*item.SeriesID]; ok {
			return info, true
		}
	}
	return requestInfo{}, false
}

func (h *Handlers) matchByExternalID(ctx context.Context, item *QueueItem, maps *requestMaps) (requestInfo, bool) {
	if h.mediaLookup == nil {
		return requestInfo{}, false
	}

	if item.MovieID != nil {
		return h.matchMovieByExternalID(ctx, *item.MovieID, maps)
	}
	if item.SeriesID != nil {
		return h.matchSeriesByExternalID(ctx, *item.SeriesID, maps)
	}

	return requestInfo{}, false
}

func (h *Handlers) matchMovieByExternalID(ctx context.Context, movieID int64, maps *requestMaps) (requestInfo, bool) {
	tmdbID, err := h.mediaLookup.GetMovieTmdbID(ctx, movieID)
	if err != nil || tmdbID == nil {
		return requestInfo{}, false
	}
	info, found := maps.byTmdbID[*tmdbID]
	return info, found
}

func (h *Handlers) matchSeriesByExternalID(ctx context.Context, seriesID int64, maps *requestMaps) (requestInfo, bool) {
	tvdbID, err := h.mediaLookup.GetSeriesTvdbID(ctx, seriesID)
	if err != nil || tvdbID == nil {
		return requestInfo{}, false
	}
	info, found := maps.byTvdbID[*tvdbID]
	return info, found
}
