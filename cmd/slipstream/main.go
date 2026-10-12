package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/notices"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
	_ "github.com/jatassi/slipstream/web"
)

const usageText = `usage: slipstream <command>
  run    --data-dir <dir>  run a Node
  stop   --data-dir <dir>  stop the Node on <dir> and wait until it releases <dir>
  config --data-dir <dir>  print the resolved configuration and endpoints census as JSON
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
