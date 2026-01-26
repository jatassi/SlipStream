//go:build !windows

package platform

import (
	"os/exec"
	"runtime"
)

type app struct {
	config AppConfig
	done   chan struct{}
}

func NewApp(cfg AppConfig) App {
	return &app{
		config: cfg,
		done:   make(chan struct{}),
	}
}

func (a *app) Run() error {
	<-a.done
	return nil
}

func (a *app) OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func (a *app) Stop() {
	select {
	case a.done <- struct{}{}:
	default:
	}
	if a.config.OnQuit != nil {
		a.config.OnQuit()
	}
}
