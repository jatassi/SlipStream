package module

// QualityItem represents a quality tier defined by a module.
type QualityItem struct {
	ID   int
	Name string
}

// QualityResult is the result of parsing quality from a release title.
type QualityResult struct {
	Quality QualityItem
	Proper  bool
}
