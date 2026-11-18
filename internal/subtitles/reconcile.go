package subtitles

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// TaskID is Subtitle search's task, whose cadence is the interval a new
// want's first search spreads over.
const TaskID tasks.TaskID = "subtitle-search"

const day = 24 * time.Hour

// Every is Subtitle search's cadence, which tasks stores.
var Every = setting.Duration{
	Key: "tasks." + string(TaskID) + ".cadence", Label: "Subtitle search runs every",
	Default: 8 * time.Hour, Min: time.Hour, Max: 7 * day, Step: time.Hour,
	Floor: "Every install shares Gestdown's servers, and providers limit how often each key searches.",
}

// Cadences is the tasks' stored cadences.
type Cadences interface {
	EveryIn(ctx context.Context, r storage.Reader[storage.Library], id tasks.TaskID) (*int64, error)
	SetCadence(ctx context.Context, id tasks.TaskID, v setting.Stored[int64]) error
}

// Unscheduled is the Cadences of a Controller whose catalog has no
// Subtitle search task yet: every cadence is its default.
type Unscheduled struct{}

func (Unscheduled) EveryIn(context.Context, storage.Reader[storage.Library], tasks.TaskID) (*int64, error) {
	return nil, nil
}

// Reconciled is what one bulk pass covered.
type Reconciled struct{ Chunks, Files int }

// Reconcile is the bulk pass: every library file's facts and want rows,
// ReconcileChunk files to one Read and one Write. While the library
// Reconciles it does nothing.
func (s *Subtitles) Reconcile(ctx context.Context) (Reconciled, error) {
	var out Reconciled
	var after files.ID
	for {
		var in chunk
		var halt bool
		err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
			var err error
			if halt, err = s.cfg.Reconciling(ctx, r); err != nil || halt {
				return err
			}
			subjects, err := s.cfg.Files.Subjects(ctx, r, after, s.cfg.Timings.ReconcileChunk, works.Every[works.WorkID](nil))
			if err != nil {
				return err
			}
			in, err = s.read(ctx, r, subjects)
			return err
		})
		if err != nil || halt || len(in.subjects) == 0 {
			return out, err
		}
		if err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return s.write(ctx, tx, in)
		}); err != nil {
			return out, err
		}
		out.Chunks++
		out.Files += len(in.subjects)
		after = in.subjects[len(in.subjects)-1].ID
		if len(in.subjects) < s.cfg.Timings.ReconcileChunk {
			return out, nil
		}
	}
}

// chunk is what one Write of the bulk pass, or one hook, reads of its
// files.
type chunk struct {
	subjects []files.Subject
	wants    map[files.ID][]db.SubtitlesWant
	sidecars map[files.ID][]Sidecar
	extras   map[files.ID][]Extra
	ignored  map[files.ID][]db.SubtitlesTrackIgnored
	langs    map[targets.ProfileID][]targets.WantedLanguage
	added    map[targets.ProfileID]map[Want]storage.Time
	gained   map[subtitling.Kind]storage.Time
	interval time.Duration
	now      storage.Time
}

func (s *Subtitles) read(ctx context.Context, r storage.Reader[storage.Library], subjects []files.Subject) (chunk, error) {
	c := chunk{
		subjects: subjects, wants: map[files.ID][]db.SubtitlesWant{}, sidecars: map[files.ID][]Sidecar{}, extras: map[files.ID][]Extra{},
		ignored: map[files.ID][]db.SubtitlesTrackIgnored{}, now: storage.FromTime(s.cfg.Clock()),
	}
	ids := make([]files.ID, len(subjects))
	for i, sub := range subjects {
		ids[i] = sub.ID
	}
	q := db.New(r)
	wants, err := q.ListWantsIn(ctx, ids)
	if err != nil {
		return chunk{}, err
	}
	for _, w := range wants {
		c.wants[w.FileID] = append(c.wants[w.FileID], w)
	}
	if c.sidecars, err = sidecarsIn(ctx, q, ids); err != nil {
		return chunk{}, err
	}
	if c.extras, err = extrasIn(ctx, q, ids); err != nil {
		return chunk{}, err
	}
	ignored, err := q.ListIgnoredIn(ctx, ids)
	if err != nil {
		return chunk{}, err
	}
	for _, ig := range ignored {
		c.ignored[ig.FileID] = append(c.ignored[ig.FileID], ig)
	}
	if c.langs, err = s.cfg.Targets.LanguagesOf(ctx, r); err != nil {
		return chunk{}, err
	}
	if c.added, err = addedIn(ctx, q); err != nil {
		return chunk{}, err
	}
	if c.gained, err = s.gainedIn(ctx, q); err != nil {
		return chunk{}, err
	}
	every, err := s.cfg.Tasks.EveryIn(ctx, r, TaskID)
	c.interval = Every.Value(every)
	return c, err
}

// addedIn is when each Profile's Wanted Languages were added, which a new
// want's first search spreads from.
func addedIn(ctx context.Context, q *db.Queries) (map[targets.ProfileID]map[Want]storage.Time, error) {
	added, err := q.ListLanguagesAdded(ctx)
	if err != nil {
		return nil, err
	}
	out := map[targets.ProfileID]map[Want]storage.Time{}
	for _, a := range added {
		l, err := language.Parse(a.Language)
		if err != nil {
			return nil, err
		}
		if out[a.ProfileID] == nil {
			out[a.ProfileID] = map[Want]storage.Time{}
		}
		out[a.ProfileID][Want{l, a.Forced}] = a.AddedAt
	}
	return out, nil
}

