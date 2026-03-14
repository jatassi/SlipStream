package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/slipstream/slipstream/internal/portal"
)

const (
	PortalUserKey = "portalUser"
)

type PortalEnabledChecker interface {
	IsPortalEnabled(ctx context.Context) bool
}

type TokenValidator interface {
	ValidateToken(tokenString string) (*portal.Claims, error)
	ValidatePortalToken(tokenString string) (*portal.Claims, error)
	ValidateAdminToken(tokenString string) (*portal.Claims, error)
}

type UserExistenceChecker interface {
	UserExists(ctx context.Context, userID int64) (bool, error)
}

type AuthMiddleware struct {
	validator      TokenValidator
	enabledChecker PortalEnabledChecker
	userChecker    UserExistenceChecker
}

func NewAuthMiddleware(validator TokenValidator, enabledChecker PortalEnabledChecker, userChecker UserExistenceChecker) *AuthMiddleware {
	return &AuthMiddleware{
		validator:      validator,
		enabledChecker: enabledChecker,
		userChecker:    userChecker,
	}
}

func (m *AuthMiddleware) PortalEnabled() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if m.enabledChecker != nil && !m.enabledChecker.IsPortalEnabled(c.Request().Context()) {
				return echo.NewHTTPError(http.StatusServiceUnavailable, "external requests portal is disabled")
			}
			return next(c)
		}
	}
}

func (m *AuthMiddleware) PortalAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := extractBearerToken(c)
			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization token")
			}

			claims, err := m.validator.ValidatePortalToken(token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}

			if err := m.verifyUserExists(c.Request().Context(), claims.UserID); err != nil {
				return err
			}

			c.Set(PortalUserKey, claims)
			return next(c)
		}
	}
}

func (m *AuthMiddleware) AdminAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := extractBearerToken(c)
			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization token")
			}

			claims, err := m.validator.ValidateAdminToken(token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}

			if err := m.verifyUserExists(c.Request().Context(), claims.UserID); err != nil {
				return err
			}

			c.Set(PortalUserKey, claims)
			return next(c)
		}
	}
}

func (m *AuthMiddleware) AnyAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := extractBearerToken(c)
			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization token")
			}

			claims, err := m.validator.ValidateToken(token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}

			if err := m.verifyUserExists(c.Request().Context(), claims.UserID); err != nil {
				return err
			}

			c.Set(PortalUserKey, claims)
			return next(c)
		}
	}
}

func (m *AuthMiddleware) verifyUserExists(ctx context.Context, userID int64) error {
	if m.userChecker == nil {
		return nil
	}
	exists, err := m.userChecker.UserExists(ctx, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to verify user")
	}
	if !exists {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
	}
	return nil
}

func GetPortalUser(c echo.Context) *portal.Claims {
	claims, ok := c.Get(PortalUserKey).(*portal.Claims)
	if !ok {
		return nil
	}
	return claims
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
