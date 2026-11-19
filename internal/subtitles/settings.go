package subtitles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/subtitles/timing"
	"github.com/jatassi/slipstream/internal/works"
)

// The Subtitles section's settings, stored in subtitles_settings. Subtitle
// search's cadence is tasks' (Every), which the section shows and sets too.
var (
	SearchOn       = setting.Switch{Key: "subtitles.searchOn", Label: "Subtitle search", Default: true}
	SearchUpgrades = setting.Switch{Key: "subtitles.searchUpgrades", Label: "Subtitle search looks for better subtitles", Default: true}
	LongestGap     = setting.Duration{
		Key: "subtitles.longestGap", Label: "Search for a subtitle that keeps coming up empty at least every",
		Default: 7 * day, Min: day, Max: 90 * day, Step: time.Hour,
	}
	FullPace = setting.Duration{
		Key: "subtitles.fullPace", Label: "Don't space out searches for anything released in the last",
		Default: 14 * day, Min: 0, Max: 90 * day, Step: day,
	}
	// ReportGap is stored now. The section shows it once S47 (Subtitle
	// reports) supplies reports.
	ReportGap = setting.Duration{
		Key: "subtitles.reportGap", Label: "While a Member's report is open, search for its subtitle at least every",
		Default: day, Min: time.Hour, Max: 7 * day, Step: time.Hour,
	}
	TriesPerSearch = setting.Count{
		Key: "subtitles.triesPerSearch", Label: "Try at most this many subtitles per search",
		Default: 3, Min: 1, Max: 10, Unit: setting.Noun{One: "subtitle", Many: "subtitles"},
	}
	Match = setting.Count{
		Key: "subtitles.sync.match", Label: "Keep a subtitle only when its timing match is at least",
		Default: timing.DefaultMatchPercent, Min: 10, Max: 95, Unit: setting.Noun{One: "%", Many: "%"},
	}
	MaxOffset = setting.Duration{
		Key: "subtitles.sync.maxOffset", Label: "Largest timing shift to accept",
		Default: 30 * time.Second, Min: 5 * time.Second, Max: 5 * time.Minute, Step: time.Second,
	}
	Framerates = setting.Choice[string]{
		Key: "subtitles.sync.framerates", Label: "Framerate changes to try", Default: "standard",
		Options: []setting.Option[string]{
			{Value: "none", Label: "None"},
			{Value: "standard", Label: "Standard rates"},
			{Value: "range", Label: "Standard rates and any between 0.9 and 1.1 times"},
		},
	}
	WarnAfter = setting.Duration{
		Key: "subtitles.providerWarnAfter", Label: "Warn when a Subtitle Provider keeps failing for",
		Default: time.Hour, Min: 10 * time.Minute, Max: day, Step: time.Minute,
	}
	// MapSlots is a Storage Node's own setting, on its card.
	MapSlots = setting.Count{
		Key: "subtitles.mapSlots", Label: "Audio scans at once on this Node",
		Default: 1, Min: 1, Max: 8, Unit: setting.Noun{One: "scan", Many: "scans"},
	}
)

// Settings is the Subtitles section.
func (s *Subtitles) Settings() setting.Panel { return panel{s} }

type panel struct{ s *Subtitles }

func (p panel) Fields(ctx context.Context) ([]setting.Field, error) {
	var row db.SubtitlesSetting
	var every *int64
	err := p.s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		if row, err = db.New(r).GetSettings(ctx); err != nil {
			return err
		}
		every, err = p.s.cfg.Tasks.EveryIn(ctx, r, TaskID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return []setting.Field{
		SearchOn.Field(row.SearchOn), SearchUpgrades.Field(row.SearchUpgrades), Every.Field(every),
		LongestGap.Field(row.LongestGapSeconds), FullPace.Field(row.FullPaceSeconds), TriesPerSearch.Field(row.TriesPerSearch),
		Match.Field(row.SyncMatch), MaxOffset.Field(row.SyncMaxOffsetSeconds), Framerates.Field(row.SyncFramerates),
		WarnAfter.Field(row.ProviderWarnAfterSeconds),
	}, nil
}

// Apply stores the patch, all or none: subtitles_settings in one Write,
// then Subtitle search's cadence through tasks. A longest gap shorter than
// the interval the patch leaves is *GapTooShortError on the key that
// moved.
func (p panel) Apply(ctx context.Context, patch setting.Patch) error {
	if len(patch) == 0 {
		return nil
	}
	var every *int64
	var cadence *setting.Stored[int64]
	err := p.s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		if every, err = p.s.cfg.Tasks.EveryIn(ctx, tx, TaskID); err != nil {
			return err
		}
		if raw, ok := patch[Every.Key]; ok {
			v, err := Every.ParseJSON(raw)
			if err != nil {
				return err
			}
			every, cadence = v.Override(), &v
		}
		return store(ctx, tx, patch, Every.Value(every))
	})
	if err != nil {
		return err
	}
	p.s.cfg.ProvidersChanged.Publish(ctx, ProvidersChanged{})
	if cadence == nil {
		return nil
	}
	return p.s.cfg.Tasks.SetCadence(ctx, TaskID, *cadence)
}

