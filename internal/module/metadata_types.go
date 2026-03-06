package module

// RefreshResult contains the structured diff from a metadata refresh operation.
// The framework uses this to determine post-refresh actions (artwork download, status updates, broadcasting).
type RefreshResult struct {
	EntityID        int64
	Updated         bool
	FieldsChanged   []string
	ChildrenAdded   []RefreshChildEntry
	ChildrenUpdated []RefreshChildEntry
	ChildrenRemoved []RefreshChildEntry
	ArtworkURLs     ArtworkURLs
	Metadata        any
}

// RefreshChildEntry represents a child entity in a refresh diff.
type RefreshChildEntry struct {
	EntityType EntityType
	Identifier string
	EntityID   int64
	Title      string
}

// ArtworkURLs holds URLs for artwork that should be downloaded after a metadata refresh.
type ArtworkURLs struct {
	PosterURL     string
	BackdropURL   string
	LogoURL       string
	StudioLogoURL string
}
