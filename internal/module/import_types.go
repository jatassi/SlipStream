package module

// CompletedDownload represents a download that finished and is ready for import.
type CompletedDownload struct {
	DownloadID string
	Title      string
	OutputPath string
	Category   string
}

// MatchedEntity is an entity matched to a download/file.
type MatchedEntity struct {
	EntityType EntityType
	EntityID   int64
	Title      string
}

// QualityInfo contains quality details of a file being imported.
type QualityInfo struct {
	QualityID   int
	QualityName string
	Proper      bool
}

// ImportResult is the result of importing a file.
type ImportResult struct {
	FilePath string
	EntityID int64
	Success  bool
}

// MediaInfoFieldDecl declares a media info field relevant to the module.
type MediaInfoFieldDecl struct {
	Name     string
	Label    string
	DataType string
}
