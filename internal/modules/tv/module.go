package tv

import (
	"github.com/google/wire"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/modules/shared"
)

var _ module.QualityDefinition = (*Descriptor)(nil)

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

// QualityItems returns the video quality tiers for this module.
func (d *Descriptor) QualityItems() []module.QualityItem {
	return shared.VideoQualityItems()
}

func (d *Descriptor) ParseQuality(_ string) (*module.QualityResult, error) {
	// Stub — delegates to existing parser in Phase 5
	return &module.QualityResult{}, nil
}

func (d *Descriptor) ScoreQuality(item module.QualityItem) int {
	// Stub — delegates to existing scorer in Phase 5
	return item.Weight
}

func (d *Descriptor) IsUpgrade(current, candidate module.QualityItem, profileID int64) (bool, error) {
	// Stub — upgrade logic lives on Profile, not the module.
	return false, nil
}
