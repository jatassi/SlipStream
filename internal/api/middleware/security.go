package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
)

// SameOriginCORS allows CORS requests only from the same host the server is accessed on.
// This allows access from any port on the same host (e.g., Vite dev server on :3000
// accessing API on :8080) while blocking requests from different hosts.
func SameOriginCORS() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			origin := c.Request().Header.Get("Origin")
			if origin == "" {
				return next(c)
			}

			originURL, err := url.Parse(origin)
			if err != nil {
				return next(c)
			}

			requestHost := c.Request().Host
			// Strip port from request host for comparison
			requestHostname := requestHost
			if idx := strings.LastIndex(requestHost, ":"); idx != -1 {
				requestHostname = requestHost[:idx]
			}

			// Allow if origin hostname matches request hostname (any port)
			if originURL.Hostname() == requestHostname {
				h := c.Response().Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
				h.Set("Access-Control-Allow-Credentials", "true")

				if c.Request().Method == http.MethodOptions {
					return c.NoContent(http.StatusNoContent)
				}
			}

			return next(c)
		}
	}
}

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
