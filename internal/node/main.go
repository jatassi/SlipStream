package node

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/notices"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
)

const usageText = `usage: slipstream <command>
  run    [--data-dir <dir>]  run a Node
  app    [--data-dir <dir>]  run a Node with the tray, opening the setup page
                             while it is Unclaimed
  stop   [--data-dir <dir>]  stop the Node on <dir> and wait until it releases <dir>
  restart [--data-dir <dir>] restart the Node on <dir> and print its address once
                             it answers again
  status [--data-dir <dir>]  print the running Node's status as JSON
  config [--data-dir <dir>]  print the resolved configuration and endpoints census as JSON
  config set [--data-dir <dir>] <key> <value>
  config unset [--data-dir <dir>] <key>
                             change a Node settings key in config.yaml; a running
                             Node applies a live key at once
  join   [--data-dir <dir>] <code>
                             join this Node to the Controller a Join Code names
  service install|uninstall --user [--data-dir <dir>]
                             run the Node as a systemd user service, or remove it
                             (tarball and AppImage installs)
  version                    print the build stamp
  notices                    print the third-party notices
The data dir is --data-dir, else SLIPSTREAM_DATA_DIR, else the install's default.`

// Main is the slipstream command: args are os.Args, and types the Content
// Types the binary registers. It returns the exit code. Two types sharing an
// id panic before any subcommand runs.
func Main(args []string, types ...contenttype.Registration) int {
	return run(args[1:], contenttype.NewSet(types...))
}

func run(args []string, types contenttype.Set) int {
	if len(args) == 0 {
		return usage(nil)
	}
	name, rest := args[0], args[1:]
	switch name {
	case "version", "notices":
		if len(rest) > 0 {
			return usage(fmt.Errorf("%s takes no arguments", name))
		}
		if name == "version" {
			printVersion()
		} else {
			fmt.Print(notices.Text)
		}
		return ExitOK
	case "run":
		env, err := nodeEnv(name, rest, types)
		if err != nil {
			return usage(err)
		}
		env.Mode = ModeRun
		return ExitCode(Run(context.Background(), env))
	case "stop":
		env, err := nodeEnv(name, rest, types)
		if err != nil {
			return usage(err)
		}
		if err := StopNode(context.Background(), env.Net, env.DataDir); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "slipstream stop: %v\n", err)
			return ExitCode(err)
		}
		return ExitOK
	case "config":
		return configCommand(rest, types)
	case "app":
		return app(rest, types)
	case "status":
		return status(rest, types)
	case "restart":
		return restart(rest, types)
	case "join":
		return join(rest, types)
	case "service":
		return service(rest)
	default:
		return usage(fmt.Errorf("unknown command %q", name))
	}
}

func nodeEnv(name string, args []string, types contenttype.Set) (Env, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dataDir := fs.String("data-dir", "", "")
	if err := fs.Parse(args); err != nil {
		return Env{}, fmt.Errorf("%s: %w", name, err)
	}
	if fs.NArg() > 0 {
		return Env{}, fmt.Errorf("%s: unexpected argument %q", name, fs.Arg(0))
	}
	env, err := ProcessEnv(name, *dataDir)
	if err != nil {
		return Env{}, fmt.Errorf("%s: %w", name, err)
	}
	env.Types = types
	return env, nil
}

