package notices

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
)

const releaseMain = "./cmd/slipstream"

type goModule struct {
	Path      string
	Version   string
	Dir       string
	Main      bool
	GoVersion string
	Replace   *goModule
}

type goPackage struct {
	Standard bool
	Module   *goModule
}

// moduleSet collects dependencies by name and version, each with every
// target that ships it.
type moduleSet struct {
	deps  []Dependency
	byKey map[string]int
}

func (s *moduleSet) add(source Source, name, version, dir, target string) error {
	key := name + " " + version
	if i, ok := s.byKey[key]; ok {
		s.deps[i].Targets = append(s.deps[i].Targets, target)
		return nil
	}
	license, text, err := licenseFiles(dir)
	if err != nil {
		return err
	}
	s.byKey[key] = len(s.deps)
	s.deps = append(s.deps, Dependency{source, name, version, license, []string{target}, text})
	return nil
}

// goModules lists the Go standard library and every module the release
// build of cmd/slipstream compiles in, per target.
func goModules(ctx context.Context, root string, targets []native.Target) ([]Dependency, error) {
	set := moduleSet{byKey: map[string]int{}}
	goroot, err := goCommand(ctx, root, nil, "env", "GOROOT")
	if err != nil {
		return nil, err
	}
	for _, t := range targets {
		pkgs, err := listDeps(ctx, root, t)
		if err != nil {
			return nil, err
		}
		for _, p := range pkgs {
			switch {
			case p.Standard:
				continue
			case p.Module.Main:
				err = set.add(SourceRuntime, "go", p.Module.GoVersion, strings.TrimSpace(string(goroot)), t.Name)
			case p.Module.Replace != nil:
				err = set.add(SourceGo, p.Module.Path, p.Module.Version, p.Module.Replace.Dir, t.Name)
			default:
				err = set.add(SourceGo, p.Module.Path, p.Module.Version, p.Module.Dir, t.Name)
			}
			if err != nil {
				return nil, err
			}
		}
	}
	return set.deps, nil
}

func listDeps(ctx context.Context, root string, t native.Target) ([]goPackage, error) {
	env := []string{"GOOS=" + t.GOOS, "GOARCH=" + t.GOARCH, "CGO_ENABLED=1"}
	out, err := goCommand(ctx, root, env, "list", "-deps", "-json=Standard,Module", "-tags", "release", releaseMain)
	if err != nil {
		return nil, err
	}
	var pkgs []goPackage
	seen := map[string]bool{}
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var p goPackage
		if err := dec.Decode(&p); err != nil {
			return nil, fmt.Errorf("go list %s for %s: %w", releaseMain, t.Name, err)
		}
		key := "std"
		if p.Module != nil {
			key = p.Module.Path
		}
		if !seen[key] {
			seen[key] = true
			pkgs = append(pkgs, p)
		}
	}
	return pkgs, nil
}

func goCommand(ctx context.Context, root string, env []string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), env...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go %s %s: %w\n%s", strings.Join(args, " "), strings.Join(env, " "), err, stderr.Bytes())
	}
	return out, nil
}

func isLicenseFile(name string) bool {
	upper := strings.ToUpper(name)
	return strings.HasPrefix(upper, "LICENSE") ||
		strings.HasPrefix(upper, "LICENCE") || //nolint:misspell // some packages name their license file with the British spelling
		strings.HasPrefix(upper, "COPYING")
}

func licenseFiles(dir string) (license, text string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", "", fmt.Errorf("find the license files: %w", err)
	}
	var licenses, texts []string
	for _, e := range entries {
		if !e.Type().IsRegular() || !isLicenseFile(e.Name()) {
			continue
		}
		content, err := readText(filepath.Join(dir, e.Name()))
		if err != nil {
			return "", "", err
		}
		texts = append(texts, strings.TrimSpace(content))
		if l := Classify(content); !slices.Contains(licenses, l) {
			licenses = append(licenses, l)
		}
	}
	if len(texts) == 0 {
		return noLicense, "", nil
	}
	slices.Sort(licenses)
	return strings.Join(licenses, " AND "), strings.Join(texts, "\n\n"), nil
}
