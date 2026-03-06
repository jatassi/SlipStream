package module

// ArrImportItem is an item from an external *arr DB.
type ArrImportItem struct {
	ExternalID string
	Title      string
	Path       string
	ProfileID  int
	Monitored  bool
}

// ProvisionInput contains input for provisioning media via the portal.
type ProvisionInput struct {
	ExternalID string
	Title      string
	ProfileID  int64
	RootFolder string
}

// AvailabilityCheckInput contains input for checking media availability.
type AvailabilityCheckInput struct {
	EntityType EntityType
	EntityID   int64
}

// AvailabilityResult contains the result of an availability check.
type AvailabilityResult struct {
	Available bool
	Status    string
}

// RequestValidationInput contains input for validating a portal request.
type RequestValidationInput struct {
	ExternalID string
	EntityType EntityType
}
