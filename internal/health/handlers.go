package health

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
)

// DownloaderService defines the interface for the downloader service.
type DownloaderService interface {
	Test(ctx context.Context, id int64) (TestResult, error)
}

// TestResult represents a test result with success and message.
type TestResult interface {
	GetSuccess() bool
	GetMessage() string
}

// IndexerService defines the interface for the indexer service.
type IndexerService interface {
	Test(ctx context.Context, id int64) (TestResult, error)
}

// RootFolderService defines the interface for the root folder service.
type RootFolderService interface {
	GetPath(ctx context.Context, id int64) (string, error)
}

// MetadataService defines the interface for the metadata service.
type MetadataService interface {
	IsTMDBConfigured() bool
	IsTVDBConfigured() bool
}

// Handlers provides HTTP handlers for health endpoints.
type Handlers struct {
	health       *Service
	testFuncs    *TestFunctions
	fsChecker    *FilesystemChecker
}

// TestFunctions holds the test functions for each category.
type TestFunctions struct {
	TestDownloadClient func(ctx context.Context, id int64) (success bool, message string)
	TestIndexer        func(ctx context.Context, id int64) (success bool, message string)
	GetRootFolderPath  func(ctx context.Context, id int64) (string, error)
	IsTMDBConfigured   func() bool
	IsTVDBConfigured   func() bool
	TestTMDB           func(ctx context.Context) error
	TestTVDB           func(ctx context.Context) error
}

// NewHandlers creates new health handlers with test functions.
func NewHandlers(health *Service, testFuncs *TestFunctions) *Handlers {
	return &Handlers{
		health:    health,
		testFuncs: testFuncs,
		fsChecker: NewFilesystemChecker(),
	}
}

// RegisterRoutes registers health routes.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("", h.GetAll)
	g.GET("/summary", h.GetSummary)
	g.GET("/:category", h.GetByCategory)
	g.POST("/:category/test", h.TestCategory)
	g.POST("/:category/:id/test", h.TestItem)
}

// GetAll returns all health items grouped by category.
// GET /api/v1/health
func (h *Handlers) GetAll(c echo.Context) error {
	return c.JSON(http.StatusOK, h.health.GetAll())
}

// GetSummary returns summary counts for the dashboard.
// GET /api/v1/health/summary
func (h *Handlers) GetSummary(c echo.Context) error {
	return c.JSON(http.StatusOK, h.health.GetSummary())
}

// GetByCategory returns health items for a specific category.
// GET /api/v1/health/:category
func (h *Handlers) GetByCategory(c echo.Context) error {
	categoryStr := c.Param("category")
	category := HealthCategory(categoryStr)

	// Validate category
	valid := false
	for _, cat := range AllCategories() {
		if cat == category {
			valid = true
			break
		}
	}
	if !valid {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid health category")
	}

	items := h.health.GetByCategory(category)
	return c.JSON(http.StatusOK, items)
}

// TestCategory tests all items in a category.
// POST /api/v1/health/:category/test
func (h *Handlers) TestCategory(c echo.Context) error {
	ctx := c.Request().Context()
	categoryStr := c.Param("category")
	category := HealthCategory(categoryStr)

	items := h.health.GetByCategory(category)
	if len(items) == 0 {
		return c.JSON(http.StatusOK, map[string]string{"message": "no items to test"})
	}

	// Test items sequentially to avoid overwhelming external services
	results := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		result := h.testSingleItem(ctx, category, item.ID)
		results = append(results, result)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"category": category,
		"results":  results,
	})
}

// TestItem tests a specific health item.
// POST /api/v1/health/:category/:id/test
func (h *Handlers) TestItem(c echo.Context) error {
	ctx := c.Request().Context()
	categoryStr := c.Param("category")
	id := c.Param("id")
	category := HealthCategory(categoryStr)

	item := h.health.GetItem(category, id)
	if item == nil {
		return echo.NewHTTPError(http.StatusNotFound, "health item not found")
	}

	result := h.testSingleItem(ctx, category, id)
	return c.JSON(http.StatusOK, result)
}

// testSingleItem tests a single health item and updates its status.
func (h *Handlers) testSingleItem(ctx context.Context, category HealthCategory, id string) map[string]interface{} {
	result := map[string]interface{}{
		"id":      id,
		"success": false,
		"message": "",
	}

	switch category {
	case CategoryDownloadClients:
		result = h.testDownloadClient(ctx, id)
	case CategoryIndexers:
		result = h.testIndexer(ctx, id)
	case CategoryRootFolders:
		result = h.testRootFolder(ctx, id)
	case CategoryMetadata:
		result = h.testMetadataProvider(ctx, id)
	case CategoryStorage:
		result = h.testStorage(ctx, id)
	default:
		result["message"] = "unsupported category"
	}

	return result
}