// gainedIn is when each kind last gained an on provider.
func (s *Subtitles) gainedIn(ctx context.Context, q *db.Queries) (map[subtitling.Kind]storage.Time, error) {
	providers, err := q.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	out := map[subtitling.Kind]storage.Time{}
	for _, p := range providers {
		for _, k := range s.spec(p.ID).Kinds {
			if p.IsOn && p.GainedAt.Compare(out[k]) > 0 {
				out[k] = p.GainedAt
			}
		}
	}
	return out, nil
}

// extrasIn is the extras each file's last survey listed.
func extrasIn(ctx context.Context, q *db.Queries, ids []files.ID) (map[files.ID][]Extra, error) {
	rows, err := q.ListFilesIn(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := map[files.ID][]Extra{}
	for _, row := range rows {
		if row.Companions == nil {
			continue
		}
		var extras []Extra
		if err := json.Unmarshal([]byte(*row.Companions), &extras); err != nil {
			return nil, fmt.Errorf("file %s's extras: %w", row.FileID, err)
		}
		out[row.FileID] = extras
	}
	return out, nil
}

func sidecarsIn(ctx context.Context, q *db.Queries, ids []files.ID) (map[files.ID][]Sidecar, error) {
	rows, err := q.ListSidecarsIn(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := map[files.ID][]Sidecar{}
	for _, row := range rows {
		sc, err := sidecarOf(row)
		if err != nil {
			return nil, err
		}
		out[row.FileID] = append(out[row.FileID], sc)
	}
	return out, nil
}

// write makes c's files' facts, want rows and ignores what their
// Subjects and Profiles say, through records.go.
func (s *Subtitles) write(ctx context.Context, tx *storage.Tx[storage.Library], c chunk) error {
	q := db.New(tx)
	byKind := map[subtitling.Kind][]files.ID{}
	for _, sub := range c.subjects {
		f, err := s.writeFile(ctx, q, c, sub)
		if err != nil {
			return err
		}
		if f.Kind != 0 {
			byKind[f.Kind] = append(byKind[f.Kind], sub.ID)
		}
	}
	for k, ids := range byKind {
		if gained, ok := c.gained[k]; ok {
			if err := q.StampGained(ctx, db.StampGainedParams{Now: c.now, Ids: ids, Kind: kindNames[k], Gained: gained}); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeFile makes sub's facts and want rows what its Subject and Profile
// say, deleting the ignores whose track changed. A Missing want an extra
// names marks sub for a survey, which adopts it: the last survey ran
// before the want was.
func (s *Subtitles) writeFile(ctx context.Context, q *db.Queries, c chunk, sub files.Subject) (facts, error) {
	ignored := make([]ignore, 0, len(c.ignored[sub.ID]))
	for _, row := range c.ignored[sub.ID] {
		ig := ignore{Ordinal: int(row.StreamIndex), Codec: row.Codec, Language: release.Language(row.Language), Forced: row.Forced}
		if !ig.holds(sub.Text) {
			if err := q.DeleteIgnored(ctx, db.DeleteIgnoredParams{FileID: sub.ID, StreamIndex: row.StreamIndex}); err != nil {
				return facts{}, err
			}
			continue
		}
		ignored = append(ignored, ig)
	}
	f := factsOf(sub, kindOfType(s.cfg.Types[sub.Type]), ignored)
	if err := putFacts(ctx, q, sub, f, c.now); err != nil {
		return facts{}, err
	}
	scs := c.sidecars[sub.ID]
	ws := wantsOf(f, c.langs[sub.Profile], func(w Want) bool { return sidecarOfWant(scs, w) != nil })
	final := func(w Want) bool { sc := sidecarOfWant(scs, w); return sc != nil && sc.Final }
	p := pace{interval: c.interval, added: func(w Want) storage.Time { return c.added[sub.Profile][w] }}
	if err := putWants(ctx, q, sub, f, ws, c.wants[sub.ID], final, p); err != nil {
		return facts{}, err
	}
	if slices.ContainsFunc(ws, func(w want) bool { return w.Kind == Missing && extraNames(c.extras[sub.ID], w.Want()) }) {
		if err := q.MarkSurveyDue(ctx, db.MarkSurveyDueParams{FileID: sub.ID}); err != nil {
			return facts{}, err
		}
		s.keeper.soon(sub.ID)
	}
	return f, nil
}

func extraNames(extras []Extra, w Want) bool {
	return slices.ContainsFunc(extras, func(e Extra) bool { return e.Want == w.String() })
}

func sidecarOfWant(scs []Sidecar, w Want) *Sidecar {
	for i := range scs {
		if scs[i].want() == w {
			return &scs[i]
		}
	}
	return nil
}

// reconcileIn is the bulk pass for ids inside a hook's Write.
func (s *Subtitles) reconcileIn(ctx context.Context, tx *storage.Tx[storage.Library], ids []files.ID) error {
	if len(ids) == 0 {
		return nil
	}
	subjects, err := s.cfg.Files.SubjectsOf(ctx, tx, ids)
	if err != nil {
		return err
	}
	c, err := s.read(ctx, tx, subjects)
	if err != nil {
		return err
	}
	return s.write(ctx, tx, c)
}
