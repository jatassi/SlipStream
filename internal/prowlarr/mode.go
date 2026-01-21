package prowlarr

import (
	"context"
)

// ModeManager provides indexer mode management with developer mode awareness.
type ModeManager struct {
	service    *Service
	devModeFunc func() bool
}

// NewModeManager creates a new mode manager.
func NewModeManager(service *Service, devModeFunc func() bool) *ModeManager {
	return &ModeManager{
		service:    service,
		devModeFunc: devModeFunc,
	}
}

// GetEffectiveMode returns the current effective indexer mode.
// Developer mode forces SlipStream mode regardless of user setting.
func (m *ModeManager) GetEffectiveMode(ctx context.Context) (IndexerMode, error) {
	// Developer mode always forces SlipStream mode
	// This ensures mock indexer is available for testing
	if m.devModeFunc != nil && m.devModeFunc() {
		return ModeSlipStream, nil
	}

	enabled, err := m.service.IsEnabled(ctx)
	if err != nil {
		return ModeSlipStream, err
	}

	if enabled {
		return ModeProwlarr, nil
	}

	return ModeSlipStream, nil
}

// IsProwlarrMode returns whether Prowlarr mode is currently effective.
func (m *ModeManager) IsProwlarrMode(ctx context.Context) (bool, error) {
	mode, err := m.GetEffectiveMode(ctx)
	if err != nil {
		return false, err
	}
	return mode == ModeProwlarr, nil
}

// IsSlipStreamMode returns whether SlipStream mode is currently effective.
func (m *ModeManager) IsSlipStreamMode(ctx context.Context) (bool, error) {
	mode, err := m.GetEffectiveMode(ctx)
	if err != nil {
		return false, err
	}
	return mode == ModeSlipStream, nil
}

// SetMode sets the indexer mode (subject to developer mode override).
func (m *ModeManager) SetMode(ctx context.Context, mode IndexerMode) error {
	enabled := mode == ModeProwlarr
	return m.service.SetEnabled(ctx, enabled)
}

// GetModeInfo returns information about the current mode state.
type ModeInfo struct {
	EffectiveMode   IndexerMode `json:"effectiveMode"`
	ConfiguredMode  IndexerMode `json:"configuredMode"`
	DevModeOverride bool        `json:"devModeOverride"`
}

// GetModeInfo returns detailed mode state information.
func (m *ModeManager) GetModeInfo(ctx context.Context) (*ModeInfo, error) {
	devMode := m.devModeFunc != nil && m.devModeFunc()

	prowlarrEnabled, err := m.service.IsEnabled(ctx)
	if err != nil && err != ErrNotConfigured {
		return nil, err
	}

	configuredMode := ModeSlipStream
	if prowlarrEnabled {
		configuredMode = ModeProwlarr
	}

	effectiveMode := configuredMode
	if devMode {
		effectiveMode = ModeSlipStream
	}

	return &ModeInfo{
		EffectiveMode:   effectiveMode,
		ConfiguredMode:  configuredMode,
		DevModeOverride: devMode && configuredMode == ModeProwlarr,
	}, nil
}
