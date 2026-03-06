package module

// ParseResult is the result of parsing a filename.
type ParseResult struct {
	Title      string
	Year       int
	EntityType EntityType
	Fields     map[string]any
}

// MonitoringPreset defines a monitoring strategy.
type MonitoringPreset struct {
	ID          string
	Label       string
	Description string
	HasOptions  bool
}
