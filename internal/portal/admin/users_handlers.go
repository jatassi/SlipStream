package admin

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
	"github.com/slipstream/slipstream/internal/portal/quota"
	"github.com/slipstream/slipstream/internal/portal/users"
)

type UserWithQuota struct {
	*users.User
	Quota *quota.QuotaStatus `json:"quota,omitempty"`
}

type UpdateUserRequest struct {
	Username         *string `json:"username"`
	QualityProfileID *int64  `json:"qualityProfileId"`
	AutoApprove      *bool   `json:"autoApprove"`
}

type UpdateQuotaRequest struct {
	MoviesLimit   *int64 `json:"moviesLimit"`
	SeasonsLimit  *int64 `json:"seasonsLimit"`
	EpisodesLimit *int64 `json:"episodesLimit"`
}

type UsersHandlers struct {
	usersService *users.Service
	quotaService *quota.Service
}

func NewUsersHandlers(usersService *users.Service, quotaService *quota.Service) *UsersHandlers {
	return &UsersHandlers{
		usersService: usersService,
		quotaService: quotaService,
	}
}

func (h *UsersHandlers) RegisterRoutes(g *echo.Group, authMiddleware *portalmw.AuthMiddleware) {
	protected := g.Group("")
	protected.Use(authMiddleware.AdminAuth())

	protected.GET("", h.List)
	protected.GET("/:id", h.Get)
	protected.PUT("/:id", h.Update)
	protected.POST("/:id/enable", h.Enable)
	protected.POST("/:id/disable", h.Disable)
	protected.DELETE("/:id", h.Delete)
	protected.GET("/:id/quota", h.GetQuota)
	protected.PUT("/:id/quota", h.UpdateQuota)
	protected.DELETE("/:id/quota", h.ClearQuota)
}

// List returns all portal users with quota status
// GET /api/v1/admin/requests/users
func (h *UsersHandlers) List(c echo.Context) error {
	usersList, err := h.usersService.List(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	results := make([]*UserWithQuota, len(usersList))
	for i, user := range usersList {
		results[i] = &UserWithQuota{User: user}
		quotaStatus, err := h.quotaService.GetUserQuota(c.Request().Context(), user.ID)
		if err == nil {
			results[i].Quota = quotaStatus
		}
	}

	return c.JSON(http.StatusOK, results)
}

// Get returns a single user by ID
// GET /api/v1/admin/requests/users/:id
func (h *UsersHandlers) Get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	user, err := h.usersService.Get(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result := &UserWithQuota{User: user}
	quotaStatus, err := h.quotaService.GetUserQuota(c.Request().Context(), user.ID)
	if err == nil {
		result.Quota = quotaStatus
	}

	return c.JSON(http.StatusOK, result)
}

// Update updates a user's settings
// PUT /api/v1/admin/requests/users/:id
func (h *UsersHandlers) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var req UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	ctx := c.Request().Context()
	user, err := h.applyUserUpdates(ctx, id, &req)
	if err != nil {
		return err
	}

	if user == nil {
		user, err = h.usersService.Get(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
	}

	result := &UserWithQuota{User: user}
	quotaStatus, err := h.quotaService.GetUserQuota(ctx, user.ID)
	if err == nil {
		result.Quota = quotaStatus
	}

	return c.JSON(http.StatusOK, result)
}

func (h *UsersHandlers) applyUserUpdates(ctx context.Context, id int64, req *UpdateUserRequest) (*users.User, error) {
	var user *users.User

	if req.Username != nil {
		u, err := h.usersService.Update(ctx, id, users.UpdateInput{Username: req.Username})
		if err != nil {
			return nil, mapUserError(err)
		}
		user = u
	}

	if req.QualityProfileID != nil {
		u, err := h.usersService.SetQualityProfile(ctx, id, req.QualityProfileID)
		if err != nil {
			return nil, mapUserError(err)
		}
		user = u
	}

	if req.AutoApprove != nil {
		u, err := h.usersService.SetAutoApprove(ctx, id, *req.AutoApprove)
		if err != nil {
			return nil, mapUserError(err)
		}
		user = u
	}

	return user, nil
}

func mapUserError(err error) *echo.HTTPError {
	if errors.Is(err, users.ErrUserNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	if errors.Is(err, users.ErrUsernameExists) {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}
	if errors.Is(err, users.ErrInvalidUsername) {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
}

// Enable enables a user account
// POST /api/v1/admin/requests/users/:id/enable
func (h *UsersHandlers) Enable(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	user, err := h.usersService.SetEnabled(c.Request().Context(), id, true)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, user)
}

// Disable disables a user account
// POST /api/v1/admin/requests/users/:id/disable
func (h *UsersHandlers) Disable(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	user, err := h.usersService.SetEnabled(c.Request().Context(), id, false)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, user)
}

// Delete deletes a user
// DELETE /api/v1/admin/requests/users/:id
func (h *UsersHandlers) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	if err := h.usersService.Delete(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// GetQuota returns a user's quota status
// GET /api/v1/admin/requests/users/:id/quota
func (h *UsersHandlers) GetQuota(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	quotaStatus, err := h.quotaService.GetUserQuota(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, quotaStatus)
}

// UpdateQuota updates a user's quota limits
// PUT /api/v1/admin/requests/users/:id/quota
func (h *UsersHandlers) UpdateQuota(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var req UpdateQuotaRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	quotaStatus, err := h.quotaService.SetUserOverride(c.Request().Context(), id, quota.QuotaLimits{
		MoviesLimit:   req.MoviesLimit,
		SeasonsLimit:  req.SeasonsLimit,
		EpisodesLimit: req.EpisodesLimit,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, quotaStatus)
}

// ClearQuota clears a user's quota overrides
// DELETE /api/v1/admin/requests/users/:id/quota
func (h *UsersHandlers) ClearQuota(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	quotaStatus, err := h.quotaService.ClearUserOverride(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, quota.ErrQuotaNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, quotaStatus)
}
