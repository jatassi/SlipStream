// Package archive is the Backup file format: a tar of five fixed entries,
// compressed with zstd and encrypted to an age scrypt passphrase. S50
// (Handover) imports it to send and load a library.
package archive

import (
	"encoding/hex"
	"fmt"
	"time"
)

// Format is the manifest's format. Read refuses a higher one with
// *TooNewError.
const Format = 1

// Entry is one of the five fixed entries, in archive order. The order is
// the type: a reader that sees any other name, order or count refuses.
type Entry int

const (
	ManifestEntry   Entry = iota // manifest.json, at most 1 MiB
	LibraryEntry                 // library.db, bounded by Limits.Library
	LibraryKeyEntry              // library.key, at most 1 MiB
	BackupKeyEntry               // backup.key, at most 1 MiB
	IndexEntry                   // backups.json, at most 1 MiB
	entries
)

var entryNames = [entries]string{"manifest.json", "library.db", "library.key", "backup.key", "backups.json"}

func (e Entry) Name() string { return entryNames[e] }

// smallEntry bounds every entry but library.db. The largest, backups.json,
// lists a few hundred Backups in tens of KiB.
const smallEntry = 1 << 20

type Trigger string

const (
	Scheduled      Trigger = "scheduled"
	Manual         Trigger = "manual"
	Update         Trigger = "update"
	BeforeRestore  Trigger = "before-restore"
	BeforeImport   Trigger = "before-import"
	BeforeHandover Trigger = "before-handover"
	Superseded     Trigger = "superseded"
	Restored       Trigger = "restored"
)

var triggers = []Trigger{Scheduled, Manual, Update, BeforeRestore, BeforeImport, BeforeHandover, Superseded, Restored}

// ID is a Backup's 128-bit id. Its last four Crockford base32 characters
// end the file name.
type ID [16]byte

func (id ID) MarshalText() ([]byte, error) { return []byte(hex.EncodeToString(id[:])), nil }

func (id *ID) UnmarshalText(b []byte) error {
	if hex.DecodedLen(len(b)) != len(id) {
		return fmt.Errorf("a Backup id is %d hex characters, not %d", hex.EncodedLen(len(id)), len(b))
	}
	if _, err := hex.Decode(id[:], b); err != nil {
		return fmt.Errorf("read a Backup id: %w", err)
	}
	return nil
}

// Manifest holds schema-stable fields only, so the hook fills it before
// any of today's queries can run.
type Manifest struct {
	Format        int              `json:"format"`
	Version       string           `json:"version"`
	Schemas       map[string]int64 `json:"schemas"`
	BackupID      ID               `json:"backupId"`
	TakenAt       time.Time        `json:"takenAt"`
	Trigger       Trigger          `json:"trigger"`
	ControllerKey []byte           `json:"controllerKey"`
	Generation    uint32           `json:"generation"`
}

// Params is the fixed encoding, from timings.backups.
type Params struct {
	ScryptLog  int
	ZstdWindow int
}

// Sources is what Write packs. LibraryDB is a path, so a 1 GB snapshot
// streams from disk.
type Sources struct {
	LibraryDB  string
	LibraryKey []byte
	BackupKey  []byte
	Index      []byte
}

// Sums is BLAKE3 of each entry's plaintext and of the whole ciphertext.
type Sums struct {
	Entries    [entries][32]byte
	Ciphertext [32]byte
	Size       int64
}

// Limits bound the reader. Library is the free space checked first.
type Limits struct {
	Library    int64
	ZstdWindow int
	ZstdMemory int64
}
