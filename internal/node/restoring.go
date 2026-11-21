package node

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/lifecycle"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/restoreapi"
)

// The Docker restore's variables. ProcessEnv takes them out of the
// environment before the config loader reads it, so no .env file can hold
// them, and no variable holds the passphrase, which docker inspect shows.
const (
	restoreVar               = "SLIPSTREAM_RESTORE"
	restorePassphraseFileVar = "SLIPSTREAM_RESTORE_PASSPHRASE_FILE"
	restoreOldOffVar         = "SLIPSTREAM_RESTORE_OLD_CONTROLLER_OFF"
)

// EnvRestore is a restore the environment asks for: the Backup at File,
// its passphrase in PassphraseFile.
type EnvRestore struct {
	File             string
	PassphraseFile   string
	OldControllerOff bool
}

// envRestore takes the restore variables out of environ.
func envRestore(environ []string) ([]string, *EnvRestore, error) {
	var r EnvRestore
	found := false
	rest := slices.DeleteFunc(slices.Clone(environ), func(kv string) bool {
		name, value, _ := strings.Cut(kv, "=")
		switch name {
		case restoreVar:
			r.File = value
		case restorePassphraseFileVar:
			r.PassphraseFile = value
		case restoreOldOffVar:
			r.OldControllerOff = value == "1"
		default:
			return false
		}
		found = true
		return true
	})
	if !found {
		return rest, nil, nil
	}
	if r.File == "" || r.PassphraseFile == "" {
		return nil, nil, fmt.Errorf("%s and %s are both needed to restore a Backup", restoreVar, restorePassphraseFileVar)
	}
	return rest, &r, nil
}

// restoreFromEnv stages the restore Env.Restore names and writes its
// record, so the swap follows in this same start. It acts only on a data
// dir no Node has set up: no library.db, pairing.json or lifecycle.json.
func (n *Node) restoreFromEnv(ctx context.Context) error {
	r := n.env.Restore
	if r == nil {
		return nil
	}
	for _, name := range []string{n.env.DataDir.LibraryDB(), filepath.Join(n.env.DataDir.Path(), pairing.RecordFile), filepath.Join(n.env.DataDir.Path(), lifecycle.FileName)} {
		if _, err := os.Stat(name); !errors.Is(err, fs.ErrNotExist) {
			n.log.Info("this data dir is already set up, so SlipStream ignores " + restoreVar)
			return nil
		}
	}
	engine, err := backupsEngine(n.env.DataDir, n.cfg, n.disk, n.key.Public(), n.env.Version, n.librarySchemas(), n.rootLog.With("subsystem", "backups"))
	if err != nil {
		return err
	}
	pass, err := readPassphraseFile(r.PassphraseFile)
	if err != nil {
		return &RestoreError{Err: err}
	}
	s := &backups.Stager{Engine: engine, Storage: n.cfg.Timings.Storage, Log: n.rootLog.With("subsystem", "backups")}
	staged, err := s.Stage(ctx, r.File, pass, true, lifecycle.ByRestore)
	if err != nil {
		return &RestoreError{Err: err}
	}
	if err := commitRestore(ctx, s, staged, r.OldControllerOff, nil, "set "+restoreOldOffVar+"=1"); err != nil {
		return &RestoreError{Err: err}
	}
	n.log.Info("prepared the restore of the Backup from " + n.backupDay(staged.Summary))
	return nil
}

// RestoreError is a restore that refused before anything live changed.
type RestoreError struct{ Err error }

func (e *RestoreError) Error() string { return e.Err.Error() }

func (e *RestoreError) Unwrap() error { return e.Err }

func oldControllerOn(controller, how string) error {
	return fmt.Errorf("this Backup is of %s, another machine. Make sure %s stays off, then %s to confirm it", controller, controller, how)
}

func readPassphraseFile(path string) (archive.Passphrase, error) {
	b, err := os.ReadFile(path) //nolint:gosec // the operator named the file
	if err != nil {
		return archive.Passphrase{}, fmt.Errorf("read the passphrase file: %w", err)
	}
	return parsePassphrase(string(b))
}

// parsePassphrase reads a typed passphrase. One that isn't ten words of
// the list is wrong for every Backup.
func parsePassphrase(typed string) (archive.Passphrase, error) {
	pass, err := archive.ParsePassphrase(typed)
	if err != nil {
		return archive.Passphrase{}, &backups.StageError{Code: backups.WrongPassphrase, Err: err, Text: wrongPassphrase}
	}
	return pass, nil
}

const wrongPassphrase = "That passphrase doesn't open this Backup. If you replaced the passphrase after this Backup was taken, use the earlier one."

func (n *Node) backupDay(s backups.Summary) string {
	return s.TakenAt.In(n.zone()).Format("2 Jan, 15:04")
}

