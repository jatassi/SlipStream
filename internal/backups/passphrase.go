package backups

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/storage"
)

// ErrNewestVerified is a delete of the newest Backup that passed its
// check, which retention keeps too.
var ErrNewestVerified = errors.New("the newest good Backup can't be deleted")

// Passphrase is the passphrase every new Backup is encrypted with, made
// on first use.
func (b *Backups) Passphrase() (archive.Passphrase, error) {
	v, err := b.cfg.Engine.Vault(b.cfg.Keys)
	if err != nil {
		return archive.Passphrase{}, err
	}
	_, pass, err := v.Current(b.cfg.Engine.cfg.Rand)
	return pass, err
}

// Saved records that the Admin saved the current passphrase.
func (b *Backups) Saved() error {
	now := b.cfg.Engine.cfg.Clock().UTC()
	return b.cfg.Engine.Index().Update(func(l *Ledger) error { l.PassphraseConfirmedAt = now; return nil })
}

// Check is whether typed is the current passphrase, which a match records
// as saved.
func (b *Backups) Check(typed string) (bool, error) {
	cur, err := b.Passphrase()
	if err != nil {
		return false, err
	}
	pass, perr := archive.ParsePassphrase(typed)
	if perr != nil || subtle.ConstantTimeCompare([]byte(pass.Reveal()), []byte(cur.Reveal())) != 1 {
		return false, nil //nolint:nilerr // words that aren't a passphrase don't match it
	}
	return true, b.Saved()
}

// Replace makes a new passphrase, and a new fence secret with it, so the
// current one is no longer saved, then starts a Backup with it. With
// deleteOlder, every Backup of an earlier passphrase is deleted, copies
// included, once the first Backup of the new one passes its check, which
// the index records so a stop before that Backup keeps the choice.
func (b *Backups) Replace(ctx context.Context, by storage.UUID, deleteOlder bool) (*Run, error) {
	v, err := b.cfg.Engine.Vault(b.cfg.Keys)
	if err != nil {
		return nil, err
	}
	gen, err := v.Replace(b.cfg.Engine.cfg.Rand)
	if err != nil {
		return nil, err
	}
	if err := b.cfg.Engine.Index().Update(func(l *Ledger) error {
		l.PassphraseConfirmedAt, l.Sheet, l.DeleteBefore = time.Time{}, Sheet{}, 0
		if deleteOlder {
			l.DeleteBefore = gen
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return b.Take(ctx, Request{Trigger: archive.Manual, By: by})
}

// Delete deletes a listed Backup and its copies. The newest Backup that
// passed its check is refused.
func (b *Backups) Delete(id archive.ID) error {
	return b.cfg.Engine.delete(id)
}

func (e *Engine) delete(id archive.ID) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.cfg.Index.Update(func(l *Ledger) error {
		entry, ok := l.Entry(id)
		switch {
		case !ok, entry.State == Taking:
			return ErrNoBackup
		case entry.State == Deleting:
			return nil
		case entry.State == Verified && entry.ID == newestVerified(*l):
			return ErrNewestVerified
		}
		entry.State = Deleting
		l.set(entry)
		return nil
	}); err != nil {
		return err
	}
	return e.deleteMarked()
}

func newestVerified(l Ledger) archive.ID {
	var newest Entry
	for _, e := range l.Backups {
		if e.State == Verified && e.TakenAt.After(newest.TakenAt) {
			newest = e
		}
	}
	return newest.ID
}

// SheetFile is the recovery sheet's file name.
const SheetFile = "slipstream-recovery-sheet.txt"

// RecoverySheet is the recovery sheet: the passphrase, the Controller's
// Backups folder and each destination's, the file name pattern and how to
// restore. Its download is recorded with the destinations it lists.
func (b *Backups) RecoverySheet() ([]byte, error) {
	pass, err := b.Passphrase()
	if err != nil {
		return nil, err
	}
	l := b.Ledger()
	self := b.cfg.Self()
	name, _ := b.cfg.Roster(self)
	var s strings.Builder
	fmt.Fprintf(&s, "SlipStream recovery sheet for %s\n\n", name)
	fmt.Fprintf(&s, "Backup passphrase: %s\n\n", pass.Reveal())
	s.WriteString("Backups are kept in these folders:\n")
	fmt.Fprintf(&s, "%s: %s\n", name, b.cfg.Engine.cfg.Folder)
	var listed []DestinationID
	for _, d := range l.Active() {
		node, _ := b.cfg.Roster(d.Node)
		fmt.Fprintf(&s, "%s: %s\n", node, d.Folder)
		listed = append(listed, d.ID)
	}
	s.WriteString("\nEach Backup's file name holds the date it was taken, such as slipstream-20261114T040012Z-2.1.0-scheduled-7k3m.tar.zst.age.\n\n")
	s.WriteString("To restore a Backup on a new machine:\n")
	s.WriteString("1. Install SlipStream.\n")
	s.WriteString("2. Open it in a browser and enter the Setup code from its log.\n")
	s.WriteString("3. Pick Restore a Backup, choose the newest Backup you have and enter the passphrase above.\n")
	now := b.cfg.Engine.cfg.Clock().UTC()
	if err := b.cfg.Engine.Index().Update(func(l *Ledger) error {
		l.Sheet = Sheet{DownloadedAt: now, Destinations: listed}
		return nil
	}); err != nil {
		return nil, err
	}
	return []byte(s.String()), nil
}

// Unlisted is each active destination the last recovery sheet didn't
// list, all of them before one was downloaded.
func (l *Ledger) Unlisted() []Destination {
	var out []Destination
	for _, d := range l.Active() {
		listed := false
		for _, id := range l.Sheet.Destinations {
			listed = listed || id == d.ID
		}
		if !listed {
			out = append(out, d)
		}
	}
	return out
}
