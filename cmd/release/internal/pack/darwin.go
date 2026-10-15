package pack

import (
	"bytes"
	"context"
	"embed"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jatassi/slipstream/internal/install"
)

//go:embed icon
var iconFS embed.FS

func copyIcon(size int, dst string) error {
	png, err := iconFS.ReadFile(fmt.Sprintf("icon/icon%d.png", size))
	if err != nil {
		return fmt.Errorf("read the %d-pixel icon: %w", size, err)
	}
	return writeFile(dst, bytes.NewReader(png), 0o644)
}

// icns is the app icon as an Apple icon file, whose ic08, ic09 and ic10
// entries each hold a PNG.
func icns() ([]byte, error) {
	var entries bytes.Buffer
	for _, e := range []struct {
		kind string
		size int
	}{{"ic08", 256}, {"ic09", 512}, {"ic10", 1024}} {
		png, err := iconFS.ReadFile(fmt.Sprintf("icon/icon%d.png", e.size))
		if err != nil {
			return nil, fmt.Errorf("read the %d-pixel icon: %w", e.size, err)
		}
		entries.WriteString(e.kind)
		_ = binary.Write(&entries, binary.BigEndian, uint32(8+len(png))) //nolint:gosec // G115: an icon is far below 4 GiB
		entries.Write(png)
	}
	var out bytes.Buffer
	out.WriteString("icns")
	_ = binary.Write(&out, binary.BigEndian, uint32(8+entries.Len())) //nolint:gosec // G115: an icon is far below 4 GiB
	out.Write(entries.Bytes())
	return out.Bytes(), nil
}

type plistData struct {
	names
	Numeric string
	MinOS   string
}

// infoPlist is the bundle's Info.plist. The trampoline is the bundle's
// executable, because a bundle launch passes no arguments.
func infoPlist(in Input) string {
	minOS := strings.TrimPrefix(in.Target.MinOS, "macos ")
	return render("Info.plist", plistData{installNames, in.Version.Numeric(), minOS})
}

// buildApp writes SlipStream.app as a DMG for people and as a zip for
// the updater, each ad-hoc signed inside out.
func buildApp(ctx context.Context, in Input, stage string) ([]string, error) {
	app := filepath.Join(stage, "dmg", install.ProductName+".app")
	contents := filepath.Join(app, "Contents")
	macOS := filepath.Join(contents, "MacOS")
	if err := in.stagePrograms(macOS, install.App); err != nil {
		return nil, err
	}
	resources := filepath.Join(contents, "Resources")
	if err := in.stageLegal(resources); err != nil {
		return nil, err
	}
	icon, err := icns()
	if err != nil {
		return nil, err
	}
	if err := writeFile(filepath.Join(resources, install.ProductName+".icns"), bytes.NewReader(icon), 0o644); err != nil {
		return nil, err
	}
	if err := writeText(filepath.Join(contents, "Info.plist"), infoPlist(in), 0o644); err != nil {
		return nil, err
	}
	for _, p := range in.programs() {
		if err := in.run(ctx, stage, nil, "codesign", "--force", "--sign", "-", "--timestamp=none", filepath.Join(macOS, p)); err != nil {
			return nil, err
		}
	}
	if err := in.run(ctx, stage, nil, "codesign", "--force", "--sign", "-", "--timestamp=none", app); err != nil {
		return nil, err
	}
	if err := in.run(ctx, stage, nil, "codesign", "--verify", "--deep", "--strict", app); err != nil {
		return nil, err
	}
	if err := os.Symlink("/Applications", filepath.Join(stage, "dmg", "Applications")); err != nil {
		return nil, fmt.Errorf("link Applications into the disk image: %w", err)
	}
	dmg := fmt.Sprintf("%s-%s-darwin-arm64.dmg", install.ProductName, in.Version)
	zip := fmt.Sprintf("%s-%s-darwin-arm64.app.zip", install.ProductName, in.Version)
	if err := in.run(ctx, stage, nil, "hdiutil", "create", "-volname", install.ProductName, "-srcfolder", filepath.Join(stage, "dmg"),
		"-fs", "HFS+", "-format", "UDZO", "-ov", filepath.Join(in.Out, dmg)); err != nil {
		return nil, err
	}
	if err := in.run(ctx, stage, nil, "ditto", "-c", "-k", "--sequesterRsrc", "--keepParent", app, filepath.Join(in.Out, zip)); err != nil {
		return nil, err
	}
	return []string{dmg, zip}, nil
}
