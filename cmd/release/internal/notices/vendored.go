package notices

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Vendored names a folder of third-party source copied into the repository.
// The folder holds the upstream LICENSE and a VENDORED.md whose "Commit:"
// line names the upstream commit it was copied from.
type Vendored struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

const commitPrefix = "Commit: "

func vendoredSources(root string, vendored []Vendored, targets []string) ([]Dependency, error) {
	deps := make([]Dependency, 0, len(vendored))
	for _, v := range vendored {
		dir := filepath.Join(root, filepath.FromSlash(v.Path))
		record, err := readText(filepath.Join(dir, "VENDORED.md"))
		if err != nil {
			return nil, err
		}
		commit, err := commitOf(record)
		if err != nil {
			return nil, fmt.Errorf("%s/VENDORED.md: %w", v.Path, err)
		}
		license, text, err := licenseFiles(dir)
		if err != nil {
			return nil, err
		}
		deps = append(deps, Dependency{SourceVendored, v.Name, commit, license, targets, text, ""})
	}
	return deps, nil
}

func commitOf(record string) (string, error) {
	for line := range strings.Lines(record) {
		if commit, ok := strings.CutPrefix(strings.TrimSpace(line), commitPrefix); ok && commit != "" {
			return commit, nil
		}
	}
	return "", fmt.Errorf("no %q line", commitPrefix)
}
