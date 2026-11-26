package node

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/playback"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// german is the fixture with every Work's original language German.
type german struct{ *contenttypetest.Fixture }

func (german) Originals(_ context.Context, _ storage.Reader[storage.Library], ws works.Among[works.WorkID]) (map[works.WorkID]release.Fact[release.Language], error) {
	out := map[works.WorkID]release.Fact[release.Language]{}
	for _, w := range ws.IDs() {
		out[w] = release.Known(release.Language("de"))
	}
	return out, nil
}

type nop[T any] struct{}

func (nop[T]) Publish(context.Context, T) {}

// items is more Items than SQLite binds in one statement.
const items = 33000

func TestPlaybackReadsOverTheLibrary(t *testing.T) {
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(),
		downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema(), subtitles.Schema(), naming.Schema())
	fx := contenttypetest.New()
	tree, err := works.Open(ctx, works.Config{
		ItemsAdded: works.NoItemsAdded, Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx},
		Source: fx.Source(), Artwork: fx, Changed: nop[works.Changed]{}, TypesChanged: nop[works.ContentTypesChanged]{},
	})
	must(t, err)
	goals, err := targets.Open(ctx, targets.Config{
		Library: lib, Works: tree, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: german{fx}}, Changed: nop[targets.Changed]{},
		Subtitled: func(works.TypeID) bool { return true },
	})
	must(t, err)
	node := role.NodeID{UUID: storage.NewUUID()}
	must(t, lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return nodestest.Insert(ctx, tx, node, "Node A")
	}))
	root := rootstest.Root(t, lib, node, contenttypetest.ID, t.TempDir())

	made := map[string]bool{}
	target := func(name string, q release.Quality, f targets.Filters) targets.Target {
		made[name] = true
		tg, err := goals.CreateTarget(ctx, identity.Admin{}, targets.Spec{Name: name, Tiers: [][]release.Quality{{q}}, Cutoff: q, Filters: f, Scores: targets.Scores{Increment: 1}})
		must(t, err)
		return tg
	}
	hd := target("HD", release.WEBDL1080p, targets.Filters{})
	uhd := target("UHD", release.WEBDL2160p, targets.Filters{})
	target("Any 1080p", release.WEBDL1080p, targets.Filters{})
	target("Original audio", release.WEBDL1080p, targets.Filters{Language: targets.Filter[release.Language]{Mode: targets.Only, Values: []release.Language{release.Original}}})
	profile, err := goals.CreateProfile(ctx, identity.Admin{}, targets.ProfileSpec{
		Name: "Standard", ContentType: contenttypetest.ID, Entries: []targets.Entry{{Target: uhd.ID, Sourcing: targets.Download}, {Target: hd.ID, Sourcing: targets.Download}},
	})
	must(t, err)

	entries := make([]works.Entry, items)
	for i := range entries {
		entries[i] = works.Entry{Key: works.Key(fmt.Sprint("i", i)), Label: fmt.Sprint("Part ", i)}
	}
	fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{{Key: "l1", Label: "Parts", Children: entries}}}})
	work, _, err := tree.Add(ctx, identity.Admin{}, contenttypetest.ID, "w1", works.Adding{
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], s works.Summary) error {
			if err := targets.Assign(ctx, tx, s, &profile.ID); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, s, nil)
		},
	})
	must(t, err)
	first, second, last := work.Items[0].ID, work.Items[1].ID, work.Items[items-1].ID

	probe := media.Record{Version: media.Version, Container: "matroska,webm", Video: media.Video{Codec: "hevc", Width: 1920, Height: 1080, HDR: []release.HDR{"SDR"}},
		Audio: []media.Audio{{Codec: "eac3", Channels: 6, Language: "fr", Default: true}}}
	add := func(tx *storage.Tx[storage.Library], path string, its []works.ItemID, label string, extra int64, fills ...targets.TargetID) files.ID {
		f, err := files.Add(ctx, tx, files.File{
			WorkID: work.ID, Items: its, Cut: release.Standard, RootID: root.ID, Path: path, Size: 1024, Media: probe, Quality: release.WEBDL1080p, Label: label, Extra: extra,
		}, fills)
		must(t, err)
		return f.ID
	}
	var played, kept, moving, gone, far files.ID
	must(t, lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		played = add(tx, "Attic/played.mkv", []works.ItemID{first, second}, "", 0, hd.ID)
		kept = add(tx, "Attic/kept.mkv", []works.ItemID{first}, "HD", 2)
		moving = add(tx, "Attic/moving.mkv", []works.ItemID{first}, "", 0)
		gone = add(tx, "Attic/gone.mkv", []works.ItemID{first}, "", 0)
		far = add(tx, "Attic/far.mkv", []works.ItemID{last}, "", 0)
		for _, s := range []struct {
			query string
			args  []any
		}{
			{"INSERT INTO files_move (file_id, sits, origin) VALUES (?, ?, ?)", []any{moving, "Attic/elsewhere.mkv", "Attic/moving.mkv"}},
			{"UPDATE files_file SET gone_at = ? WHERE id = ?", []any{storage.Now(), gone}},
			{"INSERT INTO subtitles_file (file_id, work_id, root_id, map_since) VALUES (?, ?, ?, ?)", []any{kept, work.ID, root.ID, storage.Now()}},
			{"INSERT INTO subtitles_sidecar (id, file_id, language, forced, hi, tail, source, final, timed, sha256, size, cues, added_at) VALUES (?, ?, 'en', 0, 0, 'en.srt', 'found', 1, 0, ?, 1, 1, ?)",
				[]any{storage.NewUUID(), kept, make([]byte, 32), storage.Now()}},
		} {
			if _, err := tx.ExecContext(ctx, s.query, s.args...); err != nil {
				return err
			}
		}
		return nil
	}))

	reads := playbackReads{&goals}
	all := make([]works.ItemID, items)
	for i, it := range work.Items {
		all[i] = it.ID
	}
	var got playbackGot
	must(t, lib.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		p, err := reads.played(ctx, r, played)
		if err != nil {
			return err
		}
		k, err := reads.played(ctx, r, kept)
		if err != nil {
			return err
		}
		_, got.goneErr = reads.played(ctx, r, gone)
		_, got.unknownErr = reads.played(ctx, r, files.ID{UUID: storage.NewUUID()})
		s, err := reads.siblings(ctx, r, all)
		got.played, got.kept, got.siblings = p, k, s
		return err
	}))

	names := map[files.ID]string{played: "played", kept: "kept", moving: "moving", gone: "gone", far: "far"}
	for _, c := range []struct{ name, got, want string }{
		{"played Work", got.played.Work.String(), work.ID.String()},
		{"played Item, its first by position", got.played.Item.String(), first.String()},
		{"played label", got.played.Label, "HD"},
		{"played meets", refs(got.played.Met, made), "Any 1080p, HD"},
		{"played probe", got.played.Media.Container, "matroska,webm"},
		{"Kept label", got.kept.Label, "HD (extra 2)"},
		{"gone file", strconv.FormatBool(errors.Is(got.goneErr, files.ErrNoFile)), "true"},
		{"unknown file", strconv.FormatBool(errors.Is(got.unknownErr, files.ErrNoFile)), "true"},
		{"first Item's siblings", siblingsOf(got.siblings.Of(first), names), "kept Kept HD (extra 2) [en SRT]; moving; played"},
		{"second Item's siblings", siblingsOf(got.siblings.Of(second), names), "played"},
		{"last Item's siblings", siblingsOf(got.siblings.Of(last), names), "far"},
		{"sibling meets", refs(got.siblings.Of(second)[0].Met, made), "Any 1080p, HD"},
		{"Profile", got.siblings.Goals[work.ID].Profile.Name, "Standard"},
		{"Profile's Targets", fmt.Sprint(got.siblings.Goals[work.ID].Targets), fmt.Sprint([]targets.TargetID{uhd.ID, hd.ID})},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
}

type playbackGot struct {
	played, kept        mediaservers.PlayedFile
	siblings            playback.Siblings
	goneErr, unknownErr error
}

// refs names the Targets of rs among made, leaving out the defaults the
// Targets open with.
func refs(rs []playback.TargetRef, made map[string]bool) string {
	var out []string
	for _, r := range rs {
		if made[r.Name] {
			out = append(out, r.Name)
		}
	}
	slices.Sort(out)
	return strings.Join(out, ", ")
}

func siblingsOf(sibs []playback.Sibling, names map[files.ID]string) string {
	var out []string
	for _, s := range sibs {
		var line strings.Builder
		line.WriteString(names[s.File])
		if s.Kept {
			line.WriteString(" Kept " + s.Label)
		}
		for _, sc := range s.Sidecars {
			fmt.Fprintf(&line, " [%s %s]", sc.Language, sc.Codec)
		}
		out = append(out, line.String())
	}
	slices.Sort(out)
	return strings.Join(out, "; ")
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
