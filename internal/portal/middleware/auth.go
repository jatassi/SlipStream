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

type AuthMiddleware struct {
	validator      TokenValidator
	enabledChecker PortalEnabledChecker
}

func NewAuthMiddleware(validator TokenValidator) *AuthMiddleware {
	return &AuthMiddleware{
		validator: validator,
	}
}

func (m *AuthMiddleware) SetEnabledChecker(checker PortalEnabledChecker) {
	m.enabledChecker = checker
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

			c.Set(PortalUserKey, claims)
			return next(c)
		}
	}
}

func GetPortalUser(c echo.Context) *portal.Claims {
	claims, ok := c.Get(PortalUserKey).(*portal.Claims)
	if !ok {
		return nil
	}
	return claims
}

func IsAdmin(c echo.Context) bool {
	claims := GetPortalUser(c)
	if claims == nil {
		return false
	}
	return claims.Audience == portal.AudienceAdmin
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
