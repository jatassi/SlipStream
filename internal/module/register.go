package module

import (
	"github.com/rs/zerolog"
)

// RegisterAll registers all built-in modules with the registry.
// Called once at application startup.
func RegisterAll(registry *Registry, logger *zerolog.Logger) {
	// Module descriptors are registered here.
	// In Phase 0, we only register descriptors — full Module implementations come in later phases.
	//
	// Future:
	//   registry.Register(movie.New())
	//   registry.Register(tv.New())
	logger.Info().Msg("Module registry initialized (no full module implementations registered yet)")
}
