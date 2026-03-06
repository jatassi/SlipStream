package module

import "context"

// PortalProvisioner enables portal request support.
type PortalProvisioner interface {
	EnsureInLibrary(ctx context.Context, input *ProvisionInput) (entityID int64, err error)
	CheckAvailability(ctx context.Context, input *AvailabilityCheckInput) (*AvailabilityResult, error)
	ValidateRequest(ctx context.Context, input *RequestValidationInput) error
}

// SlotSupport enables version slot support.
type SlotSupport interface {
	SlotEntityType() EntityType
	SlotTableSchema() SlotTableDecl
}

// ArrImportAdapter enables import from external *arr apps.
type ArrImportAdapter interface {
	ExternalAppName() string
	ReadExternalDB(ctx context.Context, dbPath string) ([]ArrImportItem, error)
	ConvertToCreateInput(item ArrImportItem) (any, error)
}
