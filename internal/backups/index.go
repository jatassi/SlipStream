package backups

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/atomicfile"
	"github.com/jatassi/slipstream/internal/backups/archive"
)

const IndexFile = "backups.json"

const indexFormat = 1

// Index owns backups.json, outside library.db, so a restore in place and a
// rollback keep every Backup taken since. The Node opens it once, before
// library.db; the hook and the service share the one value, so the file
// has one writer.
type Index struct {
	mu     sync.Mutex
	path   string
	retry  time.Duration
	ledger Ledger
	watch  chan struct{}
}

type Ledger struct {
	// StartedVersion is the version of the last binary that started on the
	// library.
	StartedVersion string
	// PassphraseConfirmedAt is when the Admin last showed they hold the
	// current passphrase, zero until then.
	PassphraseConfirmedAt time.Time
	// Sheet is the recovery sheet's last download, zero before one.
	Sheet Sheet
	// Failure is the newest Backup that failed since the newest one that
	// passed, nil when the newest passed.
	Failure *Failure
	// DeleteBefore is the passphrase generation a Replace made with Delete
	// older Backups on, zero for none: the first Backup of it or a later
	// one that passes its check deletes every Backup of an earlier one.
	DeleteBefore Generation
	// Destinations is in the order they were added.
	Destinations []Destination
	// Backups is newest first.
	Backups []Entry
}

// Sheet is when the recovery sheet was last downloaded and the
// destinations it listed.
type Sheet struct {
	DownloadedAt time.Time       `json:"downloadedAt"`
	Destinations []DestinationID `json:"destinations"`
}

// Failure is a run of failed Backups: when the first and the newest
// failed, and the room the newest lacked when room was the cause.
type Failure struct {
	Since time.Time   `json:"since"`
	At    time.Time   `json:"at"`
	Space *SpaceError `json:"space,omitempty"`
}

type EntryState string

const (
	// Taking is listed before its file exists; recover checks or drops it.
	Taking   EntryState = "taking"
	Verified EntryState = "verified"
	// Failed counts for nothing and goes at the next run.
	Failed EntryState = "failed"
	// Deleting is pruned; it goes once its file is gone.
	Deleting EntryState = "deleting"
)

type Entry struct {
	ID               archive.ID       `json:"id"`
	Name             archive.Name     `json:"name"`
	TakenAt          time.Time        `json:"takenAt"`
	Trigger          archive.Trigger  `json:"trigger"`
	Version          string           `json:"version"`
	Schemas          map[string]int64 `json:"schemas"`
	Size             int64            `json:"size"`
	Sum              Sum              `json:"sum"`
	Generation       Generation       `json:"generation"`
	WhileReconciling bool             `json:"whileReconciling"`
	State            EntryState       `json:"state"`
	VerifiedAt       time.Time        `json:"verifiedAt,omitzero"`
	Local            bool             `json:"local"`
	// Cause is a Failed entry's, in words.
	Cause string `json:"cause,omitempty"`
	// By is the Account that pressed Back up now.
	By string `json:"by,omitempty"`
	// Copies is the Backup's copy on each destination it was queued to.
	Copies []Copy `json:"copies,omitempty"`
	// Pull is a restore pulling the Backup back from a destination into
	// the Backups folder, nil for none.
	Pull *Pull `json:"pull,omitempty"`
}

// Sum is BLAKE3 of a Backup's ciphertext.
type Sum [32]byte

func (s Sum) MarshalText() ([]byte, error) { return []byte(hex.EncodeToString(s[:])), nil }

func (s *Sum) UnmarshalText(b []byte) error {
	if hex.DecodedLen(len(b)) != len(s) {
		return fmt.Errorf("a sum is %d hex characters, not %d", hex.EncodedLen(len(s)), len(b))
	}
	if _, err := hex.Decode(s[:], b); err != nil {
		return fmt.Errorf("read a sum: %w", err)
	}
	return nil
}

type indexFile struct {
	Format                int           `json:"format"`
	StartedVersion        string        `json:"startedVersion"`
	PassphraseConfirmedAt time.Time     `json:"passphraseConfirmedAt,omitzero"`
	Sheet                 Sheet         `json:"sheet,omitzero"`
	Failure               *Failure      `json:"failure,omitempty"`
	DeleteBefore          Generation    `json:"deleteBefore,omitzero"`
	Destinations          []Destination `json:"destinations"`
	Backups               []Entry       `json:"backups"`
}

