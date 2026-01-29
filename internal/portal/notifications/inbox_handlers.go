package notifications

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
)

type InboxHandlers struct {
	service *Service
}

func NewInboxHandlers(service *Service) *InboxHandlers {
	return &InboxHandlers{service: service}
}

func (h *InboxHandlers) RegisterRoutes(g *echo.Group, authMiddleware *portalmw.AuthMiddleware) {
	protected := g.Group("")
	protected.Use(authMiddleware.AnyAuth())

	protected.GET("", h.List)
	protected.GET("/count", h.UnreadCount)
	protected.POST("/read", h.MarkAllRead)
	protected.POST("/:id/read", h.MarkRead)
}

// List returns all in-app notifications for the authenticated user
// GET /api/v1/requests/inbox
func (h *InboxHandlers) List(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	limit := int64(50)
	offset := int64(0)

	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.ParseInt(l, 10, 64); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.ParseInt(o, 10, 64); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	result, err := h.service.ListPortalNotifications(c.Request().Context(), claims.UserID, limit, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// UnreadCount returns the count of unread notifications
// GET /api/v1/requests/inbox/count
func (h *InboxHandlers) UnreadCount(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	count, err := h.service.CountUnreadPortalNotifications(c.Request().Context(), claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]int64{"count": count})
}

// MarkAllRead marks all notifications as read
// POST /api/v1/requests/inbox/read
func (h *InboxHandlers) MarkAllRead(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	if err := h.service.MarkAllPortalNotificationsRead(c.Request().Context(), claims.UserID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// MarkRead marks a single notification as read
// POST /api/v1/requests/inbox/:id/read
func (h *InboxHandlers) MarkRead(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	if err := h.service.MarkPortalNotificationRead(c.Request().Context(), claims.UserID, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}
