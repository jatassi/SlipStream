package plex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	plexTVBaseURL = "https://plex.tv"
	userAgent     = "SlipStream"
	product       = "SlipStream"
)

// Client handles communication with the Plex API
type Client struct {
	httpClient   *http.Client
	logger       zerolog.Logger
	clientID     string
	version      string
}

// NewClient creates a new Plex API client
func NewClient(httpClient *http.Client, logger zerolog.Logger, version string) *Client {
	clientID := generateClientID()
	return &Client{
		httpClient: httpClient,
		logger:     logger.With().Str("component", "plex-client").Logger(),
		clientID:   clientID,
		version:    version,
	}
}

func generateClientID() string {
	return uuid.New().String()
}

func (c *Client) getHeaders(token string) map[string]string {
	headers := map[string]string{
		"X-Plex-Client-Identifier": c.clientID,
		"X-Plex-Product":           product,
		"X-Plex-Version":           c.version,
		"X-Plex-Platform":          runtime.GOOS,
		"X-Plex-Platform-Version":  runtime.GOARCH,
		"X-Plex-Device":            runtime.GOOS,
		"X-Plex-Device-Name":       product,
		"Accept":                   "application/json",
		"Content-Type":             "application/x-www-form-urlencoded",
	}
	if token != "" {
		headers["X-Plex-Token"] = token
	}
	return headers
}

func (c *Client) doRequest(ctx context.Context, method, url string, token string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range c.getHeaders(token) {
		req.Header.Set(key, value)
	}

	return c.httpClient.Do(req)
}

// CreatePIN creates a new PIN for authentication
func (c *Client) CreatePIN(ctx context.Context) (*PINResponse, error) {
	url := fmt.Sprintf("%s/api/v2/pins", plexTVBaseURL)

	data := strings.NewReader("strong=true")
	resp, err := c.doRequest(ctx, http.MethodPost, url, "", data)
	if err != nil {
		return nil, fmt.Errorf("failed to create PIN: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create PIN: status %d, body: %s", resp.StatusCode, string(body))
	}

	var pin PINResponse
	if err := json.NewDecoder(resp.Body).Decode(&pin); err != nil {
		return nil, fmt.Errorf("failed to decode PIN response: %w", err)
	}

	return &pin, nil
}

// CheckPIN checks the status of a PIN authentication
func (c *Client) CheckPIN(ctx context.Context, pinID int) (*PINStatus, error) {
	url := fmt.Sprintf("%s/api/v2/pins/%d", plexTVBaseURL, pinID)

	resp, err := c.doRequest(ctx, http.MethodGet, url, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check PIN: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to check PIN: status %d, body: %s", resp.StatusCode, string(body))
	}

	var status PINStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode PIN status: %w", err)
	}

	return &status, nil
}

// GetAuthURL returns the URL for user authentication
func (c *Client) GetAuthURL(pinCode string) string {
	params := url.Values{}
	params.Set("clientID", c.clientID)
	params.Set("code", pinCode)
	params.Set("context[device][product]", product)
	params.Set("context[device][version]", c.version)
	params.Set("context[device][platform]", runtime.GOOS)
	params.Set("context[device][platformVersion]", runtime.GOARCH)
	params.Set("context[device][device]", runtime.GOOS)
	params.Set("context[device][deviceName]", product)

	return fmt.Sprintf("https://app.plex.tv/auth#?%s", params.Encode())
}

