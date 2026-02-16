//go:build windows

package platform

import (
	"context"
	"embed"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/tailscale/walk"
)

//go:embed resources/SlipStream.ico
var iconFS embed.FS

type windowsApp struct {
	config     AppConfig
	app        *walk.Application
	notifyIcon *walk.NotifyIcon
	running    bool
	mu         sync.Mutex
	stopOnce   sync.Once
}

func NewApp(cfg AppConfig) App {
	return &windowsApp{config: cfg}
}

func (a *windowsApp) Run() error {
	if a.config.NoTray {
		a.mu.Lock()
		a.running = true
		a.mu.Unlock()
		select {}
	}

	var err error

	// Initialize Walk application - must be called before any other Walk functions
	a.app, err = walk.InitApp()
	if err != nil {
		return err
	}

	walk.App().SetOrganizationName("SlipStream")
	walk.App().SetProductName("SlipStream")

	// Create notify icon (no parent window required)
	a.notifyIcon, err = walk.NewNotifyIcon()
	if err != nil {
		return err
	}

	if err := a.notifyIcon.SetToolTip("SlipStream - Media Management"); err != nil {
		return err
	}

	if icon := loadEmbeddedIcon(); icon != nil {
		a.notifyIcon.SetIcon(icon)
	}

	a.notifyIcon.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			a.OpenBrowser(a.config.ServerURL)
		}
	})

	openAction := walk.NewAction()
	openAction.SetText("Open SlipStream")
	openAction.Triggered().Attach(func() {
		a.OpenBrowser(a.config.ServerURL)
	})

	quitAction := walk.NewAction()
	quitAction.SetText("Quit")
	quitAction.Triggered().Attach(func() {
		a.Stop()
	})

	a.notifyIcon.ContextMenu().Actions().Add(openAction)
	a.notifyIcon.ContextMenu().Actions().Add(walk.NewSeparatorAction())
	a.notifyIcon.ContextMenu().Actions().Add(quitAction)

	if err := a.notifyIcon.SetVisible(true); err != nil {
		return err
	}

	a.mu.Lock()
	a.running = true
	a.mu.Unlock()

	a.app.Run()
	return nil
}

func (a *windowsApp) OpenBrowser(url string) error {
	cmd := exec.CommandContext(context.Background(), "cmd", "/c", "start", "", url)
	return cmd.Start()
}

func (a *windowsApp) Stop() {
	a.stopOnce.Do(func() {
		a.mu.Lock()
		if a.running {
			if a.notifyIcon != nil {
				a.notifyIcon.Dispose()
			}
			if a.app != nil {
				a.app.Exit(0)
			}
		}
		a.mu.Unlock()

		if a.config.OnQuit != nil {
			a.config.OnQuit()
		}
	})
}

func loadEmbeddedIcon() *walk.Icon {
	iconData, err := iconFS.ReadFile("resources/SlipStream.ico")
	if err != nil {
		return nil
	}

	tmpDir := os.TempDir()
	iconPath := filepath.Join(tmpDir, "slipstream-icon.ico")

	if err := os.WriteFile(iconPath, iconData, 0644); err != nil {
		return nil
	}

	icon, err := walk.NewIconFromFile(iconPath)
	if err != nil {
		return nil
	}

	return icon
}
