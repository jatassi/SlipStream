// Package lifecycle owns lifecycle.json, the data dir's one lock on an
// operation that crosses a restart: a restore, Wipe and join, and later
// an update or a Handover. An operation starts only while no record
// exists, and the Node resumes the named one at its next start, before
// library.db opens.
//
// It also owns the swap a restore finishes with, which is a plan derived
// from the files in the data dir and nothing else, so a crash between any
// two renames resumes with exactly the renames left.
//
// It imports no SlipStream package but atomicfile, so the updater and
// Handover can write records without the Node.
package lifecycle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/jatassi/slipstream/internal/atomicfile"
)

const FileName = "lifecycle.json"

// Format is the newest record format this binary reads. A record with a
// higher one is refused, so a binary rolled back past a format change
// stops rather than guessing.
const Format = 1

// Op is one lifecycle operation, and a record holds exactly one: *Restore
// or *Wipe. A new operation is a new variant here and one arm in the
// Node's resume.
type Op interface {
	name() string
	check() error
}

// Stage is how far a restore's finish at start has gone. The renames of
// the swap aren't stages: Plan reads them from the files.
type Stage string

const (
	// Staged is every .restore file synced, with nothing live changed.
	Staged Stage = "staged"
	// Swapped is the live files being the restored ones, so library.db
	// may open.
	Swapped Stage = "swapped"
	// Written is the restore's Write committed, so the .pre-restore files
	// may go.
	Written Stage = "written"
)

// Cause is why a restore runs, which decides what its Write keeps.
type Cause string

const (
	ByRestore      Cause = "restore"
	ManualRollback Cause = "manual_rollback"
	// AutomaticRollback only swaps and opens: only migrations wrote
	// between its Backup and the rollback.
	AutomaticRollback Cause = "automatic_rollback"
)

// Backup is the manifest fields a resume needs.
type Backup struct {
	Name          string    `json:"name"`
	TakenAt       time.Time `json:"takenAt"`
	Version       string    `json:"version"`
	ControllerKey []byte    `json:"controllerKey"`
}

type Restore struct {
	// ID is random, and the restore's Write stamps it, so a Write that
	// already ran is never run again.
	ID         string `json:"id"`
	Stage      Stage  `json:"stage"`
	Cause      Cause  `json:"cause"`
	Backup     Backup `json:"backup"`
	NewMachine bool   `json:"newMachine"`
	// Remove is the ids of the Nodes the Admin unticked.
	Remove []string `json:"remove"`
	// RemovedKeys is, in place, the keys of Nodes removed after the
	// Backup was taken.
	RemovedKeys [][]byte `json:"removedKeys"`
	// Typed is whether the Admin typed the passphrase.
	Typed bool `json:"typed"`
	// Superseded is the proof for the old Controller key, minted at
	// staging with the archive's fence secret. Empty in place.
	Superseded []Proof `json:"superseded"`
}

type Proof struct {
	OldKey []byte `json:"oldKey"`
	Proof  []byte `json:"proof"`
}

func (*Restore) name() string { return "restore" }

func (r *Restore) check() error {
	switch r.Stage {
	case Staged, Swapped, Written:
	default:
		return fmt.Errorf("restore stage %q", r.Stage)
	}
	switch r.Cause {
	case ByRestore, ManualRollback, AutomaticRollback:
	default:
		return fmt.Errorf("restore cause %q", r.Cause)
	}
	return nil
}

// WipeStage is how far Wipe and join has gone.
type WipeStage string

// Deleting is the superseded Backup listed, with the library's files
// left to delete, library.db first.
const Deleting WipeStage = "deleting"

// Wipe is Wipe and join on a Superseded Controller. The resume deletes
// library.db, its keys and the Backups index, then joins with JoinCode.
type Wipe struct {
	Stage    WipeStage `json:"stage"`
	JoinCode string    `json:"joinCode"`
}

func (*Wipe) name() string { return "wipe" }

func (w *Wipe) check() error {
	if w.Stage != Deleting {
		return fmt.Errorf("wipe stage %q", w.Stage)
	}
	return nil
}

// ErrBusy is Begin while a record exists.
var ErrBusy = errors.New("a restore or an update is already running")

// RefusedError is a record this binary can't resume: a newer format, an
// operation it doesn't know, or bytes that aren't a record.
type RefusedError struct {
	Path string
	Err  error
}

func (e *RefusedError) Error() string { return fmt.Sprintf("lifecycle record %s: %v", e.Path, e.Err) }

func (e *RefusedError) Unwrap() error { return e.Err }

// Dir is a data dir as this package reaches it. Retry is how long each
// rename and delete retries on Windows, as atomicfile.Rename does.
type Dir struct {
	Path  string
	Retry time.Duration
}

func (d Dir) record() string { return filepath.Join(d.Path, FileName) }

type record struct {
	Format  int      `json:"format"`
	Op      string   `json:"op"`
	Restore *Restore `json:"restore,omitempty"`
	Wipe    *Wipe    `json:"wipe,omitempty"`
}

func encode(op Op) ([]byte, error) {
	r := record{Format: Format, Op: op.name()}
	switch op := op.(type) {
	case *Restore:
		r.Restore = op
	case *Wipe:
		r.Wipe = op
	}
	b, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("encode the lifecycle record: %w", err)
	}
	return b, nil
}

// Begin writes the record for op only when none exists, so two operations
// never both start, even across a restart. A record already there is
// ErrBusy.
func (d Dir) Begin(op Op) error {
	b, err := encode(op)
	if err != nil {
		return err
	}
	err = atomicfile.Create(d.record(), b, d.Retry)
	if errors.Is(err, fs.ErrExist) {
		return ErrBusy
	}
	return err
}

// Advance writes op over the record, as its next stage.
func (d Dir) Advance(op Op) error {
	b, err := encode(op)
	if err != nil {
		return err
	}
	return atomicfile.Replace(d.record(), b, d.Retry)
}

// Finish deletes the record, which ends the operation.
func (d Dir) Finish() error {
	if err := atomicfile.Remove(d.record(), d.Retry); err != nil {
		return err
	}
	return atomicfile.SyncDir(d.Path)
}

// Read is the record's operation, ok false when no record exists. A
// record this binary can't resume is a *RefusedError.
func (d Dir) Read() (op Op, ok bool, err error) {
	path := d.record()
	b, err := os.ReadFile(path) //nolint:gosec // the data dir's own record
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read the lifecycle record: %w", err)
	}
	op, err = decode(b)
	if err != nil {
		return nil, false, &RefusedError{Path: path, Err: err}
	}
	return op, true, nil
}

func decode(b []byte) (Op, error) {
	var head struct {
		Format int `json:"format"`
	}
	if err := json.Unmarshal(b, &head); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if head.Format > Format {
		return nil, fmt.Errorf("format %d, but this binary reads only up to %d", head.Format, Format)
	}
	if head.Format < 1 {
		return nil, fmt.Errorf("format %d", head.Format)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var r record
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if dec.More() {
		return nil, errors.New("data after the record")
	}
	var op Op
	var alone bool
	switch r.Op {
	case "restore":
		op, alone = r.Restore, r.Restore != nil && r.Wipe == nil
	case "wipe":
		op, alone = r.Wipe, r.Wipe != nil && r.Restore == nil
	default:
		return nil, fmt.Errorf("operation %q, which this binary doesn't know", r.Op)
	}
	if !alone {
		return nil, fmt.Errorf("operation %s with its fields missing or beside another operation's", r.Op)
	}
	if err := op.check(); err != nil {
		return nil, err
	}
	return op, nil
}
