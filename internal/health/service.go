package health

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Broadcaster defines the interface for sending WebSocket messages.
type Broadcaster interface {
	Broadcast(msgType string, payload interface{}) error
}

// NotificationDispatcher defines the interface for sending health notifications.
type NotificationDispatcher interface {
	DispatchHealthIssue(ctx context.Context, source, healthType, message string)
	DispatchHealthRestored(ctx context.Context, source, healthType, message string)
}

// Service manages the health state of all tracked items.
// All state is in-memory and resets on application restart.
type Service struct {
	items        map[HealthCategory]map[string]*HealthItem
	mu           sync.RWMutex
	broadcaster  Broadcaster
	notifier     NotificationDispatcher
	logger       zerolog.Logger
}

// NewService creates a new health service.
func NewService(logger zerolog.Logger) *Service {
	s := &Service{
		items:  make(map[HealthCategory]map[string]*HealthItem),
		logger: logger.With().Str("component", "health").Logger(),
	}

	// Initialize maps for all categories
	for _, cat := range AllCategories() {
		s.items[cat] = make(map[string]*HealthItem)
	}

	return s
}

// SetBroadcaster sets the WebSocket broadcaster for real-time updates.
func (s *Service) SetBroadcaster(b Broadcaster) {
	s.broadcaster = b
}

// SetNotifier sets the notification dispatcher for health alerts.
func (s *Service) SetNotifier(n NotificationDispatcher) {
	s.notifier = n
}

// RegisterItemStr is a string-based wrapper for RegisterItem.
// This allows services to use string category names without importing the health types.
func (s *Service) RegisterItemStr(category, id, name string) {
	s.RegisterItem(HealthCategory(category), id, name)
}

// UnregisterItemStr is a string-based wrapper for UnregisterItem.
func (s *Service) UnregisterItemStr(category, id string) {
	s.UnregisterItem(HealthCategory(category), id)
}

// SetErrorStr is a string-based wrapper for SetError.
func (s *Service) SetErrorStr(category, id, message string) {
	s.SetError(HealthCategory(category), id, message)
}

// SetWarningStr is a string-based wrapper for SetWarning.
func (s *Service) SetWarningStr(category, id, message string) {
	s.SetWarning(HealthCategory(category), id, message)
}

// ClearStatusStr is a string-based wrapper for ClearStatus.
func (s *Service) ClearStatusStr(category, id string) {
	s.ClearStatus(HealthCategory(category), id)
}

// RegisterItem adds a new item to health tracking with OK status.
func (s *Service) RegisterItem(category HealthCategory, id, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := &HealthItem{
		ID:       id,
		Category: category,
		Name:     name,
		Status:   StatusOK,
	}

	s.items[category][id] = item

	s.logger.Debug().
		Str("category", string(category)).
		Str("id", id).
		Str("name", name).
		Msg("Registered health item")

	s.broadcastUpdate(item)
}

// UnregisterItem removes an item from health tracking.
func (s *Service) UnregisterItem(category HealthCategory, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[category][id]; exists {
		delete(s.items[category], id)

		s.logger.Debug().
			Str("category", string(category)).
			Str("id", id).
			Msg("Unregistered health item")
	}
}

// SetError sets an item to Error status with a message.
func (s *Service) SetError(category HealthCategory, id, message string) {
	s.setStatus(category, id, StatusError, message)
}

// SetWarning sets an item to Warning status with a message.
// For binary categories (download clients, root folders), this is a no-op.
func (s *Service) SetWarning(category HealthCategory, id, message string) {
	if IsBinaryCategory(category) {
		s.logger.Debug().
			Str("category", string(category)).
			Str("id", id).
			Msg("Ignoring warning for binary health category")
		return
	}
	s.setStatus(category, id, StatusWarning, message)
}

// ClearStatus resets an item to OK status.
func (s *Service) ClearStatus(category HealthCategory, id string) {
	s.setStatus(category, id, StatusOK, "")
}

// setStatus updates the status of an item.
func (s *Service) setStatus(category HealthCategory, id string, status HealthStatus, message string) {
	s.mu.Lock()

	item, exists := s.items[category][id]
	if !exists {
		s.mu.Unlock()
		s.logger.Warn().
			Str("category", string(category)).
			Str("id", id).
			Msg("Attempted to update status for unregistered item")
		return
	}

	// Only update if status changed
	if item.Status == status && item.Message == message {
		s.mu.Unlock()
		return
	}

	oldStatus := item.Status
	item.Status = status
	item.Message = message
	itemName := item.Name

	// Set timestamp for non-OK statuses
	if status != StatusOK {
		now := time.Now()
		item.Timestamp = &now
	} else {
		item.Timestamp = nil
	}

	s.logger.Info().
		Str("category", string(category)).
		Str("id", id).
		Str("name", itemName).
		Str("oldStatus", string(oldStatus)).
		Str("newStatus", string(status)).
		Str("message", message).
		Msg("Health status changed")

	s.broadcastUpdate(item)
	s.mu.Unlock()

	// Dispatch health notifications (outside lock to avoid blocking)
	if s.notifier != nil {
		source := string(category) + ": " + itemName
		healthType := "error"
		if status == StatusWarning {
			healthType = "warning"
		}

		// Status transition: OK -> Error/Warning = issue detected
		if oldStatus == StatusOK && (status == StatusError || status == StatusWarning) {
			s.notifier.DispatchHealthIssue(context.Background(), source, healthType, message)
		}

		// Status transition: Error/Warning -> OK = issue resolved
		if (oldStatus == StatusError || oldStatus == StatusWarning) && status == StatusOK {
			s.notifier.DispatchHealthRestored(context.Background(), source, healthType, "Issue resolved")
		}
	}
}

