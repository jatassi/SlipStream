package subtitles

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/works"
)

// Audit is the records invariant over every library file in r: its row's
// fact columns and root are what factsOf reads of its Subject now, it has
// a want row for exactly its searchable wants, and each row's unmet is
// whether no Sidecar meets it.
func Audit(ctx context.Context, r storage.Reader[storage.Library], s *Subtitles) error {
	var unmet int64
	err := r.QueryRowContext(ctx, `SELECT count(*) FROM subtitles_want w WHERE w.unmet <> NOT EXISTS (
		SELECT 1 FROM subtitles_sidecar c WHERE c.file_id = w.file_id AND c.language = w.language AND c.forced = w.forced)`).Scan(&unmet)
	if err != nil {
		return fmt.Errorf("count unmet: %w", err)
	}
	if unmet != 0 {
		return fmt.Errorf("%d want rows' unmet disagrees with their Sidecars", unmet)
	}
	var after files.ID
	for {
		subjects, err := s.cfg.Files.Subjects(ctx, r, after, 500, works.Every[works.WorkID](nil))
		if err != nil || len(subjects) == 0 {
			return err
		}
		c, err := s.read(ctx, r, subjects)
		if err != nil {
			return err
		}
		ids := make([]files.ID, len(subjects))
		for i, sub := range subjects {
			ids[i] = sub.ID
		}
		rows, err := db.New(r).ListFilesIn(ctx, ids)
		if err != nil {
			return err
		}
		for i, sub := range subjects {
			if err := audit(c, sub, rows, s.cfg.Types); err != nil {
				return fmt.Errorf("file %d (%s): %w", i, sub.Path, err)
			}
		}
		after = subjects[len(subjects)-1].ID
	}
}

func audit(c chunk, sub files.Subject, rows []db.SubtitlesFile, types map[works.TypeID]Facet) error {
	i := slices.IndexFunc(rows, func(r db.SubtitlesFile) bool { return r.FileID == sub.ID })
	if i < 0 {
		return errors.New("no subtitles_file row")
	}
	row := rows[i]
	var ignored []ignore
	for _, ig := range c.ignored[sub.ID] {
		ignored = append(ignored, ignore{Ordinal: int(ig.StreamIndex), Codec: ig.Codec, Language: release.Language(ig.Language), Forced: ig.Forced})
	}
	f := factsOf(sub, kindOfType(types[sub.Type]), ignored)
	got := fmt.Sprint(kindOf(row.Kind), row.Wantless, row.Audio, row.Embedded, row.RootID)
	if want := fmt.Sprint(f.Kind, f.Wantless, f.audioColumn(), f.embeddedColumn(), sub.Root); got != want {
		return fmt.Errorf("facts read %s, want %s", got, want)
	}
	scs := c.sidecars[sub.ID]
	var want, have []string
	for _, w := range wantsOf(f, c.langs[sub.Profile], func(w Want) bool { return sidecarOfWant(scs, w) != nil }) {
		if w.searchable() {
			want = append(want, w.Want().String())
		}
	}
	for _, w := range c.wants[sub.ID] {
		have = append(have, Want{Language: mustParse(w.Language), Forced: w.Forced}.String())
	}
	slices.Sort(want)
	slices.Sort(have)
	if !slices.Equal(want, have) {
		return fmt.Errorf("want rows %v, want %v", have, want)
	}
	return nil
}

func mustParse(s string) language.Code {
	l, err := language.Parse(s)
	if err != nil {
		panic(err)
	}
	return l
}
