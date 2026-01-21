package api

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
)

type AdminSetupRequest struct {
	Password string `json:"password"`
}

type AdminSetupResponse struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
}

type AuthStatusResponse struct {
	RequiresSetup bool `json:"requiresSetup"`
	RequiresAuth  bool `json:"requiresAuth"`
}

func isLocalRequest(c echo.Context) bool {
	ip := c.RealIP()
	return ip == "127.0.0.1" || ip == "::1" || strings.HasPrefix(ip, "localhost")
}

// POST /api/v1/auth/setup - First-time admin password setup (local only)
func (s *Server) adminSetup(c echo.Context) error {
	ctx := c.Request().Context()

	if !isLocalRequest(c) {
		return echo.NewHTTPError(http.StatusForbidden, "setup must be performed from localhost")
	}

	exists, err := s.portalUsersService.AdminExists(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to check admin status")
	}
	if exists {
		return echo.NewHTTPError(http.StatusBadRequest, "admin already configured")
	}

	var req AdminSetupRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if len(req.Password) != 4 {
		return echo.NewHTTPError(http.StatusBadRequest, "PIN must be exactly 4 digits")
	}

	user, err := s.portalUsersService.CreateAdmin(ctx, req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create admin: "+err.Error())
	}

	token, err := s.portalAuthService.GenerateAdminToken(user.ID, user.Username)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(http.StatusCreated, AdminSetupResponse{
		Token: token,
		User:  user,
	})
}

// GET /api/v1/auth/status - Check if setup is required
func (s *Server) getAuthStatus(c echo.Context) error {
	ctx := c.Request().Context()

	exists, err := s.portalUsersService.AdminExists(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to check admin status")
	}

	return c.JSON(http.StatusOK, AuthStatusResponse{
		RequiresSetup: !exists,
		RequiresAuth:  true,
	})
}

// DELETE /api/v1/auth/admin - Delete admin user (debug only, local only)
func (s *Server) deleteAdmin(c echo.Context) error {
	ctx := c.Request().Context()

	if !isLocalRequest(c) {
		return echo.NewHTTPError(http.StatusForbidden, "must be performed from localhost")
	}

	admin, err := s.portalUsersService.GetAdmin(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "admin not found")
	}

	if err := s.portalUsersService.Delete(ctx, admin.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete admin")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "admin deleted"})
}

// adminAuthMiddleware protects main API routes with admin JWT authentication
func (s *Server) adminAuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := extractBearerToken(c)
			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization token")
			}

			claims, err := s.portalAuthService.ValidateAdminToken(token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			c.Set(portalmw.PortalUserKey, claims)
			return next(c)
		}
	}
}

func extractBearerToken(c echo.Context) string {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}

	return parts[1]
}
