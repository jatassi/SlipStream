package grabs

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// The Automatic search section's own settings, stored in grabs_settings.
// Backlog search's and RSS sync's cadences are tasks' (BacklogEvery and
// RSSEvery), which the section shows and sets too. The wait for each
// Indexer is indexers.SearchWait, which the section sets in the same Write.
var (
	BacklogOn       = setting.Switch{Key: "search.backlogOn", Label: "Backlog search", Default: true}
	BacklogUpgrades = setting.Switch{Key: "search.backlogUpgrades", Label: "Backlog search looks for upgrades", Default: true}
	RSSOn           = setting.Switch{Key: "search.rssOn", Label: "RSS sync", Default: true}
	LongestGap      = setting.Duration{
		Key: "search.longestGap", Label: "Search a Want that keeps coming up empty at least every",
		Default: 7 * day, Min: day, Max: 90 * day, Step: time.Hour,
	}
	FullPace = setting.Duration{
		Key: "search.fullPace", Label: "Don't space out searches for Items released in the last",
		Default: 14 * day, Min: 0, Max: 90 * day, Step: day,
	}
	StopCount = setting.Count{
		Key: "search.stopCount", Label: "Stop searching again after",
		Default: 3, Min: 1, Max: 20, Unit: setting.Noun{One: "failed Grab", Many: "failed Grabs"},
	}
	StopWindow = setting.Duration{
		Key: "search.stopWindow", Label: "Count failed Grabs in the last",
		Default: 7 * day, Min: time.Hour, Max: 30 * day, Step: time.Hour,
	}
)

// Settings is the Automatic search section.
func (gr *Grabs) Settings() setting.Panel { return panel{gr} }

type panel struct{ gr *Grabs }

func (p panel) Fields(ctx context.Context) ([]setting.Field, error) {
	var row db.GrabsSetting
	var wait *int64
	err := p.gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if row, err = db.New(r).GetSettings(ctx); err != nil {
			return err
		}
		wait, err = indexers.SearchWaitIn(ctx, r)
		return err
	})
	if err != nil {
		return nil, err
	}
	every, err := p.gr.cfg.Tasks.Every(ctx, BacklogTaskID)
	if err != nil {
		return nil, err
	}
	rssEvery, err := p.gr.cfg.Tasks.Every(ctx, RSSTaskID)
	if err != nil {
		return nil, err
	}
	return []setting.Field{
		BacklogOn.Field(row.BacklogOn), BacklogUpgrades.Field(row.BacklogUpgrades), BacklogEvery.Field(every),
		RSSOn.Field(row.RssOn), RSSEvery.Field(rssEvery),
		LongestGap.Field(row.LongestGapSeconds), FullPace.Field(row.FullPaceSeconds),
		StopCount.Field(row.StopCount), StopWindow.Field(row.StopWindowSeconds),
		indexers.SearchWait.Field(wait),
	}, nil
}

// Apply stores the patch, all or none: grabs_settings and
// indexers.SearchWait in one Write, then the two tasks' cadences through
// tasks. A longest gap shorter than the Backlog interval the patch leaves
// is *GapTooShortError on the key that moved.
func (p panel) Apply(ctx context.Context, patch setting.Patch) error {
	if len(patch) == 0 {
		return nil
	}
	every, backlog, err := p.cadence(ctx, patch, BacklogTaskID, BacklogEvery)
	if err != nil {
		return err
	}
	_, rss, err := p.cadence(ctx, patch, RSSTaskID, RSSEvery)
	if err != nil {
		return err
	}
	err = p.gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return store(ctx, tx, patch, BacklogEvery.Value(every))
	})
	if err != nil {
		return err
	}
	if backlog != nil {
		if err := p.gr.cfg.Tasks.SetCadence(ctx, BacklogTaskID, *backlog); err != nil {
			return err
		}
	}
	if rss == nil {
		return nil
	}
	return p.gr.cfg.Tasks.SetCadence(ctx, RSSTaskID, *rss)
}

