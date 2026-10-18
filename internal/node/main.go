package node

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

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
  join   [--data-dir <dir>] [--node-port <port>] <code>
                             join this Node to the Controller a Join Code names;
                             --node-port sets the port link.listen binds. When the
                             Controller can't be reached, print a Node address and
                             wait for the Controller to dial it
  node-address [--data-dir <dir>]
                             print the Node address a Controller dials this Node at
  admin reset [--data-dir <dir>] [--address <origin>] <email>
                             print a Reset link for the Account with <email>,
                             ending its sessions; --address sets the address
                             links use when none is set
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
	case "node-address":
		return nodeAddress(rest, types)
	case "admin":
		return admin(rest, types)
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

// joinFlags are slipstream join's arguments.
type joinFlags struct {
	dataDir string
	// port is --node-port, 0 when absent.
	port uint16
	code string
}

func parseJoin(args []string) (joinFlags, error) {
	var out joinFlags
	fs := flag.NewFlagSet("join", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&out.dataDir, "data-dir", "", "")
	fs.Func("node-port", "", func(s string) error {
		port, err := strconv.ParseUint(s, 10, 16)
		if err != nil || port == 0 {
			return fmt.Errorf("%s is not a port", s)
		}
		out.port = uint16(port)
		return nil
	})
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return joinFlags{}, fmt.Errorf("join: %w", err)
		}
		if fs.NArg() == 0 {
			break
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
	if len(positional) != 1 {
		return joinFlags{}, errors.New("join takes one Join Code")
	}
	out.code = positional[0]
	return out, nil
}

func join(args []string, types contenttype.Set) int {
	flags, err := parseJoin(args)
	if err != nil {
		return usage(err)
	}
	code, err := joincode.Parse(flags.code)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream join: %v\n", err)
		return ExitUsage
	}
	env, err := ProcessEnv("join", flags.dataDir)
	if err != nil {
		return usage(fmt.Errorf("join: %w", err))
	}
	env.Types = types
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	port := flags.port
	printed := false
	printAwaiting := func(lines []string) {
		if !printed {
			fmt.Println(strings.Join(lines, "\n"))
			printed = true
		}
	}
	joined, err := Join(ctx, env, code, port, func(p pairing.Pending) { printAwaiting(p.Lines()) })
	if _, held := errors.AsType[*InUseError](err); held {
		var line string
		err = Stream(ctx, env.Net, env.DataDir, pairing.JoinOp, pairing.JoinRequest{Code: code.String(), NodePort: port}, func(reply pairing.JoinReply) bool {
			switch {
			case reply.Awaiting != nil:
				printAwaiting(reply.Awaiting)
				return false
			case reply.Refused != nil:
				err = reply.Refused
			default:
				line = reply.Joined
			}
			return true
		})
		if err == nil {
			fmt.Println(line)
			return ExitOK
		}
	} else if err == nil {
		fmt.Println(joined)
		return ExitOK
	}
	return joinFailed(err)
}

// nodeAddress is slipstream node-address: the SLN1 text of this Node's key
// and the addresses its link advertises, read from config and node.key.
func nodeAddress(args []string, types contenttype.Set) int {
	env, err := nodeEnv("node-address", args, types)
	if err != nil {
		return usage(err)
	}
	addr, err := NodeAddress(env)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream node-address: %v\n", err)
		return ExitCode(err)
	}
	fmt.Println(addr)
	return ExitOK
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
