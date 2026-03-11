package module

import "context"

// CascadeMonitoredForModule dispatches a monitoring cascade to the appropriate
// module via the MonitoringCascader interface. Modules that don't implement the
// interface (or don't exist in the registry) are silently skipped.
func CascadeMonitoredForModule(ctx context.Context, registry *Registry, moduleType Type, entityType EntityType, entityID int64, monitored bool) error {
	mod := registry.Get(moduleType)
	if mod == nil {
		return nil
	}
	if cascader, ok := mod.(MonitoringCascader); ok {
		return cascader.CascadeMonitored(ctx, entityType, entityID, monitored)
	}
	return nil
}