// cadence is task id's cadence override the patch leaves, by its setting
// d, and the change to store when the patch moves it.
func (p panel) cadence(ctx context.Context, patch setting.Patch, id tasks.TaskID, d setting.Duration) (every *int64, change *setting.Stored[int64], err error) {
	every, err = p.gr.cfg.Tasks.Every(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	raw, ok := patch[d.Key]
	if !ok {
		return every, nil, nil
	}
	v, err := d.ParseJSON(raw)
	return v.Override(), &v, err
}

// store writes the patch's grabs_settings and indexers.SearchWait in tx,
// under the Backlog interval the patch leaves.
func store(ctx context.Context, tx *storage.Tx[file], patch setting.Patch, interval time.Duration) error {
	q := db.New(tx)
	row, err := q.GetSettings(ctx)
	if err != nil {
		return err
	}
	next := db.SetSettingsParams{
		BacklogOn: row.BacklogOn, BacklogUpgrades: row.BacklogUpgrades, RssOn: row.RssOn,
		LongestGapSeconds: row.LongestGapSeconds, FullPaceSeconds: row.FullPaceSeconds,
		StopCount: row.StopCount, StopWindowSeconds: row.StopWindowSeconds,
	}
	var wait *setting.Stored[int64]
	stored := map[string]func(json.RawMessage) error{
		BacklogOn.Key:       into(BacklogOn.ParseJSON, &next.BacklogOn),
		BacklogUpgrades.Key: into(BacklogUpgrades.ParseJSON, &next.BacklogUpgrades),
		BacklogEvery.Key:    func(json.RawMessage) error { return nil }, // tasks stores it, after the Write
		RSSOn.Key:           into(RSSOn.ParseJSON, &next.RssOn),
		RSSEvery.Key:        func(json.RawMessage) error { return nil },
		LongestGap.Key:      into(LongestGap.ParseJSON, &next.LongestGapSeconds),
		FullPace.Key:        into(FullPace.ParseJSON, &next.FullPaceSeconds),
		StopCount.Key:       into(StopCount.ParseJSON, &next.StopCount),
		StopWindow.Key:      into(StopWindow.ParseJSON, &next.StopWindowSeconds),
		indexers.SearchWait.Key: func(raw json.RawMessage) error {
			v, err := indexers.SearchWait.ParseJSON(raw)
			wait = &v
			return err
		},
	}
	for _, key := range slices.Sorted(maps.Keys(patch)) {
		parse, ok := stored[key]
		if !ok {
			return &setting.UnknownKeyError{Key: key}
		}
		if err := parse(patch[key]); err != nil {
			return err
		}
	}
	if err := gapRefusal(patch, LongestGap.Value(next.LongestGapSeconds), interval); err != nil {
		return err
	}
	if err := q.SetSettings(ctx, next); err != nil {
		return err
	}
	if wait == nil {
		return nil
	}
	return indexers.SetSearchWaitIn(ctx, tx, *wait)
}

// GapTooShortError is a patch that leaves the longest gap shorter than
// Backlog search's interval. Key is the setting it moved: the longest gap,
// else the interval. A patch that moves neither is never refused.
type GapTooShortError struct {
	Key      string
	Gap      time.Duration
	Interval time.Duration
}

func (e *GapTooShortError) Error() string {
	return fmt.Sprintf("the longest gap %s is shorter than Backlog search's interval %s", e.Gap, e.Interval)
}

// Words is the refusal as the Admin reads it.
func (e *GapTooShortError) Words() string {
	return "Use a longest gap of at least the Backlog search interval, " + spelled(e.Interval) + "."
}

func gapRefusal(patch setting.Patch, gap, interval time.Duration) error {
	if gap >= interval {
		return nil
	}
	for _, key := range []string{LongestGap.Key, BacklogEvery.Key} {
		if _, moved := patch[key]; moved {
			return &GapTooShortError{Key: key, Gap: gap, Interval: interval}
		}
	}
	return nil
}

// spelled is an interval in whole hours or days, in words: "8 hours",
// "1 day".
func spelled(d time.Duration) string {
	n, unit := int64(d/time.Hour), "hour"
	if d%day == 0 {
		n, unit = int64(d/day), "day"
	}
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

// into parses a value with parse and keeps its override in to.
func into[T any](parse func(json.RawMessage) (setting.Stored[T], error), to **T) func(json.RawMessage) error {
	return func(raw json.RawMessage) error {
		v, err := parse(raw)
		*to = v.Override()
		return err
	}
}
