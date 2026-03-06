package movie

import (
	"github.com/google/wire"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/modules/shared"
)

var _ module.QualityDefinition = (*Descriptor)(nil)

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
