package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/slipstream/slipstream/internal/portal/invitations"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
)

type CreateInvitationRequest struct {
	Username string `json:"username"`
}

type InvitationsHandlers struct {
	invitationsService *invitations.Service
}

func NewInvitationsHandlers(invitationsService *invitations.Service) *InvitationsHandlers {
	return &InvitationsHandlers{
		invitationsService: invitationsService,
	}
}

func (h *InvitationsHandlers) RegisterRoutes(g *echo.Group, authMiddleware *portalmw.AuthMiddleware) {
	protected := g.Group("")
	protected.Use(authMiddleware.AdminAuth())

	protected.GET("", h.List)
	protected.POST("", h.Create)
	protected.GET("/:id", h.Get)
	protected.DELETE("/:id", h.Delete)
	protected.POST("/:id/resend", h.Resend)
}

// List returns all invitations
// GET /api/v1/admin/requests/invitations
func (h *InvitationsHandlers) List(c echo.Context) error {
	invitationsList, err := h.invitationsService.List(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, invitationsList)
}

// Get returns a single invitation by ID
// GET /api/v1/admin/requests/invitations/:id
func (h *InvitationsHandlers) Get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	invitation, err := h.invitationsService.Get(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, invitations.ErrInvitationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, invitation)
}

// Create creates a new invitation
// POST /api/v1/admin/requests/invitations
func (h *InvitationsHandlers) Create(c echo.Context) error {
	var req CreateInvitationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Username == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "username is required")
	}

	invitation, err := h.invitationsService.Create(c.Request().Context(), req.Username)
	if err != nil {
		if errors.Is(err, invitations.ErrInvalidUsername) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, invitation)
}

// Delete deletes an invitation
// DELETE /api/v1/admin/requests/invitations/:id
func (h *InvitationsHandlers) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	if err := h.invitationsService.Delete(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// Resend regenerates the invitation token and extends expiry
// POST /api/v1/admin/requests/invitations/:id/resend
func (h *InvitationsHandlers) Resend(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	existing, err := h.invitationsService.Get(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, invitations.ErrInvitationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	invitation, err := h.invitationsService.ResendLink(c.Request().Context(), existing.Username)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, invitation)
}
