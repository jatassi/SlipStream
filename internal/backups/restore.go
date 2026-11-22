package backups

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jatassi/slipstream/internal/atomicfile"
	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/lifecycle"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

// UploadFile is where an uploaded Backup waits in the data dir until it is
// staged and, once the restore is confirmed, moved into the Backups folder.
const UploadFile = "restore.part"

// Stager stages one restore. Every entry point makes one: the first-run
// page, `slipstream restore` offline and online, and the Docker variables.
// It writes nothing live until Commit, and Commit writes only the
// before-restore Backup, the Backups index and lifecycle.json.
type Stager struct {
	Engine *Engine
	// Live is the library.db the restore replaces and Keys its library.key,
	// both nil when the data dir holds none.
	Live *storage.DB[storage.Library]
	Keys *secret.Keys
	// Before takes the before-restore Backup of Live and waits for it.
	Before  func(ctx context.Context) error
	Storage storage.Timings
	Log     *slog.Logger
}

// Summary is what the Admin confirms.
type Summary struct {
	TakenAt time.Time
	Version string
	// Controller is the staged Controller row's name, and Nodes every Node
	// the Backup lists, the Controller first.
	Controller string
	Nodes      []nodes.Named
	NewMachine bool
}

// Staged is a Backup decoded to the .restore files in the data dir.
type Staged struct {
	Summary Summary
	opened  archive.Opened
	archive string
	keys    *secret.Keys
	fenced  bool
	cause   lifecycle.Cause
	typed   bool
	upload  bool
}

// StageCode names why a Backup can't be staged, as the API's problem code.
type StageCode string

const (
	WrongPassphrase StageCode = "backup_wrong_passphrase"
	Damaged         StageCode = "backup_damaged"
	NotABackup      StageCode = "backup_not_a_backup"
	TooNew          StageCode = "backup_too_new"
	NoSpace         StageCode = "backup_no_space"
)

// StageError is a Backup that can't be restored, with the sentence the
// Admin reads.
type StageError struct {
	Code StageCode
	Text string
	Err  error
}

func (e *StageError) Error() string { return e.Text }

func (e *StageError) Unwrap() error { return e.Err }

func (s *Stager) refuse(err error, m archive.Manifest) error {
	if space, ok := errors.AsType[*SpaceError](err); ok {
		return &StageError{Code: NoSpace, Err: err, Text: fmt.Sprintf(
			"Restoring needs %s free in %s, and its volume has %s free. Free some space, then try again.",
			Size(space.Need), space.Folder, Size(space.Free))}
	}
	if newer, ok := errors.AsType[*archive.TooNewError](err); ok {
		m.Version = newer.Version
		return s.tooNew(err, m)
	}
	if _, ok := errors.AsType[*storage.RefusedError](err); ok {
		return s.tooNew(err, m)
	}
	switch {
	case errors.Is(err, archive.ErrWrongPassphrase):
		return &StageError{Code: WrongPassphrase, Err: err, Text: "That passphrase doesn't open this Backup. If you replaced the passphrase after this Backup was taken, use the earlier one."}
	case errors.Is(err, archive.ErrNotBackup):
		return &StageError{Code: NotABackup, Err: err, Text: "This file isn't a SlipStream Backup. Pick a file whose name ends in .tar.zst.age."}
	}
	if _, ok := errors.AsType[*archive.DamagedError](err); ok || errors.Is(err, storage.ErrDamaged) || errors.Is(err, errForeignKey) {
		return &StageError{Code: Damaged, Err: err, Text: "This Backup is damaged or incomplete. Copy it again, or pick another Backup."}
	}
	return err
}

func (s *Stager) tooNew(err error, m archive.Manifest) error {
	this := s.Engine.cfg.Version
	return &StageError{Code: TooNew, Err: err, Text: fmt.Sprintf(
		"This Backup is from SlipStream %s, which is newer than this version, %s. Install %s or later, then restore it.", m.Version, this, m.Version)}
}

var errForeignKey = errors.New("the Backup's library.key doesn't open its library")

