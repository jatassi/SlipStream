package clip

import (
	"context"
	"embed"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/example/clip/internal/db"
	"github.com/jatassi/slipstream/internal/storage/example/shelf"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("clip", schemaFiles) }

type (
	ID       = db.ClipID
	PublicID = db.ClipPublicID
)

type Clip struct {
	ID        ID
	ShelfID   shelf.ID
	PublicID  PublicID
	Title     string
	CreatedAt storage.Time
	WatchedAt storage.Time
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
