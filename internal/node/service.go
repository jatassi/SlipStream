package node

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/install"
)

const serviceStartWait = 60 * time.Second

func service(args []string) int {
	if len(args) == 0 || (args[0] != "install" && args[0] != "uninstall") {
		return usage(errors.New("service takes install or uninstall"))
	}
	action := args[0]
	flags := flag.NewFlagSet("service "+action, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	user := flags.Bool("user", false, "")
	dataDir := flags.String("data-dir", "", "")
	if err := flags.Parse(args[1:]); err != nil {
		return usage(fmt.Errorf("service %s: %w", action, err))
	}
	if flags.NArg() > 0 {
		return usage(fmt.Errorf("service %s: unexpected argument %q", action, flags.Arg(0)))
	}
	if !*user {
		return usage(fmt.Errorf("service %s takes --user; the deb and rpm install the system service", action))
	}
	env, err := ProcessEnv("service "+action, *dataDir)
	if err != nil {
		return usage(fmt.Errorf("service %s: %w", action, err))
	}
	layout, err := processLayout(env.Environ)
	if err != nil {
		return serviceFailed(err)
	}
	if action == "uninstall" {
		return uninstallUserService(env, layout)
	}
	return installUserService(env, layout)
}

func installUserService(env Env, layout install.Layout) int {
	current, err := readUnit(env.Install, layout)
	var steps []install.Step
	if err == nil {
		steps, err = install.UserServiceInstall(env.Install, layout, env.DataDir.Path(), current, lingerOf(os.Getuid()))
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream service install: %v\n", err)
		return ExitUsage
	}
	if err := applyService(env, steps); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream service install: %v\nsee: journalctl --user -u slipstream\n", err)
		return ExitFailed
	}
	return ExitOK
}

func uninstallUserService(env Env, layout install.Layout) int {
	current, err := readUnit(env.Install, layout)
	var steps []install.Step
	if err == nil {
		steps, err = install.UserServiceUninstall(env.Install, layout, current != nil)
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream service uninstall: %v\n", err)
		return ExitUsage
	}
	if err := applyService(env, steps); err != nil {
		return serviceFailed(err)
	}
	if len(steps) == 0 {
		fmt.Println("no slipstream user service is installed")
		return ExitOK
	}
	fmt.Println("slipstream service removed; linger stays on, and loginctl disable-linger turns it off")
	return ExitOK
}

// lingerOf asks logind whether uid lingers. logind knows no user that has
// neither a session nor linger, so a failed query means off.
func lingerOf(uid int) install.Linger {
	out, err := exec.CommandContext(context.Background(), "loginctl", "show-user", strconv.Itoa(uid), "--property=Linger", "--value").Output() //nolint:gosec // G204: a UID, from this process
	return install.Linger{UID: uid, On: err == nil && strings.TrimSpace(string(out)) == "yes"}
}

func readUnit(kind install.Kind, layout install.Layout) ([]byte, error) {
	p, err := install.UserUnitPath(kind, layout)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p) //nolint:gosec // G304: the user's own unit file
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the user unit: %w", err)
	}
	return b, nil
}

func applyService(env Env, steps []install.Step) error {
	for _, s := range steps {
		switch s := s.(type) {
		case install.WriteFile:
			if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil { //nolint:gosec // G301: systemd's user unit folder is not secret
				return fmt.Errorf("write the user unit: %w", err)
			}
			if err := os.WriteFile(s.Path, []byte(s.Content), 0o644); err != nil { //nolint:gosec // G306: a unit file is not secret
				return fmt.Errorf("write the user unit: %w", err)
			}
		case install.RemoveFile:
			if err := os.Remove(s.Path); err != nil {
				return fmt.Errorf("remove the user unit: %w", err)
			}
		case install.Command:
			cmd := exec.CommandContext(context.Background(), s.Args[0], s.Args[1:]...) //nolint:gosec // G204: the plan names systemctl and loginctl
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("%v: %w", s.Args, err)
			}
		case install.AwaitRunning:
			reply, err := awaitStatus(env)
			if err != nil {
				return err
			}
			fmt.Printf("slipstream service running (PID %d) on %s\n", reply.PID, env.DataDir)
		}
	}
	return nil
}

func awaitStatus(env Env) (StatusReply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), serviceStartWait)
	defer cancel()
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		reply, err := Call(ctx, env.Net, env.DataDir, StatusOp, StatusRequest{})
		if err == nil {
			return reply, nil
		}
		select {
		case <-ctx.Done():
			return StatusReply{}, fmt.Errorf("the service did not answer status within %s: %w", serviceStartWait, err)
		case <-tick.C:
		}
	}
}

func serviceFailed(err error) int {
	_, _ = fmt.Fprintf(os.Stderr, "slipstream service: %v\n", err)
	return ExitFailed
}
