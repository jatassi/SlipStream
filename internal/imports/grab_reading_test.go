package imports

import (
	"context"
	"path"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// numberlessFacet places a file as shelfFacet does, and reads a name
// that is a bare number as having no Item number.
type numberlessFacet struct{ shelfFacet }

func (numberlessFacet) Place(ctx context.Context, r storage.Reader[storage.Library], work works.WorkID) (scanning.Placer, error) {
	place, err := shelfFacet{}.Place(ctx, r, work)
	if err != nil {
		return nil, err
	}
	return func(rel string) scanning.Placement {
		if _, err := strconv.Atoi(strings.TrimSuffix(path.Base(rel), path.Ext(rel))); err == nil {
			return scanning.Unplaced(scanning.Reason{Code: scanning.NoNumber})
		}
		return place(rel)
	}, nil
}

// TestAGrabsFileItsNameCantPlaceWaitsWithItsSentenceAndActions grabs a
// Work's Box and downloads a bare-numbered file and a file naming a clip
// the Work lacks. Each waits with a sentence and the actions that move it
// on: the bare number names the Box the Grab's Items lie in.
func TestAGrabsFileItsNameCantPlaceWaitsWithItsSentenceAndActions(t *testing.T) {
	w := newWorld(t)
	w.cfg.Facets = map[works.TypeID]scanning.Facet{contenttypetest.ID: numberlessFacet{}}
	cfg := w.cfg
	cfg.Grabs = grabs.New(grabs.Config{Library: w.lib})
	im := w.newImports(cfg)
	work := w.add(t, "s1", w.root.ID)
	grab, torrent := storage.NewUUID(), storage.NewUUID()
	completion := `{"files":[{"Index":0,"Path":"Shelf/01.mkv","Size":1073741824},{"Index":1,"Path":"Shelf/Clip 9.mkv","Size":1073741824}]}`
	stmts := [][]any{
		{`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), ?, 0)`, w.node, "Media server"},
		{`INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers, content_type, wanted, added_order, added_at,
			completion, completed_at) VALUES (?, ?, randomblob(20), x'00', 'Shelf', 2147483648, 0, '[]', 'fixture', 'started', 1, 0, ?, 1)`, torrent, w.node, completion},
		{`INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, release_magnet, stage, torrent_id,
			created_at, updated_at) VALUES (?, ?, ?, 'any', 'search', ?, 'Shelf.1080p.BluRay', 2147483648, 'magnet:?xt=urn:btih:00', 'importing', ?, 0, 0)`,
			grab, work.ID, work.Items[0].ID, w.node, torrent},
		{`INSERT INTO grabs_carry (grab_id, was, fills) VALUES (?, 'probing', '[]')`, grab},
	}
	for i, it := range work.Items {
		stmts = append(stmts,
			[]any{`INSERT INTO grabs_grab_item (grab_id, item_id, position) VALUES (?, ?, ?)`, grab, it.ID, i},
			[]any{`INSERT INTO grabs_target (grab_id, position, item_id, target_id) VALUES (?, ?, ?, ?)`, grab, i, it.ID, w.hd},
		)
	}
	exec(t, w.lib, stmts...)
	if err := im.Carry(t.Context()); err != nil {
		t.Fatal(err)
	}

	imp := w.stored(t, ImportID{UUID: grab})
	said, err := im.Words(t.Context(), identity.Admin{}, *imp)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		base, sentence string
		recommended    ActionKind
		also           []ActionKind
	}{
		{"01.mkv", "This file's name has no Thing number. The folder says box.", ActMatch, []ActionKind{ActDecline}},
		{"Clip 9.mkv", "This file's name doesn't name a Thing of Shelf s1.", ActMatch, []ActionKind{ActDecline}},
	} {
		p := positionOf(imp, c.base)
		f := imp.Files[p]
		recommended, also := f.Actions(*imp)
		if f.Outcome != Waiting || said[p].Sentence != c.sentence || recommended != c.recommended || !slices.Equal(also, c.also) {
			t.Errorf("%s %s says %q offering %q then %q, want waiting saying %q offering %q then %q",
				c.base, f.Outcome, said[p].Sentence, recommended, also, c.sentence, c.recommended, c.also)
		}
	}
}
