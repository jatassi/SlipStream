package tv

import (
	"context"
	"fmt"

	tvlib "github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/module"
)

var tvMonitoringPresets = []module.MonitoringPreset{
	{ID: "all", Label: "All Episodes", Description: "Monitor all episodes (excludes specials by default)", HasOptions: true},
	{ID: "none", Label: "None", Description: "Unmonitor all episodes and seasons"},
	{ID: "future", Label: "Future Episodes", Description: "Only monitor episodes that haven't aired yet", HasOptions: true},
	{ID: "first_season", Label: "First Season", Description: "Only monitor season 1 episodes"},
	{ID: "latest_season", Label: "Latest Season", Description: "Only monitor the most recent season"},
}

func (m *Module) AvailablePresets() []module.MonitoringPreset {
	return tvMonitoringPresets
}

func (m *Module) ApplyPreset(ctx context.Context, rootEntityID int64, presetID string, options map[string]any) error {
	monitorType, err := toMonitorType(presetID)
	if err != nil {
		return err
	}

	includeSpecials := false
	if v, ok := options["includeSpecials"]; ok {
		if b, ok := v.(bool); ok {
			includeSpecials = b
		}
	}

	return m.tvService.BulkMonitor(ctx, rootEntityID, tvlib.BulkMonitorInput{
		MonitorType:     monitorType,
		IncludeSpecials: includeSpecials,
	})
}

func toMonitorType(presetID string) (tvlib.MonitorType, error) {
	switch presetID {
	case "all":
		return tvlib.MonitorTypeAll, nil
	case "none":
		return tvlib.MonitorTypeNone, nil
	case "future":
		return tvlib.MonitorTypeFuture, nil
	case "first_season":
		return tvlib.MonitorTypeFirstSeason, nil
	case "latest_season":
		return tvlib.MonitorTypeLatest, nil
	default:
		return "", fmt.Errorf("unknown TV monitoring preset: %s", presetID)
	}
}
