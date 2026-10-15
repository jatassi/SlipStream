package rules

import (
	"fmt"
	"go/build"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "desktop-confined",
		Doc:   "fyne.io/systray and internal/node/internal/desktop are imported only by the desktop package and app, and no file importing systray builds for Linux (spec #181, Lints and linkage)",
		Check: checkDesktopConfined,
	})
}

const (
	systrayModule = "fyne.io/systray"
	desktopPkg    = "internal/node/internal/desktop"
	// appFile is the app subcommand, the one importer of the desktop
	// package, so run never reaches the tray.
	appFile = "internal/node/app.go"
)

// linuxRelease is the Linux release build, which must never link systray.
var linuxRelease = func() build.Context {
	c := build.Default
	c.GOOS, c.GOARCH, c.CgoEnabled, c.BuildTags = "linux", "amd64", true, []string{"release"}
	return c
}()

func checkDesktopConfined(repo *lint.Repo) ([]lint.Finding, error) {
	module, err := modulePath(repo)
	if err != nil {
		return nil, err
	}
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, f := range files {
		for _, imp := range f.Syntax.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			message, err := desktopImport(repo, module, f.Path, p)
			if err != nil {
				return nil, err
			}
			if message != "" {
				findings = append(findings, lint.Finding{Path: f.Path, Line: repo.Fset.Position(imp.Pos()).Line, Message: message})
			}
		}
	}
	return findings, nil
}

// desktopImport is what is wrong with file importing p, or "".
func desktopImport(repo *lint.Repo, module, file, p string) (string, error) {
	dir := path.Dir(file)
	inDesktop := dir == desktopPkg || strings.HasPrefix(dir, desktopPkg+"/")
	systray := p == systrayModule || strings.HasPrefix(p, systrayModule+"/")
	switch {
	case systray && !inDesktop:
		return "imports " + p + " outside " + desktopPkg, nil
	case systray:
		linux, err := linuxRelease.MatchFile(filepath.Join(repo.Root, filepath.FromSlash(dir)), path.Base(file))
		if err != nil {
			return "", fmt.Errorf("%s: %w", file, err)
		}
		if linux {
			return "imports " + p + " in a file the Linux build compiles; constrain it to windows || darwin", nil
		}
	case p == path.Join(module, desktopPkg) && !inDesktop && file != appFile:
		return "imports the desktop package, which only " + appFile + " imports", nil
	}
	return "", nil
}
