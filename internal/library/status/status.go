package status

// These constants will be replaced by module.Status* once movie/TV services
// are refactored to use the module framework (Phase 4+).
const (
	Available   = "available"
	Upgradable  = "upgradable"
	Missing     = "missing"
	Downloading = "downloading"
	Failed      = "failed"
	Unreleased  = "unreleased"
)
