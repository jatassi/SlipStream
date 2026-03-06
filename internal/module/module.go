package module

import (
	"fmt"
	"sync"
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
