package firstrun

// One const per step. A later spec adds one here, in its place, and its
// line in internal/node/firstrun.go; New panics unless the two match.
const (
	AddressID   StepID = "address"
	PasskeyID   StepID = "passkey"
	MediaID     StepID = "media"
	LanguageID  StepID = "language"
	RootsID     StepID = "roots"
	DownloadsID StepID = "downloads"
	TargetsID   StepID = "targets"
)

// IDs is every step, in order.
func IDs() []StepID {
	return []StepID{AddressID, PasskeyID, MediaID, LanguageID, RootsID, DownloadsID, TargetsID}
}
