package pack

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
	"github.com/jatassi/slipstream/internal/install"
)

func buildTarball(_ context.Context, in Input, stage string) ([]string, error) {
	top := install.Executable + "-" + in.Version.String()
	dir := filepath.Join(stage, top)
	if err := in.stagePrograms(dir, install.Tarball); err != nil {
		return nil, err
	}
	if err := in.stageLegal(dir); err != nil {
		return nil, err
	}
	name := fmt.Sprintf("%s-%s-linux-%s.tar.gz", install.Executable, in.Version, in.Target.GOARCH)
	return []string{name}, writeTarGz(filepath.Join(in.Out, name), stage, in.programs())
}

const (
	sysusersPath = "/usr/lib/sysusers.d/slipstream.conf"
	unitPath     = "/usr/lib/systemd/system/" + install.UnitName
	docDir       = "/usr/share/doc/slipstream"
	maintainer   = "jatassi <38116120+jatassi@users.noreply.github.com>"
	homepage     = "https://github.com/jatassi/SlipStream"
	description  = "SlipStream manages a movie and TV library across your machines."
)

type systemPackage struct {
	kind    install.Kind
	arch    map[string]string
	depends string
	// fresh is the shell test that is true on a first install and false
	// on an upgrade, in the packager's maintainer script arguments.
	fresh string
	name  func(v Version, arch string) string
}

var (
	debPackage = systemPackage{
		kind:    install.Deb,
		arch:    map[string]string{"amd64": "amd64", "arm64": "arm64"},
		depends: "libc6 (>= " + install.GlibcFloor + ")",
		fresh:   `[ -z "$2" ]`,
		name:    func(v Version, arch string) string { return fmt.Sprintf("slipstream_%s_%s.deb", v, arch) },
	}
	rpmPackage = systemPackage{
		kind:    install.RPM,
		arch:    map[string]string{"amd64": "x86_64", "arm64": "aarch64"},
		depends: "glibc >= " + install.GlibcFloor,
		fresh:   `[ "$1" -eq 1 ]`,
		name:    func(v Version, arch string) string { return fmt.Sprintf("slipstream-%s-1.%s.rpm", v, arch) },
	}
)

func buildDeb(ctx context.Context, in Input, stage string) ([]string, error) {
	return buildSystemPackage(ctx, in, stage, debPackage)
}

func buildRPM(ctx context.Context, in Input, stage string) ([]string, error) {
	return buildSystemPackage(ctx, in, stage, rpmPackage)
}

type scriptData struct {
	names
	Deb      bool
	Fresh    string
	Unit     string
	Sysusers string
	Glibc    string
	Kernel   string
}

func maintainerScripts(p systemPackage) map[string]string {
	data := scriptData{
		names: installNames, Deb: p.kind == install.Deb, Fresh: p.fresh, Unit: install.UnitName,
		Sysusers: sysusersPath, Glibc: install.GlibcFloor, Kernel: install.KernelFloor,
	}
	return map[string]string{
		"preinstall":  render("preinst.sh", data),
		"postinstall": render("postinst.sh", data),
		"preremove":   render("prerm.sh", data),
	}
}

func sysusers() string {
	return render("sysusers.conf", struct {
		names
		Home string
	}{installNames, "/var/lib/slipstream"})
}

// nfpmConfig is nfpm's package description. nfpm reads YAML, of which
// JSON is a part.
type nfpmConfig struct {
	Name          string            `json:"name"`
	Arch          string            `json:"arch"`
	Platform      string            `json:"platform"`
	Version       string            `json:"version"`
	VersionSchema string            `json:"version_schema"`
	Release       string            `json:"release"`
	Maintainer    string            `json:"maintainer"`
	Description   string            `json:"description"`
	Vendor        string            `json:"vendor"`
	Homepage      string            `json:"homepage"`
	License       string            `json:"license"`
	Depends       []string          `json:"depends"`
	Contents      []nfpmContent     `json:"contents"`
	Scripts       map[string]string `json:"scripts"`
}

type nfpmContent struct {
	Src  string `json:"src"`
	Dst  string `json:"dst"`
	Type string `json:"type,omitempty"`
}

func (c nfpmConfig) encode() (string, error) {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(c); err != nil {
		return "", fmt.Errorf("encode the nfpm config: %w", err)
	}
	return b.String(), nil
}

func systemPackageConfig(in Input, p systemPackage, root string, scripts map[string]string) nfpmConfig {
	libDir := path.Dir(install.SystemExecutable)
	root = filepath.ToSlash(root)
	var contents []nfpmContent
	for _, name := range append(in.programs(), install.MarkerName) {
		contents = append(contents, nfpmContent{Src: path.Join(root, "lib", name), Dst: path.Join(libDir, name)})
	}
	contents = append(contents,
		nfpmContent{Src: install.SystemExecutable, Dst: "/usr/bin/" + install.Executable, Type: "symlink"},
		nfpmContent{Src: path.Join(root, install.UnitName), Dst: unitPath},
		nfpmContent{Src: path.Join(root, "sysusers.conf"), Dst: sysusersPath},
		nfpmContent{Src: path.Join(root, "doc", "LICENSE"), Dst: docDir + "/LICENSE"},
		nfpmContent{Src: path.Join(root, "doc", "NOTICES.txt"), Dst: docDir + "/NOTICES.txt"},
	)
	return nfpmConfig{
		Name: install.Executable, Arch: p.arch[in.Target.GOARCH], Platform: "linux",
		Version: in.Version.String(), VersionSchema: "semver", Release: "1",
		Maintainer: maintainer, Description: description, Vendor: install.ProductName, Homepage: homepage,
		License: "GPL-3.0", Depends: []string{p.depends}, Contents: contents, Scripts: scripts,
	}
}

