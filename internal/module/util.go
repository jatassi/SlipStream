package module

// GenerateSortTitle strips leading articles for sort ordering.
// Used by module services when creating/updating entities.
func GenerateSortTitle(title string) string {
	prefixes := []string{"The ", "A ", "An "}
	for _, prefix := range prefixes {
		if len(title) > len(prefix) && title[:len(prefix)] == prefix {
			return title[len(prefix):]
		}
	}
	return title
}

// ResolveField returns the input value if non-nil, otherwise the current value.
// Used in Update operations to apply optional fields.
func ResolveField[T any](current T, input *T) T {
	if input != nil {
		return *input
	}
	return current
}
