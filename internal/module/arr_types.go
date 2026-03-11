package module

// ArrImportItem is an item from an external *arr DB.
type ArrImportItem struct {
	ExternalID string
	Title      string
	Path       string
	ProfileID  int
	Monitored  bool
}
