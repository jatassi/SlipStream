//go:build windows || darwin

package desktop

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"fyne.io/systray"

	"github.com/jatassi/slipstream/internal/install"
)

// HasLoginItem reports whether the OS has a login item the app manages.
const HasLoginItem = true

const statusPoll = 2 * time.Second

var trayReady atomic.Bool

func Tray() TrayState {
	if trayReady.Load() {
		return TrayReady
	}
	return TrayStarting
}

// Run shows the tray on the main thread, which fyne.io/systray locks at
// init, and returns once app is done. systray calls the exit hook on Quit
// and when AppKit terminates the app at a macOS logout or a Finder quit;
// AppKit exits only once the hook returns, so the stop finishes first.
func Run(app App, opts Options) {
	t := &tray{app: app, log: opts.Log}
	systray.Run(t.ready, func() {
		app.Stop()
		<-app.Done()
		app.Leave()
	})
}

type tray struct {
	app App
	log *slog.Logger
}

type menu struct {
	status, open, login, quit *systray.MenuItem
}

func (t *tray) ready() {
	t.osReady()
	systray.SetTooltip(install.ProductName)
	m := menu{status: systray.AddMenuItem(install.ProductName, "")}
	m.status.Disable()
	m.open = systray.AddMenuItem("Open "+install.ProductName, "")
	on, err := StartAtLogin()
	if err != nil {
		t.log.Warn("read the login item", "err", err.Error())
	}
	m.login = systray.AddMenuItemCheckbox("Start at login", "", on)
	systray.AddSeparator()
	m.quit = systray.AddMenuItem("Quit", "")
	trayReady.Store(true)
	t.log.Info("tray", "state", TrayReady)
	go t.serve(m)
}

func (t *tray) serve(m menu) {
	tick := time.NewTicker(statusPoll)
	defer tick.Stop()
	var url string
	stopping := false
	refresh := func() {
		ctx, cancel := context.WithTimeout(context.Background(), statusPoll)
		defer cancel()
		s, err := t.app.Status(ctx)
		if err != nil {
			t.log.Debug("tray status", "err", err.Error())
			return
		}
		m.status.SetTitle(s.Line)
		url = s.URL
	}
	refresh()
	for {
		select {
		case <-tick.C:
			if !stopping {
				refresh()
			}
		case <-m.open.ClickedCh:
			if url != "" {
				go t.open(url)
			}
		case <-m.login.ClickedCh:
			t.toggleLogin(m.login)
		case <-m.quit.ClickedCh:
			stopping = true
			m.status.SetTitle("Stopping " + install.ProductName)
			m.quit.Disable()
			t.app.Stop()
		case <-t.app.Done():
			systray.Quit()
			return
		}
	}
}

func (t *tray) open(url string) {
	if err := Open(url); err != nil {
		t.log.Warn("open the browser", "err", err.Error())
	}
}

func (t *tray) toggleLogin(item *systray.MenuItem) {
	on := !item.Checked()
	if err := SetStartAtLogin(on); err != nil {
		t.log.Warn("change the login item", "on", on, "err", err.Error())
		return
	}
	if on {
		item.Check()
	} else {
		item.Uncheck()
	}
}

func trampoline() (string, error) {
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	if err != nil {
		return "", fmt.Errorf("find the executable: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), install.Trampoline+filepath.Ext(exe)), nil
}
