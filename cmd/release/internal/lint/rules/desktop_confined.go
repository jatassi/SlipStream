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
		dir := path.Dir(f.Path)
		inDesktop := dir == desktopPkg || strings.HasPrefix(dir, desktopPkg+"/")
		for _, imp := range f.Syntax.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			var message string
			switch {
			case (p == systrayModule || strings.HasPrefix(p, systrayModule+"/")) && !inDesktop:
				message = "imports " + p + " outside " + desktopPkg
			case p == systrayModule || strings.HasPrefix(p, systrayModule+"/"):
				linux, err := linuxRelease.MatchFile(filepath.Join(repo.Root, filepath.FromSlash(dir)), path.Base(f.Path))
				if err != nil {
					return nil, fmt.Errorf("%s: %w", f.Path, err)
				}
				if linux {
					message = "imports " + p + " in a file the Linux build compiles; constrain it to windows || darwin"
				}
			case p == path.Join(module, desktopPkg) && !inDesktop && f.Path != appFile:
				message = "imports the desktop package, which only " + appFile + " imports"
			}
			if message != "" {
				findings = append(findings, lint.Finding{Path: f.Path, Line: repo.Fset.Position(imp.Pos()).Line, Message: message})
			}
		}
	}
	return findings, nil
}
