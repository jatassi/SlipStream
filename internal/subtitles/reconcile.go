package subtitles

import (
	"context"
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
	ignored  map[files.ID][]db.SubtitlesTrackIgnored
	langs    map[targets.ProfileID][]targets.WantedLanguage
	added    map[targets.ProfileID]map[Want]storage.Time
	gained   map[subtitling.Kind]storage.Time
	interval time.Duration
	now      storage.Time
}

func (s *Subtitles) read(ctx context.Context, r storage.Reader[storage.Library], subjects []files.Subject) (chunk, error) {
	c := chunk{
		subjects: subjects, wants: map[files.ID][]db.SubtitlesWant{}, sidecars: map[files.ID][]Sidecar{}, ignored: map[files.ID][]db.SubtitlesTrackIgnored{},
		added: map[targets.ProfileID]map[Want]storage.Time{}, gained: map[subtitling.Kind]storage.Time{}, now: storage.FromTime(s.cfg.Clock()),
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
	added, err := q.ListLanguagesAdded(ctx)
	if err != nil {
		return chunk{}, err
	}
	for _, a := range added {
		l, err := language.Parse(a.Language)
		if err != nil {
			return chunk{}, err
		}
		if c.added[a.ProfileID] == nil {
			c.added[a.ProfileID] = map[Want]storage.Time{}
		}
		c.added[a.ProfileID][Want{l, a.Forced}] = a.AddedAt
	}
	providers, err := q.ListProviders(ctx)
	if err != nil {
		return chunk{}, err
	}
	for _, p := range providers {
		for _, k := range s.spec(subtitling.ProviderID(p.ID)).Kinds {
			if p.IsOn && p.GainedAt.Compare(c.gained[k]) > 0 {
				c.gained[k] = p.GainedAt
			}
		}
	}
	every, err := s.cfg.Tasks.EveryIn(ctx, r, TaskID)
	c.interval = Every.Value(every)
	return c, err
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
		ignored := make([]ignore, 0, len(c.ignored[sub.ID]))
		for _, row := range c.ignored[sub.ID] {
			ig := ignore{Ordinal: int(row.StreamIndex), Codec: row.Codec, Language: release.Language(row.Language), Forced: row.Forced}
			if !ig.holds(sub.Text) {
				if err := q.DeleteIgnored(ctx, db.DeleteIgnoredParams{FileID: sub.ID, StreamIndex: row.StreamIndex}); err != nil {
					return err
				}
				continue
			}
			ignored = append(ignored, ig)
		}
		f := factsOf(sub, kindOfType(s.cfg.Types[sub.Type]), ignored)
		if err := putFacts(ctx, q, sub, f, c.now); err != nil {
			return err
		}
		scs := c.sidecars[sub.ID]
		ws := wantsOf(f, c.langs[sub.Profile], func(w Want) bool { return sidecarOfWant(scs, w) != nil })
		final := func(w Want) bool { sc := sidecarOfWant(scs, w); return sc != nil && sc.Final }
		p := pace{interval: c.interval, added: func(w Want) storage.Time { return c.added[sub.Profile][w] }}
		if err := putWants(ctx, q, sub, f, ws, c.wants[sub.ID], final, p); err != nil {
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
