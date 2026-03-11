package module

import "context"

// ArrImportAdapter enables import from external *arr apps (generic interface).
// Movie and TV modules implement the richer, type-safe MovieArrImportAdapter
// and TVArrImportAdapter interfaces (arr_import.go) instead.
type ArrImportAdapter interface {
	ExternalAppName() string
	ReadExternalDB(ctx context.Context, dbPath string) ([]ArrImportItem, error)
	ConvertToCreateInput(item ArrImportItem) (any, error)
}