func buildSystemPackage(ctx context.Context, in Input, stage string, p systemPackage) ([]string, error) {
	root := filepath.Join(stage, "root")
	lib := filepath.Join(root, "lib")
	if err := in.stagePrograms(lib, p.kind); err != nil {
		return nil, err
	}
	if err := in.stageLegal(filepath.Join(root, "doc")); err != nil {
		return nil, err
	}
	if err := writeText(filepath.Join(root, install.UnitName), install.SystemUnit(), 0o644); err != nil {
		return nil, err
	}
	if err := writeText(filepath.Join(root, "sysusers.conf"), sysusers(), 0o644); err != nil {
		return nil, err
	}
	scripts := map[string]string{}
	for script, text := range maintainerScripts(p) {
		scripts[script] = filepath.Join(stage, script+".sh")
		if err := writeText(scripts[script], text, 0o755); err != nil {
			return nil, err
		}
	}
	config, err := systemPackageConfig(in, p, root, scripts).encode()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(stage, "nfpm.yaml")
	if err := writeText(configPath, config, 0o644); err != nil {
		return nil, err
	}
	name := p.name(in.Version, p.arch[in.Target.GOARCH])
	return []string{name}, in.run(ctx, in.Root, nil, "go", "tool", "-modfile=tools/go.mod", "nfpm", "package",
		"--config", configPath, "--packager", p.kind.String(), "--target", filepath.Join(in.Out, name))
}

const desktopArgs = "app"

type desktopData struct {
	names
	DesktopArgs string
}

func buildAppImage(ctx context.Context, in Input, stage string) ([]string, error) {
	appDir := filepath.Join(stage, install.ProductName+".AppDir")
	if err := in.stagePrograms(filepath.Join(appDir, "usr", "bin"), install.AppImage); err != nil {
		return nil, err
	}
	if err := in.stageLegal(filepath.Join(appDir, "usr", "share", "doc", install.Executable)); err != nil {
		return nil, err
	}
	data := desktopData{installNames, desktopArgs}
	if err := writeText(filepath.Join(appDir, "AppRun"), render("AppRun", data), 0o755); err != nil {
		return nil, err
	}
	if err := writeText(filepath.Join(appDir, install.Executable+".desktop"), render("slipstream.desktop", data), 0o644); err != nil {
		return nil, err
	}
	if err := copyIcon(256, filepath.Join(appDir, install.Executable+".png")); err != nil {
		return nil, err
	}
	tool, err := in.packager(ctx, "appimagetool")
	if err != nil {
		return nil, err
	}
	runtime, err := in.packager(ctx, "appimage-runtime")
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf("%s-%s-x86_64.AppImage", install.ProductName, in.Version)
	// appimagetool is itself an AppImage; extracting it needs no FUSE.
	env := []string{"ARCH=x86_64", "APPIMAGE_EXTRACT_AND_RUN=1"}
	return []string{name}, in.run(ctx, stage, env, tool, "--no-appstream", "--runtime-file", runtime, appDir, filepath.Join(in.Out, name))
}

func (in Input) packager(ctx context.Context, name string) (string, error) {
	p, err := in.Inputs.Packager(name, in.Target.Name)
	if err != nil {
		return "", err
	}
	file, err := native.FetchFile(ctx, in.CacheRoot, p.Download)
	if err != nil {
		return "", err
	}
	if err := os.Chmod(file, 0o755); err != nil { //nolint:gosec // G302: a program the recipe runs
		return "", fmt.Errorf("make %s runnable: %w", name, err)
	}
	return file, nil
}

const imageDir = "/usr/local/lib/slipstream"

type dockerData struct {
	names
	Version string
	Dir     string
}

func buildDocker(ctx context.Context, in Input, _ string) ([]string, error) {
	const contextDir = "docker"
	dir := filepath.Join(in.Out, contextDir)
	program := filepath.Join(dir, "program")
	if err := copyFile(filepath.Join(in.Dist, install.Executable), filepath.Join(program, install.Executable), 0o755); err != nil {
		return nil, err
	}
	if err := in.writeMarker(filepath.Join(program, install.Executable), install.Docker); err != nil {
		return nil, err
	}
	if err := in.stageLegal(program); err != nil {
		return nil, err
	}
	for _, tool := range in.Inputs.Tools {
		if err := in.stageImageTool(ctx, tool, dir, program); err != nil {
			return nil, err
		}
	}
	data := dockerData{installNames, in.Version.String(), imageDir}
	if err := writeText(filepath.Join(dir, "Dockerfile"), render("Dockerfile", data), 0o644); err != nil {
		return nil, err
	}
	return []string{contextDir}, writeText(filepath.Join(program, "entrypoint"), render("entrypoint", data), 0o755)
}

func (in Input) stageImageTool(ctx context.Context, tool native.Tool, dir, program string) error {
	if tool.Debian == nil {
		for _, p := range tool.Programs {
			if err := copyFile(filepath.Join(in.Dist, p), filepath.Join(program, p), 0o755); err != nil {
				return err
			}
		}
		return nil
	}
	file, err := native.FetchFile(ctx, in.CacheRoot, tool.Debian.Packages[in.Target.Name])
	if err != nil {
		return err
	}
	if err := copyFile(file, filepath.Join(dir, filepath.Base(file)), 0o644); err != nil {
		return err
	}
	for _, p := range tool.Programs {
		if err := os.Symlink(path.Join(tool.Debian.Dir, p), filepath.Join(program, p)); err != nil {
			return fmt.Errorf("link %s: %w", p, err)
		}
	}
	return nil
}
