package backups

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/atomicfile"
	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

// SnapshotFile is where a Backup's snapshot of library.db is written, in
// the data dir, and deleted once the archive is checked.
const SnapshotFile = "library.snapshot.db"

const partSuffix = ".part"

// recheckLimit bounds library.db when recover checks a Backup again. The
// check writes nothing, so free space doesn't bound it.
const recheckLimit = 1 << 40

// EngineConfig is what every Backup needs, known before library.db opens.
type EngineConfig struct {
	// Dir is the data dir, which holds library.key, backup.key and the
	// snapshot.
	Dir string
	// Folder is the Backups folder, resolved.
	Folder  string
	Index   *Index
	Schemas []storage.Schema[storage.Library]
	// ControllerKey is this Node's public key, which the manifest records.
	ControllerKey []byte
	Version       string
	Timings       Timings
	// Free is how many bytes the volume holding path has free.
	Free  func(ctx context.Context, path string) (int64, error)
	Clock func() time.Time
	// Rand draws Backup ids, passphrases and fence secrets. Nil is
	// crypto/rand.
	Rand io.Reader
	Log  *slog.Logger
}

// Engine takes one Backup at a time, since scrypt alone needs about
// 256 MiB. The hook and the service share one Engine through the Node.
type Engine struct {
	cfg   EngineConfig
	mu    sync.Mutex
	vault *Vault
}

func NewEngine(cfg EngineConfig) *Engine {
	if cfg.Rand == nil {
		cfg.Rand = rand.Reader
	}
	return &Engine{cfg: cfg}
}

func (e *Engine) Index() *Index { return e.cfg.Index }

// Vault opens backup.key with keys once. Every later call answers the
// same Vault, so the file has one writer.
func (e *Engine) Vault(keys *secret.Keys) (*Vault, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.vault == nil {
		v, err := OpenVault(filepath.Join(e.cfg.Dir, VaultFile), keys, e.cfg.Timings.WindowsRetry)
		if err != nil {
			return nil, err
		}
		e.vault = v
	}
	return e.vault, nil
}

// source is the library.db a Backup snapshots: *storage.DB[Library] once
// it is open, *storage.Unmigrated in the hook.
type source interface {
	Estimate(ctx context.Context) (int64, error)
	Snapshot(ctx context.Context, dst string) error
}

// request is one Backup to take.
type request struct {
	Trigger          archive.Trigger
	By               string
	WhileReconciling bool
	// Retention prunes once the Backup passes; nil prunes nothing, as in
	// the hook, which can't read stored settings yet.
	Retention *Retention
}

// SpaceError is a volume with less free space than the Backup needs.
type SpaceError struct {
	Folder     string
	Need, Free int64
}

func (e *SpaceError) Error() string {
	return fmt.Sprintf("%s has %s free, and the Backup needs about %s", e.Folder, Size(e.Free), Size(e.Need))
}

// take is every Backup:
//
//  1. refuse when the data dir's or the Backups folder's volume has less
//     free than the snapshot's estimate;
//  2. list the Backup as Taking, before any file exists;
//  3. snapshot library.db into the data dir and inspect the snapshot;
//  4. write <name>.part with O_EXCL, sync and close it;
//  5. read the .part back, whose sums must equal Write's;
//  6. rename it, sync the folder and delete the snapshot;
//  7. mark it Verified, then prune.
//
// A failure after step 2 marks the entry Failed and deletes what it wrote.
func (e *Engine) take(ctx context.Context, src source, req request) (Entry, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.vault == nil {
		panic("backups: a Backup taken before the Vault opened")
	}
	if err := e.recover(); err != nil {
		return Entry{}, err
	}
	need, err := src.Estimate(ctx)
	if err != nil {
		return Entry{}, err
	}
	if err := os.MkdirAll(e.cfg.Folder, 0o700); err != nil {
		return Entry{}, fmt.Errorf("make the Backups folder: %w", err)
	}
	for _, dir := range []string{e.cfg.Dir, e.cfg.Folder} {
		free, err := e.cfg.Free(ctx, dir)
		if err != nil {
			return Entry{}, fmt.Errorf("read the free space of %s: %w", dir, err)
		}
		if free < need {
			return Entry{}, &SpaceError{Folder: dir, Need: need, Free: free}
		}
	}
	gen, pass, err := e.vault.Current(e.cfg.Rand)
	if err != nil {
		return Entry{}, err
	}
	m := archive.Manifest{Version: e.cfg.Version, TakenAt: e.cfg.Clock().UTC().Truncate(time.Second), Trigger: req.Trigger, ControllerKey: e.cfg.ControllerKey, Generation: uint32(gen)}
	if _, err := io.ReadFull(e.cfg.Rand, m.BackupID[:]); err != nil {
		return Entry{}, fmt.Errorf("draw a Backup id: %w", err)
	}
	entry := Entry{ID: m.BackupID, Name: archive.NameOf(m), TakenAt: m.TakenAt, Trigger: m.Trigger, Version: m.Version, Generation: gen, WhileReconciling: req.WhileReconciling, State: Taking, By: req.By}
	if err := e.cfg.Index.Update(func(l *Ledger) error { l.set(entry); return nil }); err != nil {
		return Entry{}, err
	}
	done, err := e.write(ctx, src, m, entry, pass)
	if err != nil {
		e.fail(entry, err)
		return Entry{}, err
	}
	if err := e.cfg.Index.Update(func(l *Ledger) error {
		l.set(done)
		if req.Retention != nil {
			for _, id := range Prune(*l, *req.Retention) {
				pruned, _ := l.Entry(id)
				pruned.State = Deleting
				l.set(pruned)
			}
		}
		return nil
	}); err != nil {
		return Entry{}, err
	}
	return done, e.deleteMarked()
}

