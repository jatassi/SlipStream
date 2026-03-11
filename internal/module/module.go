package module

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/slipstream/slipstream/internal/module/parseutil"
)

// Module is the full interface bundle that a module implementation must satisfy.
type Module interface {
	Descriptor
	MetadataProvider
	SearchStrategy
	ImportHandler
	PathGenerator
	NamingProvider
	CalendarProvider
	QualityDefinition
	WantedCollector
	MonitoringPresets
	FileParser
	MockFactory
	NotificationEvents
	ReleaseDateResolver
	RouteProvider
	TaskProvider
}

// Registry holds all registered modules and provides lookup.
type Registry struct {
	mu      sync.RWMutex
	modules map[Type]Module
	order   []Type
	enabled map[Type]bool // nil means all enabled (default)
}

// NewRegistry creates an empty module registry.
func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[Type]Module),
	}
}

// Register adds a module to the registry. Panics on duplicate ID or conflicting entity types.
func (r *Registry) Register(m Module) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := m.ID()
	if _, exists := r.modules[id]; exists {
		panic(fmt.Sprintf("module already registered: %s", id))
	}

	schema := m.NodeSchema()
	if err := schema.Validate(); err != nil {
		panic(fmt.Sprintf("module %s has invalid schema: %v", id, err))
	}

	for _, existingMod := range r.modules {
		existingTypes := existingMod.EntityTypes()
		for _, newET := range m.EntityTypes() {
			for _, existingET := range existingTypes {
				if newET == existingET {
					panic(fmt.Sprintf("entity type %q conflicts between modules %s and %s", newET, id, existingMod.ID()))
				}
			}
		}
	}

	r.modules[id] = m
	r.order = append(r.order, id)
}

// Get returns a module by type, or nil if not found.
func (r *Registry) Get(t Type) Module {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.modules[t]
}

// All returns all registered modules in registration order.
func (r *Registry) All() []Module {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Module, 0, len(r.order))
	for _, t := range r.order {
		result = append(result, r.modules[t])
	}
	return result
}

// Types returns all registered module types in registration order.
func (r *Registry) Types() []Type {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]Type(nil), r.order...)
}

// Enabled returns only enabled modules in registration order.
// If no enabled state has been set, all modules are considered enabled.
func (r *Registry) Enabled() []Module {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Module, 0, len(r.order))
	for _, t := range r.order {
		if r.isEnabledLocked(t) {
			result = append(result, r.modules[t])
		}
	}
	return result
}

// IsEnabled checks whether a specific module type is enabled.
// Returns true if no enabled state has been configured (default: all enabled).
func (r *Registry) IsEnabled(t Type) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isEnabledLocked(t)
}

// isEnabledLocked checks enabled state while holding the lock.
func (r *Registry) isEnabledLocked(t Type) bool {
	if r.enabled == nil {
		return true
	}
	enabled, exists := r.enabled[t]
	if !exists {
		return true
	}
	return enabled
}

// SetEnabledModules updates the cached enabled state for all modules.
func (r *Registry) SetEnabledModules(enabledMap map[Type]bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = enabledMap
}

// SetModuleEnabled sets the enabled state for a single module in the cache.
func (r *Registry) SetModuleEnabled(t Type, enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.enabled == nil {
		r.enabled = make(map[Type]bool)
		for _, mt := range r.order {
			r.enabled[mt] = true
		}
	}
	r.enabled[t] = enabled
}

// LoadEnabledState reads module enabled settings from the database and populates the cache.
func (r *Registry) LoadEnabledState(ctx context.Context, db *sql.DB) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	enabledMap := make(map[Type]bool)
	for _, t := range r.order {
		enabled, err := IsModuleEnabled(ctx, db, t)
		if err != nil {
			return fmt.Errorf("checking module %s enabled state: %w", t, err)
		}
		enabledMap[t] = enabled
	}
	r.enabled = enabledMap
	return nil
}

// EnabledState returns a copy of the current enabled state map.
// If no state has been configured, returns all modules as enabled.
func (r *Registry) EnabledState() map[Type]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[Type]bool, len(r.order))
	for _, t := range r.order {
		result[t] = r.isEnabledLocked(t)
	}
	return result
}

// CollectNotificationEvents returns all notification events grouped by source.
// Framework events come first, then each registered module's events.
func (r *Registry) CollectNotificationEvents() []NotificationEventGroup {
	r.mu.RLock()
	defer r.mu.RUnlock()

	groups := []NotificationEventGroup{
		{ID: "framework", Label: "General", Events: FrameworkNotificationEvents()},
	}
	for _, t := range r.order {
		mod := r.modules[t]
		events := mod.DeclareEvents()
		if len(events) > 0 {
			groups = append(groups, NotificationEventGroup{
				ID:     string(mod.ID()),
				Label:  mod.Name(),
				Events: events,
			})
		}
	}
	return groups
}

