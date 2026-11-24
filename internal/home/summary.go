package home

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Summary is what Home shows of the Grabs, the Imports and their Works,
// derived on each read.
type Summary struct {
	grabs.Pipeline
	Recent imports.Recent
	// Works is every Work a Job, Need, Arrival or Move names.
	Works    map[works.WorkID]Work
	Torrents map[downloadclients.ID]downloadclients.Row
	// Moves is the Moves that need the Admin, newest first, at most Rows
	// on Home.
	Moves []moves.Need
}

// Work is a Work as a Home row shows it.
type Work struct {
	works.Row
	// Backdrop is zero when it has none.
	Backdrop artwork.Ref
	// Failed is whether its status is failed, so it never tints Home.
	Failed bool
}

// Read reads Home in one snapshot, Recent's totals counting the Imports
// that landed a file at or after since. Every row's Work is in it, since a
// Grab's and a landing's rows go with their Work.
func (s *Service) Read(ctx context.Context, _ identity.Admin, since time.Time) (Summary, error) {
	stored, err := s.stored(ctx)
	if err != nil {
		return Summary{}, err
	}
	rows := int(Rows.Value(stored))
	var out Summary
	err = s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if out.Pipeline, err = s.cfg.Grabs.Pipeline(ctx, r, rows); err != nil {
			return err
		}
		if out.Recent, err = s.cfg.Imports.Recent(ctx, r, rows, since); err != nil {
			return err
		}
		if out.Moves, err = s.cfg.Moves.NeedsYou(ctx, r, rows); err != nil {
			return err
		}
		if out.Works, err = s.worksOf(ctx, r, out); err != nil {
			return err
		}
		out.Torrents, err = s.torrentsOf(ctx, r, out.Jobs)
		return err
	})
	return out, err
}

// worksOf is each Work sum's rows name.
func (s *Service) worksOf(ctx context.Context, r storage.Reader[file], sum Summary) (map[works.WorkID]Work, error) {
	seen := map[works.WorkID]bool{}
	var ids []works.WorkID
	add := func(id works.WorkID) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, j := range sum.Jobs {
		add(j.WorkID)
	}
	for _, n := range sum.Needs {
		add(n.WorkID)
	}
	for _, a := range sum.Recent.Arrivals {
		add(a.Work)
	}
	for _, n := range sum.Moves {
		add(n.Work)
	}
	if len(ids) == 0 {
		return map[works.WorkID]Work{}, nil
	}
	scope := works.Scope{Works: ids}
	rows, err := s.cfg.Works.RowsIn(ctx, r, scope)
	if err != nil {
		return nil, err
	}
	backdrops, err := s.cfg.Works.BackdropsIn(ctx, r, ids)
	if err != nil {
		return nil, err
	}
	bares := make([]works.Bare, len(rows))
	for i, row := range rows {
		bares[i] = row.Bare
	}
	rolled, err := s.cfg.Files.Rollups(ctx, r, bares, scope.Among(ids))
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]Work, len(rows))
	for i, row := range rows {
		out[row.ID] = Work{Row: row, Backdrop: backdrops[row.ID], Failed: rolled[i].Status == files.Failed}
	}
	return out, nil
}

// torrentsOf is the Torrent of each of jobs that has one.
func (s *Service) torrentsOf(ctx context.Context, r storage.Reader[file], jobs []grabs.View) (map[downloadclients.ID]downloadclients.Row, error) {
	var hashes []bencode.InfoHash
	for _, j := range jobs {
		if j.Torrent != nil {
			hashes = append(hashes, j.Torrent.InfoHash)
		}
	}
	out := map[downloadclients.ID]downloadclients.Row{}
	if len(hashes) == 0 {
		return out, nil
	}
	rows, err := s.cfg.Clients.Rows(ctx, r, downloadclients.Filter{Hashes: hashes})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}