// write is steps 3 to 6 of take.
func (e *Engine) write(ctx context.Context, src source, m archive.Manifest, entry Entry, pass archive.Passphrase) (Entry, error) {
	snap := filepath.Join(e.cfg.Dir, SnapshotFile)
	if err := e.remove(snap); err != nil {
		return Entry{}, err
	}
	if err := src.Snapshot(ctx, snap); err != nil {
		return Entry{}, err
	}
	inspected, err := storage.Inspect(ctx, snap, e.cfg.Schemas)
	if err != nil {
		return Entry{}, fmt.Errorf("check the snapshot: %w", err)
	}
	m.Schemas = inspected.Versions
	sources, err := e.sources(snap)
	if err != nil {
		return Entry{}, err
	}
	part := e.path(entry.Name) + partSuffix
	sums, err := writePart(part, m, sources, pass, e.cfg.Timings.params())
	if err != nil {
		return Entry{}, err
	}
	info, err := os.Stat(snap)
	if err != nil {
		return Entry{}, fmt.Errorf("check the snapshot: %w", err)
	}
	opened, err := checkFile(part, pass, e.cfg.Timings.limits(info.Size()))
	if err != nil {
		return Entry{}, err
	}
	if opened.Sums() != sums {
		return Entry{}, errors.New("the Backup read back differs from what was written")
	}
	if err := atomicfile.Rename(part, e.path(entry.Name), e.cfg.Timings.WindowsRetry); err != nil {
		return Entry{}, fmt.Errorf("name the Backup: %w", err)
	}
	if err := atomicfile.SyncDir(e.cfg.Folder); err != nil {
		return Entry{}, fmt.Errorf("sync the Backups folder: %w", err)
	}
	if err := e.remove(snap); err != nil {
		return Entry{}, err
	}
	entry.Schemas, entry.Size, entry.Sum = m.Schemas, sums.Size, sums.Ciphertext
	entry.State, entry.VerifiedAt, entry.Local = Verified, e.cfg.Clock().UTC(), true
	return entry, nil
}

func (e *Engine) sources(snap string) (archive.Sources, error) {
	libraryKey, err := os.ReadFile(filepath.Join(e.cfg.Dir, secret.KeyFile))
	if err != nil {
		return archive.Sources{}, fmt.Errorf("read %s: %w", secret.KeyFile, err)
	}
	backupKey, err := e.vault.Bytes()
	if err != nil {
		return archive.Sources{}, err
	}
	index, err := e.cfg.Index.Bytes()
	if err != nil {
		return archive.Sources{}, err
	}
	return archive.Sources{LibraryDB: snap, LibraryKey: libraryKey, BackupKey: backupKey, Index: index}, nil
}

func writePart(part string, m archive.Manifest, s archive.Sources, pass archive.Passphrase, p archive.Params) (_ archive.Sums, err error) {
	f, err := os.OpenFile(part, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) //nolint:gosec // part is a Backup name in the Backups folder
	if err != nil {
		return archive.Sums{}, fmt.Errorf("create the Backup: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("write the Backup: %w", cerr))
		}
	}()
	sums, err := archive.Write(f, m, s, pass, p)
	if err != nil {
		return archive.Sums{}, err
	}
	if err := f.Sync(); err != nil {
		return archive.Sums{}, fmt.Errorf("write the Backup: %w", err)
	}
	return sums, nil
}

func checkFile(path string, pass archive.Passphrase, l archive.Limits) (archive.Opened, error) {
	f, err := os.Open(path) //nolint:gosec // path is a Backup name in the Backups folder
	if err != nil {
		return archive.Opened{}, fmt.Errorf("read the Backup back: %w", err)
	}
	defer func() { _ = f.Close() }()
	opened, err := archive.Check(f, pass, l)
	if err != nil {
		return archive.Opened{}, fmt.Errorf("read the Backup back: %w", err)
	}
	return opened, nil
}

// fail marks entry Failed with cause and deletes the files it left. The
// Backup failed already, so a failure here goes to the log alone.
func (e *Engine) fail(entry Entry, cause error) {
	entry.State, entry.Cause = Failed, cause.Error()
	err := errors.Join(
		e.remove(e.path(entry.Name)+partSuffix),
		e.remove(filepath.Join(e.cfg.Dir, SnapshotFile)),
		e.cfg.Index.Update(func(l *Ledger) error { l.set(entry); return nil }),
	)
	if err != nil {
		e.cfg.Log.Error("record a failed Backup", "backup", entry.Name, "err", err)
	}
}

