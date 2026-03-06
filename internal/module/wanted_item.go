package module

// WantedItem is a concrete implementation of SearchableItem for items
// collected by WantedCollector (missing and upgradable media).
type WantedItem struct {
	moduleType       Type
	mediaType        string
	entityID         int64
	title            string
	externalIDs      map[string]string
	qualityProfileID int64
	currentQualityID *int64
	searchParams     SearchParams
}

func NewWantedItem(
	moduleType Type,
	mediaType string,
	entityID int64,
	title string,
	externalIDs map[string]string,
	qualityProfileID int64,
	currentQualityID *int64,
	searchParams SearchParams,
) *WantedItem {
	return &WantedItem{
		moduleType:       moduleType,
		mediaType:        mediaType,
		entityID:         entityID,
		title:            title,
		externalIDs:      externalIDs,
		qualityProfileID: qualityProfileID,
		currentQualityID: currentQualityID,
		searchParams:     searchParams,
	}
}

func (w *WantedItem) GetModuleType() string             { return string(w.moduleType) }
func (w *WantedItem) GetMediaType() string              { return w.mediaType }
func (w *WantedItem) GetEntityID() int64                { return w.entityID }
func (w *WantedItem) GetTitle() string                  { return w.title }
func (w *WantedItem) GetExternalIDs() map[string]string { return w.externalIDs }
func (w *WantedItem) GetQualityProfileID() int64        { return w.qualityProfileID }
func (w *WantedItem) GetCurrentQualityID() *int64       { return w.currentQualityID }
func (w *WantedItem) GetSearchParams() SearchParams     { return w.searchParams }
