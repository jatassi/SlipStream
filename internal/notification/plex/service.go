package plex

import "strings"

// MapPath applies path mappings to transform a local path to a Plex path
func MapPath(path string, mappings []PathMapping) string {
	for _, m := range mappings {
		if strings.HasPrefix(path, m.From) {
			return m.To + strings.TrimPrefix(path, m.From)
		}
	}
	return path
}

// FindMatchingSection finds the library section that contains the given path
func FindMatchingSection(path string, sections []LibrarySection, targetSectionIDs []int) *LibrarySection {
	for _, section := range sections {
		if !containsInt(targetSectionIDs, section.Key) {
			continue
		}

		for _, loc := range section.Locations {
			if strings.HasPrefix(path, loc.Path) {
				return &section
			}
		}
	}
	return nil
}

func containsInt(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
