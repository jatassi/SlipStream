package cardigann

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/rs/zerolog"
)

// SearchEngine executes searches using a Cardigann definition.
type SearchEngine struct {
	def            *Definition
	templateEngine *TemplateEngine
	httpClient     *http.Client
	logger         zerolog.Logger
	baseURL        string
	userAgent      string
}

// SearchResult represents a parsed search result.
type SearchResult struct {
	Title                string
	GUID                 string
	DownloadURL          string
	InfoURL              string
	Size                 int64
	PublishDate          time.Time
	Category             string
	CategoryID           int
	Seeders              int
	Leechers             int
	Grabs                int
	InfoHash             string
	MagnetURL            string
	IMDBID               string
	TMDBID               int
	TVDBID               int
	DownloadVolumeFactor float64
	UploadVolumeFactor   float64
	MinimumRatio         float64
	MinimumSeedTime      int64
}

// NewSearchEngine creates a new search engine for a definition.
func NewSearchEngine(def *Definition, httpClient *http.Client, logger zerolog.Logger) *SearchEngine {
	return &SearchEngine{
		def:            def,
		templateEngine: NewTemplateEngine(),
		httpClient:     httpClient,
		logger:         logger.With().Str("component", "search").Str("indexer", def.ID).Logger(),
		baseURL:        def.GetBaseURL(),
		userAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}

// Search executes a search and returns parsed results.
func (e *SearchEngine) Search(ctx context.Context, query SearchQuery, settings map[string]string) ([]SearchResult, error) {
	// Merge definition defaults with user settings
	mergedSettings := e.mergeSettingsWithDefaults(settings)

	// Build template context
	tmplCtx := e.buildTemplateContext(query, mergedSettings)

	// Process keywords through filters (with template support for filter args)
	keywords := query.Query
	if len(e.def.Search.KeywordsFilters) > 0 {
		filtered, err := ApplyFiltersWithContext(keywords, e.def.Search.KeywordsFilters, e.templateEngine, tmplCtx)
		if err != nil {
			e.logger.Warn().Err(err).Msg("Failed to apply keyword filters")
		} else {
			keywords = filtered
		}
	}
	tmplCtx.Query.Keywords = keywords
	tmplCtx.Keywords = keywords // Keep top-level in sync

	var allResults []SearchResult

	// Execute search for each applicable path
	for _, searchPath := range e.def.Search.Paths {
		// Check if this path applies to the requested categories
		if !e.pathMatchesCategories(searchPath, query.Categories) {
			continue
		}

		results, err := e.executeSearchPath(ctx, searchPath, tmplCtx)
		if err != nil {
			e.logger.Error().Err(err).Str("path", searchPath.Path).Msg("Search path failed")
			continue
		}

		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// buildTemplateContext creates a template context from the search query.
func (e *SearchEngine) buildTemplateContext(query SearchQuery, settings map[string]string) *TemplateContext {
	ctx := NewTemplateContext()
	ctx.Config = settings
	ctx.Categories = e.mapCategoriesToIndexer(query.Categories)
	ctx.Keywords = query.Query // Top-level for Cardigann compatibility

	ctx.Query = QueryContext{
		Q:        query.Query,
		Keywords: query.Query,
		Year:     query.Year,
		Season:   query.Season,
		Ep:       query.Episode,
		Episode:  query.Episode,
		IMDBID:   query.IMDBID,
		TMDBID:   query.TMDBID,
		TVDBID:   query.TVDBID,
		Album:    query.Album,
		Artist:   query.Artist,
		Author:   query.Author,
		Title:    query.Title,
		Limit:    query.Limit,
		Offset:   query.Offset,
	}

	// Handle IMDB ID variants
	if query.IMDBID != "" {
		if strings.HasPrefix(query.IMDBID, "tt") {
			ctx.Query.IMDBID = query.IMDBID
			ctx.Query.IMDBIDShort = strings.TrimPrefix(query.IMDBID, "tt")
		} else {
			ctx.Query.IMDBIDShort = query.IMDBID
			ctx.Query.IMDBID = "tt" + query.IMDBID
		}
	}

	return ctx
}

// mergeSettingsWithDefaults combines user settings with definition defaults.
func (e *SearchEngine) mergeSettingsWithDefaults(settings map[string]string) map[string]string {
	merged := make(map[string]string)

	// First, apply defaults from definition
	for _, setting := range e.def.Settings {
		// For checkbox types, use empty string for false (Go templates treat non-empty strings as truthy)
		// Only set to "true" if user explicitly enabled it
		if setting.Type == "checkbox" {
			if val, ok := settings[setting.Name]; ok && val == "true" {
				merged[setting.Name] = "true"
			}
			// Don't set anything for false - empty/missing is falsy in Go templates
		} else if setting.Default != "" {
			merged[setting.Name] = setting.Default
		}
	}

	// Then override with user-provided settings (for non-checkbox types)
	for k, v := range settings {
		// Skip checkboxes as they're already handled above
		isCheckbox := false
		for _, s := range e.def.Settings {
			if s.Name == k && s.Type == "checkbox" {
				isCheckbox = true
				break
			}
		}
		if !isCheckbox {
			merged[k] = v
		}
	}

	return merged
}

// newznabCategoryNames maps Newznab category IDs to their names.
var newznabCategoryNames = map[string]string{
	// Movies
	"2000": "Movies",
	"2010": "Movies/Foreign",
	"2020": "Movies/Other",
	"2030": "Movies/SD",
	"2040": "Movies/HD",
	"2045": "Movies/UHD",
	"2050": "Movies/BluRay",
	"2060": "Movies/3D",
	"2070": "Movies/DVD",
	"2080": "Movies/WEB-DL",
	// TV
	"5000": "TV",
	"5010": "TV/WEB-DL",
	"5020": "TV/Foreign",
	"5030": "TV/SD",
	"5040": "TV/HD",
	"5045": "TV/UHD",
	"5050": "TV/Other",
	"5060": "TV/Sport",
	"5070": "TV/Anime",
	"5080": "TV/Documentary",
}

// mapCategoriesToIndexer converts Newznab category IDs to indexer-native IDs.
func (e *SearchEngine) mapCategoriesToIndexer(newznabCategories []string) []string {
	if len(newznabCategories) == 0 {
		return nil
	}

	// Build reverse mapping: Newznab cat name -> indexer ID
	catNameToIndexerID := make(map[string]string)
	for _, mapping := range e.def.Caps.CategoryMappings {
		catNameToIndexerID[mapping.Cat] = mapping.ID
	}

	// Map each Newznab category
	var indexerCategories []string
	seen := make(map[string]bool)

	for _, nzCat := range newznabCategories {
		// Get the Newznab category name
		catName, ok := newznabCategoryNames[nzCat]
		if !ok {
			continue
		}

		// Try exact match first
		if indexerID, ok := catNameToIndexerID[catName]; ok {
			if !seen[indexerID] {
				indexerCategories = append(indexerCategories, indexerID)
				seen[indexerID] = true
			}
			continue
		}

		// Try parent category match (e.g., "Movies/HD" -> "Movies")
		if idx := strings.Index(catName, "/"); idx > 0 {
			parentCat := catName[:idx]
			// Find all indexer categories that start with this parent
			for mappingCat, indexerID := range catNameToIndexerID {
				if strings.HasPrefix(mappingCat, parentCat) && !seen[indexerID] {
					indexerCategories = append(indexerCategories, indexerID)
					seen[indexerID] = true
				}
			}
		}
	}

	return indexerCategories
}

// pathMatchesCategories checks if a search path applies to the requested categories.
func (e *SearchEngine) pathMatchesCategories(path SearchPath, categories []string) bool {
	// If no categories specified on path, it applies to all
	if len(path.Categories) == 0 {
		return true
	}

	// If no categories in query, use default path
	if len(categories) == 0 {
		return true
	}

	// Check for overlap
	for _, queryCat := range categories {
		for _, pathCat := range path.Categories {
			if queryCat == pathCat {
				return true
			}
		}
	}

	return false
}

// executeSearchPath executes a single search path and returns results.
func (e *SearchEngine) executeSearchPath(ctx context.Context, path SearchPath, tmplCtx *TemplateContext) ([]SearchResult, error) {
	// Build search URL
	searchURL, err := e.buildSearchURL(path, tmplCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build search URL: %w", err)
	}

	e.logger.Debug().Str("url", searchURL).Msg("Executing search")

	// Create request
	method := "GET"
	if path.Method != "" {
		method = strings.ToUpper(path.Method)
	}

	var req *http.Request
	if method == "POST" {
		formData := url.Values{}
		for key, tmpl := range e.def.Search.Inputs {
			val, _ := e.templateEngine.Evaluate(tmpl, tmplCtx)
			formData.Set(key, val)
		}
		req, err = http.NewRequestWithContext(ctx, method, searchURL, strings.NewReader(formData.Encode()))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, err = http.NewRequestWithContext(ctx, method, searchURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
	}

	req.Header.Set("User-Agent", e.userAgent)

	// Add custom headers
	for key, val := range e.def.Search.Headers {
		evaluated, _ := e.templateEngine.Evaluate(string(val), tmplCtx)
		req.Header.Set(key, evaluated)
	}

	// Execute request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("search returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response based on type
	responseType := "html"
	if path.Response != nil && path.Response.Type != "" {
		responseType = strings.ToLower(path.Response.Type)
	}

	switch responseType {
	case "json":
		return e.parseJSONResponse(body, tmplCtx)
	default:
		return e.parseHTMLResponse(body, tmplCtx)
	}
}

// buildSearchURL constructs the search URL with query parameters.
func (e *SearchEngine) buildSearchURL(path SearchPath, tmplCtx *TemplateContext) (string, error) {
	// Evaluate path template
	pathStr, err := e.templateEngine.Evaluate(path.Path, tmplCtx)
	if err != nil {
		return "", err
	}

	// Build base URL - ensure exactly one slash between base and path
	baseURL := strings.TrimSuffix(e.baseURL, "/")
	pathStr = strings.TrimPrefix(pathStr, "/")
	searchURL := baseURL + "/" + pathStr

	// Parse URL
	u, err := url.Parse(searchURL)
	if err != nil {
		return "", err
	}

	// Add query parameters from inputs
	q := u.Query()

	// Combine definition inputs with path-specific inputs
	allInputs := make(map[string]string)
	for k, v := range e.def.Search.Inputs {
		allInputs[k] = v
	}
	for k, v := range path.Inputs {
		allInputs[k] = v
	}

	for key, tmpl := range allInputs {
		val, err := e.templateEngine.Evaluate(tmpl, tmplCtx)
		if err != nil {
			continue
		}
		// Only add non-empty values
		if val != "" {
			q.Set(key, val)
		}
	}

	u.RawQuery = q.Encode()

	return u.String(), nil
}

// parseHTMLResponse parses an HTML search response.
func (e *SearchEngine) parseHTMLResponse(body []byte, tmplCtx *TemplateContext) ([]SearchResult, error) {
	htmlSel, err := NewHTMLSelector(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Check for error selectors
	for _, errSel := range e.def.Search.Error {
		if htmlSel.Exists(errSel.Selector) {
			errMsg := "Search error"
			if errSel.Message != nil {
				if errSel.Message.Text != "" {
					errMsg = errSel.Message.Text
				} else if errSel.Message.Selector != "" {
					errMsg = htmlSel.FindText(errSel.Message.Selector)
				}
			}
			return nil, fmt.Errorf("%s", errMsg)
		}
	}

	// Extract rows
	rows := htmlSel.ExtractRows(e.def.Search.Rows)
	if len(rows) == 0 {
		e.logger.Debug().Msg("No rows found in search results")
		return nil, nil
	}

	e.logger.Debug().Int("rows", len(rows)).Msg("Found search result rows")

	var results []SearchResult

	for _, row := range rows {
		result, err := e.extractResultFromRow(row, tmplCtx)
		if err != nil {
			e.logger.Debug().Err(err).Msg("Failed to extract result from row")
			continue
		}
		if result != nil {
			results = append(results, *result)
		}
	}

	return results, nil
}

// parseJSONResponse parses a JSON search response.
func (e *SearchEngine) parseJSONResponse(body []byte, tmplCtx *TemplateContext) ([]SearchResult, error) {
	jsonSel, err := NewJSONSelector(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Get the rows array using the selector
	rowsPath := e.def.Search.Rows.Selector
	rowsData, err := jsonSel.SelectArray(rowsPath)
	if err != nil {
		e.logger.Debug().Err(err).Str("path", rowsPath).Msg("Failed to select rows")
		return nil, nil
	}

	e.logger.Debug().Int("rows", len(rowsData)).Msg("Found JSON result rows")

	var results []SearchResult

	for i, rowData := range rowsData {
		// Skip header rows
		if i < e.def.Search.Rows.After {
			continue
		}

		result, err := e.extractResultFromJSON(rowData, tmplCtx)
		if err != nil {
			e.logger.Debug().Err(err).Msg("Failed to extract result from JSON row")
			continue
		}
		if result != nil {
			results = append(results, *result)
		}
	}

	return results, nil
}

// extractResultFromRow extracts a SearchResult from an HTML row.
func (e *SearchEngine) extractResultFromRow(row *goquery.Selection, tmplCtx *TemplateContext) (*SearchResult, error) {
	result := &SearchResult{
		DownloadVolumeFactor: 1,
		UploadVolumeFactor:   1,
	}

	// Create a local context that can store extracted values
	localCtx := *tmplCtx
	localCtx.Result = make(map[string]string)

	// Two-pass extraction: first extract fields with selectors, then fields with templates
	// This ensures that fields like "title_test" are available when "title" template is evaluated
	// Pass 1: Fields with selectors (no .Result references in templates)
	for fieldName, fieldDef := range e.def.Search.Fields {
		// Skip fields that use templates referencing .Result - process in second pass
		if fieldDef.Text != "" && strings.Contains(fieldDef.Text, ".Result") {
			continue
		}
		val, err := ExtractField(row, fieldDef, &localCtx)
		if err != nil {
			if !fieldDef.Optional {
				return nil, fmt.Errorf("failed to extract %s: %w", fieldName, err)
			}
			continue
		}
		localCtx.Result[fieldName] = val
		e.mapFieldToResult(result, fieldName, val)
	}

	// Pass 2: Fields with templates that reference .Result
	for fieldName, fieldDef := range e.def.Search.Fields {
		if fieldDef.Text == "" || !strings.Contains(fieldDef.Text, ".Result") {
			continue
		}
		val, err := ExtractField(row, fieldDef, &localCtx)
		if err != nil {
			if !fieldDef.Optional {
				return nil, fmt.Errorf("failed to extract %s: %w", fieldName, err)
			}
			continue
		}
		localCtx.Result[fieldName] = val
		e.mapFieldToResult(result, fieldName, val)
	}

	// Validate required fields
	if result.Title == "" {
		return nil, fmt.Errorf("missing title")
	}
	if result.DownloadURL == "" && result.MagnetURL == "" && result.InfoHash == "" {
		return nil, fmt.Errorf("missing download URL")
	}

	// Generate GUID if not present
	if result.GUID == "" {
		result.GUID = result.DownloadURL
		if result.GUID == "" {
			result.GUID = result.InfoHash
		}
	}

	// Resolve relative URLs
	result.DownloadURL = e.resolveURL(result.DownloadURL)
	result.InfoURL = e.resolveURL(result.InfoURL)

	return result, nil
}

// extractResultFromJSON extracts a SearchResult from a JSON row.
func (e *SearchEngine) extractResultFromJSON(rowData interface{}, tmplCtx *TemplateContext) (*SearchResult, error) {
	result := &SearchResult{
		DownloadVolumeFactor: 1,
		UploadVolumeFactor:   1,
	}

	// Create a local context
	localCtx := *tmplCtx
	localCtx.Result = make(map[string]string)

	// Two-pass extraction: first extract fields with selectors, then fields with templates
	// This ensures that fields like "title_test" are available when "title" template is evaluated
	// Pass 1: Fields with selectors (no .Result references in templates)
	for fieldName, fieldDef := range e.def.Search.Fields {
		// Skip fields that use templates referencing .Result - process in second pass
		if fieldDef.Text != "" && strings.Contains(fieldDef.Text, ".Result") {
			continue
		}
		val, err := ExtractJSONField(rowData, fieldDef, &localCtx)
		if err != nil {
			if !fieldDef.Optional {
				return nil, fmt.Errorf("failed to extract %s: %w", fieldName, err)
			}
			continue
		}
		localCtx.Result[fieldName] = val
		e.mapFieldToResult(result, fieldName, val)
	}

	// Pass 2: Fields with templates that reference .Result
	for fieldName, fieldDef := range e.def.Search.Fields {
		if fieldDef.Text == "" || !strings.Contains(fieldDef.Text, ".Result") {
			continue
		}
		val, err := ExtractJSONField(rowData, fieldDef, &localCtx)
		if err != nil {
			if !fieldDef.Optional {
				return nil, fmt.Errorf("failed to extract %s: %w", fieldName, err)
			}
			continue
		}
		localCtx.Result[fieldName] = val
		e.mapFieldToResult(result, fieldName, val)
	}

	// Validate and process same as HTML
	if result.Title == "" {
		return nil, fmt.Errorf("missing title")
	}
	if result.DownloadURL == "" && result.MagnetURL == "" && result.InfoHash == "" {
		return nil, fmt.Errorf("missing download URL")
	}

	if result.GUID == "" {
		result.GUID = result.DownloadURL
		if result.GUID == "" {
			result.GUID = result.InfoHash
		}
	}

	result.DownloadURL = e.resolveURL(result.DownloadURL)
	result.InfoURL = e.resolveURL(result.InfoURL)

	return result, nil
}

// mapFieldToResult maps an extracted field value to the result struct.
func (e *SearchEngine) mapFieldToResult(result *SearchResult, fieldName, value string) {
	switch strings.ToLower(fieldName) {
	case "title":
		result.Title = value
	case "download":
		result.DownloadURL = value
	case "details", "comments", "info":
		result.InfoURL = value
	case "size":
		result.Size = parseSize(value)
	case "date", "publish_date", "publishdate":
		result.PublishDate = parseDate(value)
	case "seeders":
		result.Seeders = parseInt(value)
	case "leechers", "peers":
		result.Leechers = parseInt(value)
	case "grabs", "snatched":
		result.Grabs = parseInt(value)
	case "category", "cat":
		result.Category = value
		result.CategoryID = parseInt(value)
	case "infohash":
		result.InfoHash = value
	case "magnet", "magneturl", "magnet_url":
		result.MagnetURL = value
	case "imdb", "imdbid":
		result.IMDBID = value
	case "tmdb", "tmdbid":
		result.TMDBID = parseInt(value)
	case "tvdb", "tvdbid":
		result.TVDBID = parseInt(value)
	case "downloadvolumefactor", "freeleech":
		result.DownloadVolumeFactor = parseFloat(value)
	case "uploadvolumefactor":
		result.UploadVolumeFactor = parseFloat(value)
	case "minimumratio":
		result.MinimumRatio = parseFloat(value)
	case "minimumseedtime":
		result.MinimumSeedTime = parseInt64(value)
	case "guid":
		result.GUID = value
	}
}

// resolveURL resolves a potentially relative URL against the base URL.
func (e *SearchEngine) resolveURL(urlStr string) string {
	if urlStr == "" {
		return ""
	}
	if strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://") {
		return urlStr
	}
	if strings.HasPrefix(urlStr, "magnet:") {
		return urlStr
	}
	if strings.HasPrefix(urlStr, "/") {
		return e.baseURL + urlStr
	}
	return e.baseURL + "/" + urlStr
}

// SearchQuery represents search parameters.
type SearchQuery struct {
	Query      string
	Type       string // search, tvsearch, movie
	Categories []string
	Year       int
	Season     int
	Episode    int
	IMDBID     string
	TMDBID     int
	TVDBID     int
	Album      string
	Artist     string
	Author     string
	Title      string
	Limit      int
	Offset     int
}

// Helper functions

func parseSize(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	// First try to parse as direct bytes
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}

	// Otherwise use the size filter
	result, _ := filterSize(s, nil)
	n, _ := strconv.ParseInt(result, 10, 64)
	return n
}

func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}

	// Try RFC3339 first (common for API responses)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}

	// Try common formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		"Jan 02 2006",
		"Jan 2 2006",
		"02 Jan 2006",
		"January 2, 2006",
		time.RFC1123,
		time.RFC1123Z,
	}

	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}

	return time.Time{}
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	n, _ := strconv.Atoi(s)
	return n
}

func parseInt64(s string) int64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
