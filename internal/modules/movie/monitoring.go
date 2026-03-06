package movie

import (
	"context"

	"github.com/slipstream/slipstream/internal/module"
)

func (m *Module) AvailablePresets() []module.MonitoringPreset {
	return nil
}

func (m *Module) ApplyPreset(_ context.Context, _ int64, _ string, _ map[string]any) error {
	return nil
}
