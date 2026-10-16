// Package desktop is the app subcommand's desktop layer: the tray on
// Windows, the menu bar item on macOS, the browser opener, the native
// alert and the login item. Only app imports it, and only its
// windows-or-darwin files import fyne.io/systray, so neither run nor the
// Linux binary carries the tray.
package desktop

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
)

// App is the running Node as the desktop layer drives it.
type App interface {
	// Status is the tray's status line and the UI's address.
	Status(ctx context.Context) (Status, error)
	// Stop asks the Node to stop. Done closes once the process may exit.
	Stop()
	Done() <-chan struct{}
	// Leave runs after Done, on the main thread when RestartOnMainThread:
	// it starts the successor a restart asked for there.
	Leave()
}

type Status struct {
	Line string
	URL  string
}

type Options struct {
	Log *slog.Logger
}

// TrayState is the tray as status reports it in app mode.
type TrayState string

const (
	// TrayNone is an OS the app shows no tray on: Linux.
	TrayNone     TrayState = "none"
	TrayStarting TrayState = "starting"
	TrayReady    TrayState = "ready"
)

// Open opens url in the person's browser and returns once the opener has
// exited. On Linux $BROWSER, when set, names the browser.
func Open(url string) error {
	browser, _ := os.LookupEnv("BROWSER")
	args := opener(runtime.GOOS, browser, url)
	if err := exec.CommandContext(context.Background(), args[0], args[1:]...).Run(); err != nil { //nolint:gosec // G204: the OS's opener, or the browser the person named
		return fmt.Errorf("open %s with %s: %w", url, args[0], err)
	}
	return nil
}

// opener is the command that opens url on goos. rundll32 hands the URL to
// its protocol handler with no shell between, so the fragment survives.
func opener(goos, browser, url string) []string {
	switch {
	case goos == "windows":
		return []string{"rundll32", "url.dll,FileProtocolHandler", url}
	case goos == "darwin":
		return []string{"/usr/bin/open", url}
	case browser != "":
		return []string{browser, url}
	}
	return []string{"xdg-open", url}
}
