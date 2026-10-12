package notices

import (
	"bytes"
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

// goModules lists the Go standard library and every module the release
// build of cmd/slipstream compiles in, per target.
func goModules(root string, targets []native.Target) ([]Dependency, error) {
	var deps []Dependency
	byKey := map[string]int{}
	add := func(source Source, name, version, dir, target string) error {
		key := name + " " + version
		if i, ok := byKey[key]; ok {
			deps[i].Targets = append(deps[i].Targets, target)
			return nil
		}
		license, text, err := licenseFiles(dir)
		if err != nil {
			return err
		}
		byKey[key] = len(deps)
		deps = append(deps, Dependency{source, name, version, license, []string{target}, text})
		return nil
	}
	goroot, err := goCommand(root, nil, "env", "GOROOT")
	if err != nil {
		return nil, err
	}
	for _, t := range targets {
		pkgs, err := listDeps(root, t)
		if err != nil {
			return nil, err
		}
		for _, p := range pkgs {
			switch {
			case p.Standard:
				continue
			case p.Module.Main:
				if err := add(SourceRuntime, "go", p.Module.GoVersion, strings.TrimSpace(string(goroot)), t.Name); err != nil {
					return nil, err
				}
			default:
				dir := p.Module.Dir
				if p.Module.Replace != nil {
					dir = p.Module.Replace.Dir
				}
				if err := add(SourceGo, p.Module.Path, p.Module.Version, dir, t.Name); err != nil {
					return nil, err
				}
			}
		}
	}
	return deps, nil
}

func listDeps(root string, t native.Target) ([]goPackage, error) {
	env := []string{"GOOS=" + t.GOOS, "GOARCH=" + t.GOARCH, "CGO_ENABLED=1"}
	out, err := goCommand(root, env, "list", "-deps", "-json=Standard,Module", "-tags", "release", releaseMain)
	if err != nil {
		return nil, err
	}
	var pkgs []goPackage
	seen := map[string]bool{}
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var p goPackage
		if err := dec.Decode(&p); err != nil {
			return nil, err
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

func goCommand(root string, env []string, args ...string) ([]byte, error) {
	cmd := exec.Command("go", args...)
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

func licenseFiles(dir string) (string, string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", "", err
	}
	var licenses, texts []string
	for _, e := range entries {
		upper := strings.ToUpper(e.Name())
		if !e.Type().IsRegular() || !(strings.HasPrefix(upper, "LICENSE") || strings.HasPrefix(upper, "LICENCE") || strings.HasPrefix(upper, "COPYING")) {
			continue
		}
		text, err := readText(filepath.Join(dir, e.Name()))
		if err != nil {
			return "", "", err
		}
		texts = append(texts, strings.TrimSpace(text))
		if l := Classify(text); !slices.Contains(licenses, l) {
			licenses = append(licenses, l)
		}
	}
	if len(texts) == 0 {
		return noLicense, "", nil
	}
	slices.Sort(licenses)
	return strings.Join(licenses, " AND "), strings.Join(texts, "\n\n"), nil
}