// ModuleForEntityType returns the module that owns the given entity type.
func (r *Registry) ModuleForEntityType(et EntityType) Module {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.modules {
		for _, met := range m.EntityTypes() {
			if met == et {
				return m
			}
		}
	}
	return nil
}

// GetProvisioner returns the PortalProvisioner for the given module type, or nil.
func (r *Registry) GetProvisioner(moduleType string) PortalProvisioner {
	r.mu.RLock()
	defer r.mu.RUnlock()
	mod, ok := r.modules[Type(moduleType)]
	if !ok {
		return nil
	}
	if pp, ok := mod.(PortalProvisioner); ok {
		return pp
	}
	return nil
}

// GetProvisionerForEntityType returns the PortalProvisioner for a given entity type
// (e.g., "movie" -> movie module's provisioner, "episode" -> tv module's provisioner).
func (r *Registry) GetProvisionerForEntityType(entityType string) PortalProvisioner {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, mod := range r.modules {
		pp, ok := mod.(PortalProvisioner)
		if !ok {
			continue
		}
		for _, et := range pp.SupportedEntityTypes() {
			if et == entityType {
				return pp
			}
		}
	}
	return nil
}

// IsLeafEntityType returns true if the given entity type is a leaf node in any
// registered module's schema (e.g., "movie" for flat modules, "episode" for TV).
func (r *Registry) IsLeafEntityType(entityType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, mod := range r.modules {
		entityTypes := mod.EntityTypes()
		schema := mod.NodeSchema()
		for i, et := range entityTypes {
			if string(et) == entityType && i < len(schema.Levels) {
				return schema.Levels[i].IsLeaf
			}
		}
	}
	return false
}

// GetMovieArrAdapter returns the MovieArrImportAdapter from the registered modules, or nil if none.
func (r *Registry) GetMovieArrAdapter() MovieArrImportAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, mod := range r.modules {
		if adapter, ok := mod.(MovieArrImportAdapter); ok {
			return adapter
		}
	}
	return nil
}

// GetTVArrAdapter returns the TVArrImportAdapter from the registered modules, or nil if none.
func (r *Registry) GetTVArrAdapter() TVArrImportAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, mod := range r.modules {
		if adapter, ok := mod.(TVArrImportAdapter); ok {
			return adapter
		}
	}
	return nil
}

// ParseReleaseForFilter parses a raw release title (which typically lacks a file
// extension) into a ReleaseForFilter. It iterates registered module file parsers;
// the first successful parse wins. If no module claims the title, a minimal
// ReleaseForFilter is returned with quality info only.
func (r *Registry) ParseReleaseForFilter(rawTitle string, size int64, categories []int) *ReleaseForFilter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Strip any extension-like suffix (.x264, .mkv, etc.) before trying module parsers.
	name := strings.TrimSuffix(rawTitle, filepath.Ext(rawTitle))

	for _, t := range r.order {
		mod := r.modules[t]
		result, err := mod.ParseFilename(name)
		if err != nil || result == nil {
			continue
		}
		return parseResultToReleaseForFilter(result, mod.ID() == TypeTV, size, categories)
	}

	// Fallback: quality-only extraction from the raw title.
	attrs := parseutil.DetectQualityAttributes(rawTitle)
	return &ReleaseForFilter{
		Title:      parseutil.CleanTitle(name),
		Quality:    attrs.Quality,
		Source:     attrs.Source,
		Languages:  parseutil.ParseLanguages(rawTitle),
		Size:       size,
		Categories: categories,
	}
}

func parseResultToReleaseForFilter(result *ParseResult, isTV bool, size int64, categories []int) *ReleaseForFilter {
	rff := &ReleaseForFilter{
		Title:      result.Title,
		Year:       result.Year,
		IsTV:       isTV,
		Quality:    result.Quality,
		Source:     result.Source,
		Languages:  result.Languages,
		Size:       size,
		Categories: categories,
	}
	if result.Extra != nil {
		if accessor, ok := result.Extra.(TVExtraAccessor); ok {
			rff.Season = accessor.TVSeason()
			rff.EndSeason = accessor.TVEndSeason()
			rff.Episode = accessor.TVEpisode()
			rff.EndEpisode = accessor.TVEndEpisode()
			rff.IsSeasonPack = accessor.TVIsSeasonPack()
			rff.IsCompleteSeries = accessor.TVIsCompleteSeries()
		}
	}
	return rff
}
