// Package pack builds a target's packages from what release build wrote:
// one recipe per install Kind, each staging the program files and the
// Kind's marker, then archiving them the way its users install them.
package pack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
	"github.com/jatassi/slipstream/internal/install"
)

// Input is what one runner's packages are built from.
type Input struct {
	// Root is the repository, which holds LICENSE and the notices file.
	Root string
	// Dist is release build's output for Target.
	Dist      string
	Out       string
	Version   Version
	Target    native.Target
	Inputs    native.Inputs
	CacheRoot string
	// Log receives the output of every program a recipe runs.
	Log io.Writer
}

type recipe struct {
	targets []string
	// build writes the Kind's packages into in.Out from stage, an empty
	// directory of its own, and returns their names there.
	build func(ctx context.Context, in Input, stage string) ([]string, error)
}

var (
	linuxTargets   = []string{"linux-amd64", "linux-arm64"}
	windowsTargets = []string{"windows-amd64"}
	darwinTargets  = []string{"darwin-arm64"}
)

var recipes = map[install.Kind]recipe{
	install.MSI:      {windowsTargets, buildMSI},
	install.ZIP:      {windowsTargets, buildZIP},
	install.App:      {darwinTargets, buildApp},
	install.Deb:      {linuxTargets, buildDeb},
	install.RPM:      {linuxTargets, buildRPM},
	install.AppImage: {[]string{"linux-amd64"}, buildAppImage},
	install.Tarball:  {linuxTargets, buildTarball},
	install.Docker:   {linuxTargets, buildDocker},
}

// Targets names the targets each Kind with a recipe is packaged on.
func Targets() map[install.Kind][]string {
	out := map[install.Kind][]string{}
	for k, r := range recipes {
		out[k] = slices.Clone(r.targets)
	}
	return out
}

// Manifest is what release package wrote into its output directory, which
// release smoke and the publish job read.
type Manifest struct {
	Version string `json:"version"`
	Target  string `json:"target"`
	// Packages names each Kind's files, or the Docker build context's
	// directory, relative to the output directory.
	Packages map[install.Kind][]string `json:"packages"`
	// Assets are the files the GitHub release uploads.
	Assets []string `json:"assets"`
}

// ManifestName is the Manifest's file in the output directory.
const ManifestName = "packages.json"

// Build runs every recipe for in.Target and returns what they wrote.
func Build(ctx context.Context, in Input) (Manifest, error) {
	m := Manifest{Version: in.Version.String(), Target: in.Target.Name, Packages: map[install.Kind][]string{}, Assets: []string{}}
	for _, k := range slices.Sorted(maps.Keys(recipes)) {
		r := recipes[k]
		if !slices.Contains(r.targets, in.Target.Name) {
			continue
		}
		stage, err := os.MkdirTemp("", "slipstream-"+k.String()+"-")
		if err != nil {
			return Manifest{}, fmt.Errorf("stage %s: %w", k, err)
		}
		files, err := r.build(ctx, in, stage)
		_ = os.RemoveAll(stage)
		if err != nil {
			return Manifest{}, fmt.Errorf("package %s: %w", k, err)
		}
		m.Packages[k] = files
		if k != install.Docker {
			m.Assets = append(m.Assets, files...)
		}
	}
	return m, nil
}

func (m Manifest) Write(dir string) error {
	data, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		return fmt.Errorf("encode the package manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ManifestName), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write the package manifest: %w", err)
	}
	return nil
}

func ReadManifest(dir string) (Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, ManifestName))
	if err != nil {
		return Manifest{}, fmt.Errorf("read the package manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("%s: %w", ManifestName, err)
	}
	return m, nil
}