func store(ctx context.Context, tx *storage.Tx[storage.Library], patch setting.Patch, interval time.Duration) error {
	q := db.New(tx)
	row, err := q.GetSettings(ctx)
	if err != nil {
		return err
	}
	next := db.SetSettingsParams{
		SearchOn: row.SearchOn, SearchUpgrades: row.SearchUpgrades, LongestGapSeconds: row.LongestGapSeconds, FullPaceSeconds: row.FullPaceSeconds,
		ReportGapSeconds: row.ReportGapSeconds, TriesPerSearch: row.TriesPerSearch, SyncMatch: row.SyncMatch,
		SyncMaxOffsetSeconds: row.SyncMaxOffsetSeconds, SyncFramerates: row.SyncFramerates, ProviderWarnAfterSeconds: row.ProviderWarnAfterSeconds,
	}
	stored := map[string]func(json.RawMessage) error{
		SearchOn.Key:       into(SearchOn.ParseJSON, &next.SearchOn),
		SearchUpgrades.Key: into(SearchUpgrades.ParseJSON, &next.SearchUpgrades),
		Every.Key:          func(json.RawMessage) error { return nil }, // tasks stores it, after the Write
		LongestGap.Key:     into(LongestGap.ParseJSON, &next.LongestGapSeconds),
		FullPace.Key:       into(FullPace.ParseJSON, &next.FullPaceSeconds),
		ReportGap.Key:      into(ReportGap.ParseJSON, &next.ReportGapSeconds),
		TriesPerSearch.Key: into(TriesPerSearch.ParseJSON, &next.TriesPerSearch),
		Match.Key:          into(Match.ParseJSON, &next.SyncMatch),
		MaxOffset.Key:      into(MaxOffset.ParseJSON, &next.SyncMaxOffsetSeconds),
		Framerates.Key:     into(Framerates.ParseJSON, &next.SyncFramerates),
		WarnAfter.Key:      into(WarnAfter.ParseJSON, &next.ProviderWarnAfterSeconds),
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
	return q.SetSettings(ctx, next)
}

// GapTooShortError is a patch that leaves the longest gap shorter than
// Subtitle search's interval. Key is the setting it moved: the longest
// gap, else the interval. A patch that moves neither is never refused.
type GapTooShortError struct {
	Key      string
	Gap      time.Duration
	Interval time.Duration
}

func (e *GapTooShortError) Error() string {
	return fmt.Sprintf("the longest subtitle gap %s is shorter than Subtitle search's interval %s", e.Gap, e.Interval)
}

// Words is the refusal as the Admin reads it under the setting Key names.
func (e *GapTooShortError) Words() string {
	if e.Key == LongestGap.Key {
		return "Use " + spelled(e.Interval) + " or longer. Subtitle search runs every " + spelled(e.Interval) + "."
	}
	return "Use " + spelled(e.Gap) + " or shorter. A subtitle that keeps coming up empty is searched for at least every " + spelled(e.Gap) + "."
}

func gapRefusal(patch setting.Patch, gap, interval time.Duration) error {
	if gap >= interval {
		return nil
	}
	for _, key := range []string{LongestGap.Key, Every.Key} {
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

func into[T any](parse func(json.RawMessage) (setting.Stored[T], error), to **T) func(json.RawMessage) error {
	return func(raw json.RawMessage) error {
		v, err := parse(raw)
		*to = v.Override()
		return err
	}
}

// NotStorageError is a Node settings call for a Node without the Storage
// Role.
type NotStorageError struct{ Node role.NodeID }

func (e *NotStorageError) Error() string {
	return fmt.Sprintf("node %s doesn't have the Storage Role", e.Node)
}

// NodeSettings is the Subtitles settings of one Storage Node, on its card.
func (s *Subtitles) NodeSettings(node role.NodeID) setting.Panel { return nodePanel{s, node} }

type nodePanel struct {
	s    *Subtitles
	node role.NodeID
}

func (p nodePanel) storage() error {
	if !slices.Contains(p.s.cfg.Roster.Holding(role.Storage), p.node) {
		return &NotStorageError{Node: p.node}
	}
	return nil
}

func (p nodePanel) Fields(ctx context.Context) ([]setting.Field, error) {
	if err := p.storage(); err != nil {
		return nil, err
	}
	var slots *int64
	err := p.s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		slots, err = db.New(r).GetNodeSlots(ctx, p.node)
		if errors.Is(err, storage.ErrNoRows) {
			return nil
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return []setting.Field{MapSlots.Field(slots)}, nil
}

func (p nodePanel) Apply(ctx context.Context, patch setting.Patch) error {
	if err := p.storage(); err != nil {
		return err
	}
	for _, key := range slices.Sorted(maps.Keys(patch)) {
		if key != MapSlots.Key {
			return &setting.UnknownKeyError{Key: key}
		}
	}
	raw, ok := patch[MapSlots.Key]
	if !ok {
		return nil
	}
	v, err := MapSlots.ParseJSON(raw)
	if err != nil {
		return err
	}
	return p.s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		if err := q.SetNodeSlots(ctx, db.SetNodeSlotsParams{NodeID: p.node, MapSlots: v.Override()}); err != nil {
			return err
		}
		return q.DropIdleNode(ctx, p.node)
	})
}

// LeadKind is which sentence leads the Subtitles section.
type LeadKind uint8

const (
	NoLanguages LeadKind = iota + 1
	Uncovered
	SearchOff
	SearchReady
)

// Lead is what the Subtitles section leads with: the first kind that
// holds, the Content Types that no provider that can search covers, the
// unmet wants, Subtitle search's interval and each Storage Node's audio
// scans still due.
type Lead struct {
	Kind      LeadKind
	Uncovered []works.TypeID
	Missing   int
	Interval  time.Duration
	Scans     []Scans
}

// Scans is one Storage Node's files whose audio is due a scan.
type Scans struct {
	Node role.NodeID
	Name string
	Left int
}

// Lead reads the section's lead in one Read.
func (s *Subtitles) Lead(ctx context.Context) (Lead, error) {
	var l Lead
	var due []db.MapsDueByRootRow
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		if due, err = db.New(r).MapsDueByRoot(ctx); err != nil {
			return err
		}
		l, err = s.leadIn(ctx, r)
		return err
	})
	if err != nil {
		return Lead{}, err
	}
	l.Scans, err = s.scansOf(ctx, due)
	return l, err
}

