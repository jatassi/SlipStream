package movie

import (
	"github.com/google/wire"
	"github.com/slipstream/slipstream/internal/module"
)

// Descriptor implements module.Descriptor for the Movie module.
type Descriptor struct{}

func (d *Descriptor) ID() module.Type    { return module.TypeMovie }
func (d *Descriptor) Name() string       { return "Movies" }
func (d *Descriptor) PluralName() string { return "Movies" }
func (d *Descriptor) Icon() string       { return "film" }
func (d *Descriptor) ThemeColor() string { return "blue" }

func (d *Descriptor) NodeSchema() module.NodeSchema {
	return module.NodeSchema{
		Levels: []module.NodeLevel{
			{
				Name:         "movie",
				PluralName:   "movies",
				IsRoot:       true,
				IsLeaf:       true,
				HasMonitored: true,
				Searchable:   true,
			},
		},
	}
}

func (d *Descriptor) EntityTypes() []module.EntityType {
	return []module.EntityType{module.EntityMovie}
}

func (d *Descriptor) Wire() wire.ProviderSet {
	return wire.NewSet()
}