// commandArgs parses --data-dir anywhere among args and returns the rest.
func commandArgs(name string, args []string) (dataDir string, positional []string, err error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&dataDir, "data-dir", "", "")
	for {
		if err := fs.Parse(args); err != nil {
			return "", nil, fmt.Errorf("%s: %w", name, err)
		}
		if fs.NArg() == 0 {
			return dataDir, positional, nil
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

func join(args []string, types contenttype.Set) int {
	dataDir, positional, err := commandArgs("join", args)
	if err != nil {
		return usage(err)
	}
	if len(positional) != 1 {
		return usage(errors.New("join takes one Join Code"))
	}
	code, err := joincode.Parse(positional[0])
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream join: %v\n", err)
		return ExitUsage
	}
	env, err := ProcessEnv("join", dataDir)
	if err != nil {
		return usage(fmt.Errorf("join: %w", err))
	}
	env.Types = types
	ctx := context.Background()
	joined, err := Join(ctx, env, code)
	if _, held := errors.AsType[*InUseError](err); held {
		var reply pairing.JoinReply
		if reply, err = Call(ctx, env.Net, env.DataDir, pairing.JoinOp, pairing.JoinRequest{Code: code.String()}); err == nil {
			if reply.Refused == nil {
				fmt.Println(reply.Joined)
				return ExitOK
			}
			err = reply.Refused
		}
	} else if err == nil {
		fmt.Println(joined)
		return ExitOK
	}
	return joinFailed(err)
}

func configCommand(args []string, types contenttype.Set) int {
	if len(args) > 0 && (args[0] == "set" || args[0] == "unset") {
		return configSet(args[0], args[1:], types)
	}
	env, err := nodeEnv("config", args, types)
	if err != nil {
		return usage(err)
	}
	resolved, err := Resolve(env)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream config: %v\n", err)
		return ExitCode(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(resolved); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream config: %v\n", err)
		return ExitFailed
	}
	return ExitOK
}

// configSet is config set and config unset. It prints each key as it
// resolves after the change.
func configSet(verb string, args []string, types contenttype.Set) int {
	name := "config " + verb
	dataDir, positional, err := commandArgs(name, args)
	if err != nil {
		return usage(err)
	}
	var change ConfigChange
	switch {
	case verb == "set" && len(positional) == 2:
		change = ConfigChange{Key: positional[0], Value: &positional[1]}
	case verb == "unset" && len(positional) == 1:
		change = ConfigChange{Key: positional[0]}
	case verb == "set":
		return usage(errors.New("config set takes a key and a value"))
	default:
		return usage(errors.New("config unset takes a key"))
	}
	env, err := ProcessEnv(name, dataDir)
	if err != nil {
		return usage(fmt.Errorf("%s: %w", name, err))
	}
	env.Types = types
	ctx, cancel := context.WithTimeout(context.Background(), serviceStartWait)
	defer cancel()
	keys, online, err := SetConfig(ctx, env, []ConfigChange{change})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream %s: %v\n", name, err)
		if _, refused := errors.AsType[*nodesystem.Refusal](err); refused {
			return ExitUsage
		}
		return ExitCode(err)
	}
	fmt.Println(changedLine(change.Key, keys, online))
	return ExitOK
}

// changedLine is key's resolved value, and whether a running Node waits
// for a restart to use it.
func changedLine(key string, keys []Key, online bool) string {
	line := key + ": " + keysByName(keys)[key].Value
	if r, _ := settingRowOf(key); online && !r.live() {
		line += ", applies after restart"
	}
	return line
}

func restart(args []string, types contenttype.Set) int {
	env, err := nodeEnv("restart", args, types)
	if err != nil {
		return usage(err)
	}
	reply, err := RestartNode(context.Background(), env)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream restart: %v\n", err)
		return ExitCode(err)
	}
	fmt.Println(reply.URL)
	return ExitOK
}

func status(args []string, types contenttype.Set) int {
	env, err := nodeEnv("status", args, types)
	if err != nil {
		return usage(err)
	}
	reply, err := Call(context.Background(), env.Net, env.DataDir, StatusOp, StatusRequest{})
	if err == nil {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		err = enc.Encode(reply)
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream status: %v\n", err)
		return ExitFailed
	}
	return ExitOK
}

func joinFailed(err error) int {
	refused, ok := errors.AsType[*pairing.Error](err)
	switch {
	case ok && refused.Kind == pairing.AlreadyJoined:
		fmt.Println("join code ignored: " + refused.Reason)
		return ExitOK
	case ok && refused.Kind.Ignored():
		_, _ = fmt.Fprintln(os.Stderr, "join code ignored: "+refused.Reason)
	case ok && refused.Kind == pairing.Unreachable:
		_, _ = fmt.Fprintln(os.Stderr, "join failed: "+refused.Reason)
	case ok:
		_, _ = fmt.Fprintln(os.Stderr, "join refused: "+refused.Reason)
	default:
		_, _ = fmt.Fprintf(os.Stderr, "slipstream join: %v\n", err)
		return ExitCode(err)
	}
	return ExitFailed
}

func usage(err error) int {
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream: %v\n", err)
	}
	_, _ = fmt.Fprintln(os.Stderr, usageText)
	return ExitUsage
}

func printVersion() {
	s, ok := buildinfo.Current()
	if !ok {
		fmt.Println("slipstream dev (unstamped build)")
		return
	}
	fmt.Printf("slipstream %s\n", s.Version)
	fmt.Printf("commit         %s\n", s.Commit)
	fmt.Printf("target         %s\n", s.Target)
	fmt.Printf("libtorrent     %s\n", libtorrent.Version())
	fmt.Printf("fingerprint    %s\n", s.Fingerprint)
	fmt.Printf("native bundle  %s\n", s.NativeBundle)
}
