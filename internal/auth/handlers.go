package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/slipstream/slipstream/internal/portal/invitations"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
	"github.com/slipstream/slipstream/internal/portal/users"
)

// AccountLockoutChecker provides account lockout functionality.
type AccountLockoutChecker interface {
	IsAccountLocked(username string) bool
	GetLockoutRemaining(username string) time.Duration
	RecordFailedAttempt(username string)
	RecordSuccessfulLogin(username string)
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AdminLoginResponse struct {
	Token   string      `json:"token"`
	User    *users.User `json:"user"`
	IsAdmin bool        `json:"isAdmin"`
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  *users.User `json:"user"`
}

type SignupRequest struct {
	Token       string `json:"token"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName,omitempty"`
}

type ResendRequest struct {
	Username string `json:"username"`
}

type ValidateInvitationResponse struct {
	Valid     bool   `json:"valid"`
	Username  string `json:"username"`
	ExpiresAt string `json:"expiresAt"`
}

type UpdateProfileRequest struct {
	Username    *string `json:"username,omitempty"`
	Password    *string `json:"password,omitempty"`
	DisplayName *string `json:"displayName,omitempty"`
}

type VerifyPinRequest struct {
	Pin string `json:"pin"`
}

type VerifyPinResponse struct {
	Valid bool `json:"valid"`
}

type Handlers struct {
	authService        *Service
	usersService       *users.Service
	invitationsService *invitations.Service
	lockoutChecker     AccountLockoutChecker
}

func NewHandlers(authService *Service, usersService *users.Service, invitationsService *invitations.Service) *Handlers {
	return &Handlers{
		authService:        authService,
		usersService:       usersService,
		invitationsService: invitationsService,
	}
}

func (h *Handlers) SetLockoutChecker(checker AccountLockoutChecker) {
	h.lockoutChecker = checker
}

func (h *Handlers) RegisterRoutes(g *echo.Group, authMiddleware *portalmw.AuthMiddleware) {
	g.POST("/login", h.Login)
	g.POST("/signup", h.Signup)
	g.POST("/resend", h.ResendInvitation)
	g.POST("/logout", h.Logout)
	g.GET("/validate-invitation", h.ValidateInvitation)

	protected := g.Group("")
	protected.Use(authMiddleware.AnyAuth())
	protected.GET("/profile", h.GetProfile)
	protected.PUT("/profile", h.UpdateProfile)
	protected.POST("/verify-pin", h.VerifyPin)
}

// POST /api/v1/requests/auth/login
func (h *Handlers) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Username == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "username and password are required")
	}

	// Check account lockout
	if h.lockoutChecker != nil && h.lockoutChecker.IsAccountLocked(req.Username) {
		remaining := h.lockoutChecker.GetLockoutRemaining(req.Username)
		minutes := int(remaining.Minutes()) + 1
		return echo.NewHTTPError(http.StatusTooManyRequests,
			fmt.Sprintf("account temporarily locked due to too many failed attempts, try again in %d minute(s)", minutes))
	}

	// Check if this is an admin login (username is "Administrator")
	if req.Username == "Administrator" {
		return h.handleAdminLogin(c, req.Password)
	}

	dbUser, err := h.usersService.ValidateCredentials(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			if h.lockoutChecker != nil {
				h.lockoutChecker.RecordFailedAttempt(req.Username)
			}
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid username or password")
		}
		if errors.Is(err, ErrUserDisabled) {
			return echo.NewHTTPError(http.StatusForbidden, "account is disabled")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "authentication failed")
	}

	// Clear lockout on successful login
	if h.lockoutChecker != nil {
		h.lockoutChecker.RecordSuccessfulLogin(req.Username)
	}

	token, err := h.authService.GeneratePortalToken(dbUser)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	user, err := h.usersService.Get(c.Request().Context(), dbUser.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user")
	}

	return c.JSON(http.StatusOK, AdminLoginResponse{
		Token:   token,
		User:    user,
		IsAdmin: false,
	})
}

func (h *Handlers) handleAdminLogin(c echo.Context, password string) error {
	ctx := c.Request().Context()
	username := "Administrator"

	dbAdmin, err := h.usersService.GetDBAdmin(ctx)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			if h.lockoutChecker != nil {
				h.lockoutChecker.RecordFailedAttempt(username)
			}
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid username or password")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "authentication failed")
	}

	if err := ValidatePassword(dbAdmin.PasswordHash, password); err != nil {
		if h.lockoutChecker != nil {
			h.lockoutChecker.RecordFailedAttempt(username)
		}
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid username or password")
	}

	if dbAdmin.Enabled == 0 {
		return echo.NewHTTPError(http.StatusForbidden, "account is disabled")
	}

	// Clear lockout on successful login
	if h.lockoutChecker != nil {
		h.lockoutChecker.RecordSuccessfulLogin(username)
	}

	token, err := h.authService.GenerateAdminToken(dbAdmin.ID, dbAdmin.Username)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	admin, err := h.usersService.GetAdmin(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user")
	}

	return c.JSON(http.StatusOK, AdminLoginResponse{
		Token:   token,
		User:    admin,
		IsAdmin: true,
	})
}

