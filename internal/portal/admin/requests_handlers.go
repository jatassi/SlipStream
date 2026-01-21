package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
	"github.com/slipstream/slipstream/internal/portal/requests"
)

type ApproveRequestInput struct {
	Action       string `json:"action"`
	RootFolderID *int64 `json:"rootFolderId,omitempty"`
}

type DenyRequestInput struct {
	Reason string `json:"reason"`
}

type BatchApproveInput struct {
	IDs          []int64 `json:"ids"`
	Action       string  `json:"action"`
	RootFolderID *int64  `json:"rootFolderId,omitempty"`
}

type BatchDenyInput struct {
	IDs    []int64 `json:"ids"`
	Reason string  `json:"reason"`
}

type BatchDeleteInput struct {
	IDs []int64 `json:"ids"`
}

type RequestSearcher interface {
	SearchForRequestAsync(requestID int64)
}

type RequestsHandlers struct {
	requestsService *requests.Service
	requestSearcher RequestSearcher
}

func NewRequestsHandlers(requestsService *requests.Service, requestSearcher RequestSearcher) *RequestsHandlers {
	return &RequestsHandlers{
		requestsService: requestsService,
		requestSearcher: requestSearcher,
	}
}

func (h *RequestsHandlers) RegisterRoutes(g *echo.Group, authMiddleware *portalmw.AuthMiddleware) {
	protected := g.Group("")
	protected.Use(authMiddleware.AdminAuth())

	protected.GET("", h.List)
	protected.GET("/:id", h.Get)
	protected.POST("/:id/approve", h.Approve)
	protected.POST("/:id/deny", h.Deny)
	protected.DELETE("/:id", h.Delete)
	protected.POST("/batch/approve", h.BatchApprove)
	protected.POST("/batch/deny", h.BatchDeny)
	protected.POST("/batch/delete", h.BatchDelete)
}

// List returns all requests with admin filtering
// GET /api/v1/admin/requests
func (h *RequestsHandlers) List(c echo.Context) error {
	filters := requests.ListFilters{}

	if status := c.QueryParam("status"); status != "" {
		filters.Status = &status
	}
	if mediaType := c.QueryParam("mediaType"); mediaType != "" {
		filters.MediaType = &mediaType
	}
	if userIDStr := c.QueryParam("userId"); userIDStr != "" {
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err == nil {
			filters.UserID = &userID
		}
	}

	requestsList, err := h.requestsService.List(c.Request().Context(), filters)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, requestsList)
}

// Get returns a single request by ID
// GET /api/v1/admin/requests/:id
func (h *RequestsHandlers) Get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	request, err := h.requestsService.Get(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, requests.ErrRequestNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, request)
}

// Approve approves a request
// POST /api/v1/admin/requests/:id/approve
func (h *RequestsHandlers) Approve(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input ApproveRequestInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	action := requests.ApprovalActionOnly
	switch input.Action {
	case "auto_search":
		action = requests.ApprovalActionAutoSearch
	case "manual_search":
		action = requests.ApprovalActionManual
	}

	request, err := h.requestsService.Approve(c.Request().Context(), id, claims.UserID, action)
	if err != nil {
		if errors.Is(err, requests.ErrRequestNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if action == requests.ApprovalActionAutoSearch && h.requestSearcher != nil {
		h.requestSearcher.SearchForRequestAsync(id)
	}

	return c.JSON(http.StatusOK, request)
}

// Deny denies a request
// POST /api/v1/admin/requests/:id/deny
func (h *RequestsHandlers) Deny(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input DenyRequestInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	var reason *string
	if input.Reason != "" {
		reason = &input.Reason
	}

	request, err := h.requestsService.Deny(c.Request().Context(), id, reason)
	if err != nil {
		if errors.Is(err, requests.ErrRequestNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, request)
}

// Delete deletes a request
// DELETE /api/v1/admin/requests/:id
func (h *RequestsHandlers) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	if err := h.requestsService.Delete(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// BatchApprove approves multiple requests
// POST /api/v1/admin/requests/batch/approve
func (h *RequestsHandlers) BatchApprove(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var input BatchApproveInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if len(input.IDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "ids is required")
	}

	action := requests.ApprovalActionOnly
	switch input.Action {
	case "auto_search":
		action = requests.ApprovalActionAutoSearch
	case "manual_search":
		action = requests.ApprovalActionManual
	}

	results, err := h.requestsService.BatchApprove(c.Request().Context(), input.IDs, claims.UserID, action)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if action == requests.ApprovalActionAutoSearch && h.requestSearcher != nil {
		for _, req := range results {
			h.requestSearcher.SearchForRequestAsync(req.ID)
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"approved": len(results),
		"requests": results,
	})
}

// BatchDeny denies multiple requests
// POST /api/v1/admin/requests/batch/deny
func (h *RequestsHandlers) BatchDeny(c echo.Context) error {
	var input BatchDenyInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if len(input.IDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "ids is required")
	}

	var reason *string
	if input.Reason != "" {
		reason = &input.Reason
	}

	results, err := h.requestsService.BatchDeny(c.Request().Context(), input.IDs, reason)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"denied":   len(results),
		"requests": results,
	})
}

// BatchDelete permanently deletes multiple requests
// POST /api/v1/admin/requests/batch/delete
func (h *RequestsHandlers) BatchDelete(c echo.Context) error {
	var input BatchDeleteInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if len(input.IDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "ids is required")
	}

	deleted := 0
	for _, id := range input.IDs {
		if err := h.requestsService.Delete(c.Request().Context(), id); err == nil {
			deleted++
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"deleted": deleted,
	})
}
