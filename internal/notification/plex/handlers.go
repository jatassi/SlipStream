package plex

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

// Handlers provides HTTP handlers for Plex OAuth and server discovery
type Handlers struct {
	client *Client
	oauth  *OAuthFlow
	logger *zerolog.Logger
}

// NewHandlers creates a new Handlers instance
func NewHandlers(client *Client, logger *zerolog.Logger) *Handlers {
	subLogger := logger.With().Str("component", "plex-handlers").Logger()
	return &Handlers{
		client: client,
		oauth:  NewOAuthFlow(client),
		logger: &subLogger,
	}
}

// RegisterRoutes registers the Plex OAuth and discovery routes
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.POST("/auth/start", h.StartAuth)
	g.GET("/auth/status/:pinId", h.CheckAuthStatus)
	g.GET("/servers", h.ListServers)
	g.GET("/servers/:id/sections", h.ListSections)
}

// StartAuthResponse is the response from starting OAuth
type StartAuthResponse struct {
	PinID    int    `json:"pinId"`
	AuthURL  string `json:"authUrl"`
	ClientID string `json:"clientId"`
}

// StartAuth initiates the Plex OAuth flow
// POST /api/v1/notifications/plex/auth/start
func (h *Handlers) StartAuth(c echo.Context) error {
	ctx := c.Request().Context()

	result, err := h.oauth.StartAuth(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to start OAuth")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to start Plex authentication"})
	}

	return c.JSON(http.StatusOK, StartAuthResponse{
		PinID:    result.PinID,
		AuthURL:  result.AuthURL,
		ClientID: h.client.ClientID(),
	})
}

// AuthStatusResponse is the response from checking auth status
type AuthStatusResponse struct {
	Complete  bool   `json:"complete"`
	AuthToken string `json:"authToken,omitempty"`
}

// CheckAuthStatus checks if the user has completed Plex authentication
// GET /api/v1/notifications/plex/auth/status/:pinId
func (h *Handlers) CheckAuthStatus(c echo.Context) error {
	ctx := c.Request().Context()

	pinID, err := strconv.Atoi(c.Param("pinId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid pin ID"})
	}

	result, err := h.oauth.CheckAuth(ctx, pinID)
	if err != nil {
		if errors.Is(err, ErrPINExpired) {
			return c.JSON(http.StatusGone, map[string]string{"error": "PIN has expired"})
		}
		h.logger.Error().Err(err).Int("pinId", pinID).Msg("Failed to check auth status")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check authentication status"})
	}

	return c.JSON(http.StatusOK, AuthStatusResponse{
		Complete:  result.Complete,
		AuthToken: result.AuthToken,
	})
}

// ServerResponse represents a Plex server in API responses
type ServerResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Owned   bool   `json:"owned"`
	Address string `json:"address,omitempty"`
}

// ListServers returns the user's available Plex servers
// GET /api/v1/notifications/plex/servers
func (h *Handlers) ListServers(c echo.Context) error {
	ctx := c.Request().Context()

	token := c.Request().Header.Get("X-Plex-Token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "X-Plex-Token header is required"})
	}

	servers, err := h.client.GetResources(ctx, token)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get servers")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get Plex servers"})
	}

	response := make([]ServerResponse, 0, len(servers))
	for i := range servers {
		server := &servers[i]
		var address string
		for _, conn := range server.Connections {
			if !conn.Relay && !conn.Local {
				address = conn.Address
				break
			}
		}
		if address == "" && len(server.Connections) > 0 {
			address = server.Connections[0].Address
		}

		response = append(response, ServerResponse{
			ID:      server.ClientID,
			Name:    server.Name,
			Owned:   server.Owned,
			Address: address,
		})
	}

	return c.JSON(http.StatusOK, response)
}

// SectionResponse represents a library section in API responses
type SectionResponse struct {
	Key   int    `json:"key"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// ListSections returns the library sections for a server
// GET /api/v1/notifications/plex/servers/:id/sections
func (h *Handlers) ListSections(c echo.Context) error {
	ctx := c.Request().Context()

	serverID := c.Param("id")
	if serverID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Server ID is required"})
	}

	token := c.Request().Header.Get("X-Plex-Token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "X-Plex-Token header is required"})
	}

	targetServer, err := h.findServer(c, serverID, token)
	if err != nil {
		return err
	}

	serverURL, err := h.client.FindServerURL(ctx, targetServer, token)
	if err != nil {
		h.logger.Error().Err(err).Str("serverId", serverID).Msg("Failed to connect to server")
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Failed to connect to Plex server"})
	}

	sections, err := h.client.GetLibrarySections(ctx, serverURL, token)
	if err != nil {
		h.logger.Error().Err(err).Str("serverId", serverID).Msg("Failed to get library sections")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get library sections"})
	}

	return c.JSON(http.StatusOK, filterMediaSections(sections))
}

func (h *Handlers) findServer(c echo.Context, serverID, token string) (*PlexServer, error) {
	servers, err := h.client.GetResources(c.Request().Context(), token)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get servers")
		return nil, c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get Plex servers"})
	}

	for i := range servers {
		if servers[i].ClientID == serverID {
			return &servers[i], nil
		}
	}

	return nil, c.JSON(http.StatusNotFound, map[string]string{"error": "Server not found"})
}

func filterMediaSections(sections []LibrarySection) []SectionResponse {
	response := make([]SectionResponse, 0, len(sections))
	for _, section := range sections {
		if section.Type == "movie" || section.Type == "show" {
			response = append(response, SectionResponse{
				Key:   section.Key,
				Title: section.Title,
				Type:  section.Type,
			})
		}
	}
	return response
}
