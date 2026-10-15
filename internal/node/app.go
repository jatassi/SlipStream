package node

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node/internal/desktop"
	"github.com/jatassi/slipstream/internal/node/internal/logs"
	"github.com/jatassi/slipstream/internal/system"
)

// app is the desktop launch: run's lifecycle with the tray on the main
// goroutine, and the setup page in the browser while the Node is
// Unclaimed. A data dir another Node holds opens that Node's page instead.
// A failure before ready has no console to read, so it also shows an
// alert, and exits with run's code.
func app(args []string, types contenttype.Set) int {
	env, err := nodeEnv("app", args, types)
	if err != nil {
		desktop.Alert(install.ProductName, err.Error())
		return usage(err)
	}
	env.Mode = ModeApp
	_, err = os.Stat(env.DataDir.Path())
	fresh := errors.Is(err, fs.ErrNotExist)
	ready := false
	err = serve(context.Background(), env, func(n *Node, exit <-chan struct{}) {
		ready = true
		n.appReady(fresh)
		desktop.Run(appNode{n: n, exit: exit}, desktop.Options{Log: n.log})
	})
	if _, held := errors.AsType[*InUseError](err); held {
		return openRunning(env)
	}
	if code := ExitCode(err); !ready && code != ExitOK {
		desktop.Alert(install.ProductName, fmt.Sprintf("SlipStream could not start: %v\n\nThe log is %s", err, filepath.Join(env.DataDir.LogsDir(), logs.File)))
	}
	return ExitCode(err)
}

// appReady turns Start at login on for a macOS app's first launch, which
// never overrides a person who turned it off, and opens the setup page.
func (n *Node) appReady(fresh bool) {
	if fresh && n.env.Install == install.App {
		if err := desktop.SetStartAtLogin(true); err != nil {
			n.log.Warn("turn on start at login", "err", err.Error())
		}
	}
	if n.ready.Claim != system.Unclaimed {
		return
	}
	go func() {
		if err := desktop.Open(setupURL(n.ready.Listeners.HTTP, n.ready.SetupCode)); err != nil {
			n.log.Warn("open the setup page", "err", err.Error())
		}
	}()
}

// openRunning opens the page of the Node that holds env's data dir, so a
// second launch opens the app rather than failing.
func openRunning(env Env) int {
	reply, err := awaitStatus(env)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream app: %v\n", err)
		desktop.Alert(install.ProductName, fmt.Sprintf("SlipStream is already running on %s and did not answer: %v", env.DataDir, err))
		return ExitInUse
	}
	if err := desktop.Open(reply.URL); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream app: %v\n", err)
	}
	return ExitOK
}

// appStatus adds the tray's state and the login item to an app's status.
func appStatus(r *StatusReply) error {
	r.Tray = string(desktop.Tray())
	if !desktop.HasLoginItem {
		return nil
	}
	on, err := desktop.StartAtLogin()
	if err != nil {
		return err
	}
	r.StartAtLogin = &on
	return nil
}

// appNode is the Node as the tray drives it.
type appNode struct {
	n    *Node
	exit <-chan struct{}
}

func (a appNode) Status(ctx context.Context) (desktop.Status, error) {
	s, err := a.n.Status(ctx)
	if err != nil {
		return desktop.Status{}, err
	}
	return desktop.Status{Line: trayLine(s), URL: s.URL}, nil
}

func (a appNode) Stop() { a.n.Stop() }

func (a appNode) Done() <-chan struct{} { return a.exit }

// trayLine is the tray's status line.
func trayLine(s StatusReply) string {
	switch s.Claim {
	case system.Controller:
		return "Controller · running"
	case system.Worker:
		if s.Controller == nil {
			return "Worker · joining"
		}
		state := "running"
		if !s.Controller.Reachable {
			state = "Controller unreachable"
		}
		return "Worker of " + s.Controller.Name + " · " + state
	case system.Unclaimed:
	}
	return "Unclaimed · open " + install.ProductName + " to set up"
}