// Stage reads the Backup at path with pass into the .restore files, after
// checking that the data dir's volume holds the Backup's library and the
// before-restore Backup. It refuses a library from a newer version, and a
// library.key that doesn't open its library. Except for a rollback, it
// migrates the staged library to read the Summary. upload is whether path
// is UploadFile, which Commit moves into the Backups folder.
func (s *Stager) Stage(ctx context.Context, path string, pass archive.Passphrase, typed bool, cause lifecycle.Cause) (*Staged, error) {
	st, err := s.stage(ctx, path, pass, cause)
	if err != nil {
		return nil, errors.Join(err, s.Discard())
	}
	st.typed, st.upload = typed, filepath.Base(path) == UploadFile && filepath.Dir(path) == s.Engine.cfg.Dir
	return st, nil
}

func (s *Stager) stage(ctx context.Context, path string, pass archive.Passphrase, cause lifecycle.Cause) (*Staged, error) {
	dir := s.Engine.cfg.Dir
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("read the Backup: %w", err)
	}
	var before int64
	if s.Live != nil {
		if before, err = s.Live.Estimate(ctx); err != nil {
			return nil, err
		}
	}
	free, err := s.Engine.cfg.Free(ctx, dir)
	if err != nil {
		return nil, fmt.Errorf("read the free space of %s: %w", dir, err)
	}
	if need := info.Size() + before; free < need {
		return nil, s.refuse(&SpaceError{Folder: dir, Need: need, Free: free}, archive.Manifest{})
	}
	f, err := os.Open(path) //nolint:gosec // the Admin named the Backup to restore
	if err != nil {
		return nil, fmt.Errorf("read the Backup: %w", err)
	}
	defer func() { _ = f.Close() }()
	limits := s.Engine.cfg.Timings.limits(free - before)
	opened, err := archive.Read(f, pass, limits, stagedFiles{dir: dir, retry: s.Engine.cfg.Timings.WindowsRetry})
	if over, ok := errors.AsType[*archive.OversizeError](err); ok && over.Entry == archive.LibraryEntry {
		err = &SpaceError{Folder: dir, Need: over.Size + before, Free: free}
	}
	if err != nil {
		return nil, s.refuse(err, archive.Manifest{})
	}
	m := opened.Manifest()
	st := &Staged{opened: opened, archive: path, cause: cause, Summary: Summary{TakenAt: m.TakenAt, Version: m.Version, NewMachine: !bytes.Equal(m.ControllerKey, s.Engine.cfg.ControllerKey)}}
	library := restored(dir, storage.LibraryFile)
	inspected, err := storage.Inspect(ctx, library, s.Engine.cfg.Schemas)
	if err != nil {
		return nil, s.refuse(err, m)
	}
	keys, err := secret.Verify(restored(dir, secret.KeyFile), inspected.Check, inspected.Check != nil)
	if err != nil {
		return nil, s.refuse(fmt.Errorf("%w: %w", errForeignKey, err), m)
	}
	st.keys = keys
	if cause != lifecycle.ByRestore {
		return st, nil
	}
	candidate, err := storage.OpenCandidate(ctx, library, s.Engine.cfg.Schemas, s.Storage, s.Log)
	if err != nil {
		return nil, s.refuse(err, m)
	}
	st.fenced = candidate.Fenced
	err = candidate.DB.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		st.Summary.Nodes, err = nodes.Names(ctx, r)
		return err
	})
	if err := errors.Join(err, candidate.DB.Close()); err != nil {
		return nil, err
	}
	if len(st.Summary.Nodes) > 0 {
		st.Summary.Controller = st.Summary.Nodes[0].Name
	}
	return st, nil
}

// Confirm is the Admin's answer to the Summary.
type Confirm struct {
	// OldControllerOff is required on a new machine: two Controllers
	// running one library seed the same private torrents.
	OldControllerOff bool
	// Remove is the Nodes the Admin unticked.
	Remove []nodes.ID
}

// ErrOldControllerOn is a new machine's restore confirmed without the old
// Controller off.
var ErrOldControllerOn = errors.New("confirm that the old Controller stays off")

