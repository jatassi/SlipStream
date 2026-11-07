package works

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/storage"
)

// Preset is a monitoring preset: a rule over level order, the aside mark
// and each Item's facts. It is not targets.Preset, which lays out quality
// tiers.
type Preset string

const (
	PresetAll      Preset = "all"
	PresetFuture   Preset = "future"
	PresetMissing  Preset = "missing"
	PresetExisting Preset = "existing"
	PresetFirst    Preset = "first"
	PresetLatest   Preset = "latest"
	PresetNone     Preset = "none"
)

var presets = []Preset{PresetAll, PresetFuture, PresetMissing, PresetExisting, PresetFirst, PresetLatest, PresetNone}

// Presets is every Preset in the order choices show them.
func Presets() []Preset { return slices.Clone(presets) }

func ParsePreset(s string) (Preset, error) {
	if p := Preset(s); slices.Contains(presets, p) {
		return p, nil
	}
	return "", fmt.Errorf("%q is not a monitoring preset", s)
}

// MonitorsNew is the preset's monitor_new: on for All, Future, Missing and
// Latest.
func (p Preset) MonitorsNew() bool {
	return p == PresetAll || p == PresetFuture || p == PresetMissing || p == PresetLatest
}

// NeedsFacts is whether Monitors reads ItemFacts for the preset. All,
// First and None read the tree alone.
func (p Preset) NeedsFacts() bool {
	return p == PresetFuture || p == PresetMissing || p == PresetExisting || p == PresetLatest
}

// Choice is a monitoring choice for a Work.
type Choice struct {
	Preset Preset
	// Asides, when set, adds every aside Item to the set and becomes the
	// Work's MonitorAsides; nil keeps each Work's own.
	Asides *bool
	// New overrides Preset.MonitorsNew when set.
	New *bool
}

// ItemFacts is what the presets read of one Item that works doesn't hold.
type ItemFacts struct {
	// Released: out on the Controller's calendar date.
	Released bool
	// Filled: a file fills at least one of its cells, Met or Upgradable.
	Filled bool
}

type Facts map[ItemID]ItemFacts

// Monitors is the set c monitors among t's live Items, under facts. Pure.
func Monitors(t Tree, facts Facts, c Choice) map[ItemID]bool {
	panic("works: Monitors is not implemented")
}

type PresetCount struct {
	Preset      Preset
	Monitors    int
	MonitorsNew bool
}

// PresetCounts is what each Preset monitors without asides, and how many
// live aside Items Include adds to any of them.
type PresetCounts struct {
	Choices []PresetCount
	Asides  int
}

// CountPresets is Monitors' size for each Preset. Pure.
func CountPresets(t Tree, facts Facts) PresetCounts {
	panic("works: CountPresets is not implemented")
}

// Delta is what applying a Choice changes on one Work.
type Delta struct {
	Start, Stop int
	// Rule: MonitorNew or MonitorAsides moved.
	Rule bool
}

func (d Delta) Zero() bool { return d == Delta{} }

// ApplyPreset makes every live Item of each of ws monitored exactly when
// Monitors says under c, and stores c's MonitorNew and MonitorAsides on
// each, in tx. Withdrawn Items keep their flags. All and None are
// set-based over ws and read no facts; facts is read only when
// c.Preset.NeedsFacts. It publishes nothing: the composer publishes once
// after commit, or rolls back for a preview. Applying it twice changes
// nothing the second time. A Work ws names that doesn't exist is absent
// from the answer.
func (w *Works) ApplyPreset(ctx context.Context, tx *storage.Tx[file], ws Among[WorkID], c Choice, facts map[WorkID]Facts) (map[WorkID]Delta, error) {
	return nil, errors.New("not implemented")
}

// Within names part of a Work: the Items under Levels and the Items
// named.
type Within struct {
	Levels []LevelID
	Items  []ItemID
}