func (s *Subtitles) leadIn(ctx context.Context, r storage.Reader[storage.Library]) (Lead, error) {
	q := db.New(r)
	row, err := q.GetSettings(ctx)
	if err != nil {
		return Lead{}, err
	}
	every, err := s.cfg.Tasks.EveryIn(ctx, r, TaskID)
	if err != nil {
		return Lead{}, err
	}
	l := Lead{Interval: Every.Value(every)}
	missing, err := q.MissingCount(ctx)
	if err != nil {
		return Lead{}, err
	}
	l.Missing = int(missing)
	wanted, err := q.ListWantedTypes(ctx)
	if err != nil {
		return Lead{}, err
	}
	providers, err := s.providersIn(ctx, r)
	if err != nil {
		return Lead{}, err
	}
	for _, t := range wanted {
		if _, ok := s.uncovered(works.TypeID(t), providers); ok {
			l.Uncovered = append(l.Uncovered, works.TypeID(t))
		}
	}
	switch {
	case len(wanted) == 0:
		l.Kind = NoLanguages
	case len(l.Uncovered) > 0:
		l.Kind = Uncovered
	case !SearchOn.Value(row.SearchOn):
		l.Kind = SearchOff
	default:
		l.Kind = SearchReady
	}
	return l, nil
}

// scansOf is each Node's audio scans still due, its roots' due summed,
// in Node order.
func (s *Subtitles) scansOf(ctx context.Context, due []db.MapsDueByRootRow) ([]Scans, error) {
	left := map[role.NodeID]int{}
	for _, d := range due {
		root, _, err := s.cfg.Roots.StateOf(ctx, d.RootID)
		if err != nil {
			return nil, err
		}
		left[root.Node] += int(d.Due)
	}
	out := make([]Scans, 0, len(left))
	for _, n := range slices.SortedFunc(maps.Keys(left), func(a, b role.NodeID) int { return a.Compare(b.UUID) }) {
		out = append(out, Scans{Node: n, Name: s.cfg.Roster.Name(n), Left: left[n]})
	}
	return out, nil
}
