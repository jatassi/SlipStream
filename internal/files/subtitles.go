package files

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Subtitles is what files tells subtitles, each in the Write that makes
// it true.
type Subtitles interface {
	// Changed is tell's facts, the ones the Media Servers are told.
	Changed(ctx context.Context, tx *storage.Tx[storage.Library], cs []PathChange) error
	// Leaving runs in doom before d's row goes. by is the file that
	// replaces d in this same Write, or the zero ID when nothing does.
	Leaving(ctx context.Context, tx *storage.Tx[storage.Library], d File, by ID) error
	// Languages runs in an UpdateProfile's change Write when its Wanted
	// Languages differ, before the Change applies. It answers the Impact
	// and, unless preview, makes the drops so.
	Languages(ctx context.Context, tx *storage.Tx[storage.Library], p targets.ProfileID, was, now []targets.WantedLanguage, preview bool) (SubtitlesImpact, error)
}

// NoSubtitles is the Subtitles of a library nothing subtitles.
type NoSubtitles struct{}

func (NoSubtitles) Changed(context.Context, *storage.Tx[storage.Library], []PathChange) error {
	return nil
}

func (NoSubtitles) Leaving(context.Context, *storage.Tx[storage.Library], File, ID) error {
	return nil
}

func (NoSubtitles) Languages(context.Context, *storage.Tx[storage.Library], targets.ProfileID, []targets.WantedLanguage, []targets.WantedLanguage, bool) (SubtitlesImpact, error) {
	return SubtitlesImpact{}, nil
}

// SubtitlesImpact is what a change of a Profile's Wanted Languages does to
// subtitles.
type SubtitlesImpact struct {
	Gained  []GainedLanguage
	Dropped []DroppedLanguage
	// Days is how long the gained searches take at the providers' last
	// reported download allowance, zero when none reported one.
	Days int
}

// GainedLanguage is a want the change adds: Files is how many files it
// searches for, NeedScan how many of them need their audio scanned first.
type GainedLanguage struct {
	Language        language.Code
	Forced          bool
	Files, NeedScan int
}

// DroppedLanguage is a want the change removes: Deleted is how many
// subtitles SlipStream downloaded for it the save deletes.
type DroppedLanguage struct {
	Language language.Code
	Forced   bool
	Deleted  int
}

// Deleted is how many subtitles the save deletes in all.
func (s SubtitlesImpact) Deleted() int {
	n := 0
	for _, d := range s.Dropped {
		n += d.Deleted
	}
	return n
}

// Subject is one library file as subtitle wants read it.
type Subject struct {
	ID      ID
	Work    works.WorkID
	Type    works.TypeID
	Profile targets.ProfileID
	Root    roots.ID
	Path    storing.Rel
	Items   []works.ItemID
	Cut     release.CutKey
	// Kept is a file that fills nothing.
	Kept bool
	Gone bool
	// AtName is no move open: the bytes sit at Path.
	AtName bool
	// Audio is each audio track's language in stream order, "" when
	// untagged, and DefaultAudio the 1-based ordinal of the default one, 0
	// when none is.
	Audio        []release.Language
	DefaultAudio int
	// Text is the media record's subtitle tracks in stream order.
	Text     []media.Subtitle
	ProbedAt storage.Time
	Duration time.Duration
	// Seen and Size are what a scan or an import last read of its bytes.
	Seen *storing.Seen
	Size int64
	// Group is the Release group its name reads to, "" when none.
	Group string
}

// Subjects is up to n library files after the file id after, in id
// order. among narrows them to Works: a Work page, a Profile's change.
func (f *Files) Subjects(ctx context.Context, r storage.Reader[file], after ID, n int, among works.Among[works.WorkID]) ([]Subject, error) {
	q := db.New(r)
	rows, err := subjectRows(ctx, q, after, n, among)
	if err != nil {
		return nil, err
	}
	return subjectsOf(ctx, r, q, rows)
}

