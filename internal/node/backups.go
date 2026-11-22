package node

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/node/internal/control"
	"github.com/jatassi/slipstream/internal/node/op"
	"github.com/jatassi/slipstream/internal/node/surface"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// backupsEngine is the Backups engine of the Node on dir. The hook and the
// Backups service share it, so backups.json and backup.key each have one
// writer. It opens before library.db.
func backupsEngine(dir DataDir, cfg Config, disk storing.Disk, key link.Key, version string, schemas []storage.Schema[storage.Library], log *slog.Logger) (*backups.Engine, error) {
	index, err := backups.OpenIndex(filepath.Join(dir.Path(), backups.IndexFile), cfg.Timings.Backups.WindowsRetry)
	if err != nil {
		return nil, err
	}
	return backups.NewEngine(backups.EngineConfig{
		Dir:           dir.Path(),
		Folder:        dir.Resolve(cfg.Backups.Folder),
		Index:         index,
		Schemas:       schemas,
		ControllerKey: key[:],
		Version:       version,
		Timings:       cfg.Timings.Backups,
		Free: func(ctx context.Context, path string) (int64, error) {
			v, err := disk.Volume(ctx, path)
			return v.Free, err
		},
		Clock: time.Now,
		Log:   log,
	}), nil
}

// backingUp is the hook's phase, which status answers before ready.
func (n *Node) backingUp(on bool) {
	if on {
		n.setPhase(PhaseBackingUp)
		return
	}
	n.setPhase(PhaseStarting)
}

// whileReconciling is whether a Backup is taken while Reconciling runs.
func whileReconciling(ctx context.Context, r storage.Reader[storage.Library]) (bool, error) {
	p, err := reconciling.State(ctx, r)
	return p == reconciling.Reconciling, err
}

// openOffline opens library.db as a start does, for a host command that
// holds the data dir lock, with the hook's Backup before any migration.
func openOffline(ctx context.Context, env Env, cfg Config) (*libraryDB, error) {
	disk, err := env.Disk(env.DataDir.Path())
	if err != nil {
		return nil, fmt.Errorf("open the disk: %w", err)
	}
	key, err := openKey(env.DataDir)
	if err != nil {
		return nil, err
	}
	schemas := append(LibrarySchemas(env.Types), env.Test.Library...)
	log := slog.New(slog.DiscardHandler)
	engine, err := backupsEngine(env.DataDir, cfg, disk, key.Public(), env.Version, schemas, log)
	if err != nil {
		return nil, err
	}
	return openLibraryFiles(ctx, libraryFiles{
		dir:     env.DataDir,
		schemas: schemas,
		hook:    backups.BeforeMigrating(engine, func(bool) {}),
		timings: cfg.Timings,
		hub:     live.NewHub(surface.Events),
		log:     log,
		test:    env.Test,
	})
}

// BackupOp takes a manual Backup on the running Controller and answers
// once it is checked.
var BackupOp = op.Op[BackupRequest, BackupReply]{Name: "backup"}

type BackupRequest struct{}

type BackupReply struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func (n *Node) serveBackup(ctx context.Context, body json.RawMessage, send func(any) error) error {
	var req BackupRequest
	if err := control.Decode(body, &req); err != nil {
		return err
	}
	h := n.half.Load()
	if h == nil || h.backups == nil {
		return &control.Error{Code: control.CodeFailed, Message: "this Node isn't the Controller, so it keeps no Backups"}
	}
	run, err := h.backups.Take(ctx, backups.Request{Trigger: archive.Manual})
	if err != nil {
		return &control.Error{Code: control.CodeFailed, Message: err.Error()}
	}
	taken, err := run.Wait(ctx)
	if err != nil {
		return &control.Error{Code: control.CodeFailed, Message: err.Error()}
	}
	return send(BackupReply{Path: h.backups.Path(taken), Size: taken.Size})
}

func backupCommand(args []string, types contenttype.Set) int {
	env, err := nodeEnv("backup", args, types)
	if err != nil {
		return usage(err)
	}
	reply, err := Call(context.Background(), env.Net, env.DataDir, BackupOp, BackupRequest{})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "slipstream backup: %v\n", err)
		return ExitFailed
	}
	fmt.Printf("Took a %s Backup and checked it: %s\n", backups.Size(reply.Size), reply.Path)
	return ExitOK
}
