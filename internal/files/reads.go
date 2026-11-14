package files

import (
	"context"
	"fmt"
	"path"
	"slices"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Open is the files owner over cfg, once every file has its release name
// and its Title: the files the migration that added them left without
// them take them first.
func Open(ctx context.Context, cfg Config) (*Files, error) {
	if cfg.MediaServers == nil {
		cfg.MediaServers = NoMediaServers{}
	}
	f := &Files{cfg: cfg, poke: make(chan struct{}, 1)}
	if err := f.catchUpReads(ctx); err != nil {
		return nil, fmt.Errorf("open files: read release names: %w", err)
	}
	return f, nil
}

// readsPerWrite bounds how many files one of the catch-up's Writes reads.
const readsPerWrite = 500

// catchUpReads gives every file without a release name its Grab's Release
// title, or its own file name when no Grab record holds one, and that
// name's reading by its Content Type. It reads them in one Read and
// writes them in Writes of readsPerWrite that skip a file read since, so
// a repeat finds none.
func (f *Files) catchUpReads(ctx context.Context) error {
	var reads []db.SetFileReadParams
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		reads, err = f.unread(ctx, r)
		return err
	})
	if err != nil {
		return err
	}
	for batch := range slices.Chunk(reads, readsPerWrite) {
		err := f.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			q := db.New(tx)
			for _, read := range batch {
				if err := q.SetFileRead(ctx, read); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// unread reads, in r, the release name and Title of each file without
// them.
func (f *Files) unread(ctx context.Context, r storage.Reader[file]) ([]db.SetFileReadParams, error) {
	rows, err := db.New(r).ListUnreadFiles(ctx)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	summaries, err := works.Summaries(ctx, r)
	if err != nil {
		return nil, err
	}
	types := make(map[works.WorkID]works.TypeID, len(summaries))
	for _, s := range summaries {
		types[s.ID] = s.ContentType
	}
	titles, err := f.cfg.Grabs.ReleaseTitles(ctx, r)
	if err != nil {
		return nil, err
	}
	known, err := f.cfg.Targets.CutsIn(ctx, r)
	if err != nil {
		return nil, err
	}
	out := make([]db.SetFileReadParams, len(rows))
	for i, row := range rows {
		typ := types[row.WorkID]
		names, cuts := f.cfg.Names[typ], known.For(typ)
		out[i] = db.SetFileReadParams{ID: row.ID, ReleaseName: path.Base(row.Path), TitleRead: db.Reading{Read: release.File(row.Path, names, cuts)}}
		if title, ok := titles[row.GrabID]; ok {
			out[i].ReleaseName, out[i].TitleRead.Read = title, release.Title(title, names, cuts)
		}
	}
	return out, nil
}
