package imports

import (
	"context"
	"path"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// carried writes a Grab of work's first Item, importing over a download
// of Clip 1 and Clip 2 and holding the world's HD cell, with the carry row
// grabs 00012 leaves for a Grab it found at was, which chose Clip 1.
func (w *world) carried(t *testing.T, work works.Tree, was string) storage.UUID {
	t.Helper()
	grab, torrent, item := storage.NewUUID(), storage.NewUUID(), work.Items[0].ID
	completion := `{"files":[{"Index":0,"Path":"Shelf/Clip 1.mkv","Size":1073741824},{"Index":1,"Path":"Shelf/Clip 2.mkv","Size":1073741824}]}`
	var chosen [4]any
	if was != "probing" {
		chosen = [4]any{0, `{"version":1}`, w.root.ID, "Shelf/Clip 1.mkv"}
	}
	exec(t, w.lib,
		[]any{`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), ?, 0) ON CONFLICT DO NOTHING`, w.node, "Media server"},
		[]any{`INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers, content_type, wanted, added_order, added_at,
			completion, completed_at) VALUES (?, ?, randomblob(20), x'00', 'Shelf', 2147483648, 0, '[]', 'fixture', 'started',
			(SELECT count(*) + 1 FROM downloadclients_torrent), 0, ?, 1)`, torrent, w.node, completion},
		[]any{`INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, release_magnet, stage, torrent_id,
			created_at, updated_at) VALUES (?, ?, ?, 'any', 'search', ?, 'Shelf.1080p.BluRay', 2147483648, 'magnet:?xt=urn:btih:00', 'importing', ?, 0, 0)`,
			grab, work.ID, item, w.node, torrent},
		[]any{`INSERT INTO grabs_grab_item (grab_id, item_id, position) VALUES (?, ?, 0)`, grab, item},
		[]any{`INSERT INTO grabs_target (grab_id, position, item_id, target_id) VALUES (?, 0, ?, ?)`, grab, item, w.hd},
		[]any{`INSERT INTO grabs_carry (grab_id, was, file_index, probe, root_id, path, fills) VALUES (?, ?, ?, ?, ?, ?, json_array(hex(?)))`,
			grab, was, chosen[0], chosen[1], chosen[2], chosen[3], w.hd},
	)
	return grab
}

// TestCarryMakesAnImportOfEachGrabFoundPastItsDownload carries a Grab
// found at each of probing, transferring and importing into an Import
// with its id. Probing chose nothing, so its Import reads the download as
// a new Grab's does. Transferring sends the file it chose and importing
// places it, each leaving the other video out. A second Carry carries
// nothing.
func TestCarryMakesAnImportOfEachGrabFoundPastItsDownload(t *testing.T) {
	w := newWorld(t)
	cfg := w.cfg
	cfg.Grabs = grabs.New(grabs.Config{Library: w.lib})
	im := w.newImports(cfg)
	want := map[string][]string{
		"probing":      {"Clip 1.mkv: pending", "Clip 2.mkv: waiting other_items"},
		"transferring": {"Clip 1.mkv: sending holding", "Clip 2.mkv: left_out sibling_better"},
		"importing":    {"Clip 1.mkv: placing holding", "Clip 2.mkv: left_out sibling_better"},
	}
	byWas := map[string]storage.UUID{}
	for i, was := range []string{"probing", "transferring", "importing"} {
		byWas[was] = w.carried(t, w.add(t, works.Key("s"+string(rune('1'+i))), w.root.ID), was)
	}
	for range 2 {
		if err := im.Carry(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	for was, grab := range byWas {
		imp := w.stored(t, ImportID{UUID: grab})
		var got []string
		for _, f := range imp.Files {
			line := path.Base(f.Path) + ": " + string(f.Outcome)
			if f.Reason != nil {
				line += " " + string(f.Reason.Code)
			}
			if f.Dest != nil && f.Dest.Root == w.root.ID && f.Dest.Holding.String() != "" {
				line += " holding"
			}
			got = append(got, line)
		}
		if imp.Grab.UUID != grab || !slices.Equal(got, want[was]) {
			t.Errorf("the Grab carried at %s became an Import of Grab %s reading %q, want its own reading %q", was, imp.Grab, got, want[was])
		}
	}
	var imports, left int
	err := w.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		if err := r.QueryRowContext(ctx, `SELECT count(*) FROM imports_import`).Scan(&imports); err != nil {
			return err
		}
		return r.QueryRowContext(ctx, `SELECT count(*) FROM grabs_carry`).Scan(&left)
	})
	if err != nil || imports != 3 || left != 0 {
		t.Fatalf("after two Carries the library holds %d Imports and %d carry rows (%v), want 3 and none", imports, left, err)
	}
}
