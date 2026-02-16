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

const (
	usernameAdministrator = "Administrator"
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
	req, err := h.validateLoginRequest(c)
	if err != nil {
		return err
	}

	if err := h.checkAccountLockout(req.Username); err != nil {
		return err
	}

	if req.Username == usernameAdministrator {
		return h.handleAdminLogin(c, req.Password)
	}

	return h.handlePortalUserLogin(c, req)
}

func (h *Handlers) validateLoginRequest(c echo.Context) (*LoginRequest, error) {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Username == "" || req.Password == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "username and password are required")
	}

	return &req, nil
}

func (h *Handlers) checkAccountLockout(username string) error {
	if h.lockoutChecker == nil {
		return nil
	}

	if !h.lockoutChecker.IsAccountLocked(username) {
		return nil
	}

	remaining := h.lockoutChecker.GetLockoutRemaining(username)
	minutes := int(remaining.Minutes()) + 1
	return echo.NewHTTPError(http.StatusTooManyRequests,
		fmt.Sprintf("account temporarily locked due to too many failed attempts, try again in %d minute(s)", minutes))
}

func (h *Handlers) handlePortalUserLogin(c echo.Context, req *LoginRequest) error {
	dbUser, err := h.usersService.ValidateCredentials(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		return h.handleLoginError(err, req.Username)
	}

	h.recordSuccessfulLogin(req.Username)

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

func (h *Handlers) handleLoginError(err error, username string) error {
	if errors.Is(err, ErrInvalidCredentials) {
		h.recordFailedAttempt(username)
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid username or password")
	}
	if errors.Is(err, ErrUserDisabled) {
		return echo.NewHTTPError(http.StatusForbidden, "account is disabled")
	}
	return echo.NewHTTPError(http.StatusInternalServerError, "authentication failed")
}

func (h *Handlers) recordFailedAttempt(username string) {
	if h.lockoutChecker != nil {
		h.lockoutChecker.RecordFailedAttempt(username)
	}
}

func (h *Handlers) recordSuccessfulLogin(username string) {
	if h.lockoutChecker != nil {
		h.lockoutChecker.RecordSuccessfulLogin(username)
	}
}

func (h *Handlers) handleAdminLogin(c echo.Context, password string) error {
	ctx := c.Request().Context()
	username := usernameAdministrator

	dbAdmin, err := h.usersService.GetDBAdmin(ctx)
	if err != nil {
		return h.handleAdminLookupError(err, username)
	}

	if err := ValidatePassword(dbAdmin.PasswordHash, password); err != nil {
		h.recordFailedAttempt(username)
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid username or password")
	}

	if dbAdmin.Enabled == 0 {
		return echo.NewHTTPError(http.StatusForbidden, "account is disabled")
	}

	h.recordSuccessfulLogin(username)

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

func (h *Handlers) handleAdminLookupError(err error, username string) error {
	if errors.Is(err, users.ErrUserNotFound) {
		h.recordFailedAttempt(username)
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid username or password")
	}
	return echo.NewHTTPError(http.StatusInternalServerError, "authentication failed")
}

func (h *Handlers) Signup(c echo.Context) error {
	req, err := h.validateSignupRequest(c)
	if err != nil {
		return err
	}

	inv, err := h.validateInvitation(c, req.Token)
	if err != nil {
		return err
	}

	user, err := h.createUserFromInvitation(c, req, inv)
	if err != nil {
		return err
	}

	if err := h.invitationsService.MarkUsed(c.Request().Context(), inv.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to mark invitation as used")
	}

	return h.generateSignupResponse(c, user.ID)
}

func (h *Handlers) validateSignupRequest(c echo.Context) (*SignupRequest, error) {
	var req SignupRequest
	if err := c.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Token == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "invitation token is required")
	}
	if req.Password == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "password is required")
	}

	return &req, nil
}

func (h *Handlers) validateInvitation(c echo.Context, token string) (*invitations.Invitation, error) {
	inv, err := h.invitationsService.Validate(c.Request().Context(), token)
	if err != nil {
		if errors.Is(err, invitations.ErrInvitationNotFound) {
			return nil, echo.NewHTTPError(http.StatusNotFound, "invitation not found")
		}
		if errors.Is(err, invitations.ErrInvitationExpired) {
			return nil, echo.NewHTTPError(http.StatusGone, "invitation has expired")
		}
		if errors.Is(err, invitations.ErrInvitationUsed) {
			return nil, echo.NewHTTPError(http.StatusConflict, "invitation has already been used")
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "failed to validate invitation")
	}
	return inv, nil
}

func (h *Handlers) createUserFromInvitation(c echo.Context, req *SignupRequest, inv *invitations.Invitation) (*users.User, error) {
	user, err := h.usersService.Create(c.Request().Context(), users.CreateInput{
		Username:         inv.Username,
		Password:         req.Password,
		DisplayName:      req.DisplayName,
		QualityProfileID: inv.QualityProfileID,
		AutoApprove:      inv.AutoApprove,
	})
	if err != nil {
		if errors.Is(err, users.ErrUsernameExists) {
			return nil, echo.NewHTTPError(http.StatusConflict, "username already registered")
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}
	return user, nil
}

func (h *Handlers) generateSignupResponse(c echo.Context, userID int64) error {
	dbUser, err := h.usersService.GetDBUser(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user")
	}

	token, err := h.authService.GeneratePortalToken(dbUser)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	user, err := h.usersService.Get(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user")
	}

	return c.JSON(http.StatusCreated, LoginResponse{
		Token: token,
		User:  user,
	})
}
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