// Recover finishes what a stopped start or run left, at each start and
// each run:
//
//   - Failed, from an earlier run, or Deleting: delete its file, drop it;
//   - Taking with its file: check it again, then Verified or Failed;
//   - Taking with only its .part, or neither: delete the .part, drop it;
//   - a .part no entry names, and the snapshot: delete.
//
// A file in the folder that no entry names is never listed.
func (e *Engine) Recover() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.recover()
}

func (e *Engine) recover() error {
	if err := e.cfg.Index.Update(func(l *Ledger) error {
		for _, entry := range l.Backups {
			if entry.State == Failed {
				entry.State = Deleting
				l.set(entry)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if err := e.deleteMarked(); err != nil {
		return err
	}
	for _, entry := range e.cfg.Index.Read().Backups {
		if entry.State != Taking {
			continue
		}
		if err := e.recheck(entry); err != nil {
			return err
		}
	}
	return e.leftovers()
}

func (e *Engine) recheck(entry Entry) error {
	final := e.path(entry.Name)
	_, statErr := os.Stat(final)
	switch {
	case errors.Is(statErr, fs.ErrNotExist):
		if err := e.remove(final + partSuffix); err != nil {
			return err
		}
		return e.cfg.Index.Update(func(l *Ledger) error { l.drop(entry.ID); return nil })
	case statErr != nil:
		return fmt.Errorf("check a Backup: %w", statErr)
	}
	pass, ok := e.vault.Passphrase(entry.Generation)
	checked := entry
	switch opened, err := checkFile(final, pass, e.cfg.Timings.limits(recheckLimit)); {
	case !ok:
		checked.State, checked.Cause = Failed, "backup.key has no passphrase of its generation"
	case err != nil:
		checked.State, checked.Cause = Failed, err.Error()
	default:
		m := opened.Manifest()
		checked.Schemas, checked.Size, checked.Sum = m.Schemas, opened.Sums().Size, opened.Sums().Ciphertext
		checked.State, checked.VerifiedAt, checked.Local = Verified, e.cfg.Clock().UTC(), true
	}
	return e.cfg.Index.Update(func(l *Ledger) error { l.set(checked); return nil })
}

// deleteMarked deletes the file of each Deleting entry and drops it.
func (e *Engine) deleteMarked() error {
	for _, entry := range e.cfg.Index.Read().Backups {
		if entry.State != Deleting {
			continue
		}
		if err := e.remove(e.path(entry.Name)); err != nil {
			return err
		}
		if err := e.cfg.Index.Update(func(l *Ledger) error { l.drop(entry.ID); return nil }); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) leftovers() error {
	if err := e.remove(filepath.Join(e.cfg.Dir, SnapshotFile)); err != nil {
		return err
	}
	names, err := os.ReadDir(e.cfg.Folder)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("list the Backups folder: %w", err)
	}
	for _, d := range names {
		name, ok := strings.CutSuffix(d.Name(), partSuffix)
		if !ok {
			continue
		}
		if _, err := archive.ParseName(name); err != nil {
			continue
		}
		if err := e.remove(filepath.Join(e.cfg.Folder, d.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) path(n archive.Name) string { return filepath.Join(e.cfg.Folder, n.String()) }

func (e *Engine) remove(path string) error {
	if err := atomicfile.Remove(path, e.cfg.Timings.WindowsRetry); err != nil {
		return fmt.Errorf("delete %s: %w", filepath.Base(path), err)
	}
	return nil
}

var sizeUnits = [...]string{"bytes", "KB", "MB", "GB", "TB"}

// Size writes bytes as the web's bytes() does: 1024 to a unit, one
// decimal under 10.
func Size(n int64) string {
	if n == 1 {
		return "1 byte"
	}
	v, unit := float64(n), 0
	for v >= 1024 && unit < len(sizeUnits)-1 {
		v /= 1024
		unit++
	}
	digits := 0
	if unit > 0 && v < 10 {
		digits = 1
	}
	return strconv.FormatFloat(v, 'f', digits, 64) + " " + sizeUnits[unit]
}

// TakeOffline takes a Backup of db outside the Backup task, for a host
// command that holds the data dir lock while no Node runs. It prunes
// nothing.
func (e *Engine) TakeOffline(ctx context.Context, db *storage.DB[storage.Library], keys *secret.Keys, trigger archive.Trigger) (Entry, error) {
	if _, err := e.Vault(keys); err != nil {
		return Entry{}, err
	}
	return e.take(ctx, db, request{Trigger: trigger})
}

// Restored is what the restore's Write leaves in the index: this version
// as the last started, and the passphrase confirmed when the Admin typed
// it into the restore.
func Restored(x *Index, version string, typed bool, now time.Time) error {
	return x.Update(func(l *Ledger) error {
		l.StartedVersion = version
		if typed {
			l.PassphraseConfirmedAt = now.UTC()
		}
		return nil
	})
}
