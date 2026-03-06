package module

// TemplateVariable is a variable available for path templates.
type TemplateVariable struct {
	Name        string
	Description string
	Example     string
}

// ConditionalSegment is a conditional path segment declaration.
type ConditionalSegment struct {
	Name      string
	Condition string
	Template  string
}

// TokenContext is a named scope of template variables.
type TokenContext struct {
	Name      string
	Variables []TemplateVariable
}

// FormatOption is a formatting option (bool or enum).
type FormatOption struct {
	Name         string
	Label        string
	Type         string
	DefaultValue string
	Options      []string
}
