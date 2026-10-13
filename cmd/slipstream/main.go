package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/notices"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
)

const usageText = `usage: slipstream <command>
  run    --data-dir <dir>  run a Node
  stop   --data-dir <dir>  stop the Node on <dir> and wait until it releases <dir>
  config --data-dir <dir>  print the resolved configuration and endpoints census as JSON
  join   --data-dir <dir> <code>
                           join this Node to the Controller a Join Code names
  version                  print the build stamp
  notices                  print the third-party notices`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
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
		return node.ExitOK
	case "run":
		env, err := nodeEnv(name, rest)
		if err != nil {
			return usage(err)
		}
		return node.ExitCode(node.Run(context.Background(), env))
	case "stop":
		env, err := nodeEnv(name, rest)
		if err != nil {
			return usage(err)
		}
		if err := node.StopNode(context.Background(), env.DataDir); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "slipstream stop: %v\n", err)
			return node.ExitCode(err)
		}
		return node.ExitOK
	case "config":
		env, err := nodeEnv(name, rest)
		if err != nil {
			return usage(err)
		}
		resolved, err := node.Resolve(env.DataDir, env.Environ)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "slipstream config: %v\n", err)
			return node.ExitCode(err)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(resolved); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "slipstream config: %v\n", err)
			return node.ExitFailed
		}
		return node.ExitOK
	case "join":
		return join(rest)
	default:
		return usage(fmt.Errorf("unknown command %q", name))
	}
}

func nodeEnv(name string, args []string) (node.Env, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dataDir := fs.String("data-dir", "", "")
	if err := fs.Parse(args); err != nil {
		return node.Env{}, fmt.Errorf("%s: %w", name, err)
	}
	if fs.NArg() > 0 {
		return node.Env{}, fmt.Errorf("%s: unexpected argument %q", name, fs.Arg(0))
	}
	env, err := node.ProcessEnv(*dataDir)
	if err != nil {
		return node.Env{}, fmt.Errorf("%s: %w", name, err)
	}
	return env, nil
}

// join pairs the Node on the data dir: through the join control op when a
// Node holds the data dir, else offline.
func join(args []string) int {
	fs := flag.NewFlagSet("join", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dataDir := fs.String("data-dir", "", "")
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return usage(fmt.Errorf("join: %w", err))
		}
		if fs.NArg() == 0 {
			break
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
	if len(positional) != 1 {
		return usage(errors.New("join takes one Join Code"))
	}
	code, err := joincode.Parse(positional[0])
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream join: %v\n", err)
		return node.ExitUsage
	}
	env, err := node.ProcessEnv(*dataDir)
	if err != nil {
		return usage(fmt.Errorf("join: %w", err))
	}
	ctx := context.Background()
	joined, err := node.Join(ctx, env, code)
	if _, held := errors.AsType[*node.InUseError](err); held {
		var reply pairing.JoinReply
		if reply, err = node.Call(ctx, env.DataDir, pairing.JoinOp, pairing.JoinRequest{Code: code.String()}); err == nil {
			if reply.Refused == nil {
				fmt.Println(reply.Joined)
				return node.ExitOK
			}
			err = reply.Refused
		}
	} else if err == nil {
		fmt.Println(joined)
		return node.ExitOK
	}
	return joinFailed(err)
}

func joinFailed(err error) int {
	refused, ok := errors.AsType[*pairing.Error](err)
	switch {
	case ok && refused.Kind == pairing.AlreadyJoined:
		fmt.Println("join code ignored: " + refused.Reason)
		return node.ExitOK
	case ok && refused.Kind.Ignored():
		_, _ = fmt.Fprintln(os.Stderr, "join code ignored: "+refused.Reason)
	case ok && refused.Kind == pairing.Unreachable:
		_, _ = fmt.Fprintln(os.Stderr, "join failed: "+refused.Reason)
	case ok:
		_, _ = fmt.Fprintln(os.Stderr, "join refused: "+refused.Reason)
	default:
		_, _ = fmt.Fprintf(os.Stderr, "slipstream join: %v\n", err)
		return node.ExitCode(err)
	}
	return node.ExitFailed
}

func usage(err error) int {
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream: %v\n", err)
	}
	_, _ = fmt.Fprintln(os.Stderr, usageText)
	return node.ExitUsage
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
