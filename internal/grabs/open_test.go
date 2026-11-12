package grabs_test

import (
	"errors"
	"log/slog"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
)

func TestAFetchingGrabsCellReadsDownloadingAndIsNotOpenToSearch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newAttic(t, scripted{}, []twinSite{{name: "Fixture Site", priority: 25}}, nil, slog.New(slog.DiscardHandler))
		_, cells, _, err := a.cfg.Files.WorkCells(t.Context(), identity.Admin{}, a.work)
		if err != nil || len(cells) != 1 {
			t.Fatalf("The Attic has cells %+v, %v; want one", cells, err)
		}
		grab := storage.NewUUID()
		run(t, a.lib,
			sql(`INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, release_magnet, stage, created_at, updated_at)
				SELECT ?, work_id, id, 'any', 'search', (SELECT id FROM nodes_node), 'The.Attic.2020.1080p.WEB-DL', 1,
				'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567', 'fetching', 1, 1 FROM works_item WHERE work_id = ?`, grab, a.work),
			sql(`INSERT INTO grabs_target (grab_id, position, item_id, target_id) SELECT id, 0, item_id, ? FROM grabs_grab WHERE id = ?`, cells[0].Target.ID, grab),
		)
		_, cells, _, err = a.cfg.Files.WorkCells(t.Context(), identity.Admin{}, a.work)
		if err != nil {
			t.Fatal(err)
		}
		if len(cells) != 1 || cells[0].State != files.Downloading || cells[0].Wanted() || cells[0].Grab != grab {
			t.Fatalf("the cell a fetching Grab holds reads %+v, want downloading from the Grab and not wanted", cells)
		}
		if _, err := a.g.SearchWork(t.Context(), identity.Admin{}, a.work, grabs.Scope{}); !errors.Is(err, grabs.ErrNothingWanted) {
			t.Fatalf("SearchWork = %v, want %v", err, grabs.ErrNothingWanted)
		}
	})
}
