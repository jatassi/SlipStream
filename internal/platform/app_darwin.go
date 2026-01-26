//go:build darwin

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/macos/foundation"
	"github.com/progrium/darwinkit/objc"
)

const (
	launchAgentLabel = "com.slipstream.slipstream"
	launchAgentPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.AppPath}}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>`
)

type macApp struct {
	config      AppConfig
	statusItem  appkit.StatusItem
	startupItem appkit.MenuItem
	running     bool
	mu          sync.Mutex
	stopOnce    sync.Once
}

func NewApp(cfg AppConfig) App {
	return &macApp{config: cfg}
}

func (a *macApp) Run() error {
	if a.config.NoTray {
		a.mu.Lock()
		a.running = true
		a.mu.Unlock()
		select {}
	}

	objc.WithAutoreleasePool(func() {
		app := appkit.Application_SharedApplication()
		app.SetActivationPolicy(appkit.ApplicationActivationPolicyAccessory)

		a.statusItem = appkit.StatusBar_SystemStatusBar().StatusItemWithLength(appkit.VariableStatusItemLength)

		if button := a.statusItem.Button(); button.Ptr != nil {
			button.SetTitle("SlipStream")
		}

		menu := appkit.NewMenu()

		openItem := appkit.NewMenuItemWithAction("Open SlipStream", "", func(sender objc.Object) {
			a.OpenBrowser(a.config.ServerURL)
		})
		menu.AddItem(openItem)

		menu.AddItem(appkit.MenuItem_SeparatorItem())

		startupEnabled, _ := IsStartupEnabled()
		startupTitle := "Enable Start at Login"
		if startupEnabled {
			startupTitle = "Disable Start at Login"
		}
		a.startupItem = appkit.NewMenuItemWithAction(startupTitle, "", func(sender objc.Object) {
			a.toggleStartup()
		})
		menu.AddItem(a.startupItem)

		menu.AddItem(appkit.MenuItem_SeparatorItem())

		quitItem := appkit.NewMenuItemWithAction("Quit", "q", func(sender objc.Object) {
			a.Stop()
		})
		menu.AddItem(quitItem)

		a.statusItem.SetMenu(menu)

		a.mu.Lock()
		a.running = true
		a.mu.Unlock()

		app.Run()
	})

	return nil
}

func (a *macApp) toggleStartup() {
	enabled, _ := IsStartupEnabled()
	if enabled {
		DisableStartup()
	} else {
		appPath, _ := os.Executable()
		EnableStartup(appPath)
	}

	newEnabled, _ := IsStartupEnabled()
	title := "Enable Start at Login"
	if newEnabled {
		title = "Disable Start at Login"
	}

	if a.startupItem.Ptr != nil {
		a.startupItem.SetTitle(title)
	}
}

func (a *macApp) OpenBrowser(url string) error {
	nsURL := foundation.URL_URLWithString(url)
	if nsURL.Ptr == nil {
		return exec.Command("open", url).Start()
	}
	appkit.Workspace_SharedWorkspace().OpenURL(nsURL)
	return nil
}

func (a *macApp) Stop() {
	a.stopOnce.Do(func() {
		a.mu.Lock()
		wasRunning := a.running
		a.running = false
		a.mu.Unlock()

		if wasRunning {
			objc.WithAutoreleasePool(func() {
				app := appkit.Application_SharedApplication()
				app.Terminate(nil)
			})
		}

		if a.config.OnQuit != nil {
			a.config.OnQuit()
		}
	})
}

func launchAgentPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", launchAgentLabel+".plist")
}

func EnableStartup(appPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	launchAgentsDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	tmpl, err := template.New("plist").Parse(launchAgentPlist)
	if err != nil {
		return fmt.Errorf("parse plist template: %w", err)
	}

	plistPath := launchAgentPath()
	f, err := os.Create(plistPath)
	if err != nil {
		return fmt.Errorf("create plist file: %w", err)
	}
	defer f.Close()

	data := struct {
		Label   string
		AppPath string
	}{
		Label:   launchAgentLabel,
		AppPath: appPath,
	}

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	cmd := exec.Command("launchctl", "load", plistPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("launchctl load: %w", err)
	}

	return nil
}

func DisableStartup() error {
	plistPath := launchAgentPath()

	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return nil
	}

	cmd := exec.Command("launchctl", "unload", plistPath)
	_ = cmd.Run()

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}

	return nil
}

func IsStartupEnabled() (bool, error) {
	plistPath := launchAgentPath()
	_, err := os.Stat(plistPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
