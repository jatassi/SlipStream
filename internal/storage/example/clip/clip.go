// Package clip is the test-only Content Type and the owner to copy. To add
// tables: copy this folder, rename the package and the set, change the file
// alias if the tables are Role-local, then edit schema/, queries/ and the
// functions below, and run `go run ./cmd/release sqlc`.
package clip

import (
	"context"
	"embed"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/example/clip/internal/db"
	"github.com/jatassi/slipstream/internal/storage/example/shelf"
)

// file is the one line that picks this owner's database. Role-local state
// (an outbox, resume data) says storage.Local instead.
type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

// Schema is this owner's line in the composition root's list. A Content
// Type carries it on its Content Type value instead.
func Schema() storage.Schema[file] { return storage.NewSchema[file]("clip", schemaFiles) }

type (
	ID       = db.ClipID
	PublicID = db.ClipPublicID
)

// Clip mirrors the generated row field for field, so Clip(row) converts it
// and a schema change that reshapes the row fails to compile here.
type Clip struct {
	ID        ID
	ShelfID   shelf.ID
	PublicID  PublicID
	Title     string
	CreatedAt storage.Time
	WatchedAt storage.Time // zero until watched
}

func Add(ctx context.Context, tx *storage.Tx[file], on shelf.ID, title string) (Clip, error) {
	row, err := db.New(tx).InsertClip(ctx, db.InsertClipParams{
		ShelfID:   on,
		PublicID:  PublicID{UUID: storage.NewUUID()},
		Title:     title,
		CreatedAt: storage.Now(),
	})
	if err != nil {
		return Clip{}, err
	}
	return Clip(row), nil
}

// Get returns storage.ErrNoRows when no clip has id.
func Get(ctx context.Context, r storage.Reader[file], id ID) (Clip, error) {
	row, err := db.New(r).GetClip(ctx, id)
	if err != nil {
		return Clip{}, err
	}
	return Clip(row), nil
}

// OnShelf lists a shelf's clips oldest first.
func OnShelf(ctx context.Context, r storage.Reader[file], on shelf.ID) ([]Clip, error) {
	rows, err := db.New(r).ListClipsOnShelf(ctx, on)
	if err != nil {
		return nil, err
	}
	return clips(rows), nil
}

// Unwatched lists a shelf's clips never marked watched, oldest first.
func Unwatched(ctx context.Context, r storage.Reader[file], on shelf.ID) ([]Clip, error) {
	rows, err := db.New(r).ListUnwatchedClips(ctx, on)
	if err != nil {
		return nil, err
	}
	return clips(rows), nil
}

func MarkWatched(ctx context.Context, tx *storage.Tx[file], id ID, at storage.Time) error {
	return db.New(tx).MarkClipWatched(ctx, db.MarkClipWatchedParams{WatchedAt: at, ID: id})
}

func clips(rows []db.Clip) []Clip {
	out := make([]Clip, len(rows))
	for i, row := range rows {
		out[i] = Clip(row)
	}
	return out
}
