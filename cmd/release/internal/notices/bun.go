package notices

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type bunLock struct {
	Workspaces map[string]bunDeps           `json:"workspaces"`
	Packages   map[string][]json.RawMessage `json:"packages"`
}

type bunDeps struct {
	Dependencies         map[string]string `json:"dependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
}

func (d bunDeps) names() []string {
	var names []string
	for name := range d.Dependencies {
		names = append(names, name)
	}
	for name := range d.OptionalDependencies {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

type packageJSON struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	License string `json:"license"`
}

func npmPackages(web string, targets []string) ([]Dependency, error) {
	data, err := os.ReadFile(filepath.Join(web, "bun.lock"))
	if err != nil {
		return nil, fmt.Errorf("read the web lockfile: %w", err)
	}
	var lock bunLock
	if err := json.Unmarshal(stripTrailingCommas(data), &lock); err != nil {
		return nil, fmt.Errorf("web/bun.lock: %w", err)
	}
	var deps []Dependency
	seen := map[string]bool{}
	var visit func(parent, name string) error
	visit = func(parent, name string) error {
		key, entry, err := lock.resolve(parent, name)
		if err != nil {
			return err
		}
		if seen[key] {
			return nil
		}
		seen[key] = true
		var ident string
		if err := json.Unmarshal(entry[0], &ident); err != nil {
			return fmt.Errorf("web/bun.lock: package %s: %w", key, err)
		}
		version := ident[strings.LastIndex(ident, "@")+1:]
		dep, err := npmPackage(installDir(web, key), name, version, targets)
		if err != nil {
			return err
		}
		deps = append(deps, dep)
		meta, err := entryDeps(key, entry)
		if err != nil {
			return err
		}
		for _, child := range meta.names() {
			if err := visit(key, child); err != nil {
				return err
			}
		}
		return nil
	}
	for _, name := range lock.Workspaces[""].names() {
		if err := visit("", name); err != nil {
			return nil, err
		}
	}
	return deps, nil
}

func installDir(web, key string) string {
	dir := web
	for _, n := range splitKey(key) {
		dir = filepath.Join(dir, "node_modules", filepath.FromSlash(n))
	}
	return dir
}

func entryDeps(key string, entry []json.RawMessage) (bunDeps, error) {
	var meta bunDeps
	if len(entry) > 2 {
		if err := json.Unmarshal(entry[2], &meta); err != nil {
			return bunDeps{}, fmt.Errorf("web/bun.lock: package %s: %w", key, err)
		}
	}
	return meta, nil
}

// resolve finds the package a dependency of parent installs as, searching
// from parent's own nested node_modules up to the top level, as Node does.
func (l bunLock) resolve(parent, name string) (key string, entry []json.RawMessage, err error) {
	for scope := parent; ; scope = parentKey(scope) {
		key = name
		if scope != "" {
			key = scope + "/" + name
		}
		if entry, ok := l.Packages[key]; ok {
			return key, entry, nil
		}
		if scope == "" {
			return "", nil, fmt.Errorf("web/bun.lock: no package resolves %s for %q", name, parent)
		}
	}
}

func parentKey(key string) string {
	names := splitKey(key)
	return strings.Join(names[:len(names)-1], "/")
}

func splitKey(key string) []string {
	var names []string
	parts := strings.Split(key, "/")
	for i := 0; i < len(parts); i++ {
		if strings.HasPrefix(parts[i], "@") {
			names = append(names, parts[i]+"/"+parts[i+1])
			i++
			continue
		}
		names = append(names, parts[i])
	}
	return names
}

func npmPackage(dir, name, version string, targets []string) (Dependency, error) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return Dependency{}, fmt.Errorf("%w; run bun install --frozen-lockfile in web", err)
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return Dependency{}, fmt.Errorf("%s: %w", dir, err)
	}
	if pkg.Name != name || pkg.Version != version {
		return Dependency{}, fmt.Errorf("%s holds %s@%s but web/bun.lock pins %s@%s; run bun install --frozen-lockfile in web",
			dir, pkg.Name, pkg.Version, name, version)
	}
	_, text, err := licenseFiles(dir)
	if err != nil {
		return Dependency{}, err
	}
	license := pkg.License
	if license == "" {
		license = noLicense
	}
	return Dependency{SourceNPM, name, version, license, targets, text, ""}, nil
}

func stripTrailingCommas(data []byte) []byte {
	out := make([]byte, 0, len(data))
	inString := false
	for i := 0; i < len(data); i++ {
		c := data[i]
		switch {
		case inString && c == '\\':
			out = append(out, c, data[i+1])
			i++
			continue
		case c == '"':
			inString = !inString
		case !inString && c == ',':
			j := i + 1
			for j < len(data) && strings.ContainsRune(" \t\r\n", rune(data[j])) {
				j++
			}
			if j < len(data) && (data[j] == '}' || data[j] == ']') {
				continue
			}
		}
		out = append(out, c)
	}
	return out
}