// OpenIndex reads path. A missing file is an empty Ledger.
func OpenIndex(path string, retry time.Duration) (*Index, error) {
	x := &Index{path: path, retry: retry, watch: make(chan struct{}, 1)}
	b, err := os.ReadFile(path) //nolint:gosec // path is the data dir's backups.json, chosen by the Node
	if errors.Is(err, fs.ErrNotExist) {
		return x, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", IndexFile, err)
	}
	if x.ledger, err = decodeIndex(b); err != nil {
		return nil, fmt.Errorf("read %s: %w", IndexFile, err)
	}
	return x, nil
}

func decodeIndex(b []byte) (Ledger, error) {
	var head struct {
		Format int `json:"format"`
	}
	if err := json.Unmarshal(b, &head); err != nil {
		return Ledger{}, fmt.Errorf("decode: %w", err)
	}
	if head.Format != indexFormat {
		return Ledger{}, fmt.Errorf("format %d, and this version reads format %d", head.Format, indexFormat)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var f indexFile
	if err := dec.Decode(&f); err != nil {
		return Ledger{}, fmt.Errorf("decode: %w", err)
	}
	return Ledger{StartedVersion: f.StartedVersion, PassphraseConfirmedAt: f.PassphraseConfirmedAt, Sheet: f.Sheet, Failure: f.Failure, DeleteBefore: f.DeleteBefore, Destinations: f.Destinations, Backups: f.Backups}, nil
}

func encodeIndex(l Ledger) ([]byte, error) {
	backups, destinations := l.Backups, l.Destinations
	if backups == nil {
		backups = []Entry{}
	}
	if destinations == nil {
		destinations = []Destination{}
	}
	f := indexFile{Format: indexFormat, StartedVersion: l.StartedVersion, PassphraseConfirmedAt: l.PassphraseConfirmedAt, Sheet: l.Sheet, Failure: l.Failure, DeleteBefore: l.DeleteBefore, Destinations: destinations, Backups: backups}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", IndexFile, err)
	}
	return b, nil
}

// Read is a copy nothing else changes.
func (x *Index) Read() Ledger {
	x.mu.Lock()
	defer x.mu.Unlock()
	return x.ledger.clone()
}

// Update applies fn to a copy and writes it. An error from fn, or a
// Ledger fn left as it was, leaves the file alone and Changed quiet.
func (x *Index) Update(fn func(*Ledger) error) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	next := x.ledger.clone()
	if err := fn(&next); err != nil {
		return err
	}
	b, err := encodeIndex(next)
	if err != nil {
		return err
	}
	if was, err := encodeIndex(x.ledger); err == nil && bytes.Equal(was, b) {
		return nil
	}
	if err := atomicfile.Replace(x.path, b, x.retry); err != nil {
		return fmt.Errorf("write %s: %w", IndexFile, err)
	}
	x.ledger = next
	select {
	case x.watch <- struct{}{}:
	default:
	}
	return nil
}

// Changed receives after each Update, coalesced.
func (x *Index) Changed() <-chan struct{} { return x.watch }

// Bytes is the file as a Backup carries it.
func (x *Index) Bytes() ([]byte, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	return encodeIndex(x.ledger)
}

func (l *Ledger) clone() Ledger {
	out := Ledger{StartedVersion: l.StartedVersion, PassphraseConfirmedAt: l.PassphraseConfirmedAt, Sheet: l.Sheet, DeleteBefore: l.DeleteBefore, Destinations: slices.Clone(l.Destinations), Backups: slices.Clone(l.Backups)}
	out.Sheet.Destinations = slices.Clone(l.Sheet.Destinations)
	if l.Failure != nil {
		f := *l.Failure
		if f.Space != nil {
			f.Space = new(*f.Space)
		}
		out.Failure = &f
	}
	for i := range out.Backups {
		e := &out.Backups[i]
		e.Schemas, e.Copies = maps.Clone(e.Schemas), slices.Clone(e.Copies)
		if e.Pull != nil {
			e.Pull = new(*e.Pull)
		}
	}
	return out
}

func (l *Ledger) Entry(id archive.ID) (Entry, bool) {
	i := slices.IndexFunc(l.Backups, func(e Entry) bool { return e.ID == id })
	if i < 0 {
		return Entry{}, false
	}
	return l.Backups[i], true
}

func (l *Ledger) set(e Entry) {
	if i := slices.IndexFunc(l.Backups, func(o Entry) bool { return o.ID == e.ID }); i >= 0 {
		l.Backups[i] = e
		return
	}
	l.Backups = slices.Insert(l.Backups, 0, e)
}

func (l *Ledger) drop(id archive.ID) {
	l.Backups = slices.DeleteFunc(l.Backups, func(e Entry) bool { return e.ID == id })
}
