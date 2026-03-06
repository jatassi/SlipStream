package tv

import (
	"github.com/google/wire"
	"github.com/slipstream/slipstream/internal/module"
)

// Descriptor implements module.Descriptor for the TV module.
type Descriptor struct{}

func (d *Descriptor) ID() module.Type    { return module.TypeTV }
func (d *Descriptor) Name() string       { return "TV" }
func (d *Descriptor) PluralName() string { return "Series" }
func (d *Descriptor) Icon() string       { return "tv" }
func (d *Descriptor) ThemeColor() string { return "green" }

func (d *Descriptor) NodeSchema() module.NodeSchema {
	return module.NodeSchema{
		Levels: []module.NodeLevel{
			{
				Name:         "series",
				PluralName:   "series",
				IsRoot:       true,
				HasMonitored: true,
				Searchable:   true,
			},
			{
				Name:         "season",
				PluralName:   "seasons",
				HasMonitored: true,
				IsSpecial:    true,
			},
			{
				Name:                     "episode",
				PluralName:               "episodes",
				IsLeaf:                   true,
				HasMonitored:             true,
				Searchable:               true,
				SupportsMultiEntityFiles: true,
				FormatVariants:           []string{"standard", "daily", "anime"},
			},
		},
	}
}

func (d *Descriptor) EntityTypes() []module.EntityType {
	return []module.EntityType{module.EntitySeries, module.EntitySeason, module.EntityEpisode}
}

func (d *Descriptor) Wire() wire.ProviderSet {
	return wire.NewSet()
}
