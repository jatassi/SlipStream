package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/slipstream/slipstream/internal/downloader"
)

const redactedSentinel = "********"

func redactDownloadClient(client *downloader.DownloadClient) {
	if client.Password != "" {
		client.Password = redactedSentinel
	}
	if client.APIKey != "" {
		client.APIKey = redactedSentinel
	}
}

func (s *Server) restoreRedactedCredentials(ctx context.Context, id int64, input *downloader.UpdateClientInput) {
	if input.Password != redactedSentinel && input.APIKey != redactedSentinel {
		return
	}
	existing, err := s.downloaderService.Get(ctx, id)
	if err != nil {
		return
	}
	if input.Password == redactedSentinel {
		input.Password = existing.Password
	}
	if input.APIKey == redactedSentinel {
		input.APIKey = existing.APIKey
	}
}

func (s *Server) listDownloadClients(c echo.Context) error {
	ctx := c.Request().Context()

	clients, err := s.downloaderService.List(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	for _, client := range clients {
		redactDownloadClient(client)
	}

	return c.JSON(http.StatusOK, clients)
}

func (s *Server) addDownloadClient(c echo.Context) error {
	ctx := c.Request().Context()

	var input downloader.CreateClientInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	client, err := s.downloaderService.Create(ctx, &input)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	redactDownloadClient(client)
	return c.JSON(http.StatusCreated, client)
}

func (s *Server) getDownloadClient(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parseIDParam(c)
	if err != nil {
		return err
	}

	client, err := s.downloaderService.Get(ctx, id)
	if err != nil {
		if errors.Is(err, downloader.ErrClientNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "client not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	redactDownloadClient(client)
	return c.JSON(http.StatusOK, client)
}

func (s *Server) updateDownloadClient(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parseIDParam(c)
	if err != nil {
		return err
	}

	var input downloader.UpdateClientInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	s.restoreRedactedCredentials(ctx, id, &input)

	client, err := s.downloaderService.Update(ctx, id, &input)
	if err != nil {
		if errors.Is(err, downloader.ErrClientNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "client not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	redactDownloadClient(client)
	return c.JSON(http.StatusOK, client)
}

func (s *Server) deleteDownloadClient(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parseIDParam(c)
	if err != nil {
		return err
	}

	if err := s.downloaderService.Delete(ctx, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

func (s *Server) testDownloadClient(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parseIDParam(c)
	if err != nil {
		return err
	}

	result, err := s.downloaderService.Test(ctx, id)
	if err != nil {
		if errors.Is(err, downloader.ErrClientNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "client not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

func (s *Server) testNewDownloadClient(c echo.Context) error {
	ctx := c.Request().Context()

	var input downloader.CreateClientInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	result, err := s.downloaderService.TestConfig(ctx, &input)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// Queue handlers
func (s *Server) getQueue(c echo.Context) error {
	ctx := c.Request().Context()

	resp, err := s.downloaderService.GetQueue(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Trigger import check asynchronously - provides faster import triggering than scheduled task
	// The import service is efficient and only processes newly completed downloads
	go func() {
		if err := s.importService.CheckAndProcessCompletedDownloads(context.Background()); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to process completed downloads")
		}
	}()

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) pauseDownload(c echo.Context) error {
	ctx := c.Request().Context()
	torrentID := c.Param("id")

	var body struct {
		ClientID int64 `json:"clientId"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := s.downloaderService.PauseDownload(ctx, body.ClientID, torrentID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Trigger immediate broadcast of queue state
	if s.queueBroadcaster != nil {
		s.queueBroadcaster.Trigger()
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "paused"})
}

func (s *Server) resumeDownload(c echo.Context) error {
	ctx := c.Request().Context()
	torrentID := c.Param("id")

	var body struct {
		ClientID int64 `json:"clientId"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := s.downloaderService.ResumeDownload(ctx, body.ClientID, torrentID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Trigger fast polling and immediate broadcast
	if s.queueBroadcaster != nil {
		s.queueBroadcaster.Trigger()
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "resumed"})
}

func (s *Server) fastForwardDownload(c echo.Context) error {
	ctx := c.Request().Context()
	downloadID := c.Param("id")

	var body struct {
		ClientID int64 `json:"clientId"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := s.downloaderService.FastForwardMockDownload(ctx, body.ClientID, downloadID); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Trigger immediate broadcast of queue state
	if s.queueBroadcaster != nil {
		s.queueBroadcaster.Trigger()
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "completed"})
}

func (s *Server) removeFromQueue(c echo.Context) error {
	ctx := c.Request().Context()
	torrentID := c.Param("id")

	clientID, err := strconv.ParseInt(c.QueryParam("clientId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid clientId")
	}

	deleteFiles := c.QueryParam("deleteFiles") == queryTrue

	if err := s.downloaderService.RemoveDownload(ctx, clientID, torrentID, deleteFiles); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Trigger immediate broadcast of queue state
	if s.queueBroadcaster != nil {
		s.queueBroadcaster.Trigger()
	}

	return c.NoContent(http.StatusNoContent)
}

// getIndexerHistory returns indexer search and grab history.
func (s *Server) getIndexerHistory(c echo.Context) error {
	limit := 50
	offset := 0

	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	grabHistory, err := s.grabService.GetGrabHistory(c.Request().Context(), limit, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, grabHistory)
}
