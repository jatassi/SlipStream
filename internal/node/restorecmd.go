package node

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/lifecycle"
	"github.com/jatassi/slipstream/internal/node/internal/control"
	"github.com/jatassi/slipstream/internal/node/internal/lock"
	"github.com/jatassi/slipstream/internal/node/op"
	"github.com/jatassi/slipstream/internal/pairing"
)

// RestoreOp stages a restore on the running Node, which restarts to finish
// it once it has answered.
var RestoreOp = op.Op[RestoreRequest, RestoreReply]{Name: "restore"}

type RestoreRequest struct {
	// File is the Backup's absolute path.
	File             string   `json:"file"`
	Passphrase       string   `json:"passphrase"`
	OldControllerOff bool     `json:"oldControllerOff"`
	Remove           []string `json:"remove"`
}

type RestoreReply struct {
	TakenAt time.Time `json:"takenAt"`
}

// restoreFlags is slipstream restore's command line.
type restoreFlags struct {
	dataDir, passphraseFile, file string
	oldOff                        bool
	remove                        []string
}

func parseRestore(args []string) (restoreFlags, error) {
	var f restoreFlags
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&f.dataDir, "data-dir", "", "")
	fs.StringVar(&f.passphraseFile, "passphrase-file", "", "")
	fs.BoolVar(&f.oldOff, "old-controller-off", false, "")
	fs.Func("remove", "", func(s string) error { f.remove = append(f.remove, s); return nil })
	var files []string
	for {
		if err := fs.Parse(args); err != nil {
			return f, fmt.Errorf("restore: %w", err)
		}
		if fs.NArg() == 0 {
			break
		}
		files = append(files, fs.Arg(0))
		args = fs.Args()[1:]
	}
	if len(files) != 1 {
		return f, errors.New("restore takes one Backup file")
	}
	abs, err := filepath.Abs(files[0])
	if err != nil {
		return f, fmt.Errorf("restore: %w", err)
	}
	f.file = abs
	return f, nil
}

func restoreCommand(args []string, types contenttype.Set) int {
	f, err := parseRestore(args)
	if err != nil {
		return usage(err)
	}
	env, err := ProcessEnv("restore", f.dataDir)
	if err != nil {
		return usage(fmt.Errorf("restore: %w", err))
	}
	env.Types = types
	typed, err := passphraseOf(f.passphraseFile, os.Stdin, os.Stderr)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream restore: %v\n", err)
		return ExitUsage
	}
	ctx := context.Background()
	line, err := RestoreOffline(ctx, env, f, typed)
	if _, held := errors.AsType[*InUseError](err); held {
		var reply RestoreReply
		reply, err = Call(ctx, env.Net, env.DataDir, RestoreOp, RestoreRequest{File: f.file, Passphrase: typed, OldControllerOff: f.oldOff, Remove: f.remove})
		line = "Prepared the restore of the Backup from " + reply.TakenAt.Local().Format("2 Jan, 15:04") + ". SlipStream is restarting to finish it."
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream restore: %v\n", err)
		return ExitCode(err)
	}
	fmt.Println(line)
	return ExitOK
}

// passphraseOf is the passphrase in file, else the line typed at in after
// a prompt on out.
func passphraseOf(file string, in io.Reader, out io.Writer) (string, error) {
	if file != "" {
		b, err := os.ReadFile(file) //nolint:gosec // the Admin named the file
		if err != nil {
			return "", fmt.Errorf("read the passphrase file: %w", err)
		}
		return string(b), nil
	}
	_, _ = fmt.Fprint(out, "Backup passphrase: ")
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && (!errors.Is(err, io.EOF) || line == "") {
		return "", fmt.Errorf("read the passphrase: %w", err)
	}
	return line, nil
}

// RestoreOffline stages a restore while no Node holds env's data dir and
// answers the line to print. In place it backs up the library first. The
// next start finishes it. A held data dir is an *InUseError.
func RestoreOffline(ctx context.Context, env Env, f restoreFlags, typed string) (string, error) {
	lk, err := lock.Acquire(env.DataDir.RunDir())
	if held, ok := errors.AsType[*lock.HeldError](err); ok {
		err = &InUseError{Dir: env.DataDir, PID: held.PID}
	}
	if err != nil {
		return "", err
	}
	line, err := restoreOffline(ctx, env, f, typed)
	return line, errors.Join(err, lk.Release())
}

