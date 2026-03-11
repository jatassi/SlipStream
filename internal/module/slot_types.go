package module

// SlotTableDecl declares a slot assignment table schema for the framework to generate.
type SlotTableDecl struct {
	TableName    string
	ForeignKey   string
	ForeignTable string
}
