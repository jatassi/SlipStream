//go:build linux

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	serviceFileName = "slipstream.service"
	serviceName     = "slipstream"
)

type linuxApp struct {
	config AppConfig
	done   chan struct{}
}

func NewApp(cfg AppConfig) App {
	return &linuxApp{
		config: cfg,
		done:   make(chan struct{}),
	}
}

func (a *linuxApp) Run() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
	case <-a.done:
	}

	if a.config.OnQuit != nil {
		a.config.OnQuit()
	}
	return nil
}

func (a *linuxApp) OpenBrowser(url string) error {
	return exec.Command("xdg-open", url).Start()
}

func (a *linuxApp) Stop() {
	select {
	case a.done <- struct{}{}:
	default:
	}
}

func userServiceDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".config", "systemd", "user"), nil
}

func serviceFilePath() (string, error) {
	dir, err := userServiceDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, serviceFileName), nil
}

func EnableStartup(appPath string) error {
	serviceDir, err := userServiceDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return fmt.Errorf("create systemd user dir: %w", err)
	}

	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "slipstream")

	serviceContent := fmt.Sprintf(`[Unit]
Description=SlipStream Media Management
Documentation=https://github.com/slipstream/slipstream
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s
WorkingDirectory=%s
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=default.target
`, appPath, configDir)

	servicePath, err := serviceFilePath()
	if err != nil {
		return err
	}

	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("write service file: %w", err)
	}

	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}

	if err := exec.Command("systemctl", "--user", "enable", serviceName).Run(); err != nil {
		return fmt.Errorf("systemctl enable: %w", err)
	}

	if err := exec.Command("systemctl", "--user", "start", serviceName).Run(); err != nil {
		return fmt.Errorf("systemctl start: %w", err)
	}

	return nil
}

func DisableStartup() error {
	_ = exec.Command("systemctl", "--user", "stop", serviceName).Run()
	_ = exec.Command("systemctl", "--user", "disable", serviceName).Run()

	servicePath, err := serviceFilePath()
	if err != nil {
		return err
	}

	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove service file: %w", err)
	}

	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	return nil
}

func IsStartupEnabled() (bool, error) {
	cmd := exec.Command("systemctl", "--user", "is-enabled", serviceName)
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(output)) == "enabled", nil
}
