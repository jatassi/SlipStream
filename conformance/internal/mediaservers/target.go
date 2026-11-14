// Package mediaservers is what the Media Server conformance suites share:
// the Target one product is under test as, a fake or the real thing, and
// the cases every kind passes, each driving the server through the kind's
// own way of telling it about a change.
package mediaservers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/storing"
)

// Target is one product under test.
type Target struct {
	Kind   link.KindName
	Server Server
	// Library is a library SlipStream may fill and empty, and Folder one of
	// its folders as the server sees it.
	Library string
	Folder  link.ServerPath
	// Disk writes the files the server sees under Folder.
	Disk Disk
	// Fresh is a library never scanned with content, nil when the target
	// has none, as a real server SlipStream may not create libraries on.
	Fresh *Fresh
	// FreshTakesChange is a server that takes a change into a fresh library
	// without a scan, as Emby did (#101 run 2).
	FreshTakesChange bool
	// Play starts a playback of the file at p and answers its stop, nil
	// when no player is at hand.
	Play func(t *testing.T, p link.ServerPath, d link.Decision) (stop func())
	// Settle is how long the server takes a change it takes at all: the
	// kind's own delay and a little more. A case waits it out before it
	// reads a change as ignored.
	Settle time.Duration
	// Bound is how long a case waits for the server to take a change.
	Bound time.Duration
	Log   *slog.Logger
}

// Fresh is a library never scanned with content and its folder.
type Fresh struct {
	Library string
	Folder  link.ServerPath
	Disk    Disk
}

// Server is the product as a case drives it, through the kind's wire
// client.
type Server interface {
	// Reach reads the identity with no credential.
	Reach(ctx context.Context) (link.Identity, error)
	Libraries(ctx context.Context) ([]link.Library, error)
	// Tell tells the server files changed, the kind's way: a Plex folder
	// refresh, a Jellyfin or Emby path report.
	Tell(ctx context.Context, library string, told []Told) error
	// Moved tells the server files moved, in the order the kind keeps
	// watch data.
	Moved(ctx context.Context, library string, from, to []link.ServerPath) error
	// SidecarRemoved tells the server a Sidecar beside video is gone.
	SidecarRemoved(ctx context.Context, library string, video, sidecar link.ServerPath) error
	// Scan starts a full scan of library, and ScanEnded is whether the
	// scan the last Scan asked for has ended.
	Scan(ctx context.Context, library string) error
	ScanEnded(ctx context.Context, library string) (bool, error)
	ItemAt(ctx context.Context, library string, p link.ServerPath) (Item, bool, error)
	// Item is the item key, false when the server holds none.
	Item(ctx context.Context, key string) (Item, bool, error)
	// Watched plays item once and puts it in the collection name.
	Watched(ctx context.Context, item, name string) error
	Sessions(ctx context.Context) ([]link.Session, error)
}

// Told is one file a case tells the server about. Gone is a file no
// longer on disk.
type Told struct {
	Path link.ServerPath
	Gone bool
}

// Item is an item as the cases compare it.
type Item struct {
	Key         string
	Files       []link.ServerPath
	Sidecars    int
	Played      bool
	PlayCount   int
	Collections []string
}

// Disk is the server's folder on this machine.
type Disk interface {
	// Place writes a video, or a Sidecar by its extension, at rel. Each
	// video's bytes differ from every other's.
	Place(rel storing.Rel) error
	Move(from, to storing.Rel) error
	Remove(rel storing.Rel) error
}

// DirDisk is a Disk on the local folder Dir. Each video is a copy of the
// video at Seed with its name appended, or a few bytes of its name when
// Seed is "".
type DirDisk struct {
	Dir  string
	Seed string
}

func (d DirDisk) path(rel storing.Rel) string {
	return filepath.Join(d.Dir, filepath.FromSlash(rel.String()))
}

func (d DirDisk) Place(rel storing.Rel) error {
	file := d.path(rel)
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil { //nolint:gosec // the Media Server reads the folder, maybe as another user
		return fmt.Errorf("place %s: %w", rel, err)
	}
	if filepath.Ext(file) == ".srt" {
		return os.WriteFile(file, []byte("1\n00:00:01,000 --> 00:00:02,000\nSlipStream\n"), 0o644) //nolint:wrapcheck,gosec // the path names itself, and the Media Server reads it
	}
	out, err := os.Create(file) //nolint:gosec // the suite's own folder
	if err != nil {
		return fmt.Errorf("place %s: %w", rel, err)
	}
	var cerr error
	if d.Seed != "" {
		cerr = copyFile(out, d.Seed)
	}
	_, werr := out.WriteString(rel.String())
	if err := errors.Join(cerr, werr, out.Close()); err != nil {
		return fmt.Errorf("place %s: %w", rel, err)
	}
	return nil
}

func copyFile(out io.Writer, from string) error {
	in, err := os.Open(from) //nolint:gosec // the seed video the run names
	if err != nil {
		return err //nolint:wrapcheck // Place names the file
	}
	defer func() { _ = in.Close() }()
	_, err = io.Copy(out, in)
	return err //nolint:wrapcheck // Place names the file
}

func (d DirDisk) Move(from, to storing.Rel) error {
	if err := os.MkdirAll(filepath.Dir(d.path(to)), 0o755); err != nil { //nolint:gosec // the Media Server reads the folder, maybe as another user
		return fmt.Errorf("move %s: %w", from, err)
	}
	return os.Rename(d.path(from), d.path(to)) //nolint:wrapcheck // the paths name themselves
}

func (d DirDisk) Remove(rel storing.Rel) error {
	return os.Remove(d.path(rel)) //nolint:wrapcheck // the path names itself
}
