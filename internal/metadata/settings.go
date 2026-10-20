package metadata

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

const day = 24 * time.Hour

const sharedKeys = "Every SlipStream install shares the bundled keys."

var (
	LanguageChoice = setting.Choice[string]{Key: "metadata.language", Label: "Metadata language", Default: DefaultLanguage, Options: languages}
	RegionChoice   = setting.Choice[string]{Key: "metadata.region", Label: "Metadata region", Default: DefaultRegion, Options: regions}
	CacheFor       = setting.Duration{Key: "metadata.cacheFor", Label: "Metadata cache keeps answers for", Default: time.Hour, Min: 0, Max: day, Step: time.Minute}
	RatingsEvery   = setting.Duration{Key: "metadata.ratingsEvery", Label: "OMDb ratings no more often than every", Default: 7 * day, Min: day, Max: 365 * day, Step: day}
	ArtworkStore   = setting.Count{Key: "metadata.artworkStore", Label: "Artwork store size", Default: 2048, Min: 100, Max: 1 << 20, Unit: setting.Noun{One: "MB", Many: "MB"}}

	UnreleasedEvery = setting.Duration{Key: "metadata.refresh.unreleased", Label: "Unreleased Works refresh every", Default: day, Min: day, Max: 30 * day, Step: day, Floor: sharedKeys}
	GrowingEvery    = setting.Duration{Key: "metadata.refresh.growing", Label: "Growing Works refresh every", Default: day, Min: day, Max: 30 * day, Step: day, Floor: sharedKeys}
	OthersEvery     = setting.Duration{Key: "metadata.refresh.others", Label: "Every other Work refresh every", Default: 30 * day, Min: day, Max: 365 * day, Step: day, Floor: sharedKeys}
	FollowChanges   = setting.Switch{Key: "metadata.refresh.changes", Label: "Refresh what the provider reports changed", Default: true}
)

// stored is each metadata setting's override, nil at its default.
type stored = db.MetadataSetting

type column struct {
	key   string
	field func(stored) setting.Field
	set   func(*stored, json.RawMessage) error
}

func columnOf[T any, F setting.Field](key string, field func(*T) F, parse func(json.RawMessage) (setting.Stored[T], error), at func(*stored) **T) column {
	return column{
		key:   key,
		field: func(s stored) setting.Field { return field(*at(&s)) },
		set: func(s *stored, raw json.RawMessage) error {
			v, err := parse(raw)
			if err != nil {
				return err
			}
			*at(s) = v.Override()
			return nil
		},
	}
}

var settingColumns = []column{
	columnOf(LanguageChoice.Key, LanguageChoice.Field, LanguageChoice.ParseJSON, func(s *stored) **string { return &s.Language }),
	columnOf(RegionChoice.Key, RegionChoice.Field, RegionChoice.ParseJSON, func(s *stored) **string { return &s.Region }),
	columnOf(CacheFor.Key, CacheFor.Field, CacheFor.ParseJSON, func(s *stored) **int64 { return &s.CacheForSeconds }),
	columnOf(RatingsEvery.Key, RatingsEvery.Field, RatingsEvery.ParseJSON, func(s *stored) **int64 { return &s.RatingsEverySeconds }),
	columnOf(ArtworkStore.Key, ArtworkStore.Field, ArtworkStore.ParseJSON, func(s *stored) **int64 { return &s.ArtworkStoreMb }),
}

var tuningColumns = []column{
	columnOf(UnreleasedEvery.Key, UnreleasedEvery.Field, UnreleasedEvery.ParseJSON, func(s *stored) **int64 { return &s.UnreleasedEverySeconds }),
	columnOf(GrowingEvery.Key, GrowingEvery.Field, GrowingEvery.ParseJSON, func(s *stored) **int64 { return &s.GrowingEverySeconds }),
	columnOf(OthersEvery.Key, OthersEvery.Field, OthersEvery.ParseJSON, func(s *stored) **int64 { return &s.OthersEverySeconds }),
	columnOf(FollowChanges.Key, FollowChanges.Field, FollowChanges.ParseJSON, func(s *stored) **bool { return &s.FollowChanges }),
}

