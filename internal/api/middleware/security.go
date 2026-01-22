package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
)

func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()

			// Prevent MIME type sniffing
			h.Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking
			h.Set("X-Frame-Options", "SAMEORIGIN")

			// Enable XSS filter in older browsers
			h.Set("X-XSS-Protection", "1; mode=block")

			// Control referrer information
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Prevent content from being loaded in other sites' iframes
			h.Set("Content-Security-Policy", "frame-ancestors 'self'")

			// Disable caching for API responses (can be overridden per-route if needed)
			if strings.HasPrefix(c.Request().URL.Path, "/api") {
				h.Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
				h.Set("Pragma", "no-cache")
			}

			return next(c)
		}
	}
}
