package files_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// hearing is a library's files.Subtitles. It records each call as a line
// in the Write that makes it, so a Write that fails records none.
// Languages answers impact.
type hearing struct {
	impact files.SubtitlesImpact
}

func (l *library) hear(t *testing.T) *hearing {
	t.Helper()
	l.exec(t, `CREATE TABLE test_heard (seq INTEGER PRIMARY KEY, line TEXT NOT NULL)`)
	h := &hearing{}
	l.subtitles = h
	return h
}

func note(ctx context.Context, tx *storage.Tx[storage.Library], line string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO test_heard (line) VALUES (?)`, line)
	return err
}

func (*hearing) Changed(ctx context.Context, tx *storage.Tx[storage.Library], cs []files.PathChange) error {
	for _, line := range said(cs) {
		if err := note(ctx, tx, line); err != nil {
			return err
		}
	}
	return nil
}

func (*hearing) Leaving(ctx context.Context, tx *storage.Tx[storage.Library], d files.File, by files.ID) error {
	var held bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM files_file WHERE id = ?)`, d.ID).Scan(&held); err != nil {
		return fmt.Errorf("read file %s's row: %w", d.ID, err)
	}
	return note(ctx, tx, fmt.Sprintf("leaving %s for %s, row held %v", d.ID, by, held))
}

func (h *hearing) Languages(ctx context.Context, tx *storage.Tx[storage.Library], p targets.ProfileID, was, now []targets.WantedLanguage, preview bool) (files.SubtitlesImpact, error) {
	return h.impact, note(ctx, tx, fmt.Sprintf("languages %s %v -> %v preview %v", p, tags(was), tags(now), preview))
}

func tags(ls []targets.WantedLanguage) []string {
	out := []string{}
	for _, l := range ls {
		out = append(out, l.Language.String())
	}
	return out
}

