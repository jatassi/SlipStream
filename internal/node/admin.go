package node

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/node/internal/control"
	"github.com/jatassi/slipstream/internal/node/internal/lock"
	"github.com/jatassi/slipstream/internal/node/surface"
	"github.com/jatassi/slipstream/internal/system"
)

// admin is `slipstream admin reset <email> [--address <origin>]`: it
// prints a Reset link and nothing else on stdout. It issues the link
// offline when no Node holds the data dir, and through the running Node
// otherwise.
func admin(args []string, types contenttype.Set) int {
	if len(args) == 0 || args[0] != "reset" {
		return usage(errors.New("admin takes the reset command: admin reset <email>"))
	}
	fs := flag.NewFlagSet("admin reset", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dataDir := fs.String("data-dir", "", "")
	address := fs.String("address", "", "")
	var positional []string
	args = args[1:]
	for {
		if err := fs.Parse(args); err != nil {
			return usage(fmt.Errorf("admin reset: %w", err))
		}
		if fs.NArg() == 0 {
			break
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
	if len(positional) != 1 {
		return usage(errors.New("admin reset takes one email"))
	}
	env, err := ProcessEnv("admin reset", *dataDir)
	if err != nil {
		return usage(fmt.Errorf("admin reset: %w", err))
	}
	env.Types = types
	ctx := context.Background()
	req := identity.ResetRequest{Email: positional[0], Address: *address}
	url, err := AdminReset(ctx, env, req)
	if _, held := errors.AsType[*InUseError](err); held {
		var reply identity.ResetReply
		if reply, err = Call(ctx, env.Net, env.DataDir, identity.ResetOp, req); err == nil {
			url = reply.URL
			if reply.Refused != nil {
				err = reply.Refused
			}
		}
	}
	if err == nil {
		fmt.Println(url)
		return ExitOK
	}
	_, _ = fmt.Fprintf(os.Stderr, "slipstream admin reset: %v\n", err)
	if _, refused := errors.AsType[*identity.HostRefusal](err); refused {
		return ExitUsage
	}
	return ExitCode(err)
}

// AdminReset issues a Reset link while no Node holds env's data dir: it
// takes the lock, refuses a data dir that isn't a Controller's, opens
// library.db as a start does over a hub nobody subscribes to, and releases
// the lock. A held data dir is an *InUseError.
func AdminReset(ctx context.Context, env Env, req identity.ResetRequest) (string, error) {
	lk, err := lock.Acquire(env.DataDir.RunDir())
	if held, ok := errors.AsType[*lock.HeldError](err); ok {
		err = &InUseError{Dir: env.DataDir, PID: held.PID}
	}
	if err != nil {
		return "", err
	}
	url, err := resetOffline(ctx, env, req)
	return url, errors.Join(err, lk.Release())
}

func resetOffline(ctx context.Context, env Env, req identity.ResetRequest) (string, error) {
	if claim, _ := (offline{env.DataDir}).Claim(ctx); claim != system.Controller {
		return "", identity.NotAController()
	}
	cfg, _, err := LoadConfig(env.DataDir, env.Environ, env.Types)
	if err != nil {
		return "", err
	}
	lib, err := openLibraryFiles(ctx, libraryFiles{
		dir:     env.DataDir,
		schemas: append(LibrarySchemas(env.Types), env.Test.Library...),
		timings: cfg.Timings,
		hub:     live.NewHub(surface.Events),
		log:     slog.New(slog.DiscardHandler),
	})
	if err != nil {
		return "", err
	}
	url, err := lib.ids.ResetFromHost(ctx, req)
	return url, errors.Join(err, lib.close(ctx))
}

// serveAdminReset is ResetOp on a running Node: a Controller answers the
// link, and any other Node the same refusal as offline.
func (n *Node) serveAdminReset(ctx context.Context, body json.RawMessage, send func(any) error) error {
	var req identity.ResetRequest
	if err := control.Decode(body, &req); err != nil {
		return err
	}
	claim, err := n.claim(ctx)
	if err != nil {
		return err
	}
	if claim != system.Controller {
		return send(identity.ResetReply{Refused: identity.NotAController()})
	}
	url, err := n.lib.Load().ids.ResetFromHost(ctx, req)
	if refused, ok := errors.AsType[*identity.HostRefusal](err); ok {
		return send(identity.ResetReply{Refused: refused})
	}
	if err != nil {
		return &control.Error{Code: control.CodeFailed, Message: err.Error()}
	}
	return send(identity.ResetReply{URL: url})
}
