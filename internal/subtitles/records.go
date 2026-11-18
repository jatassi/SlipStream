package subtitles

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/pacing"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/subtitling"
)

// This file is the one writer of subtitles_want.unmet and of
// subtitles_file's fact columns: kind, wantless, audio and embedded.
// subtitlestest checks after every step that no other code wrote them.

// Source is where a Sidecar's bytes came from.
type Source string

const (
	FromProvider Source = "provider"
	FromUpload   Source = "upload"
	Found        Source = "found"
	FromSibling  Source = "sibling"
)

// Offer is the candidate a fetched Sidecar was.
type Offer struct {
	Provider  subtitling.ProviderID `json:"provider"`
	Key       string                `json:"key"`
	Release   string                `json:"release,omitempty"`
	AI        bool                  `json:"ai,omitempty"`
	HumanMade bool                  `json:"humanMade,omitempty"`
}

// Fit is the timing the Controller recomputed for a Sidecar.
type Fit struct {
	Offset   time.Duration `json:"offset"`
	Ratio    float64       `json:"ratio"`
	Segments int           `json:"segments"`
	Match    int           `json:"match"`
}

// Sidecar is one subtitles_sidecar row. Its bytes sit at its video's stem
// plus Tail, or at Stray while they are away from home.
type Sidecar struct {
	ID        storage.UUID `json:"-"`
	File      files.ID     `json:"-"`
	Tail      Tail         `json:"-"`
	TailText  string       `json:"tail"`
	Source    Source       `json:"source"`
	Offer     *Offer       `json:"offer,omitempty"`
	SiblingOf storage.UUID `json:"siblingOf"`
	Final     bool         `json:"final"`
	Fit       *Fit         `json:"fit,omitempty"`
	SHA256    [32]byte     `json:"sha256"`
	Size      int64        `json:"size"`
	Cues      int          `json:"cues"`
	TimingDue bool         `json:"timingDue"`
	Stray     string       `json:"stray,omitempty"`
	Added     storage.Time `json:"-"`
}

func (s Sidecar) want() Want { return s.Tail.Want() }

// at is where s's bytes sit when its video is at video.
func (s Sidecar) at(video string) string {
	if s.Stray != "" {
		return s.Stray
	}
	return besideAt(video, s.TailText)
}

// besideAt is the path of a companion with tail beside the video at video.
func besideAt(video, tail string) string { return stemOf(video) + "." + tail }

func stemOf(video string) string {
	slash := strings.LastIndex(video, "/")
	if dot := strings.LastIndex(video, "."); dot > slash {
		return video[:dot]
	}
	return video
}

