//go:build linux

package platform

import (
	"os/exec"
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
	return exec.Command("xdg-open", url).Start()
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