// panel is some of the metadata settings, read from and applied to the
// Controller's copy of metadata_settings.
type panel struct {
	m       *Metadata
	columns []column
}

// Settings is the metadata settings an Admin sets in Settings: language,
// region, the cache, OMDb's ratings and the artwork store.
func (m *Metadata) Settings() setting.Panel { return panel{m, settingColumns} }

// Tuning is the Metadata refresh task's settings: its three tiers and
// whether it follows the provider's changes list.
func (m *Metadata) Tuning() setting.Panel { return panel{m, tuningColumns} }

func (p panel) Fields(context.Context) ([]setting.Field, error) {
	s := p.m.stored()
	out := make([]setting.Field, len(p.columns))
	for i, c := range p.columns {
		out[i] = c.field(s)
	}
	return out, nil
}

// Apply stores the patch, all or none. A new language or region makes
// every Work due at the next refresh; a new region also announces that
// every Work of every type may read differently, since availability reads
// the region. A smaller artwork store trims the store at once.
func (p panel) Apply(ctx context.Context, patch setting.Patch) error {
	m := p.m
	m.applying.Lock()
	defer m.applying.Unlock()
	was := m.stored()
	next := was
	for _, key := range slices.Sorted(maps.Keys(patch)) {
		i := slices.IndexFunc(p.columns, func(c column) bool { return c.key == key })
		if i < 0 {
			return &setting.UnknownKeyError{Key: key}
		}
		if err := p.columns[i].set(&next, patch[key]); err != nil {
			return err
		}
	}
	newRegion := RegionChoice.Value(next.Region) != RegionChoice.Value(was.Region)
	refreshAll := newRegion || LanguageChoice.Value(next.Language) != LanguageChoice.Value(was.Language)
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		if err := q.SetSettings(ctx, db.SetSettingsParams{
			Language: next.Language, Region: next.Region, UnreleasedEverySeconds: next.UnreleasedEverySeconds,
			GrowingEverySeconds: next.GrowingEverySeconds, OthersEverySeconds: next.OthersEverySeconds, FollowChanges: next.FollowChanges,
			RatingsEverySeconds: next.RatingsEverySeconds, CacheForSeconds: next.CacheForSeconds, ArtworkStoreMb: next.ArtworkStoreMb,
		}); err != nil {
			return err
		}
		if !refreshAll {
			return nil
		}
		now := storage.Now()
		for _, t := range m.cfg.Types {
			if err := q.RefreshAll(ctx, db.RefreshAllParams{ContentType: string(t), RefreshAllAt: now}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.settings = next
	m.mu.Unlock()
	if newRegion {
		m.typesChanged(ctx)
	}
	if ArtworkStore.Value(next.ArtworkStoreMb) < ArtworkStore.Value(was.ArtworkStoreMb) {
		return m.gallery.store.Trim(ctx)
	}
	return nil
}

// typesChanged publishes one works.type-changed per Content Type: pages
// read every Work again and the availability timer re-arms.
func (m *Metadata) typesChanged(ctx context.Context) {
	for _, t := range m.cfg.Types {
		m.cfg.TypeChanged.Publish(ctx, works.TypeChanged{ContentType: string(t)})
	}
}

func (m *Metadata) stored() stored {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.settings
}

// Region is the release region a library half reads availability
// against, inside r.
func Region(ctx context.Context, r storage.Reader[storage.Library]) (string, error) {
	s, err := db.New(r).GetSettings(ctx)
	if err != nil {
		return "", err
	}
	return RegionChoice.Value(s.Region), nil
}

// ClearCache has every holder drop its cache on the next call it answers,
// including holders offline now: each call carries the new generation.
func (m *Metadata) ClearCache(identity.Admin) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generation = m.cfg.Rand.Uint64()
}
