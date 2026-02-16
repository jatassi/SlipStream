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

const (
	searchTypeMovie = "movie"
)

// SearchEngine executes searches using a Cardigann definition.
type SearchEngine struct {
	def            *Definition
	templateEngine *TemplateEngine
	httpClient     *http.Client
	logger         *zerolog.Logger
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
func NewSearchEngine(def *Definition, httpClient *http.Client, logger *zerolog.Logger) *SearchEngine {
	subLogger := logger.With().Str("component", "search").Str("indexer", def.ID).Logger()
	return &SearchEngine{
		def:            def,
		templateEngine: NewTemplateEngine(),
		httpClient:     httpClient,
		logger:         &subLogger,
		baseURL:        def.GetBaseURL(),
		userAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}

// Search executes a search and returns parsed results.
func (e *SearchEngine) Search(ctx context.Context, query *SearchQuery, settings map[string]string) ([]SearchResult, error) {
	mergedSettings := e.mergeSettingsWithDefaults(settings)
	e.logSettingsDebug(mergedSettings, len(settings))

	tmplCtx := e.buildTemplateContext(query, mergedSettings)

	keywords := e.resolveKeywords(query, tmplCtx)
	tmplCtx.Query.Keywords = keywords
	tmplCtx.Keywords = keywords

	var allResults []SearchResult
	for i := range e.def.Search.Paths {
		if !e.pathMatchesCategories(&e.def.Search.Paths[i], query.Categories) {
			continue
		}
		results, err := e.executeSearchPath(ctx, &e.def.Search.Paths[i], tmplCtx)
		if err != nil {
			e.logger.Error().Err(err).Str("path", e.def.Search.Paths[i].Path).Msg("Search path failed")
			continue
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

func (e *SearchEngine) logSettingsDebug(mergedSettings map[string]string, inputCount int) {
	e.logger.Debug().Int("inputSettings", inputCount).Int("mergedSettings", len(mergedSettings)).Msg("Search settings")
	if apikey, ok := mergedSettings["apikey"]; ok {
		e.logger.Debug().Str("apikey", apikey[:min(10, len(apikey))]+"...").Msg("API key found in merged settings")
	} else {
		e.logger.Warn().Msg("API key NOT found in merged settings")
	}
}

func (e *SearchEngine) resolveKeywords(query *SearchQuery, tmplCtx *TemplateContext) string {
	if query.Type == searchTypeMovie && query.TMDBID > 0 {
		e.logger.Debug().Int("tmdbId", query.TMDBID).Msg("Clearing keywords for TMDB-based movie search")
		return ""
	}
	if query.Type == searchTypeMovie && query.IMDBID != "" {
		e.logger.Debug().Str("imdbId", query.IMDBID).Msg("Clearing keywords for IMDB-based movie search")
		return ""
	}
	if len(e.def.Search.KeywordsFilters) > 0 {
		filtered, err := ApplyFiltersWithContext(query.Query, e.def.Search.KeywordsFilters, e.templateEngine, tmplCtx)
		if err != nil {
			e.logger.Warn().Err(err).Msg("Failed to apply keyword filters")
		} else {
			return filtered
		}
	}
	return query.Query
}

// buildTemplateContext creates a template context from the search query.
func (e *SearchEngine) buildTemplateContext(query *SearchQuery, settings map[string]string) *TemplateContext {
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

	for _, setting := range e.def.Settings {
		if setting.Type == "checkbox" {
			if val, ok := settings[setting.Name]; ok && val == "true" {
				merged[setting.Name] = "true"
			}
		} else if setting.Default != "" {
			merged[setting.Name] = setting.Default
		}
	}

	for k, v := range settings {
		if !e.isCheckboxSetting(k) {
			merged[k] = v
		}
	}

	return merged
}

func (e *SearchEngine) isCheckboxSetting(name string) bool {
	for _, s := range e.def.Settings {
		if s.Name == name && s.Type == "checkbox" {
			return true
		}
	}
	return false
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

// buildCategoryNameToIDMap creates reverse mapping from Newznab category names to indexer IDs
func (e *SearchEngine) buildCategoryNameToIDMap() map[string]string {
	catNameToIndexerID := make(map[string]string)
	for _, mapping := range e.def.Caps.CategoryMappings {
		catNameToIndexerID[mapping.Cat] = mapping.ID
	}
	return catNameToIndexerID
}

// mapNewznabCategory maps a single Newznab category to indexer categories
func (e *SearchEngine) mapNewznabCategory(nzCat string, catNameToIndexerID map[string]string, seen map[string]bool) []string {
	catName, ok := newznabCategoryNames[nzCat]
	if !ok {
		return nil
	}

	// Try exact match
	if indexerID, ok := catNameToIndexerID[catName]; ok {
		if !seen[indexerID] {
			seen[indexerID] = true
			return []string{indexerID}
		}
		return nil
	}

	// Try parent category match
	return e.mapParentCategory(catName, catNameToIndexerID, seen)
}

// mapParentCategory maps parent category to all matching subcategories
func (e *SearchEngine) mapParentCategory(catName string, catNameToIndexerID map[string]string, seen map[string]bool) []string {
	idx := strings.Index(catName, "/")
	if idx <= 0 {
		return nil
	}

	parentCat := catName[:idx]
	var result []string

	for mappingCat, indexerID := range catNameToIndexerID {
		if strings.HasPrefix(mappingCat, parentCat) && !seen[indexerID] {
			result = append(result, indexerID)
			seen[indexerID] = true
		}
	}

	return result
}

// mapCategoriesToIndexer converts Newznab category IDs to indexer-native IDs.
func (e *SearchEngine) mapCategoriesToIndexer(newznabCategories []string) []string {
	if len(newznabCategories) == 0 {
		return nil
	}

	catNameToIndexerID := e.buildCategoryNameToIDMap()
	var indexerCategories []string
	seen := make(map[string]bool)

	for _, nzCat := range newznabCategories {
		mapped := e.mapNewznabCategory(nzCat, catNameToIndexerID, seen)
		indexerCategories = append(indexerCategories, mapped...)
	}

	return indexerCategories
}

// pathMatchesCategories checks if a search path applies to the requested categories.
func (e *SearchEngine) pathMatchesCategories(path *SearchPath, categories []string) bool {
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

// createSearchRequest builds an HTTP request for search
func (e *SearchEngine) createSearchRequest(ctx context.Context, method, searchURL string, _path *SearchPath, tmplCtx *TemplateContext) (*http.Request, error) {
	var req *http.Request
	var err error

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
		req, err = http.NewRequestWithContext(ctx, method, searchURL, http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
	}

	req.Header.Set("User-Agent", e.userAgent)
	e.setSearchHeaders(req, tmplCtx)

	return req, nil
}

// setSearchHeaders applies custom headers from definition
func (e *SearchEngine) setSearchHeaders(req *http.Request, tmplCtx *TemplateContext) {
	for key, val := range e.def.Search.Headers {
		evaluated, err := e.templateEngine.Evaluate(string(val), tmplCtx)
		if err != nil {
			e.logger.Warn().Err(err).Str("header", key).Str("template", string(val)).Msg("Failed to evaluate header template")
		}
		e.logger.Debug().Str("header", key).Str("template", string(val)).Str("evaluated", evaluated).Msg("Setting search header")
		req.Header.Set(key, evaluated)
	}
}

// executeSearchRequest performs the HTTP request and reads response
func (e *SearchEngine) executeSearchRequest(req *http.Request) ([]byte, error) {
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

	return body, nil
}

// executeSearchPath executes a single search path and returns results.
func (e *SearchEngine) executeSearchPath(ctx context.Context, path *SearchPath, tmplCtx *TemplateContext) ([]SearchResult, error) {
	searchURL, err := e.buildSearchURL(path, tmplCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build search URL: %w", err)
	}

	e.logger.Debug().Str("url", searchURL).Msg("Executing search")

	method := "GET"
	if path.Method != "" {
		method = strings.ToUpper(path.Method)
	}

	req, err := e.createSearchRequest(ctx, method, searchURL, path, tmplCtx)
	if err != nil {
		return nil, err
	}

	body, err := e.executeSearchRequest(req)
	if err != nil {
		return nil, err
	}

	responseType := "html"
	if path.Response != nil && path.Response.Type != "" {
		responseType = strings.ToLower(path.Response.Type)
	}

	if responseType == "json" {
		return e.parseJSONResponse(body, tmplCtx)
	}
	return e.parseHTMLResponse(body, tmplCtx)
}

// buildSearchURL constructs the search URL with query parameters.
func (e *SearchEngine) buildSearchURL(path *SearchPath, tmplCtx *TemplateContext) (string, error) {
	pathStr, err := e.templateEngine.Evaluate(path.Path, tmplCtx)
	if err != nil {
		return "", err
	}

	baseURL := strings.TrimSuffix(e.baseURL, "/")
	pathStr = strings.TrimPrefix(pathStr, "/")

	u, err := url.Parse(baseURL + "/" + pathStr)
	if err != nil {
		return "", err
	}

	allInputs := e.combineInputs(path)
	q, rawQuery := e.evaluateInputs(allInputs, tmplCtx, u.Query())

	u.RawQuery = q.Encode()
	if rawQuery != "" {
		u.RawQuery += rawQuery
	}

	return u.String(), nil
}

func (e *SearchEngine) combineInputs(path *SearchPath) map[string]string {
	allInputs := make(map[string]string)
	for k, v := range e.def.Search.Inputs {
		allInputs[k] = v
	}
	for k, v := range path.Inputs {
		allInputs[k] = v
	}
	return allInputs
}

func (e *SearchEngine) evaluateInputs(allInputs map[string]string, tmplCtx *TemplateContext, q url.Values) (values url.Values, rawQuery string) {
	for key, tmpl := range allInputs {
		val, err := e.templateEngine.Evaluate(tmpl, tmplCtx)
		if err != nil {
			continue
		}
		if key == "$raw" {
			rawQuery = val
			continue
		}
		if val == "" || val == "0" {
			continue
		}
		q.Set(key, val)
	}
	return q, rawQuery
}

// checkHTMLErrors looks for error selectors in HTML response
func (e *SearchEngine) checkHTMLErrors(htmlSel *HTMLSelector) error {
	for _, errSel := range e.def.Search.Error {
		if !htmlSel.Exists(errSel.Selector) {
			continue
		}

		errMsg := "Search error"
		if errSel.Message != nil {
			if errSel.Message.Text != "" {
				errMsg = errSel.Message.Text
			} else if errSel.Message.Selector != "" {
				errMsg = htmlSel.FindText(errSel.Message.Selector)
			}
		}
		return fmt.Errorf("%s", errMsg)
	}
	return nil
}

// parseHTMLResponse parses an HTML search response.
func (e *SearchEngine) parseHTMLResponse(body []byte, tmplCtx *TemplateContext) ([]SearchResult, error) {
	htmlSel, err := NewHTMLSelector(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	if err := e.checkHTMLErrors(htmlSel); err != nil {
		return nil, err
	}

	rows := htmlSel.ExtractRows(&e.def.Search.Rows)
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

	rowsData, err := jsonSel.SelectArray(e.def.Search.Rows.Selector)
	if err != nil {
		e.logger.Debug().Err(err).Str("path", e.def.Search.Rows.Selector).Msg("Failed to select rows")
		return nil, nil
	}

	e.logger.Debug().Int("rows", len(rowsData)).Msg("Found JSON result rows")

	var results []SearchResult
	for i, rowData := range rowsData {
		if i < e.def.Search.Rows.After {
			continue
		}

		actualRowData := e.extractJSONRowData(rowData)
		if actualRowData == nil {
			continue
		}

		result, err := e.extractResultFromJSON(actualRowData, tmplCtx)
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

// extractJSONRowData extracts the actual row data, handling nested attributes
func (e *SearchEngine) extractJSONRowData(rowData interface{}) interface{} {
	if e.def.Search.Rows.Attribute == "" {
		return rowData
	}

	rowMap, ok := rowData.(map[string]interface{})
	if !ok {
		return rowData
	}

	attrData, ok := rowMap[e.def.Search.Rows.Attribute]
	if !ok {
		e.logger.Debug().Str("attribute", e.def.Search.Rows.Attribute).Msg("Row missing attribute field")
		return nil
	}

	return attrData
}

// extractFields performs two-pass field extraction to handle .Result template references
func (e *SearchEngine) extractFields(fields map[string]Field, extractFunc func(Field, *TemplateContext) (string, error), localCtx *TemplateContext) error {
	if err := e.extractFieldsPass(fields, extractFunc, localCtx, false); err != nil {
		return err
	}
	return e.extractFieldsPass(fields, extractFunc, localCtx, true)
}

func (e *SearchEngine) extractFieldsPass(fields map[string]Field, extractFunc func(Field, *TemplateContext) (string, error), localCtx *TemplateContext, resultRefs bool) error {
	for fieldName, fieldDef := range fields {
		hasResultRef := fieldDef.Text != "" && strings.Contains(fieldDef.Text, ".Result")
		if hasResultRef != resultRefs {
			continue
		}
		val, err := extractFunc(fieldDef, localCtx)
		if err != nil {
			if !fieldDef.Optional {
				return fmt.Errorf("failed to extract %s: %w", fieldName, err)
			}
			continue
		}
		localCtx.Result[fieldName] = val
	}
	return nil
}

// applyFieldsToResult maps all extracted fields to result struct
func (e *SearchEngine) applyFieldsToResult(result *SearchResult, localCtx *TemplateContext) {
	for fieldName, val := range localCtx.Result {
		e.mapFieldToResult(result, fieldName, val)
	}
}

// validateResult checks that required fields are present
func (e *SearchEngine) validateResult(result *SearchResult) error {
	if result.Title == "" {
		return fmt.Errorf("missing title")
	}
	if result.DownloadURL == "" && result.MagnetURL == "" && result.InfoHash == "" {
		return fmt.Errorf("missing download URL")
	}
	return nil
}

// finalizeResult sets defaults and resolves URLs
func (e *SearchEngine) finalizeResult(result *SearchResult) {
	if result.GUID == "" {
		result.GUID = result.DownloadURL
		if result.GUID == "" {
			result.GUID = result.InfoHash
		}
	}
	result.DownloadURL = e.resolveURL(result.DownloadURL)
	result.InfoURL = e.resolveURL(result.InfoURL)
}

// extractResultFromRow extracts a SearchResult from an HTML row.
func (e *SearchEngine) extractResultFromRow(row *goquery.Selection, tmplCtx *TemplateContext) (*SearchResult, error) {
	result := &SearchResult{
		DownloadVolumeFactor: 1,
		UploadVolumeFactor:   1,
	}

	localCtx := *tmplCtx
	localCtx.Result = make(map[string]string)

	extractFunc := func(fieldDef Field, ctx *TemplateContext) (string, error) {
		return ExtractField(row, &fieldDef, ctx)
	}

	if err := e.extractFields(e.def.Search.Fields, extractFunc, &localCtx); err != nil {
		return nil, err
	}

	e.applyFieldsToResult(result, &localCtx)

	if err := e.validateResult(result); err != nil {
		return nil, err
	}

	e.finalizeResult(result)
	return result, nil
}

// extractResultFromJSON extracts a SearchResult from a JSON row.
func (e *SearchEngine) extractResultFromJSON(rowData interface{}, tmplCtx *TemplateContext) (*SearchResult, error) {
	result := &SearchResult{
		DownloadVolumeFactor: 1,
		UploadVolumeFactor:   1,
	}

	localCtx := *tmplCtx
	localCtx.Result = make(map[string]string)

	extractFunc := func(fieldDef Field, ctx *TemplateContext) (string, error) {
		return ExtractJSONField(rowData, &fieldDef, ctx)
	}

	if err := e.extractFields(e.def.Search.Fields, extractFunc, &localCtx); err != nil {
		return nil, err
	}

	e.applyFieldsToResult(result, &localCtx)

	if err := e.validateResult(result); err != nil {
		return nil, err
	}

	e.finalizeResult(result)
	return result, nil
}

type fieldMapper func(*SearchResult, string)

var fieldMappers = map[string]fieldMapper{
	"title":                func(r *SearchResult, v string) { r.Title = v },
	"download":             func(r *SearchResult, v string) { r.DownloadURL = v },
	"details":              func(r *SearchResult, v string) { r.InfoURL = v },
	"comments":             func(r *SearchResult, v string) { r.InfoURL = v },
	"info":                 func(r *SearchResult, v string) { r.InfoURL = v },
	"size":                 func(r *SearchResult, v string) { r.Size = parseSize(v) },
	"date":                 func(r *SearchResult, v string) { r.PublishDate = parseDate(v) },
	"publish_date":         func(r *SearchResult, v string) { r.PublishDate = parseDate(v) },
	"publishdate":          func(r *SearchResult, v string) { r.PublishDate = parseDate(v) },
	"seeders":              func(r *SearchResult, v string) { r.Seeders = parseInt(v) },
	"leechers":             func(r *SearchResult, v string) { r.Leechers = parseInt(v) },
	"peers":                func(r *SearchResult, v string) { r.Leechers = parseInt(v) },
	"grabs":                func(r *SearchResult, v string) { r.Grabs = parseInt(v) },
	"snatched":             func(r *SearchResult, v string) { r.Grabs = parseInt(v) },
	"category":             func(r *SearchResult, v string) { r.Category = v; r.CategoryID = parseInt(v) },
	"cat":                  func(r *SearchResult, v string) { r.Category = v; r.CategoryID = parseInt(v) },
	"infohash":             func(r *SearchResult, v string) { r.InfoHash = v },
	"magnet":               func(r *SearchResult, v string) { r.MagnetURL = v },
	"magneturl":            func(r *SearchResult, v string) { r.MagnetURL = v },
	"magnet_url":           func(r *SearchResult, v string) { r.MagnetURL = v },
	"imdb":                 func(r *SearchResult, v string) { r.IMDBID = v },
	"imdbid":               func(r *SearchResult, v string) { r.IMDBID = v },
	"tmdb":                 func(r *SearchResult, v string) { r.TMDBID = parseInt(v) },
	"tmdbid":               func(r *SearchResult, v string) { r.TMDBID = parseInt(v) },
	"tvdb":                 func(r *SearchResult, v string) { r.TVDBID = parseInt(v) },
	"tvdbid":               func(r *SearchResult, v string) { r.TVDBID = parseInt(v) },
	"downloadvolumefactor": func(r *SearchResult, v string) { r.DownloadVolumeFactor = parseFloat(v) },
	"freeleech":            func(r *SearchResult, v string) { r.DownloadVolumeFactor = parseFloat(v) },
	"uploadvolumefactor":   func(r *SearchResult, v string) { r.UploadVolumeFactor = parseFloat(v) },
	"minimumratio":         func(r *SearchResult, v string) { r.MinimumRatio = parseFloat(v) },
	"minimumseedtime":      func(r *SearchResult, v string) { r.MinimumSeedTime = parseInt64(v) },
	"guid":                 func(r *SearchResult, v string) { r.GUID = v },
}

func (e *SearchEngine) mapFieldToResult(result *SearchResult, fieldName, value string) {
	if mapper, ok := fieldMappers[strings.ToLower(fieldName)]; ok {
		mapper(result, value)
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
