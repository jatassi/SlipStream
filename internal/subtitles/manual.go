package subtitles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// WorkView is every library file of one Work and what each want of it
// reads as, from one snapshot.
type WorkView struct{ Files []FileView }

// FileView is one library file's subtitles. Season and Episodes are set
// for an episode file with Aired numbers.
type FileView struct {
	File       files.ID
	Path       string
	Season     int
	Episodes   []int
	Wants      []WantView
	Embedded   []Track
	Companions []Extra
}

// WantView is one want in Profile order, its State and its last search.
type WantView struct {
	Wanted targets.WantedLanguage
	State  State
	Last   *LastSearch
}

// LastSearch is what the want's last search found.
type LastSearch struct {
	At        time.Time
	Found     int
	BestMatch *int
}

// Track is one of the media record's subtitle tracks, by its ordinal.
// Meets is the want it meets, nil when it meets none.
type Track struct {
	Ordinal  int
	Codec    string
	Language release.Language
	Forced   bool
	Image    bool
	Ignored  bool
	Meets    *Want
}

// WorkSubtitles is w's files' subtitles in one Read.
func (s *Subtitles) WorkSubtitles(ctx context.Context, _ identity.Admin, w works.WorkID) (WorkView, error) {
	var out WorkView
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var after files.ID
		for {
			subjects, err := s.cfg.Files.Subjects(ctx, r, after, s.cfg.Timings.ReconcileChunk, works.Only([]works.WorkID{w}))
			if err != nil || len(subjects) == 0 {
				return err
			}
			views, err := s.fileViews(ctx, r, subjects)
			if err != nil {
				return err
			}
			out.Files = append(out.Files, views...)
			after = subjects[len(subjects)-1].ID
		}
	})
	return out, err
}

func (s *Subtitles) fileViews(ctx context.Context, r storage.Reader[storage.Library], subjects []files.Subject) ([]FileView, error) {
	c, err := s.read(ctx, r, subjects)
	if err != nil {
		return nil, err
	}
	ids := make([]files.ID, len(subjects))
	for i, sub := range subjects {
		ids[i] = sub.ID
	}
	rows, err := db.New(r).ListFilesIn(ctx, ids)
	if err != nil {
		return nil, err
	}
	worlds := map[subtitling.Kind]world{}
	out := make([]FileView, 0, len(subjects))
	for _, sub := range subjects {
		i := slices.IndexFunc(rows, func(row db.SubtitlesFile) bool { return row.FileID == sub.ID })
		if i < 0 {
			out = append(out, FileView{File: sub.ID, Path: sub.Path.String(), Embedded: tracksOf(sub, nil, facts{})})
			continue
		}
		v, err := s.fileView(ctx, r, c, sub, &rows[i], worlds)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *Subtitles) fileView(ctx context.Context, r storage.Reader[storage.Library], c chunk, sub files.Subject, row *db.SubtitlesFile, worlds map[subtitling.Kind]world) (FileView, error) {
	ignored := ignoresOf(c.ignored[sub.ID])
	facet := s.cfg.Types[sub.Type]
	fa := factsOf(sub, kindOfType(facet), ignored)
	v := FileView{File: sub.ID, Path: sub.Path.String(), Embedded: tracksOf(sub, ignored, fa)}
	if row.Companions != nil {
		if err := json.Unmarshal([]byte(*row.Companions), &v.Companions); err != nil {
			return FileView{}, fmt.Errorf("file %s's companions: %w", sub.ID, err)
		}
	}
	if fa.Kind == 0 {
		return v, nil
	}
	wd, ok := worlds[fa.Kind]
	if !ok {
		var err error
		if wd, err = s.worldOf(ctx, r, fa.Kind); err != nil {
			return FileView{}, err
		}
		worlds[fa.Kind] = wd
	}
	subject, err := facet.SubtitleSubject(ctx, r, sub.Work, sub.Items)
	wd.noAired = errors.Is(err, ErrNoAiredNumber)
	if err != nil && !wd.noAired {
		return FileView{}, err
	}
	v.Season, v.Episodes = subject.Season, subject.Episodes
	scs := c.sidecars[sub.ID]
	for _, w := range wantsOf(fa, c.langs[sub.Profile], func(w Want) bool { return sidecarOfWant(scs, w) != nil }) {
		wr := wantRow(c.wants[sub.ID], w.Want())
		v.Wants = append(v.Wants, WantView{Wanted: w.WantedLanguage, State: stateOf(w, wr, row, sidecarOfWant(scs, w.Want()), wd), Last: lastOf(wr)})
	}
	return v, nil
}

func wantRow(rows []db.SubtitlesWant, w Want) *db.SubtitlesWant {
	for i := range rows {
		if rows[i].Language == w.Language.String() && rows[i].Forced == w.Forced {
			return &rows[i]
		}
	}
	return nil
}

func lastOf(wr *db.SubtitlesWant) *LastSearch {
	if wr == nil || wr.LastSearchedAt.IsZero() {
		return nil
	}
	last := &LastSearch{At: wr.LastSearchedAt.Time(), Found: int(deref(wr.LastFound))}
	if wr.BestMatch != nil {
		last.BestMatch = new(int(*wr.BestMatch))
	}
	return last
}

func ignoresOf(rows []db.SubtitlesTrackIgnored) []ignore {
	out := make([]ignore, 0, len(rows))
	for _, ig := range rows {
		out = append(out, ignore{Ordinal: int(ig.StreamIndex), Codec: ig.Codec, Language: release.Language(ig.Language), Forced: ig.Forced})
	}
	return out
}

func tracksOf(sub files.Subject, ignored []ignore, fa facts) []Track {
	out := make([]Track, len(sub.Text))
	for i, t := range sub.Text {
		out[i] = Track{Ordinal: i, Codec: t.Codec, Language: t.Language, Forced: t.Forced, Image: t.Image,
			Ignored: slices.ContainsFunc(ignored, func(ig ignore) bool { return ig.Ordinal == i && ig.holds(sub.Text) })}
		for w, at := range fa.Embedded {
			if at == i {
				out[i].Meets = &w
			}
		}
	}
	return out
}

// LibraryMark is how a Work's subtitles read in the Library list.
type LibraryMark uint8

const (
	// MarkMissing is a Work with an unmet want.
	MarkMissing LibraryMark = iota + 1
	// MarkNonePassed is a Work with an unmet want whose last search found
	// subtitles and none matched the timing.
	MarkNonePassed
)

// LibraryRows is the mark of every Work with an unmet want, in r.
func (s *Subtitles) LibraryRows(ctx context.Context, r storage.Reader[storage.Library]) (map[works.WorkID]LibraryMark, error) {
	rows, err := db.New(r).ListLibraryMarks(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]LibraryMark, len(rows))
	for _, row := range rows {
		out[row.WorkID] = MarkMissing
		if row.NonePassed != 0 {
			out[row.WorkID] = MarkNonePassed
		}
	}
	return out, nil
}