// Commit is the last step before the restart. In place it takes the
// before-restore Backup, merges the Backup's passphrase generations into
// this backup.key and keeps this Backups index. On a new machine it mints
// the proof that this Node replaced the Backup's Controller from the
// Backup's fence secret, then gives the staged backup.key a fresh one, and
// keeps the Backup's index with no local files. Either way a fenced
// library gets a fresh fence secret too. An upload becomes a restored
// Backup in the Backups folder. Then it writes lifecycle.json, which the
// next start finishes.
func (st *Staged) Commit(ctx context.Context, s *Stager, c Confirm) error {
	if st.Summary.NewMachine && !c.OldControllerOff {
		return ErrOldControllerOn
	}
	dir := s.Engine.cfg.Dir
	m := st.opened.Manifest()
	op := &lifecycle.Restore{
		Stage: lifecycle.Staged, Cause: st.cause, NewMachine: st.Summary.NewMachine, Typed: st.typed,
		Backup: lifecycle.Backup{Name: archive.NameOf(m).String(), TakenAt: m.TakenAt, Version: m.Version, ControllerKey: m.ControllerKey},
	}
	for _, id := range c.Remove {
		op.Remove = append(op.Remove, id.String())
	}
	var id [16]byte
	if _, err := io.ReadFull(rand.Reader, id[:]); err != nil {
		return fmt.Errorf("draw a restore id: %w", err)
	}
	op.ID = hex.EncodeToString(id[:])
	stagedKey, err := os.ReadFile(restored(dir, VaultFile))
	if err != nil {
		return fmt.Errorf("read the staged %s: %w", VaultFile, err)
	}
	var vault []byte
	if st.Summary.NewMachine {
		archived, err := decode(stagedKey, st.keys.Sealer())
		if err != nil {
			return err
		}
		op.Superseded = []lifecycle.Proof{{OldKey: m.ControllerKey, Proof: Prove(archived.fence.derive(), m.ControllerKey, s.Engine.cfg.ControllerKey)}}
		if vault, err = VaultForNewMachine(stagedKey, st.keys, s.Engine.cfg.Rand); err != nil {
			return err
		}
		if err := st.newMachineIndex(s); err != nil {
			return err
		}
	} else {
		if vault, err = st.inPlace(ctx, s, op, stagedKey); err != nil {
			return err
		}
	}
	if err := atomicfile.Replace(restored(dir, VaultFile), vault, s.Engine.cfg.Timings.WindowsRetry); err != nil {
		return fmt.Errorf("write the staged %s: %w", VaultFile, err)
	}
	if st.upload {
		if err := st.adopt(s); err != nil {
			return err
		}
	}
	return lifecycle.Dir{Path: dir, Retry: s.Engine.cfg.Timings.WindowsRetry}.Begin(op)
}

// inPlace takes the before-restore Backup into the kept index and answers
// the staged backup.key: this one's passphrases and fence secret, plus each
// of the Backup's generations it lacks. A fenced library's fence secret is
// replaced, since the Workers paired with the Controller that fenced it
// still carry proofs naming this key.
func (st *Staged) inPlace(ctx context.Context, s *Stager, op *lifecycle.Restore, stagedKey []byte) ([]byte, error) {
	dir := s.Engine.cfg.Dir
	vault := stagedKey
	if s.Live != nil {
		if err := s.Before(ctx); err != nil {
			return nil, err
		}
		err := s.Live.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
			keys, err := nodes.RemovedSince(ctx, r, op.Backup.TakenAt)
			for _, k := range keys {
				op.RemovedKeys = append(op.RemovedKeys, k[:])
			}
			return err
		})
		if err != nil {
			return nil, err
		}
		current, err := s.Engine.Vault(s.Keys)
		if err != nil {
			return nil, err
		}
		if vault, err = MergeVault(current, stagedKey, st.keys); err != nil {
			return nil, err
		}
	}
	if err := atomicfile.Remove(restored(dir, IndexFile), s.Engine.cfg.Timings.WindowsRetry); err != nil {
		return nil, fmt.Errorf("drop the staged %s: %w", IndexFile, err)
	}
	if !st.fenced {
		return vault, nil
	}
	return VaultForNewMachine(vault, st.keys, s.Engine.cfg.Rand)
}