func restoreOffline(ctx context.Context, env Env, f restoreFlags, typed string) (string, error) {
	cfg, _, err := LoadConfig(env.DataDir, env.Environ, env.Types, env.Install)
	if err != nil {
		return "", err
	}
	_, busy, err := readLifecycle(env.DataDir.lifecycle(cfg.Timings.Backups.WindowsRetry))
	if err != nil {
		return "", err
	}
	if busy {
		return "", lifecycle.ErrBusy
	}
	worker, err := pairing.IsWorker(env.DataDir.Path())
	if err != nil {
		return "", refusedRecord(err)
	}
	if worker {
		return "", errWorkerRestore
	}
	pass, err := parsePassphrase(typed)
	if err != nil {
		return "", err
	}
	disk, err := env.Disk(env.DataDir.Path())
	if err != nil {
		return "", fmt.Errorf("open the disk: %w", err)
	}
	key, err := openKey(env.DataDir)
	if err != nil {
		return "", err
	}
	schemas := append(LibrarySchemas(env.Types), env.Test.Library...)
	log := slog.New(slog.DiscardHandler)
	engine, err := backupsEngine(env.DataDir, cfg, disk, key.Public(), env.Version, schemas, log)
	if err != nil {
		return "", err
	}
	s := &backups.Stager{Engine: engine, Storage: cfg.Timings.Storage, Log: log}
	if _, err := os.Stat(env.DataDir.LibraryDB()); err == nil {
		lib, err := openOffline(ctx, env, cfg)
		if err != nil {
			return "", err
		}
		defer func() { _ = lib.close(ctx) }()
		s.Live, s.Keys = lib.db, lib.keys
		s.Before = func(ctx context.Context) error {
			_, err := engine.TakeOffline(ctx, lib.db, lib.keys, archive.BeforeRestore)
			return err
		}
	}
	staged, err := s.Stage(ctx, f.file, pass, true, lifecycle.ByRestore)
	if err != nil {
		return "", err
	}
	if err := commitRestore(ctx, s, staged, f.oldOff, f.remove, "run slipstream restore again with --old-controller-off"); err != nil {
		return "", err
	}
	return "Prepared the restore of the Backup from " + staged.Summary.TakenAt.Local().Format("2 Jan, 15:04") + ". Start SlipStream to finish it.", nil
}

// commitRestore commits staged with the Admin's flags, or discards it.
func commitRestore(ctx context.Context, s *backups.Stager, staged *backups.Staged, oldOff bool, remove []string, how string) error {
	ids, err := removeByName(staged.Summary, remove)
	if err == nil {
		err = staged.Commit(ctx, s, backups.Confirm{OldControllerOff: oldOff, Remove: ids})
	}
	if errors.Is(err, backups.ErrOldControllerOn) {
		err = oldControllerOn(staged.Summary.Controller, how)
	}
	if err != nil {
		return errors.Join(err, s.Discard())
	}
	return nil
}

// serveRestore is RestoreOp: it stages and commits on this Node, answers,
// then restarts.
func (n *Node) serveRestore(ctx context.Context, body json.RawMessage, send func(any) error) error {
	var req RestoreRequest
	if err := control.Decode(body, &req); err != nil {
		return err
	}
	fail := func(err error) error { return &control.Error{Code: control.CodeFailed, Message: err.Error()} }
	pass, err := parsePassphrase(req.Passphrase)
	if err != nil {
		return fail(err)
	}
	n.claimMu.Lock()
	defer n.claimMu.Unlock()
	if err := n.lifecycleIdle(); err != nil {
		return fail(err)
	}
	s, err := n.stager()
	if err != nil {
		return fail(err)
	}
	staged, err := s.Stage(ctx, req.File, pass, true, lifecycle.ByRestore)
	if err != nil {
		return fail(err)
	}
	if err := commitRestore(ctx, s, staged, req.OldControllerOff, req.Remove, "run slipstream restore again with --old-controller-off"); err != nil {
		return fail(err)
	}
	if err := send(RestoreReply{TakenAt: staged.Summary.TakenAt}); err != nil {
		return err
	}
	n.log.Info("restarting to finish the restore")
	return n.Restart()
}