// GetAll returns all health items grouped by category.
func (s *Service) GetAll() *HealthResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp := &HealthResponse{
		DownloadClients: s.itemsToSlice(CategoryDownloadClients),
		Indexers:        s.itemsToSlice(CategoryIndexers),
		RootFolders:     s.itemsToSlice(CategoryRootFolders),
		Metadata:        s.itemsToSlice(CategoryMetadata),
		Storage:         s.itemsToSlice(CategoryStorage),
		Import:          s.itemsToSlice(CategoryImport),
	}

	return resp
}

// GetByCategory returns all items in a specific category.
func (s *Service) GetByCategory(category HealthCategory) []HealthItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.itemsToSlice(category)
}

// GetItem returns a single item by category and ID.
func (s *Service) GetItem(category HealthCategory, id string) *HealthItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if item, exists := s.items[category][id]; exists {
		copy := *item
		return &copy
	}
	return nil
}

// GetSummary returns counts per category for the dashboard.
func (s *Service) GetSummary() *HealthSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := &HealthSummary{
		Categories: make([]CategorySummary, 0, len(AllCategories())),
		HasIssues:  false,
	}

	for _, cat := range AllCategories() {
		catSummary := CategorySummary{Category: cat}

		for _, item := range s.items[cat] {
			switch item.Status {
			case StatusOK:
				catSummary.OK++
			case StatusWarning:
				catSummary.Warning++
			case StatusError:
				catSummary.Error++
			}
		}

		if catSummary.HasIssues() {
			summary.HasIssues = true
		}

		summary.Categories = append(summary.Categories, catSummary)
	}

	return summary
}

// IsHealthy returns true if the specified item is OK.
func (s *Service) IsHealthy(category HealthCategory, id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if item, exists := s.items[category][id]; exists {
		return item.Status == StatusOK
	}
	return false
}

// IsCategoryHealthy returns true if all items in the category are OK.
func (s *Service) IsCategoryHealthy(category HealthCategory) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.items[category] {
		if item.Status != StatusOK {
			return false
		}
	}
	return true
}

// GetUnhealthyItems returns all items in a category that are not OK.
func (s *Service) GetUnhealthyItems(category HealthCategory) []HealthItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var unhealthy []HealthItem
	for _, item := range s.items[category] {
		if item.Status != StatusOK {
			unhealthy = append(unhealthy, *item)
		}
	}
	return unhealthy
}

// GetHealthyItems returns all items in a category that are OK.
func (s *Service) GetHealthyItems(category HealthCategory) []HealthItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var healthy []HealthItem
	for _, item := range s.items[category] {
		if item.Status == StatusOK {
			healthy = append(healthy, *item)
		}
	}
	return healthy
}

// itemsToSlice converts the map of items to a slice.
func (s *Service) itemsToSlice(category HealthCategory) []HealthItem {
	items := make([]HealthItem, 0, len(s.items[category]))
	for _, item := range s.items[category] {
		items = append(items, *item)
	}
	return items
}

// broadcastUpdate sends a health update via WebSocket.
func (s *Service) broadcastUpdate(item *HealthItem) {
	if s.broadcaster == nil {
		return
	}

	payload := HealthUpdatePayload{
		Category:  item.Category,
		ID:        item.ID,
		Name:      item.Name,
		Status:    item.Status,
		Message:   item.Message,
		Timestamp: item.Timestamp,
	}

	if err := s.broadcaster.Broadcast("health:updated", payload); err != nil {
		s.logger.Error().Err(err).Msg("Failed to broadcast health update")
	}
}

// RegisterImportItem registers a new import item for health tracking.
func (s *Service) RegisterImportItem(id, name string) {
	s.RegisterItem(CategoryImport, id, name)
}

// SetImportError sets an import item to Error status.
func (s *Service) SetImportError(id, message string) {
	s.SetError(CategoryImport, id, message)
}

// SetImportWarning sets an import item to Warning status.
func (s *Service) SetImportWarning(id, message string) {
	s.SetWarning(CategoryImport, id, message)
}

// ClearImportStatus resets an import item to OK status.
func (s *Service) ClearImportStatus(id string) {
	s.ClearStatus(CategoryImport, id)
}

// UnregisterImportItem removes an import item from health tracking.
func (s *Service) UnregisterImportItem(id string) {
	s.UnregisterItem(CategoryImport, id)
}
