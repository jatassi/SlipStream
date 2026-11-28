package engine

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"sync"

	"github.com/zeebo/blake3"

	"github.com/jatassi/slipstream/internal/atomicfile"
	"github.com/jatassi/slipstream/internal/backups/archive"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// backupFolders is this Node's Backup folders as its intent lists them. It
// opens, stages, renames and deletes only a Backup name in a listed
// folder, so the Controller can't name any other file on this Node, and
// it reports the Backup names each folder holds and the keyed check the
// intent asks for.
type backupFolders struct {
	mu      sync.Mutex
	folders map[string]*tv1.BackupFolder
	facts   []*tv1.BackupFolderFacts
	// checked is the last answer to each folder's check, so a check key
	// is answered once.
	checked map[string]*tv1.BackupCheck
}

// errNotListed is a folder this Node's intent doesn't list.
var errNotListed = errors.New("the folder isn't a Backup folder this Node keeps")

// at is the path of the Backup name in folder, while the intent lists
// folder and name is a Backup name.
func (e *Engine) at(folder, name string) (string, error) {
	e.backups.mu.Lock()
	_, ok := e.backups.folders[folder]
	e.backups.mu.Unlock()
	if !ok {
		return "", errNotListed
	}
	if _, err := archive.ParseName(name); err != nil {
		return "", fmt.Errorf("%q: %w", name, err)
	}
	return e.cfg.Disk.Host().Style.Join(folder, name), nil
}

// open resolves a sender's source: a Backup in a listed folder, a file
// under the scratch folder, or what the Opener answers.
func (e *Engine) open(ref source.Ref) (string, error) {
	switch r := ref.(type) {
	case source.Backup:
		path, err := e.at(r.Folder, r.Name)
		if err != nil {
			return "", &source.Absent{Reason: source.Gone, Detail: err.Error()}
		}
		return path, nil
	case source.Scratch:
		return e.scratchSource(r)
	}
	return e.cfg.Sources.Open(ref)
}

// backupStaging is the folder a Backup kind's Transfer stages in: a
// hidden folder in the Backup folder, so the rename into place stays on
// one volume.
func (e *Engine) backupStaging(ctx context.Context, b *tv1.BackupPlace, id ID) (string, error) {
	if _, err := e.at(b.GetFolder(), b.GetName()); err != nil {
		return "", err
	}
	style := e.cfg.Disk.Host().Style
	staging := style.Join(b.GetFolder(), storing.StagingDir)
	folder := style.Join(staging, id.String())
	for _, p := range []string{staging, folder} {
		if err := e.cfg.Disk.Mkdir(ctx, p); err != nil && !errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("make the staging folder: %w", err)
		}
	}
	return folder, nil
}

// place renames a verified staged Backup to its name in its folder.
func (e *Engine) place(ctx context.Context, staged string, b *tv1.BackupPlace) error {
	to, err := e.at(b.GetFolder(), b.GetName())
	if err != nil {
		return err
	}
	if err := atomicfile.Retried(e.cfg.Timings.Retry, func() error { return e.cfg.Disk.Rename(ctx, staged, to) }); err != nil {
		return fmt.Errorf("name the Backup: %w", err)
	}
	e.scanFolders(ctx)
	return nil
}

// followFolders takes the intent's Backup folders: it deletes each name
// a folder's delete lists, then reads every folder again.
func (e *Engine) followFolders(ctx context.Context, folders []*tv1.BackupFolder) {
	listed := make(map[string]*tv1.BackupFolder, len(folders))
	for _, f := range folders {
		listed[f.GetFolder()] = f
	}
	e.backups.mu.Lock()
	e.backups.folders = listed
	e.backups.mu.Unlock()
	for _, f := range folders {
		for _, name := range f.GetDelete() {
			path, err := e.at(f.GetFolder(), name)
			if err == nil {
				err = atomicfile.Retried(e.cfg.Timings.Retry, func() error { return e.cfg.Disk.Remove(ctx, path) })
			}
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				e.cfg.Log.Warn("delete a Backup", "folder", f.GetFolder(), "name", name, "err", err)
			}
		}
	}
	e.scanFolders(ctx)
}

// scanFolders reads the Backup names in each listed folder and answers
// each check the intent asks, into the facts.
func (e *Engine) scanFolders(ctx context.Context) {
	e.backups.mu.Lock()
	folders := make([]*tv1.BackupFolder, 0, len(e.backups.folders))
	for _, f := range e.backups.folders {
		folders = append(folders, f)
	}
	e.backups.mu.Unlock()
	slices.SortFunc(folders, func(a, b *tv1.BackupFolder) int { return cmp.Compare(a.GetFolder(), b.GetFolder()) })
	facts := make([]*tv1.BackupFolderFacts, 0, len(folders))
	checked := map[string]*tv1.BackupCheck{}
	for _, f := range folders {
		fact := &tv1.BackupFolderFacts{Folder: f.GetFolder()}
		entries, err := e.cfg.Disk.ReadDir(ctx, f.GetFolder())
		if err != nil {
			fact.Error = err.Error()
		}
		for _, entry := range entries {
			if _, err := archive.ParseName(entry.Name); err == nil && !entry.Dir {
				fact.Names = append(fact.Names, entry.Name)
			}
		}
		slices.Sort(fact.Names)
		if len(f.GetCheckKey()) > 0 {
			fact.Check = e.check(ctx, f)
			checked[f.GetFolder()] = fact.Check
		}
		facts = append(facts, fact)
	}
	e.backups.mu.Lock()
	e.backups.facts, e.backups.checked = facts, checked
	e.backups.mu.Unlock()
	e.facts.urgent()
}

// check is the keyed BLAKE3 of f's check under its key, read once per
// key.
func (e *Engine) check(ctx context.Context, f *tv1.BackupFolder) *tv1.BackupCheck {
	e.backups.mu.Lock()
	last := e.backups.checked[f.GetFolder()]
	e.backups.mu.Unlock()
	if last.GetName() == f.GetCheck() && bytes.Equal(last.GetKey(), f.GetCheckKey()) {
		return last
	}
	c := &tv1.BackupCheck{Name: f.GetCheck(), Key: f.GetCheckKey()}
	sum, err := e.keyedSum(ctx, f)
	if err != nil {
		c.Error = err.Error()
	} else {
		c.Sum = sum
	}
	return c
}

func (e *Engine) keyedSum(ctx context.Context, f *tv1.BackupFolder) ([]byte, error) {
	if len(f.GetCheckKey()) != 32 {
		return nil, fmt.Errorf("a check key of %d bytes", len(f.GetCheckKey()))
	}
	path, err := e.at(f.GetFolder(), f.GetCheck())
	if err != nil {
		return nil, err
	}
	in, err := e.cfg.Disk.Open(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = in.Close() }()
	h, err := blake3.NewKeyed(f.GetCheckKey())
	if err != nil {
		return nil, fmt.Errorf("start the check: %w", err)
	}
	if _, err := io.Copy(h, in); err != nil {
		return nil, fmt.Errorf("read the Backup: %w", err)
	}
	return h.Sum(nil), nil
}