// SubjectsOf is the Subjects of ids, in id order, without the ids no file
// has.
func (f *Files) SubjectsOf(ctx context.Context, r storage.Reader[file], ids []ID) ([]Subject, error) {
	q := db.New(r)
	var rows []db.ListSubjectsRow
	for chunk := range slices.Chunk(ids, works.Batch) {
		of, err := q.ListSubjectsOfFiles(ctx, chunk)
		if err != nil {
			return nil, err
		}
		for _, row := range of {
			rows = append(rows, db.ListSubjectsRow(row))
		}
	}
	slices.SortFunc(rows, func(a, b db.ListSubjectsRow) int { return a.FilesFile.ID.Compare(b.FilesFile.ID.UUID) })
	return subjectsOf(ctx, r, q, rows)
}

func subjectsOf(ctx context.Context, r storage.Reader[file], q *db.Queries, rows []db.ListSubjectsRow) ([]Subject, error) {
	ids := make([]ID, len(rows))
	var ws []works.WorkID
	for i, row := range rows {
		ids[i] = row.FilesFile.ID
		if !slices.Contains(ws, row.FilesFile.WorkID) {
			ws = append(ws, row.FilesFile.WorkID)
		}
	}
	items, err := q.ListItemsOfFiles(ctx, ids)
	if err != nil {
		return nil, err
	}
	held := map[ID][]works.ItemID{}
	for _, it := range items {
		held[it.FileID] = append(held[it.FileID], it.ItemID)
	}
	wanting, err := targets.WantingOf(ctx, r, ws)
	if err != nil {
		return nil, err
	}
	out := make([]Subject, len(rows))
	for i, row := range rows {
		g := fileOf(row.FilesFile)
		if out[i], err = subjectOf(g, row.AtName == 1, held[g.ID], wanting[g.WorkID]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// subjectRows is the page of Subjects reads: of every Work in one query,
// else of among's Works a Batch at a time, merged in id order.
func subjectRows(ctx context.Context, q *db.Queries, after ID, n int, among works.Among[works.WorkID]) ([]db.ListSubjectsRow, error) {
	if among.Every() {
		return q.ListSubjects(ctx, db.ListSubjectsParams{After: after, N: int64(n)})
	}
	var rows []db.ListSubjectsRow
	for chunk := range slices.Chunk(among.IDs(), works.Batch) {
		listed, err := json.Marshal(chunk)
		if err != nil {
			return nil, fmt.Errorf("listing Works: %w", err)
		}
		of, err := q.ListSubjectsOfWorks(ctx, db.ListSubjectsOfWorksParams{WorkIds: string(listed), After: after, N: int64(n)})
		if err != nil {
			return nil, err
		}
		for _, row := range of {
			rows = append(rows, db.ListSubjectsRow(row))
		}
	}
	slices.SortFunc(rows, func(a, b db.ListSubjectsRow) int { return a.FilesFile.ID.Compare(b.FilesFile.ID.UUID) })
	return rows[:min(n, len(rows))], nil
}

func subjectOf(g File, atName bool, items []works.ItemID, w targets.Wanting) (Subject, error) {
	path, err := storing.ParseRel(g.Path)
	if err != nil {
		return Subject{}, err
	}
	group, _ := g.Title.Facts.Group.Get()
	s := Subject{
		ID: g.ID, Work: g.WorkID, Type: w.Type, Profile: w.Profile, Root: g.RootID, Path: path, Items: items, Cut: g.Cut,
		Kept: g.Kept(), Gone: g.Gone(), AtName: atName, Text: g.Media.Subtitles, ProbedAt: g.ProbedAt, Duration: g.Media.Duration,
		Seen: g.Seen, Size: g.Size, Group: group,
	}
	for k, a := range g.Media.Audio {
		s.Audio = append(s.Audio, a.Language)
		if a.Default && s.DefaultAudio == 0 {
			s.DefaultAudio = k + 1
		}
	}
	return s, nil
}
