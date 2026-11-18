package subtitles

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/subtitling"
)

// MissingReason is why a Missing want has no subtitle yet, the first that
// holds in this order.
type MissingReason uint8

const (
	NoHolder MissingReason = iota + 1
	NoProvider
	Paused
	NoAiredNumber
	MapFailed
	MapDue
	NonePassed
	NothingFound
	NeverSearched
)

// State is what a want reads as now: the same answer the Work page,
// Subtitle search and S47 (Subtitle reports) read.
type State struct {
	Kind StateKind
	// Why is set on NotNeeded, Track on Embedded: the track's ordinal in
	// the media record's subtitle tracks.
	Why   NotNeededWhy
	Track int
	// Sidecar is set on Met. Its Final says whether search goes on.
	Sidecar *Sidecar
	Missing MissingReason
	// Next is the want's next search, zero when none is planned.
	Next time.Time
}

// world is what a want's State reads beyond its own rows.
type world struct {
	holders bool
	// ready is whether a provider that covers the kind is on, keyed if it
	// needs a key, and unpaused; covered whether one is on at all.
	ready, covered bool
	noAired        bool
}

// stateOf finishes w's State from its row, its file's row and the world.
func stateOf(w want, row *db.SubtitlesWant, file *db.SubtitlesFile, sc *Sidecar, wd world) State {
	st := State{Kind: w.Kind, Why: w.Why, Track: w.Track}
	switch w.Kind {
	case NotNeeded, Embedded:
		return st
	case Met:
		st.Sidecar = sc
	case Missing:
		st.Missing = missingOf(row, file, wd)
	}
	if row != nil && !row.DueAt.IsZero() {
		st.Next = row.DueAt.Time()
	}
	return st
}

func missingOf(row *db.SubtitlesWant, file *db.SubtitlesFile, wd world) MissingReason {
	switch {
	case !wd.holders:
		return NoHolder
	case !wd.covered:
		return NoProvider
	case !wd.ready:
		return Paused
	case wd.noAired:
		return NoAiredNumber
	case file.MapState == "failed":
		return MapFailed
	case file.MapState != "built":
		return MapDue
	}
	switch {
	case row == nil || row.LastResult == nil:
		return NeverSearched
	case *row.LastResult == "none_passed":
		return NonePassed
	case *row.LastResult == "nothing_found":
		return NothingFound
	}
	return NeverSearched
}

// Roster is which Nodes hold a Role, and each Node's name.
type Roster interface {
	Holding(name role.Name) []role.NodeID
	Name(id role.NodeID) string
}

// Outcome is what w of file f reads as now, in r.
func (s *Subtitles) Outcome(ctx context.Context, r storage.Reader[storage.Library], f files.ID, w Want) (State, error) {
	subjects, err := s.cfg.Files.SubjectsOf(ctx, r, []files.ID{f})
	if err != nil {
		return State{}, err
	}
	if len(subjects) == 0 {
		return State{}, fmt.Errorf("file %s: %w", f, files.ErrNoFile)
	}
	sub := subjects[0]
	c, err := s.read(ctx, r, subjects)
	if err != nil {
		return State{}, err
	}
	q := db.New(r)
	rows, err := q.ListFilesIn(ctx, []files.ID{f})
	if err != nil || len(rows) == 0 {
		return State{}, err
	}
	var ignored []ignore
	for _, ig := range c.ignored[f] {
		ignored = append(ignored, ignore{Ordinal: int(ig.StreamIndex), Codec: ig.Codec, Language: release.Language(ig.Language), Forced: ig.Forced})
	}
	facet := s.cfg.Types[sub.Type]
	fa := factsOf(sub, kindOfType(facet), ignored)
	scs := c.sidecars[f]
	ws := wantsOf(fa, c.langs[sub.Profile], func(w Want) bool { return sidecarOfWant(scs, w) != nil })
	i := slices.IndexFunc(ws, func(x want) bool { return x.Want() == w })
	if i < 0 {
		return State{}, fmt.Errorf("file %s doesn't want %s subtitles", f, w)
	}
	var row *db.SubtitlesWant
	for j := range c.wants[f] {
		if x := c.wants[f][j]; x.Language == w.Language.String() && x.Forced == w.Forced {
			row = &c.wants[f][j]
		}
	}
	wd, err := s.worldOf(ctx, r, fa.Kind)
	if err != nil {
		return State{}, err
	}
	_, err = facet.SubtitleSubject(ctx, r, sub.Work, sub.Items)
	wd.noAired = errors.Is(err, ErrNoAiredNumber)
	if err != nil && !wd.noAired {
		return State{}, err
	}
	return stateOf(ws[i], row, &rows[0], sidecarOfWant(scs, w), wd), nil
}

func (s *Subtitles) worldOf(ctx context.Context, r storage.Reader[storage.Library], k subtitling.Kind) (world, error) {
	wd := world{holders: len(s.cfg.Roster.Holding(role.Subtitles)) > 0}
	providers, err := db.New(r).ListProviders(ctx)
	if err != nil {
		return world{}, err
	}
	now := storage.FromTime(s.cfg.Clock())
	for _, p := range providers {
		spec := s.spec(p.ID)
		if !p.IsOn || !slices.Contains(spec.Kinds, k) {
			continue
		}
		wd.covered = true
		if (!spec.Keyed || p.ApiKeySealed.Set()) && p.PausedUntil.Compare(now) <= 0 {
			wd.ready = true
		}
	}
	return wd, nil
}
