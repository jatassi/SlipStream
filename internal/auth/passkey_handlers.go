package auth

import (
	"net/http"

	"github.com/labstack/echo/v4"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
	"github.com/slipstream/slipstream/internal/portal/users"
)

type BeginRegistrationRequest struct {
	PIN string `json:"pin"`
}

type FinishRegistrationRequest struct {
	ChallengeID string `json:"challengeId"`
	Name        string `json:"name"`
}

type FinishLoginRequest struct {
	ChallengeID string `json:"challengeId"`
}

type PasskeyHandlers struct {
	passkeyService *PasskeyService
	authService    *Service
	usersService   *users.Service
}

func NewPasskeyHandlers(passkeyService *PasskeyService, authService *Service, usersService *users.Service) *PasskeyHandlers {
	return &PasskeyHandlers{
		passkeyService: passkeyService,
		authService:    authService,
		usersService:   usersService,
	}
}

func (h *PasskeyHandlers) RegisterRoutes(g *echo.Group, authMiddleware *portalmw.AuthMiddleware) {
	// Public routes (for login)
	g.POST("/passkey/login/begin", h.BeginLogin)
	g.POST("/passkey/login/finish", h.FinishLogin)

	// Protected routes (for registration and management)
	protected := g.Group("")
	protected.Use(authMiddleware.AnyAuth())
	protected.POST("/passkey/register/begin", h.BeginRegistration)
	protected.POST("/passkey/register/finish", h.FinishRegistration)
	protected.GET("/passkey/credentials", h.ListCredentials)
	protected.PUT("/passkey/credentials/:id", h.UpdateCredential)
	protected.DELETE("/passkey/credentials/:id", h.DeleteCredential)
}

// POST /api/v1/requests/auth/passkey/register/begin
func (h *PasskeyHandlers) BeginRegistration(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req BeginRegistrationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	user, err := h.usersService.GetDBUser(c.Request().Context(), claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	// Verify PIN before allowing passkey registration
	if err := ValidatePassword(user.PasswordHash, req.PIN); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid PIN")
	}

	resp, err := h.passkeyService.BeginRegistration(c.Request().Context(), user)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, resp)
}

// POST /api/v1/requests/auth/passkey/register/finish
func (h *PasskeyHandlers) FinishRegistration(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	// Extract challengeId and name from query params since body is raw credential
	challengeID := c.QueryParam("challengeId")
	name := c.QueryParam("name")
	if challengeID == "" || name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "challengeId and name query parameters are required")
	}

	user, err := h.usersService.GetDBUser(c.Request().Context(), claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	if err := h.passkeyService.FinishRegistration(c.Request().Context(), user, challengeID, name, c.Request()); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, map[string]string{"status": "ok"})
}

// POST /api/v1/requests/auth/passkey/login/begin
func (h *PasskeyHandlers) BeginLogin(c echo.Context) error {
	resp, err := h.passkeyService.BeginLogin(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, resp)
}

// POST /api/v1/requests/auth/passkey/login/finish
func (h *PasskeyHandlers) FinishLogin(c echo.Context) error {
	challengeID := c.QueryParam("challengeId")
	if challengeID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "challengeId query parameter is required")
	}

	result, err := h.passkeyService.FinishLogin(c.Request().Context(), challengeID, c.Request())
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	// Generate JWT token based on user type
	var token string
	if result.IsAdmin {
		token, err = h.authService.GenerateAdminToken(result.UserID, result.Username)
	} else {
		user, userErr := h.usersService.GetDBUser(c.Request().Context(), result.UserID)
		if userErr != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user")
		}
		token, err = h.authService.GeneratePortalToken(user)
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	userInfo, err := h.usersService.Get(c.Request().Context(), result.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user info")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"token":   token,
		"user":    userInfo,
		"isAdmin": result.IsAdmin,
	})
}

// GET /api/v1/requests/auth/passkey/credentials
func (h *PasskeyHandlers) ListCredentials(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	creds, err := h.passkeyService.ListCredentials(c.Request().Context(), claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, creds)
}

// PUT /api/v1/requests/auth/passkey/credentials/:id
func (h *PasskeyHandlers) UpdateCredential(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	credID := c.Param("id")

	var req struct {
		Name string `json:"name"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := h.passkeyService.UpdateCredentialName(c.Request().Context(), credID, claims.UserID, req.Name); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

// DELETE /api/v1/requests/auth/passkey/credentials/:id
func (h *PasskeyHandlers) DeleteCredential(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	credID := c.Param("id")

	if err := h.passkeyService.DeleteCredential(c.Request().Context(), credID, claims.UserID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}