// ErrNoTrack is a track ordinal the file's media record has no text track
// at.
var ErrNoTrack = errors.New("the file has no text subtitle track at that position")

// IgnoreTrack sets whether the text track at ordinal of f counts toward
// its wants, by the track's ordinal and fingerprint, and rewrites f's
// facts and want rows in the same Write.
func (s *Subtitles) IgnoreTrack(ctx context.Context, _ identity.Admin, f files.ID, ordinal int, ignored bool) error {
	var work works.WorkID
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		subjects, err := s.cfg.Files.SubjectsOf(ctx, tx, []files.ID{f})
		if err != nil {
			return err
		}
		if len(subjects) == 0 {
			return fmt.Errorf("file %s: %w", f, files.ErrNoFile)
		}
		sub := subjects[0]
		if ordinal < 0 || ordinal >= len(sub.Text) || sub.Text[ordinal].Image {
			return ErrNoTrack
		}
		work = sub.Work
		q := db.New(tx)
		if !ignored {
			err = q.DeleteIgnored(ctx, db.DeleteIgnoredParams{FileID: f, StreamIndex: int64(ordinal)})
		} else {
			t := sub.Text[ordinal]
			err = q.PutIgnored(ctx, db.PutIgnoredParams{FileID: f, StreamIndex: int64(ordinal), Codec: t.Codec, Language: string(t.Language), Forced: t.Forced})
		}
		if err != nil {
			return err
		}
		return s.reconcileIn(ctx, tx, []files.ID{f})
	})
	if err != nil {
		return err
	}
	s.cfg.Changed.Publish(ctx, changedOf([]works.WorkID{work}, []files.ID{f}))
	return nil
}

// TooLargeError is an upload over Timings.UploadCap. Words says the cap.
type TooLargeError struct{ Cap int }

func (e *TooLargeError) Error() string { return fmt.Sprintf("the upload is over %d bytes", e.Cap) }

func (e *TooLargeError) Words() string {
	return fmt.Sprintf("Use a subtitle file of %d MB or smaller.", e.Cap>>20)
}

func (s *Subtitles) uploadShape(b []byte) error {
	if len(b) > s.cfg.Timings.UploadCap {
		return &TooLargeError{Cap: s.cfg.Timings.UploadCap}
	}
	return nil
}