func (h *Handlers) testDownloadClient(ctx context.Context, id string) map[string]interface{} {
	result := map[string]interface{}{
		"id":      id,
		"success": false,
		"message": "",
	}

	if h.testFuncs == nil || h.testFuncs.TestDownloadClient == nil {
		result["message"] = "download client testing not configured"
		return result
	}

	clientID, err := parseInt64(id)
	if err != nil {
		result["message"] = "invalid client ID"
		return result
	}

	success, message := h.testFuncs.TestDownloadClient(ctx, clientID)
	if success {
		h.health.ClearStatus(CategoryDownloadClients, id)
		result["success"] = true
		result["message"] = "Connection verified"
	} else {
		h.health.SetError(CategoryDownloadClients, id, message)
		result["message"] = message
	}

	return result
}

func (h *Handlers) testIndexer(ctx context.Context, id string) map[string]interface{} {
	result := map[string]interface{}{
		"id":      id,
		"success": false,
		"message": "",
	}

	if h.testFuncs == nil || h.testFuncs.TestIndexer == nil {
		result["message"] = "indexer testing not configured"
		return result
	}

	indexerID, err := parseInt64(id)
	if err != nil {
		result["message"] = "invalid indexer ID"
		return result
	}

	success, message := h.testFuncs.TestIndexer(ctx, indexerID)
	if success {
		h.health.ClearStatus(CategoryIndexers, id)
		result["success"] = true
		result["message"] = "Connection verified"
	} else {
		h.health.SetError(CategoryIndexers, id, message)
		result["message"] = message
	}

	return result
}

func (h *Handlers) testRootFolder(ctx context.Context, id string) map[string]interface{} {
	result := map[string]interface{}{
		"id":      id,
		"success": false,
		"message": "",
	}

	if h.testFuncs == nil || h.testFuncs.GetRootFolderPath == nil {
		result["message"] = "root folder testing not configured"
		return result
	}

	folderID, err := parseInt64(id)
	if err != nil {
		result["message"] = "invalid folder ID"
		return result
	}

	path, err := h.testFuncs.GetRootFolderPath(ctx, folderID)
	if err != nil {
		h.health.SetError(CategoryRootFolders, id, err.Error())
		result["message"] = err.Error()
		return result
	}

	if path == "" {
		result["message"] = "could not determine folder path"
		return result
	}

	// Check folder health
	ok, message := h.fsChecker.CheckFolderHealth(path)
	if !ok {
		h.health.SetError(CategoryRootFolders, id, message)
		result["message"] = message
		return result
	}

	h.health.ClearStatus(CategoryRootFolders, id)
	result["success"] = true
	result["message"] = "Folder is accessible and writable"
	return result
}

func (h *Handlers) testMetadataProvider(ctx context.Context, id string) map[string]interface{} {
	result := map[string]interface{}{
		"id":      id,
		"success": false,
		"message": "",
	}

	if h.testFuncs == nil {
		result["message"] = "metadata testing not configured"
		return result
	}

	switch id {
	case "tmdb":
		if h.testFuncs.IsTMDBConfigured == nil || !h.testFuncs.IsTMDBConfigured() {
			result["message"] = "TMDB not configured"
			return result
		}
		if h.testFuncs.TestTMDB == nil {
			result["message"] = "TMDB test not configured"
			return result
		}
		if err := h.testFuncs.TestTMDB(ctx); err != nil {
			h.health.SetError(CategoryMetadata, id, err.Error())
			result["message"] = err.Error()
			return result
		}
		h.health.ClearStatus(CategoryMetadata, id)
		result["success"] = true
		result["message"] = "API connection verified"

	case "tvdb":
		if h.testFuncs.IsTVDBConfigured == nil || !h.testFuncs.IsTVDBConfigured() {
			result["message"] = "TVDB not configured"
			return result
		}
		if h.testFuncs.TestTVDB == nil {
			result["message"] = "TVDB test not configured"
			return result
		}
		if err := h.testFuncs.TestTVDB(ctx); err != nil {
			h.health.SetError(CategoryMetadata, id, err.Error())
			result["message"] = err.Error()
			return result
		}
		h.health.ClearStatus(CategoryMetadata, id)
		result["success"] = true
		result["message"] = "API connection verified"

	default:
		result["message"] = "unknown metadata provider"
	}

	return result
}

func (h *Handlers) testStorage(ctx context.Context, id string) map[string]interface{} {
	result := map[string]interface{}{
		"id":      id,
		"success": true,
		"message": "Storage is checked automatically on schedule",
	}

	// Storage health is monitored by scheduled task checking disk space thresholds.
	// Manual testing would duplicate the scheduled check logic.
	// The current status reflects the last scheduled check.
	return result
}

// parseInt64 parses a string to int64.
func parseInt64(s string) (int64, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, echo.NewHTTPError(http.StatusBadRequest, "invalid number")
		}
		n = n*10 + int64(c-'0')
	}
	return n, nil
}
