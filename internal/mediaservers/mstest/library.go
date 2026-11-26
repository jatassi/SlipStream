package mstest

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/mediaservers/playback"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// File is a library file as the Playback report reads it.
type File struct {
	// ID is the file's, a new one when zero.
	ID    files.ID
	Work  works.WorkID
	Item  works.ItemID
	Label string
	Kept  bool
	Media media.Record
	// Met is the names of the Targets it meets.
	Met []string
	// Profile is its Work's Profile, whose Targets are those naming it.
	Profile string
}

// library is the files and Targets the World's reads answer from.
type library struct {
	files    []File
	failed   map[files.ID]error
	targets  []targets.Target
	profiles map[string]targets.ProfileID
}

func newLibrary() *library {
	return &library{failed: map[files.ID]error{}, profiles: map[string]targets.ProfileID{}}
}

// File puts f in the library in place of the file with its id, and
// answers its id. Played reads it again.
func (w *World) File(t *testing.T, f File) files.ID {
	t.Helper()
	w.mu.Lock()
	defer w.mu.Unlock()
	if f.ID.IsZero() {
		f.ID = files.ID{UUID: storage.NewUUID()}
	}
	l := w.library
	delete(l.failed, f.ID)
	l.files = slices.DeleteFunc(l.files, func(g File) bool { return g.ID == f.ID })
	l.files = append(l.files, f)
	return f.ID
}

// FailFile makes Played fail for id with a read that isn't the
// database's, as a probe that won't decode does, until File puts it back.
func (w *World) FailFile(t *testing.T, id files.ID) {
	t.Helper()
	w.mu.Lock()
	defer w.mu.Unlock()
	w.library.failed[id] = errors.New("the probe won't decode")
}

// Gone removes the file id from the library, as a delete does.
func (w *World) Gone(t *testing.T, id files.ID) {
	t.Helper()
	w.mu.Lock()
	defer w.mu.Unlock()
	w.library.files = slices.DeleteFunc(w.library.files, func(f File) bool { return f.ID == id })
}

// Target adds spec as the Target name, in each Profile it names, and
// answers its id.
func (w *World) Target(t *testing.T, name string, spec targets.Target) targets.TargetID {
	t.Helper()
	w.mu.Lock()
	defer w.mu.Unlock()
	spec.ID, spec.Name = targets.TargetID{UUID: storage.NewUUID()}, name
	for i, p := range spec.Profiles {
		spec.Profiles[i].ID = w.library.profile(p.Name)
	}
	w.library.targets = append(w.library.targets, spec)
	return spec.ID
}

func (l *library) profile(name string) targets.ProfileID {
	if _, ok := l.profiles[name]; !ok {
		l.profiles[name] = targets.ProfileID{UUID: storage.NewUUID()}
	}
	return l.profiles[name]
}

// Index has server's path index name file at path, as item.
func (w *World) Index(t *testing.T, server link.ServerID, path link.ServerPath, file files.ID, item string) {
	t.Helper()
	err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return db.New(tx).UpsertItem(ctx, db.UpsertItemParams{
			ServerID: server, PathKey: path.Key(), ServerPath: path.String(), FileID: file, ItemKey: item, TopKey: item, SeenAt: storage.Now(),
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Filed is play id's file and snapshot, zero and nil for none.
func (w *World) Filed(t *testing.T, server link.ServerID, id link.PlayID) (files.ID, *playback.Snapshot) {
	t.Helper()
	var rows []db.MediaserversPlay
	err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		rows, err = db.New(r).ListPlays(ctx, server)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range rows {
		if p.ID == id {
			return p.FileID, p.Snapshot
		}
	}
	t.Fatalf("server %s has no play %s", server, id)
	return files.ID{}, nil
}

func (w *World) played(_ context.Context, _ storage.Reader[storage.Library], id files.ID) (mediaservers.PlayedFile, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	l := w.library
	if err := l.failed[id]; err != nil {
		return mediaservers.PlayedFile{}, err
	}
	i := slices.IndexFunc(l.files, func(f File) bool { return f.ID == id })
	if i < 0 {
		return mediaservers.PlayedFile{}, files.ErrNoFile
	}
	f := l.files[i]
	return mediaservers.PlayedFile{Work: f.Work, Item: f.Item, Label: f.Label, Met: l.met(f), Media: f.Media}, nil
}

func (w *World) siblings(_ context.Context, _ storage.Reader[storage.Library], items []works.ItemID) (playback.Siblings, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	l := w.library
	out := playback.Siblings{Files: map[works.ItemID][]playback.Sibling{}, Goals: map[works.WorkID]playback.Goal{}}
	for _, f := range l.files {
		if !slices.Contains(items, f.Item) {
			continue
		}
		out.Files[f.Item] = append(out.Files[f.Item], playback.Sibling{
			File: f.ID, Work: f.Work, Label: f.Label, Kept: f.Kept, Met: l.met(f), Media: f.Media,
		})
		if f.Profile != "" {
			out.Goals[f.Work] = l.goal(f.Profile)
		}
	}
	return out, nil
}

func (w *World) allTargets(context.Context, storage.Reader[storage.Library]) ([]targets.Target, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.library.targets), nil
}

func (l *library) met(f File) []playback.TargetRef {
	var out []playback.TargetRef
	for _, name := range f.Met {
		i := slices.IndexFunc(l.targets, func(t targets.Target) bool { return t.Name == name })
		if i < 0 {
			panic(fmt.Sprintf("file %s meets %q, which no Target is named", f.ID, name))
		}
		out = append(out, playback.TargetRef{ID: l.targets[i].ID.String(), Name: name})
	}
	return out
}

func (l *library) goal(profile string) playback.Goal {
	g := playback.Goal{Profile: playback.TargetRef{ID: l.profile(profile).String(), Name: profile}}
	for _, t := range l.targets {
		if slices.ContainsFunc(t.Profiles, func(p targets.ProfileRef) bool { return p.Name == profile }) {
			g.Targets = append(g.Targets, t.ID)
		}
	}
	return g
}
