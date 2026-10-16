package node

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node/internal/desktop"
	"github.com/jatassi/slipstream/internal/node/internal/logs"
	"github.com/jatassi/slipstream/internal/system"
)

func app(args []string, types contenttype.Set) int {
	env, err := nodeEnv("app", args, types)
	if err != nil {
		desktop.Alert(install.ProductName, err.Error())
		return usage(err)
	}
	env.Mode = ModeApp
	leave := func() {}
	if desktop.RestartOnMainThread {
		env.Restart, leave = deferRestart(env.Restart)
	}
	_, err = os.Stat(env.DataDir.Path())
	fresh := errors.Is(err, fs.ErrNotExist)
	ready := false
	err = serve(context.Background(), env, func(n *Node, exit <-chan struct{}) {
		ready = true
		n.appReady(fresh)
		desktop.Run(appNode{n: n, exit: exit, leave: leave}, desktop.Options{Log: n.log})
	})
	if _, held := errors.AsType[*InUseError](err); held {
		return openRunning(env)
	}
	if code := ExitCode(err); !ready && code != ExitOK {
		desktop.Alert(install.ProductName, fmt.Sprintf("SlipStream could not start: %v\n\nThe log is %s", err, filepath.Join(env.DataDir.LogsDir(), logs.File)))
	}
	return ExitCode(err)
}

func (n *Node) appReady(fresh bool) {
	if fresh && n.env.Install == install.App {
		if err := desktop.SetStartAtLogin(true); err != nil {
			n.log.Warn("turn on start at login", "err", err.Error())
		}
	}
	if n.ready.Claim != system.Unclaimed || n.env.RestartOf != 0 {
		return
	}
	go func() {
		if err := desktop.Open(setupURL(n.ready.Listeners.HTTP, n.ready.SetupCode)); err != nil {
			n.log.Warn("open the setup page", "err", err.Error())
		}
	}()
}

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

// deferRestart splits restart in two: the Node's call records the request,
// and leave, on the main thread, starts the successor. A failed start has
// been reported, and exits 1.
func deferRestart(restart func() error) (record func() error, leave func()) {
	var asked atomic.Bool
	record = func() error {
		asked.Store(true)
		return nil
	}
	leave = func() {
		if asked.Load() && restart() != nil {
			os.Exit(ExitFailed) //nolint:revive // AppKit exits 0 once the tray's exit hook returns
		}
	}
	return record, leave
}

type appNode struct {
	n     *Node
	exit  <-chan struct{}
	leave func()
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

func (a appNode) Leave() { a.leave() }

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
