package health

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/filesystem"
)

// StorageInfoProvider defines the interface for getting storage information.
type StorageInfoProvider interface {
	GetStorageInfo(ctx context.Context) ([]StorageItem, error)
}

// StorageItem represents storage info for health checking.
type StorageItem struct {
	VolumeID      string
	Label         string
	FreeSpace     int64
	TotalSpace    int64
	UsedPercent   float64
	HasRootFolder bool
}

// StorageServiceAdapter adapts filesystem.StorageService to StorageInfoProvider.
type StorageServiceAdapter struct {
	storageService *filesystem.StorageService
}

// NewStorageServiceAdapter creates a new adapter.
func NewStorageServiceAdapter(storageSvc *filesystem.StorageService) *StorageServiceAdapter {
	return &StorageServiceAdapter{
		storageService: storageSvc,
	}
}

// GetStorageInfo implements StorageInfoProvider by converting filesystem.StorageInfo to StorageItem.
func (a *StorageServiceAdapter) GetStorageInfo(ctx context.Context) ([]StorageItem, error) {
	fsStorage, err := a.storageService.GetStorageInfo(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]StorageItem, len(fsStorage))
	for i, fs := range fsStorage {
		items[i] = StorageItem{
			VolumeID:      fs.Path,
			Label:         fs.Label,
			FreeSpace:     fs.FreeSpace,
			TotalSpace:    fs.TotalSpace,
			UsedPercent:   fs.UsedPercent,
			HasRootFolder: len(fs.RootFolders) > 0,
		}
	}

	return items, nil
}

// StorageChecker checks storage health based on disk space.
type StorageChecker struct {
	healthService   *Service
	storageProvider StorageInfoProvider
	config          *config.HealthConfig
	logger          *zerolog.Logger

	mu           sync.Mutex
	knownVolumes map[string]bool
}

// NewStorageChecker creates a new storage checker.
func NewStorageChecker(
	healthSvc *Service,
	storageProvider StorageInfoProvider,
	cfg *config.HealthConfig,
	logger *zerolog.Logger,
) *StorageChecker {
	subLogger := logger.With().Str("component", "storage-health").Logger()
	return &StorageChecker{
		healthService:   healthSvc,
		storageProvider: storageProvider,
		config:          cfg,
		logger:          &subLogger,
		knownVolumes:    make(map[string]bool),
	}
}

// CheckAllStorage checks health of all storage volumes with root folders.
func (c *StorageChecker) CheckAllStorage(ctx context.Context) error {
	storage, err := c.storageProvider.GetStorageInfo(ctx)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to get storage info")
		return fmt.Errorf("failed to get storage info: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	currentVolumes := make(map[string]bool)
	thresholds := c.getThresholds()

	for _, item := range storage {
		if !item.HasRootFolder {
			continue
		}

		currentVolumes[item.VolumeID] = true
		c.registerVolumeIfNew(item)
		c.checkVolumeHealth(item, thresholds)
	}

	c.unregisterRemovedVolumes(currentVolumes)
	return nil
}

type storageThresholds struct {
	warning float64
	error   float64
}

func (c *StorageChecker) getThresholds() storageThresholds {
	warning := c.config.StorageWarningThreshold
	if warning == 0 {
		warning = 0.20
	}
	errorThreshold := c.config.StorageErrorThreshold
	if errorThreshold == 0 {
		errorThreshold = 0.05
	}
	return storageThresholds{warning: warning, error: errorThreshold}
}

func (c *StorageChecker) registerVolumeIfNew(item StorageItem) {
	if c.knownVolumes[item.VolumeID] {
		return
	}
	c.healthService.RegisterItem(CategoryStorage, item.VolumeID, item.Label)
	c.knownVolumes[item.VolumeID] = true
	c.logger.Debug().
		Str("volumeId", item.VolumeID).
		Str("label", item.Label).
		Msg("Registered storage volume with health service")
}

func (c *StorageChecker) checkVolumeHealth(item StorageItem, thresholds storageThresholds) {
	freePercent := calculateFreePercent(item)

	switch {
	case freePercent < thresholds.error:
		c.setVolumeError(item, freePercent)
	case freePercent < thresholds.warning:
		c.setVolumeWarning(item, freePercent)
	default:
		c.clearVolumeStatus(item, freePercent)
	}
}

func calculateFreePercent(item StorageItem) float64 {
	if item.TotalSpace == 0 {
		return 0
	}
	return float64(item.FreeSpace) / float64(item.TotalSpace)
}

func (c *StorageChecker) setVolumeError(item StorageItem, freePercent float64) {
	message := fmt.Sprintf("Critically low disk space: %.1f%% free", freePercent*100)
	c.healthService.SetError(CategoryStorage, item.VolumeID, message)
	c.logger.Warn().
		Str("volumeId", item.VolumeID).
		Float64("freePercent", freePercent*100).
		Msg("Storage critically low")
}

func (c *StorageChecker) setVolumeWarning(item StorageItem, freePercent float64) {
	message := fmt.Sprintf("Low disk space: %.1f%% free", freePercent*100)
	c.healthService.SetWarning(CategoryStorage, item.VolumeID, message)
	c.logger.Info().
		Str("volumeId", item.VolumeID).
		Float64("freePercent", freePercent*100).
		Msg("Storage low")
}

func (c *StorageChecker) clearVolumeStatus(item StorageItem, freePercent float64) {
	c.healthService.ClearStatus(CategoryStorage, item.VolumeID)
	c.logger.Debug().
		Str("volumeId", item.VolumeID).
		Float64("freePercent", freePercent*100).
		Msg("Storage healthy")
}

func (c *StorageChecker) unregisterRemovedVolumes(currentVolumes map[string]bool) {
	for volumeID := range c.knownVolumes {
		if currentVolumes[volumeID] {
			continue
		}
		c.healthService.UnregisterItem(CategoryStorage, volumeID)
		delete(c.knownVolumes, volumeID)
		c.logger.Debug().
			Str("volumeId", volumeID).
			Msg("Unregistered storage volume from health service")
	}
}