// stager is a restore's staging on this running Node. A Controller backs
// up its library through the Backup task first.
func (n *Node) stager() (*backups.Stager, error) {
	if n.pairing.Paired() {
		return nil, errWorkerRestore
	}
	s := &backups.Stager{Engine: n.backups, Storage: n.cfg.Timings.Storage, Log: n.rootLog.With("subsystem", "backups")}
	if lib := n.lib.Load(); lib != nil {
		s.Live, s.Keys = lib.db, lib.keys
		s.Before = func(ctx context.Context) error {
			h := n.half.Load()
			if h == nil || h.backups == nil {
				return errors.New("the Controller isn't running, so it can't back up before the restore")
			}
			run, err := h.backups.Take(ctx, backups.Request{Trigger: archive.BeforeRestore})
			if err != nil {
				return err
			}
			_, err = run.Wait(ctx)
			return err
		}
	}
	return s, nil
}

var errWorkerRestore = errors.New("this Node is a Worker. Restore a Backup on a Node that isn't paired")

// pendingRestore is the one restore the first-run page is staging.
type pendingRestore struct {
	id     string
	staged *backups.Staged
}

// ReceiveRestore keeps an uploaded Backup in the data dir until it is
// opened, replacing any earlier upload.
func (s surfaceNode) ReceiveRestore(_ context.Context, body io.Reader) (string, error) {
	n := s.n
	n.restoreMu.Lock()
	defer n.restoreMu.Unlock()
	if err := n.lifecycleIdle(); err != nil {
		return "", err
	}
	if n.pending != nil && n.pending.staged != nil {
		if err := (&backups.Stager{Engine: n.backups}).Discard(); err != nil {
			return "", err
		}
	}
	n.pending = nil
	path := filepath.Join(n.env.DataDir.Path(), backups.UploadFile)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600) //nolint:gosec // a fixed name in the data dir
	if err != nil {
		return "", fmt.Errorf("keep the upload: %w", err)
	}
	_, err = io.Copy(f, body)
	if err = errors.Join(err, f.Sync(), f.Close()); err != nil {
		return "", errors.Join(err, os.Remove(path))
	}
	var id [8]byte
	_, _ = rand.Read(id[:])
	n.pending = &pendingRestore{id: hex.EncodeToString(id[:])}
	return n.pending.id, nil
}

// UploadRoom is how large an upload the data dir's volume takes.
func (s surfaceNode) UploadRoom(ctx context.Context) (int64, error) {
	v, err := s.n.disk.Volume(ctx, s.n.env.DataDir.Path())
	return v.Free, err
}

// OpenRestore stages the upload id with the typed passphrase.
func (s surfaceNode) OpenRestore(ctx context.Context, id, typed string) (backups.Summary, error) {
	n := s.n
	n.restoreMu.Lock()
	defer n.restoreMu.Unlock()
	if n.pending == nil || n.pending.id != id {
		return backups.Summary{}, restoreapi.ErrNoUpload
	}
	pass, err := parsePassphrase(typed)
	if err != nil {
		return backups.Summary{}, err
	}
	st, err := n.stager()
	if err != nil {
		return backups.Summary{}, err
	}
	staged, err := st.Stage(ctx, filepath.Join(n.env.DataDir.Path(), backups.UploadFile), pass, true, lifecycle.ByRestore)
	if err != nil {
		return backups.Summary{}, err
	}
	n.pending.staged = staged
	return staged.Summary, nil
}

// RestoreController commits the opened upload id and restarts the Node to
// finish it. It refuses as a claim does: a Node already set up, a join
// waiting, or another lifecycle operation.
func (s surfaceNode) RestoreController(ctx context.Context, id string, c backups.Confirm) error {
	n := s.n
	n.claimMu.Lock()
	defer n.claimMu.Unlock()
	n.restoreMu.Lock()
	defer n.restoreMu.Unlock()
	switch {
	case n.pairing.Paired():
		return identity.ErrAlreadyClaimed
	case n.pairing.Awaiting():
		return identity.ErrJoinPending
	case n.pending == nil || n.pending.id != id || n.pending.staged == nil:
		return restoreapi.ErrNoUpload
	}
	if err := n.lifecycleIdle(); err != nil {
		return err
	}
	st, err := n.stager()
	if err != nil {
		return err
	}
	if err := n.pending.staged.Commit(context.WithoutCancel(ctx), st, c); err != nil {
		return err
	}
	n.pending = nil
	n.log.Info("restarting to finish the restore")
	return n.Restart()
}

// removeByName is the ids of the Nodes the Admin named, from those the
// Backup lists. The Controller can't be one of them.
func removeByName(sum backups.Summary, names []string) ([]nodes.ID, error) {
	var ids []nodes.ID
	for _, name := range names {
		i := slices.IndexFunc(sum.Nodes, func(n nodes.Named) bool { return strings.EqualFold(n.Name, name) })
		switch {
		case i < 0:
			return nil, fmt.Errorf("this Backup lists no Node named %s", name)
		case i == 0:
			return nil, fmt.Errorf("%s is the Controller in this Backup, so it can't be removed", name)
		}
		ids = append(ids, sum.Nodes[i].ID)
	}
	return ids, nil
}
