package pack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
	"github.com/jatassi/slipstream/internal/install"
)

// upgradeCode is the MSI's UpgradeCode for every v2 release, so each one
// replaces the last. v1's MSI has its own, so v2 never upgrades or
// removes v1.
const upgradeCode = "{73394E63-47CD-417B-AEDD-7F50B149F574}"

// programFilesGUID is fixed because the program files' component has an
// HKCU key path, which ICE38 requires of every component under a user's
// profile folder, and WiX cannot generate the GUID of a component that
// holds files under a registry key path.
const programFilesGUID = "{C20646BA-DEE1-4ACA-9C3E-B3E9C252C265}"

const wixVersion = "5.0.2"

func buildZIP(_ context.Context, in Input, stage string) ([]string, error) {
	dir := filepath.Join(stage, install.ProductName)
	if err := in.stagePrograms(dir, install.ZIP); err != nil {
		return nil, err
	}
	if err := in.stageLegal(dir); err != nil {
		return nil, err
	}
	readme := render("README.txt", struct {
		names
		Version Version
	}{installNames, in.Version})
	if err := writeText(filepath.Join(dir, "README.txt"), crlf(readme), 0o644); err != nil {
		return nil, err
	}
	name := fmt.Sprintf("%s-%s-windows-amd64.zip", install.ProductName, in.Version)
	return []string{name}, writeZip(filepath.Join(in.Out, name), stage)
}

func crlf(s string) string { return strings.ReplaceAll(s, "\n", "\r\n") }

type wixFile struct{ ID, Name, Source string }

type wixData struct {
	names
	Version          Version
	Numeric          string
	UpgradeCode      string
	ProgramFilesGUID string
	ProgramsDir      string
	InstallDir       string
	RegistryKey      string
	RunValue         string
	TrampolineExe    string
	Files            []wixFile
	LicenseRTF       string
}

func wixSource(in Input, dir string, files []string) string {
	programs, installDir, _ := strings.Cut(install.WindowsProgramDir, `\`)
	data := wixData{
		names: installNames, Version: in.Version, Numeric: in.Version.Numeric(), UpgradeCode: upgradeCode, ProgramFilesGUID: programFilesGUID,
		ProgramsDir: programs, InstallDir: installDir, RegistryKey: install.WindowsRegistryKey, RunValue: install.WindowsRunValue,
		LicenseRTF: filepath.Join(dir, "LICENSE.rtf"), TrampolineExe: native.Program(in.Target, install.Trampoline),
	}
	for _, f := range files {
		id := strings.NewReplacer(".", "_", "-", "_").Replace(f)
		data.Files = append(data.Files, wixFile{ID: id, Name: f, Source: filepath.Join(dir, f)})
	}
	return render("SlipStream.wxs", data)
}

func buildMSI(ctx context.Context, in Input, stage string) ([]string, error) {
	dir := filepath.Join(stage, "files")
	if err := in.stagePrograms(dir, install.MSI); err != nil {
		return nil, err
	}
	if err := in.stageLegal(dir); err != nil {
		return nil, err
	}
	license, err := os.ReadFile(filepath.Join(dir, "LICENSE"))
	if err != nil {
		return nil, fmt.Errorf("read LICENSE: %w", err)
	}
	if err := writeText(filepath.Join(dir, "LICENSE.rtf"), rtf(string(license)), 0o644); err != nil {
		return nil, err
	}
	files := append(in.programs(), install.MarkerName, "LICENSE", "NOTICES.txt")
	source := filepath.Join(stage, "SlipStream.wxs")
	if err := writeText(source, wixSource(in, dir, files), 0o644); err != nil {
		return nil, err
	}
	name := fmt.Sprintf("%s-%s-windows-amd64.msi", install.ProductName, in.Version)
	ui := "WixToolset.UI.wixext/" + wixVersion
	for _, args := range [][]string{
		{"tool", "restore"},
		{"tool", "run", "wix", "extension", "add", "-g", ui},
		{"tool", "run", "wix", "build", "-arch", "x64", "-ext", ui, "-o", filepath.Join(in.Out, name), source},
	} {
		if err := in.run(ctx, in.Root, nil, "dotnet", args...); err != nil {
			return nil, err
		}
	}
	return []string{name}, nil
}

// rtf is plain text as the RTF the MSI's license dialog shows.
func rtf(text string) string {
	var b strings.Builder
	b.WriteString(`{\rtf1\ansi\deff0{\fonttbl{\f0 Consolas;}}\f0\fs18` + "\n")
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		b.WriteString(strings.NewReplacer(`\`, `\\`, "{", `\{`, "}", `\}`).Replace(line))
		b.WriteString("\\par\n")
	}
	b.WriteString("}\n")
	return b.String()
}