// newMachineIndex writes the Backup's own index as the staged one, with
// no Backup listed: none of its files is on this machine.
func (st *Staged) newMachineIndex(s *Stager) error {
	path := restored(s.Engine.cfg.Dir, IndexFile)
	b, err := os.ReadFile(path) //nolint:gosec // the data dir's staged backups.json
	if err != nil {
		return fmt.Errorf("read the staged %s: %w", IndexFile, err)
	}
	l, err := decodeIndex(b)
	if err != nil {
		return fmt.Errorf("read the staged %s: %w", IndexFile, err)
	}
	l.Backups = nil
	if b, err = encodeIndex(l); err != nil {
		return err
	}
	return atomicfile.Replace(path, b, s.Engine.cfg.Timings.WindowsRetry)
}

// adopt moves an uploaded Backup into the Backups folder and lists it in
// the index the restored library starts with: the live one in place, the
// staged one on a new machine.
func (st *Staged) adopt(s *Stager) error {
	m := st.opened.Manifest()
	m.Trigger = archive.Restored
	entry := Entry{
		ID: m.BackupID, Name: archive.NameOf(m), TakenAt: m.TakenAt, Trigger: archive.Restored, Version: m.Version, Schemas: m.Schemas,
		Size: st.opened.Sums().Size, Sum: st.opened.Sums().Ciphertext, Generation: Generation(m.Generation),
		State: Verified, VerifiedAt: s.Engine.cfg.Clock().UTC(), Local: true,
	}
	list := func(l *Ledger) error {
		l.set(entry)
		return nil
	}
	retry := s.Engine.cfg.Timings.WindowsRetry
	live := s.Engine.cfg.Index.Read()
	if listed, ok := live.Entry(entry.ID); ok && listed.Local && !st.Summary.NewMachine {
		return atomicfile.Remove(st.archive, retry)
	}
	if err := os.MkdirAll(s.Engine.cfg.Folder, 0o700); err != nil {
		return fmt.Errorf("make the Backups folder: %w", err)
	}
	if err := atomicfile.Rename(st.archive, s.Engine.path(entry.Name), retry); err != nil {
		return fmt.Errorf("keep the uploaded Backup: %w", err)
	}
	if err := atomicfile.SyncDir(s.Engine.cfg.Folder); err != nil {
		return fmt.Errorf("sync the Backups folder: %w", err)
	}
	if !st.Summary.NewMachine {
		return s.Engine.cfg.Index.Update(list)
	}
	path := restored(s.Engine.cfg.Dir, IndexFile)
	b, err := os.ReadFile(path) //nolint:gosec // the data dir's staged backups.json
	if err != nil {
		return fmt.Errorf("read the staged %s: %w", IndexFile, err)
	}
	l, err := decodeIndex(b)
	if err != nil {
		return err
	}
	if err := list(&l); err != nil {
		return err
	}
	if b, err = encodeIndex(l); err != nil {
		return err
	}
	return atomicfile.Replace(path, b, retry)
}

// Discard deletes the .restore files a staging left.
func (s *Stager) Discard() error {
	return stagedFiles{dir: s.Engine.cfg.Dir, retry: s.Engine.cfg.Timings.WindowsRetry}.Discard()
}

func restored(dir, name string) string { return filepath.Join(dir, name+lifecycle.Restored) }

// stagedFiles is the Sink that writes each entry to its .restore file in
// the data dir, synced before Read reaches the next.
type stagedFiles struct {
	dir   string
	retry time.Duration
}

var stagedNames = map[archive.Entry]string{
	archive.LibraryEntry:    storage.LibraryFile,
	archive.LibraryKeyEntry: secret.KeyFile,
	archive.BackupKeyEntry:  VaultFile,
	archive.IndexEntry:      IndexFile,
}

func (s stagedFiles) Create(e archive.Entry) (io.WriteCloser, error) {
	f, err := os.OpenFile(restored(s.dir, stagedNames[e]), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("stage %s: %w", stagedNames[e], err)
	}
	return syncedFile{f}, nil
}

func (s stagedFiles) Discard() error {
	var errs []error
	for _, name := range stagedNames {
		err := atomicfile.Remove(restored(s.dir, name), s.retry)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type syncedFile struct{ *os.File }

func (f syncedFile) Close() error {
	return errors.Join(f.Sync(), f.File.Close())
}