// POST /api/v1/requests/auth/signup
func (h *Handlers) Signup(c echo.Context) error {
	var req SignupRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Token == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "invitation token is required")
	}
	if req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "password is required")
	}

	inv, err := h.invitationsService.Validate(c.Request().Context(), req.Token)
	if err != nil {
		if errors.Is(err, invitations.ErrInvitationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "invitation not found")
		}
		if errors.Is(err, invitations.ErrInvitationExpired) {
			return echo.NewHTTPError(http.StatusGone, "invitation has expired")
		}
		if errors.Is(err, invitations.ErrInvitationUsed) {
			return echo.NewHTTPError(http.StatusConflict, "invitation has already been used")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to validate invitation")
	}

	user, err := h.usersService.Create(c.Request().Context(), users.CreateInput{
		Username:         inv.Username,
		Password:         req.Password,
		DisplayName:      req.DisplayName,
		QualityProfileID: inv.QualityProfileID,
		AutoApprove:      inv.AutoApprove,
	})
	if err != nil {
		if errors.Is(err, users.ErrUsernameExists) {
			return echo.NewHTTPError(http.StatusConflict, "username already registered")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}

	if err := h.invitationsService.MarkUsed(c.Request().Context(), inv.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to mark invitation as used")
	}

	dbUser, err := h.usersService.GetDBUser(c.Request().Context(), user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user")
	}

	token, err := h.authService.GeneratePortalToken(dbUser)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(http.StatusCreated, LoginResponse{
		Token: token,
		User:  user,
	})
}

// GET /api/v1/requests/auth/validate-invitation
func (h *Handlers) ValidateInvitation(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "token is required")
	}

	inv, err := h.invitationsService.Validate(c.Request().Context(), token)
	if err != nil {
		if errors.Is(err, invitations.ErrInvitationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "invitation not found")
		}
		if errors.Is(err, invitations.ErrInvitationExpired) {
			return echo.NewHTTPError(http.StatusGone, "invitation has expired")
		}
		if errors.Is(err, invitations.ErrInvitationUsed) {
			return echo.NewHTTPError(http.StatusConflict, "invitation has already been used")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to validate invitation")
	}

	return c.JSON(http.StatusOK, ValidateInvitationResponse{
		Valid:     true,
		Username:  inv.Username,
		ExpiresAt: inv.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// POST /api/v1/requests/auth/resend
func (h *Handlers) ResendInvitation(c echo.Context) error {
	var req ResendRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Username == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "username is required")
	}

	_, err := h.invitationsService.ResendLink(c.Request().Context(), req.Username)
	if err != nil {
		if errors.Is(err, invitations.ErrInvitationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "no invitation found for this username")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to resend invitation")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "invitation link has been regenerated",
	})
}

// POST /api/v1/requests/auth/logout
func (h *Handlers) Logout(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}

// GET /api/v1/requests/auth/profile
func (h *Handlers) GetProfile(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	user, err := h.usersService.Get(c.Request().Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get profile")
	}

	return c.JSON(http.StatusOK, user)
}

// PUT /api/v1/requests/auth/profile
func (h *Handlers) UpdateProfile(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req UpdateProfileRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	user, err := h.usersService.Update(c.Request().Context(), claims.UserID, users.UpdateInput{
		Username:    req.Username,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		if errors.Is(err, users.ErrUsernameExists) {
			return echo.NewHTTPError(http.StatusConflict, "username already in use")
		}
		if errors.Is(err, users.ErrInvalidUsername) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid username")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update profile")
	}

	return c.JSON(http.StatusOK, user)
}

// POST /api/v1/portal/auth/verify-pin
func (h *Handlers) VerifyPin(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req VerifyPinRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Pin == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "pin is required")
	}

	dbUser, err := h.usersService.GetDBUser(c.Request().Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to verify pin")
	}

	if err := ValidatePassword(dbUser.PasswordHash, req.Pin); err != nil {
		return c.JSON(http.StatusOK, VerifyPinResponse{Valid: false})
	}

	return c.JSON(http.StatusOK, VerifyPinResponse{Valid: true})
}