func sidecarOf(row db.SubtitlesSidecar) (Sidecar, error) {
	l, err := language.Parse(row.Language)
	if err != nil {
		return Sidecar{}, fmt.Errorf("read Sidecar %s: %w", row.ID, err)
	}
	t, ok := ParseTail(row.Tail)
	if !ok {
		t.Ext = row.Tail[strings.LastIndex(row.Tail, ".")+1:]
	}
	t.Language, t.Forced, t.HI = l, row.Forced, row.Hi
	s := Sidecar{
		ID: row.ID, File: row.FileID, Tail: t, TailText: row.Tail, Source: Source(row.Source), SiblingOf: row.SiblingOf, Final: row.Final,
		Size: row.Size, Cues: int(row.Cues), TimingDue: row.TimingDue, Added: row.AddedAt,
	}
	copy(s.SHA256[:], row.Sha256)
	if row.Stray != nil {
		s.Stray = *row.Stray
	}
	if row.Provider != nil {
		s.Offer = &Offer{Provider: subtitling.ProviderID(*row.Provider), Key: deref(row.CandidateKey), Release: deref(row.Release), AI: row.AiTranslated, HumanMade: row.HumanMade}
	}
	if row.Timed {
		s.Fit = &Fit{Offset: time.Duration(deref(row.OffsetMs)) * time.Millisecond, Ratio: deref(row.Ratio), Segments: int(deref(row.Segments)), Match: int(deref(row.Match))}
	}
	return s, nil
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

func some[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

// putFacts writes f's fact columns for s, inserting its row when it has
// none.
func putFacts(ctx context.Context, q *db.Queries, s files.Subject, f facts, now storage.Time) error {
	return q.PutFacts(ctx, db.PutFactsParams{
		FileID: s.ID, WorkID: s.Work, RootID: s.Root, Kind: kindColumn(f.Kind), Wantless: f.Wantless,
		Audio: f.audioColumn(), Embedded: f.embeddedColumn(), MapSince: now,
	})
}

var kindNames = map[subtitling.Kind]string{subtitling.Feature: "feature", subtitling.Episode: "episode"}

func kindColumn(k subtitling.Kind) *string {
	if n, ok := kindNames[k]; ok {
		return &n
	}
	return nil
}

func kindOf(s *string) subtitling.Kind {
	for k, n := range kindNames {
		if s != nil && *s == n {
			return k
		}
	}
	return 0
}

// pace is how a new want row's first search is spread: over interval
// from when its Wanted Language was added.
type pace struct {
	interval time.Duration
	added    func(Want) storage.Time
}

func (p pace) firstDue(file files.ID, w Want) storage.Time {
	return storage.FromTime(p.added(w).Time().Add(pacing.Spread(file.String(), w.String(), p.interval)))
}

// putWants makes s's want rows the searchable wants of ws: it inserts the
// missing ones, first due by p unless a final Sidecar meets them, and
// deletes the ones no longer searchable. had is the rows s has now.
func putWants(ctx context.Context, q *db.Queries, s files.Subject, f facts, ws []want, had []db.SubtitlesWant, final func(Want) bool, p pace) error {
	keep := map[Want]bool{}
	for _, w := range ws {
		if !w.searchable() {
			continue
		}
		keep[w.Want()] = true
		var due *storage.Time
		if !final(w.Want()) {
			at := p.firstDue(s.ID, w.Want())
			due = &at
		}
		err := q.PutWant(ctx, db.PutWantParams{
			FileID: s.ID, Language: w.Language.String(), Forced: w.Forced, WorkID: s.Work, Kind: kindNames[f.Kind],
			DueAt: deref(due), GapSeconds: int64(p.interval / time.Second),
		})
		if err != nil {
			return err
		}
	}
	for _, row := range had {
		l, err := language.Parse(row.Language)
		if err != nil || !keep[Want{l, row.Forced}] {
			if err := q.DeleteWant(ctx, db.DeleteWantParams{FileID: row.FileID, Language: row.Language, Forced: row.Forced}); err != nil {
				return err
			}
		}
	}
	return nil
}

// putSidecar writes s's row and its want's unmet flag.
func putSidecar(ctx context.Context, q *db.Queries, s Sidecar, now storage.Time) error {
	p := db.PutSidecarParams{
		ID: s.ID, FileID: s.File, Language: s.Tail.Language.String(), Forced: s.Tail.Forced, Hi: s.Tail.HI, Tail: s.TailText, Source: string(s.Source),
		SiblingOf: s.SiblingOf, Final: s.Final, Timed: s.Fit != nil, Sha256: s.SHA256[:], Size: s.Size, Cues: int64(s.Cues), TimingDue: s.TimingDue,
		Stray: some(s.Stray), AddedAt: s.Added,
	}
	if p.AddedAt.IsZero() {
		p.AddedAt = now
	}
	if o := s.Offer; o != nil {
		p.Provider, p.CandidateKey, p.Release, p.AiTranslated, p.HumanMade = some(string(o.Provider)), some(o.Key), some(o.Release), o.AI, o.HumanMade
	}
	if f := s.Fit; f != nil {
		p.OffsetMs, p.Ratio, p.Segments, p.Match = new(f.Offset.Milliseconds()), &f.Ratio, new(int64(f.Segments)), new(int64(f.Match))
	}
	if err := q.PutSidecar(ctx, p); err != nil {
		return err
	}
	return setUnmet(ctx, q, s.File, s.want(), now)
}

// dropSidecar deletes s's row while it still holds s's bytes, and sets its
// want's unmet flag. It answers false when the row changed or went.
func dropSidecar(ctx context.Context, q *db.Queries, s Sidecar, now storage.Time) (bool, error) {
	gone, err := q.DeleteSidecar(ctx, db.DeleteSidecarParams{ID: s.ID, Sha256: s.SHA256[:]})
	if err != nil || len(gone) == 0 {
		return false, err
	}
	return true, setUnmet(ctx, q, s.File, s.want(), now)
}

func setUnmet(ctx context.Context, q *db.Queries, f files.ID, w Want, now storage.Time) error {
	return q.SetUnmet(ctx, db.SetUnmetParams{FileID: f, Language: w.Language.String(), Forced: w.Forced, Now: now})
}

// homeOf is where a companion at path sits relative to video: its tail
// when it sits beside the video's own stem.
func homeOf(video string, path storing.Rel) (string, bool) {
	return strings.CutPrefix(path.String(), stemOf(video)+".")
}

func flag(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// allowanceOf is the allowance p's last OK answer reported, nil when none
// did.
func allowanceOf(p db.SubtitlesProvider) (*calls.Allowance, error) {
	if p.Allowance == nil {
		return nil, nil
	}
	var a calls.Allowance
	if err := json.Unmarshal([]byte(*p.Allowance), &a); err != nil {
		return nil, err
	}
	return &a, nil
}