// GetResources returns all resources (servers) available to the user
func (c *Client) GetResources(ctx context.Context, token string) ([]PlexServer, error) {
	url := fmt.Sprintf("%s/api/v2/resources?includeHttps=1&includeRelay=1", plexTVBaseURL)

	resp, err := c.doRequest(ctx, http.MethodGet, url, token, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get resources: status %d, body: %s", resp.StatusCode, string(body))
	}

	var resources []struct {
		Name              string       `json:"name"`
		ClientIdentifier  string       `json:"clientIdentifier"`
		AccessToken       string       `json:"accessToken"`
		Provides          string       `json:"provides"`
		Connections       []Connection `json:"connections"`
		Owned             bool         `json:"owned"`
		Home              bool         `json:"home"`
		SourceTitle       string       `json:"sourceTitle"`
		PublicAddress     string       `json:"publicAddress"`
		Product           string       `json:"product"`
		ProductVersion    string       `json:"productVersion"`
		Platform          string       `json:"platform"`
		PlatformVersion   string       `json:"platformVersion"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&resources); err != nil {
		return nil, fmt.Errorf("failed to decode resources: %w", err)
	}

	var servers []PlexServer
	for _, r := range resources {
		if !strings.Contains(r.Provides, "server") {
			continue
		}
		servers = append(servers, PlexServer{
			Name:            r.Name,
			ClientID:        r.ClientIdentifier,
			AccessToken:     r.AccessToken,
			Connections:     r.Connections,
			Owned:           r.Owned,
			Home:            r.Home,
			SourceTitle:     r.SourceTitle,
			PublicAddress:   r.PublicAddress,
			Product:         r.Product,
			ProductVersion:  r.ProductVersion,
			Platform:        r.Platform,
			PlatformVersion: r.PlatformVersion,
			Provides:        r.Provides,
		})
	}

	return servers, nil
}

// GetLibrarySections returns the library sections for a server
func (c *Client) GetLibrarySections(ctx context.Context, serverURL, token string) ([]LibrarySection, error) {
	url := fmt.Sprintf("%s/library/sections", serverURL)

	resp, err := c.doRequest(ctx, http.MethodGet, url, token, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get library sections: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get library sections: status %d, body: %s", resp.StatusCode, string(body))
	}

	var mediaContainer struct {
		MediaContainer struct {
			Directory []struct {
				Key       string `json:"key"`
				Title     string `json:"title"`
				Type      string `json:"type"`
				Agent     string `json:"agent"`
				Scanner   string `json:"scanner"`
				Language  string `json:"language"`
				Location  []struct {
					ID   int    `json:"id"`
					Path string `json:"path"`
				} `json:"Location"`
			} `json:"Directory"`
		} `json:"MediaContainer"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&mediaContainer); err != nil {
		return nil, fmt.Errorf("failed to decode library sections: %w", err)
	}

	var sections []LibrarySection
	for _, dir := range mediaContainer.MediaContainer.Directory {
		key, _ := strconv.Atoi(dir.Key)
		section := LibrarySection{
			Key:      key,
			Title:    dir.Title,
			Type:     dir.Type,
			Agent:    dir.Agent,
			Scanner:  dir.Scanner,
			Language: dir.Language,
		}
		for _, loc := range dir.Location {
			section.Locations = append(section.Locations, struct {
				ID   int    `json:"id"`
				Path string `json:"path"`
			}{ID: loc.ID, Path: loc.Path})
		}
		sections = append(sections, section)
	}

	return sections, nil
}

// RefreshSection triggers a full refresh of a library section
func (c *Client) RefreshSection(ctx context.Context, serverURL string, sectionKey int, token string) error {
	url := fmt.Sprintf("%s/library/sections/%d/refresh", serverURL, sectionKey)

	resp, err := c.doRequest(ctx, http.MethodGet, url, token, nil)
	if err != nil {
		return fmt.Errorf("failed to refresh section: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to refresh section: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// RefreshPath triggers a partial refresh of a specific path in a library section
func (c *Client) RefreshPath(ctx context.Context, serverURL string, sectionKey int, path, token string) error {
	reqURL := fmt.Sprintf("%s/library/sections/%d/refresh?path=%s", serverURL, sectionKey, url.QueryEscape(path))

	resp, err := c.doRequest(ctx, http.MethodGet, reqURL, token, nil)
	if err != nil {
		return fmt.Errorf("failed to refresh path: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to refresh path: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// TestConnection tests the connection to a Plex server
func (c *Client) TestConnection(ctx context.Context, serverURL, token string) error {
	url := fmt.Sprintf("%s/identity", serverURL)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := c.doRequest(ctx, http.MethodGet, url, token, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned error: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// FindServerURL attempts to find a working URL for a server
func (c *Client) FindServerURL(ctx context.Context, server PlexServer, token string) (string, error) {
	for _, conn := range server.Connections {
		if conn.Relay {
			continue
		}

		testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := c.TestConnection(testCtx, conn.URI, token)
		cancel()

		if err == nil {
			return conn.URI, nil
		}
		c.logger.Debug().Err(err).Str("uri", conn.URI).Msg("Connection test failed")
	}

	for _, conn := range server.Connections {
		if !conn.Relay {
			continue
		}

		testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := c.TestConnection(testCtx, conn.URI, token)
		cancel()

		if err == nil {
			return conn.URI, nil
		}
		c.logger.Debug().Err(err).Str("uri", conn.URI).Msg("Relay connection test failed")
	}

	return "", fmt.Errorf("no working connection found for server %s", server.Name)
}