// heard is every line l's files owners told subtitles, in order, each id
// replaced by its name in names.
func (l *library) heard(t *testing.T, names map[files.ID]string) []string {
	t.Helper()
	rows, err := l.lib.QueryContext(t.Context(), `SELECT line FROM test_heard ORDER BY seq`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		line = strings.ReplaceAll(line, files.ID{}.String(), "nothing")
		for id, name := range names {
			line = strings.ReplaceAll(line, id.String(), name)
		}
		out = append(out, line)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// addedAt is when the full Wanted Language of tag was added to its
// Profile.
func (l *library) addedAt(t *testing.T, tag string) int64 {
	t.Helper()
	var at int64
	if err := l.lib.QueryRowContext(t.Context(), `SELECT added_at FROM targets_profile_language WHERE language = ? AND forced = 0`, tag).Scan(&at); err != nil {
		t.Fatal(err)
	}
	return at
}

func lang(t *testing.T, s string) language.Code {
	t.Helper()
	c, err := language.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestSubtitlesHearInEachWriteWhatTheMediaServersHear(t *testing.T) {
	l := open(t)
	l.hear(t)
	best := l.target(t, "Best", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "Best", best)
	tree := l.work(t, "w1", true, &p)
	svc := l.files(files.NoGrabs{})
	name := l.judge(t, svc, tree.ID, holdingOf(tree, 1, files.HoldingOf(storage.NewUUID(), ".mkv"), 5, media.Record{}, release.WEBDL1080p)).Path
	l.put(t, name, "found")
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := l.land(ctx, tx, svc, tree.ID, holdingOf(tree, 1, name, 5, media.Record{Video: media.Video{Width: 1920, Height: 1080}}, release.WEBDL1080p))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	web := l.download(t, svc, tree, 0, release.WEBDL1080p, "webdl")
	l.settle(t, svc)
	l.download(t, svc, tree, 0, release.Bluray1080p, "bluray")
	l.settle(t, svc)
	l.renames(t, map[string]string{name: "Moved/" + name})
	l.settle(t, svc)

	told := said(l.told(t))
	want := []string{
		"added " + name, "added " + web.Path, "removed " + web.Path + " replaced", "added " + web.Path, "moved " + name + " -> Moved/" + name,
	}
	if !slices.Equal(told, want) {
		t.Fatalf("the Media Servers heard %v, want %v", told, want)
	}
	var changed []string
	for _, line := range l.heard(t, nil) {
		if !strings.HasPrefix(line, "leaving ") {
			changed = append(changed, line)
		}
	}
	if !slices.Equal(changed, told) {
		t.Fatalf("subtitles heard %v, want what the Media Servers heard, %v", changed, told)
	}
}

func TestADoomedFileLeavesForItsReplacerBeforeItsRowGoes(t *testing.T) {
	l := open(t)
	l.hear(t)
	best := l.target(t, "Best", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "Best", best)
	tree := l.work(t, "w1", true, &p)
	svc := l.files(files.NoGrabs{})
	scarf := l.download(t, svc, tree, 0, release.WEBDL1080p, "scarf")
	mittens := l.download(t, svc, tree, 1, release.WEBDL1080p, "mittens")
	l.settle(t, svc)
	l.exec(t, `DELETE FROM test_heard`)

	holding := files.HoldingOf(storage.NewUUID(), ".mkv")
	l.put(t, holding, "pair")
	var pair files.Landed
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		pair, err = svc.Import(ctx, tx, files.Landing{Batch: files.Batch{Work: tree.ID, Root: l.root.ID, Candidates: []files.Candidate{{
			Items: []works.ItemID{tree.Items[0].ID, tree.Items[1].ID}, Cut: release.Standard, Holding: holding, Size: 4,
			Media: media.Record{Video: media.Video{Width: 1920, Height: 1080}}, Quality: release.Bluray1080p,
		}}}, PlacedBy: "copy"})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	names := map[files.ID]string{scarf.ID: "scarf", mittens.ID: "mittens", pair.File.ID: "pair"}
	got := l.heard(t, names)
	slices.Sort(got)
	if want := []string{"leaving mittens for pair, row held true", "leaving scarf for pair, row held true"}; !slices.Equal(got, want) {
		t.Fatalf("an import replacing two files told subtitles %v, want each leaving for the one replacer with its row still there", got)
	}

	l.settle(t, svc)
	l.exec(t, `DELETE FROM test_heard`)
	err = l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return svc.DoomIn(ctx, tx, []works.WorkID{tree.ID})
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := l.heard(t, names), []string{"leaving pair for nothing, row held true"}; !slices.Equal(got, want) {
		t.Fatalf("deleting the Work's files told subtitles %v, want %v", got, want)
	}
}

func TestALanguagesOrNameOnlySaveRefillsNothingAndQueuesNoSearch(t *testing.T) {
	l := open(t)
	queue := record(t, l.lib)
	l.queue = queue
	l.hear(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "Both", hd)
	tree := l.work(t, "w1", true, &p)
	svc := l.files(files.NoGrabs{})
	l.download(t, svc, tree, 0, release.WEBDL1080p, "webdl")
	l.settle(t, svc)
	l.exec(t, `DELETE FROM test_heard`)
	english := targets.WantedLanguage{Language: lang(t, "en"), HI: targets.Allow, Audio: targets.NoAudio}

	save := holding(p, hd)
	save.Spec.Languages = []targets.WantedLanguage{english}
	if i := l.change(t, svc, save, deleting); i.Works != 0 || i.NewlyWanted != 0 {
		t.Fatalf("adding a language had Impact %+v, want no Work touched", i)
	}
	added := l.addedAt(t, "en")
	renamed := save
	renamed.Spec.Name = "Everything"
	if i := l.change(t, svc, renamed, deleting); i.Works != 0 {
		t.Fatalf("renaming the Profile had Impact %+v, want no Work touched", i)
	}
	if got := queue.calls(t); len(got) != 0 {
		t.Fatalf("the saves queued %q, want no release search", got)
	}
	for _, e := range l.published.events[len(l.published.events)-2:] {
		if len(e.WorkIDs) != 0 {
			t.Fatalf("a save published targets.changed naming %v, want no Work", e.WorkIDs)
		}
	}
	if got, want := l.heard(t, nil), []string{fmt.Sprintf("languages %s [] -> [en] preview false", p)}; !slices.Equal(got, want) {
		t.Fatalf("subtitles heard %v, want only the languages change, once: the rename keeps them", got)
	}
	if at := l.addedAt(t, "en"); at != added {
		t.Fatalf("English was added at %d, then at %d after a save that kept it, want it kept", added, at)
	}
	got, err := l.targets.GetProfile(t.Context(), admin, p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Everything" || !slices.Equal(got.Languages, []targets.WantedLanguage{english}) {
		t.Fatalf("the Profile reads %q with %+v, want Everything with English", got.Name, got.Languages)
	}
}

func TestASubtitleDeleteCountJoinsTheConfirmRound(t *testing.T) {
	l := open(t)
	h := l.hear(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "Both", hd)
	svc := l.files(files.NoGrabs{})
	french := targets.WantedLanguage{Language: lang(t, "fr"), HI: targets.Avoid, Audio: targets.Always}
	add := holding(p, hd)
	add.Spec.Languages = []targets.WantedLanguage{french}
	l.change(t, svc, add, deleting)
	drop := holding(p, hd)

	h.impact = files.SubtitlesImpact{Dropped: []files.DroppedLanguage{{Language: french.Language, Deleted: 88}}}
	preview, err := svc.Change(t.Context(), admin, drop, files.Mode{Unwanted: files.Delete, Preview: true})
	if err != nil || preview.Subtitles.Deleted() != 88 {
		t.Fatalf("the preview answered %+v, %v, want 88 subtitles deleted", preview.Subtitles, err)
	}
	h.impact.Dropped[0].Deleted = 90
	_, err = svc.Change(t.Context(), admin, drop, files.Mode{Unwanted: files.Delete, Confirmed: new(0), ConfirmedSubtitles: new(88)})
	changed, ok := errors.AsType[*files.ImpactChangedError](err)
	if !ok || !changed.Subtitles || changed.Files || changed.ConfirmedSubtitles != 88 || changed.Impact.Subtitles.Deleted() != 90 {
		t.Fatalf("a save deleting 90 subtitles after 88 were confirmed answered %v, want impact_changed naming subtitles", err)
	}
	if ls, err := l.targets.GetProfile(t.Context(), admin, p); err != nil || len(ls.Languages) != 1 {
		t.Fatalf("after the refused save the Profile lists %+v, %v, want French still", ls.Languages, err)
	}
	if _, err := svc.Change(t.Context(), admin, drop, files.Mode{Unwanted: files.Delete, Confirmed: new(0), ConfirmedSubtitles: new(90)}); err != nil {
		t.Fatalf("a save confirming its 90 subtitles: %v", err)
	}
	if got, err := l.targets.GetProfile(t.Context(), admin, p); err != nil || len(got.Languages) != 0 {
		t.Fatalf("after the save the Profile lists %+v, %v, want none", got.Languages, err)
	}
	want := []string{
		fmt.Sprintf("languages %s [] -> [fr] preview false", p),
		fmt.Sprintf("languages %s [fr] -> [] preview false", p),
	}
	if got := l.heard(t, nil); !slices.Equal(got, want) {
		t.Fatalf("subtitles kept %v, want only the saves that committed: a preview and a refused save roll back", got)
	}
}

func TestSubjectsPageByIDWithEachFilesFacts(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	other := l.work(t, "w2", true, nil)
	ids := l.stock(t, tree, "w1/a.mkv", "w1/b.mkv", "w1/c.mkv", "w1/d.mkv")
	l.stock(t, other, "w2/e.mkv")
	l.exec(t, `UPDATE files_file SET extra = 1 WHERE path = 'w1/b.mkv'`)
	l.exec(t, `UPDATE files_file SET gone_at = 1 WHERE path = 'w1/c.mkv'`)
	l.exec(t, `UPDATE files_file SET probe = ? WHERE path = 'w1/a.mkv'`,
		`{"version":1,"audio":[{"language":"fr"},{"language":"en","default":true}],"subtitles":[{"codec":"subrip","language":"en","forced":true}]}`)
	l.renames(t, map[string]string{"w1/d.mkv": "w1/z.mkv"})
	names := map[files.ID]string{}
	for p, id := range ids {
		names[id] = p
	}
	svc := l.files(files.NoGrabs{})

	var pages [][]string
	var after files.ID
	for {
		var page []files.Subject
		err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			var err error
			page, err = svc.Subjects(ctx, r, after, 3, works.Only([]works.WorkID{tree.ID}))
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		var lines []string
		for _, s := range page {
			if s.Work != tree.ID || s.Type != "fixture" || s.Root != l.root.ID || len(s.Items) != 1 {
				t.Fatalf("subject %+v, want the fixture Work's at its root with its Item", s)
			}
			lines = append(lines, fmt.Sprintf("%s at %s kept=%v gone=%v atName=%v audio=%v default=%d text=%d", names[s.ID], s.Path, s.Kept, s.Gone, s.AtName, s.Audio, s.DefaultAudio, len(s.Text)))
		}
		pages = append(pages, lines)
		after = page[len(page)-1].ID
	}
	if len(pages) != 2 || len(pages[0]) != 3 || len(pages[1]) != 1 {
		t.Fatalf("pages of 3 = %v, want 3 then 1 of the Work's 4 files", pages)
	}
	got := slices.Concat(pages...)
	want := []string{
		"w1/a.mkv at w1/a.mkv kept=false gone=false atName=true audio=[fr en] default=2 text=1",
		"w1/b.mkv at w1/b.mkv kept=true gone=false atName=true audio=[] default=0 text=0",
		"w1/c.mkv at w1/c.mkv kept=false gone=true atName=true audio=[] default=0 text=0",
		"w1/d.mkv at w1/z.mkv kept=false gone=false atName=false audio=[] default=0 text=0",
	}
	sorted := slices.Clone(got)
	slices.Sort(sorted)
	if !slices.Equal(sorted, want) {
		t.Fatalf("subjects =\n%s\nwant\n%s", strings.Join(sorted, "\n"), strings.Join(want, "\n"))
	}
	var all []files.Subject
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		all, err = svc.Subjects(ctx, r, files.ID{}, 10, works.Every([]works.WorkID{tree.ID, other.ID}))
		return err
	})
	if err != nil || len(all) != 5 {
		t.Fatalf("every Work's subjects = %d, %v, want the 5 files of both", len(all), err)
	}
	var order []files.ID
	for _, line := range got {
		order = append(order, ids[strings.Fields(line)[0]])
	}
	if !slices.IsSortedFunc(order, func(a, b files.ID) int { return a.Compare(b.UUID) }) {
		t.Fatalf("subjects came in %v, want id order", got)
	}
}
