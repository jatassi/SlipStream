package module

import "github.com/google/wire"

// Type identifies a module (used as discriminator in shared tables).
type Type string

const (
	TypeMovie Type = "movie"
	TypeTV    Type = "tv"
)

// EntityType identifies an entity within a module (e.g., "movie", "series", "season", "episode").
type EntityType string

const (
	EntityMovie   EntityType = "movie"
	EntitySeries  EntityType = "series"
	EntitySeason  EntityType = "season"
	EntityEpisode EntityType = "episode"
)

// Descriptor provides a module's identity and schema.
type Descriptor interface {
	ID() Type
	Name() string
	PluralName() string
	Icon() string
	ThemeColor() string
	NodeSchema() NodeSchema
	EntityTypes() []EntityType
	Wire() wire.ProviderSet
}
