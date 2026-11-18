package subtitles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/subtitles/cues"

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

var (
	ErrSearchExpired  = errors.New("the subtitle search expired")
	ErrAttemptExpired = errors.New("the subtitle attempt expired")
	ErrUploadExpired  = errors.New("the refused upload expired")
	ErrImage          = errors.New("image subtitles can't be timed")
	// ErrNotScanned is a pick or an upload on a file whose audio isn't
	// scanned yet, which Timing sync needs.
	ErrNotScanned = errors.New("the file's audio isn't scanned yet")
)

// Reason is a ranking step a candidate meets.
type Reason uint8

const (
	MadeByPerson Reason = iota + 1
	SameGroup
	SameSource
	ProviderOrder
)

// Tried is an earlier try of a candidate for the same want.
type Tried struct {
	Outcome TriedOutcome
	Match   int
	At      time.Time
}

// TriedOutcome is how a try ended, a subtitles_tried outcome.
type TriedOutcome string

const (
	TriedPassed       TriedOutcome = "passed"
	TriedFailedTiming TriedOutcome = "failed_timing"
	TriedMarkedBad    TriedOutcome = "marked_bad"
	TriedUnreadable   TriedOutcome = "unreadable"
	TriedImage        TriedOutcome = "image"
	TriedWrongEpisode TriedOutcome = "wrong_episode"
	TriedTooLarge     TriedOutcome = "too_large"
)

// Candidate is one ranked offer of a Manual subtitle search. A marked bad
// one can be picked, which undoes the Mark bad.
type Candidate struct {
	Key      string
	Provider subtitling.ProviderID
	Release  string
	Reasons  []Reason
	Tried    *Tried
	Disabled bool
}

// Candidates is a Manual subtitle search so far.
type Candidates struct {
	Done       bool
	Candidates []Candidate
	Outcomes   []calls.Outcome
}

// Step is where a pick or an upload is.
type Step uint8

const (
	Downloading Step = iota + 1
	Checking
	Finished
)

// Ended is how a finished pick or upload ended.
type Ended uint8

const (
	Added Ended = iota + 1
	FailedTiming
	Unreadable
	Image
	WrongEpisode
	MarkedBad
	// Unfetched is a download the provider didn't serve. Attempt.Answer
	// says why.
	Unfetched
)

var endedOf = map[TriedOutcome]Ended{
	TriedFailedTiming: FailedTiming, TriedUnreadable: Unreadable, TriedImage: Image, TriedWrongEpisode: WrongEpisode, TriedTooLarge: Unreadable,
}

// Attempt is a pick's or an upload's step, and its outcome once finished.
// Upload is set on an upload that failed and can be kept anyway, and
// Answer on a pick that ended Unfetched.
type Attempt struct {
	Step      Step
	Provider  subtitling.ProviderID
	Ended     Ended
	Match     int
	Threshold int
	Upload    storage.UUID
	Answer    *calls.Outcome
}

// held is what the Controller holds in memory for a Timings bound: Manual
// subtitle searches, refused uploads and the steps of picks and uploads.
type held struct {
	searches expiring[*manualSearch]
	uploads  expiring[refusedUpload]
	attempts expiring[*shownAttempt]
}

type expiring[T any] struct {
	mu sync.Mutex
	of map[storage.UUID]kept[T]
}

type kept[T any] struct {
	v     T
	until time.Time
}

// put holds v until until, and lets go of everything past its own.
func (e *expiring[T]) put(id storage.UUID, v T, now, until time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.of == nil {
		e.of = map[storage.UUID]kept[T]{}
	}
	maps.DeleteFunc(e.of, func(_ storage.UUID, k kept[T]) bool { return !now.Before(k.until) })
	e.of[id] = kept[T]{v: v, until: until}
}

func (e *expiring[T]) get(id storage.UUID, now time.Time) (T, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	k, ok := e.of[id]
	if !ok || !now.Before(k.until) {
		var zero T
		return zero, false
	}
	return k.v, true
}

func (e *expiring[T]) drop(id storage.UUID) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.of, id)
}

// manualSearch is one Manual subtitle search's ranked offers, once its
// Search answers.
type manualSearch struct {
	key   WantKey
	tried []db.SubtitlesTried

	mu       sync.Mutex
	done     bool
	ranked   []ranked
	outcomes []calls.Outcome
}

func candidateKey(o subtitling.Offer) string { return string(o.Provider) + ":" + o.Key }

func (m *manualSearch) answered(ranked []ranked, outcomes []calls.Outcome) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.done, m.ranked, m.outcomes = true, ranked, outcomes
}

func (m *manualSearch) candidate(key string) (ranked, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	i := slices.IndexFunc(m.ranked, func(x ranked) bool { return candidateKey(x.offer) == key })
	if i < 0 {
		return ranked{}, false
	}
	return m.ranked[i], true
}

// view is m's candidates best first, a paused provider's and an image
// one disabled.
func (m *manualSearch) view(ps []Provider, now time.Time) Candidates {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := Candidates{Done: m.done, Candidates: []Candidate{}, Outcomes: m.outcomes}
	top := len(ps)
	for _, x := range m.ranked {
		top = min(top, x.place)
	}
	for _, x := range m.ranked {
		o := x.offer
		c := Candidate{Key: candidateKey(o), Provider: o.Provider, Release: first(o.Releases), Reasons: reasons(x, top), Disabled: x.floor == imageOnly}
		if i := slices.IndexFunc(ps, func(p Provider) bool { return p.Spec.ID == o.Provider }); i >= 0 && !ps[i].ready(now) {
			c.Disabled = true
		}
		if i := slices.IndexFunc(m.tried, func(t db.SubtitlesTried) bool {
			return t.CandidateKey == o.Key && (t.Provider == nil || *t.Provider == string(o.Provider))
		}); i >= 0 {
			t := m.tried[i]
			c.Tried = &Tried{Outcome: TriedOutcome(t.Outcome), Match: int(deref(t.Match)), At: t.TriedAt.Time()}
		}
		out.Candidates = append(out.Candidates, c)
	}
	return out
}

// shownAttempt is a pick's or an upload's state, which getSubtitleAttempt
// reads and subtitles.attempt says moved.
type shownAttempt struct {
	id   storage.UUID
	file files.ID

	mu  sync.Mutex
	now Attempt
}

func (sh *shownAttempt) read() Attempt {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	return sh.now
}

// refusedUpload is an upload that failed Timing sync, held so the Admin
// can keep it anyway.
type refusedUpload struct {
	key   WantKey
	bytes []byte
}

// step moves a pick's or an upload's step.
func (a *attempt) step(ctx context.Context, st Step) {
	if a.shown == nil {
		return
	}
	a.shown.mu.Lock()
	a.shown.now.Step = st
	a.shown.mu.Unlock()
	a.s.moved(ctx, a.shown)
}

// finish ends a pick or an upload with x.
func (a *attempt) finish(ctx context.Context, x Attempt) {
	a.shown.mu.Lock()
	x.Step, x.Provider = Finished, a.shown.now.Provider
	a.shown.now = x
	a.shown.mu.Unlock()
	a.s.moved(ctx, a.shown)
}

func (s *Subtitles) moved(ctx context.Context, sh *shownAttempt) {
	s.cfg.Attempt.Publish(ctx, Attempted{AttemptID: sh.id.String(), FileID: sh.file.String()})
}

// show holds a new pick's or upload's state at x for Timings.AttemptHold.
func (s *Subtitles) show(k WantKey, x Attempt) *shownAttempt {
	sh := &shownAttempt{id: storage.NewUUID(), file: k.File, now: x}
	now := s.cfg.Clock()
	s.held.attempts.put(sh.id, sh, now, now.Add(s.cfg.Timings.AttemptHold))
	return sh
}

// look reads what an attempt on k decides from, holding no lease.
func (s *Subtitles) look(ctx context.Context, k WantKey) (*attempt, error) {
	a := &attempt{s: s, key: k}
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		if _, err := db.New(r).GetWant(ctx, wantParams(k)); errors.Is(err, storage.ErrNoRows) {
			return ErrNotWanted
		} else if err != nil {
			return err
		}
		return a.read(ctx, r)
	})
	if err != nil {
		return nil, err
	}
	return a, nil
}

// SearchNowFor searches k at the Admin's rank, ahead of the schedule.
func (s *Subtitles) SearchNowFor(ctx context.Context, _ identity.Admin, k WantKey) error {
	var work works.WorkID
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		if work, err = s.workOf(ctx, tx, k.File); err != nil {
			return err
		}
		return s.SearchNow(ctx, tx, []files.ID{k.File}, k.Want, ByAdmin)
	})
	if err != nil {
		return err
	}
	s.cfg.Changed.Publish(ctx, changedOf([]works.WorkID{work}, []files.ID{k.File}))
	return nil
}

func (s *Subtitles) workOf(ctx context.Context, r storage.Reader[storage.Library], f files.ID) (works.WorkID, error) {
	subjects, err := s.cfg.Files.SubjectsOf(ctx, r, []files.ID{f})
	if err != nil {
		return works.WorkID{}, err
	}
	if len(subjects) == 0 {
		return works.WorkID{}, fmt.Errorf("file %s: %w", f, files.ErrNoFile)
	}
	return subjects[0].Work, nil
}

// StartCandidates starts a Manual subtitle search on k: one Search on
// every ready provider of the file's kind, never waiting in the queue,
// held for Timings.CandidatesHold.
func (s *Subtitles) StartCandidates(ctx context.Context, _ identity.Admin, k WantKey) (storage.UUID, error) {
	a, err := s.look(ctx, k)
	if err != nil {
		return storage.UUID{}, err
	}
	id, now := storage.NewUUID(), s.cfg.Clock()
	m := &manualSearch{key: k, tried: a.tried}
	s.held.searches.put(id, m, now, now.Add(s.cfg.Timings.CandidatesHold))
	go s.searchFor(context.WithoutCancel(ctx), a, m)
	return id, nil
}

func (s *Subtitles) searchFor(ctx context.Context, a *attempt, m *manualSearch) {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Timings.AttemptBound)
	defer cancel()
	var offers []subtitling.Offer
	var outcomes []calls.Outcome
	if !a.noAired {
		if ps := a.searchers(); len(ps) > 0 {
			var err error
			if offers, outcomes, err = a.ask(ctx, ps); err != nil {
				s.cfg.Log.WarnContext(ctx, "A Manual subtitle search didn't reach the Subtitles Node", "file", a.key.File, "error", err)
			}
		}
		outcomes = a.endings(outcomes)
	}
	m.answered(a.ranking().rank(offers, true), outcomes)
}

// endings is how the search ended on each provider that searches the
// file's kind, in search order: its last call's outcome, or its pause
// when it was out and not asked. One the search didn't reach is
// unreachable.
func (a *attempt) endings(os []calls.Outcome) []calls.Outcome {
	now := a.s.cfg.Clock()
	var out []calls.Outcome
	for _, p := range a.providers {
		if !p.searches() || !slices.Contains(p.Spec.Kinds, a.kind()) {
			continue
		}
		o, ok := lastOutcome(os, p.Spec.ID)
		switch {
		case ok:
		case p.paused(now):
			o = p.Paused.outcome(p.Spec.ID)
		default:
			o = calls.Outcome{Provider: p.Spec.ID, Kind: calls.Unreachable}
		}
		out = append(out, o)
	}
	return out
}

func lastOutcome(os []calls.Outcome, id subtitling.ProviderID) (calls.Outcome, bool) {
	for i := len(os) - 1; i >= 0; i-- {
		if os[i].Provider == id {
			return os[i], true
		}
	}
	return calls.Outcome{}, false
}

// Candidates is search's candidates so far.
func (s *Subtitles) Candidates(ctx context.Context, _ identity.Admin, k WantKey, search storage.UUID) (Candidates, error) {
	m, ok := s.held.searches.get(search, s.cfg.Clock())
	if !ok || m.key != k {
		return Candidates{}, ErrSearchExpired
	}
	var ps []Provider
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		ps, err = s.providersIn(ctx, r)
		return err
	})
	if err != nil {
		return Candidates{}, err
	}
	return m.view(ps, s.cfg.Clock()), nil
}

// Pick downloads search's candidate key and times it against the file,
// taking k's lease ahead of Subtitle search. A pass becomes the want's
// Sidecar, and a pick of a candidate marked bad undoes the Mark bad. A
// failure goes to Tried before.
func (s *Subtitles) Pick(ctx context.Context, _ identity.Admin, k WantKey, search storage.UUID, key string) (storage.UUID, error) {
	m, ok := s.held.searches.get(search, s.cfg.Clock())
	if !ok || m.key != k {
		return storage.UUID{}, ErrSearchExpired
	}
	x, ok := m.candidate(key)
	if !ok {
		return storage.UUID{}, ErrSearchExpired
	}
	if x.floor == imageOnly {
		return storage.UUID{}, ErrImage
	}
	row, err := s.fileRow(ctx, k)
	if err != nil {
		return storage.UUID{}, err
	}
	if row.MapState != "built" {
		return storage.UUID{}, ErrNotScanned
	}
	sh := s.show(k, Attempt{Step: Downloading, Provider: x.offer.Provider})
	go s.manual(context.WithoutCancel(ctx), k, sh, func(ctx context.Context, a *attempt) error { return a.picked(ctx, x.offer) })
	return sh.id, nil
}

// fileRow is k's file's row, once k is a want it searches.
func (s *Subtitles) fileRow(ctx context.Context, k WantKey) (db.SubtitlesFile, error) {
	var row db.SubtitlesFile
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		q := db.New(r)
		if _, err := q.GetWant(ctx, wantParams(k)); errors.Is(err, storage.ErrNoRows) {
			return ErrNotWanted
		} else if err != nil {
			return err
		}
		rows, err := q.ListFilesIn(ctx, []files.ID{k.File})
		if err != nil {
			return err
		}
		row = rows[0]
		return nil
	})
	return row, err
}

// manual runs fn under k's lease, taken ahead of Subtitle search, with the
// file's map read. One that ends in an error is dropped, so reading it
// says it expired.
func (s *Subtitles) manual(ctx context.Context, k WantKey, sh *shownAttempt, fn func(context.Context, *attempt) error) {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Timings.AttemptBound)
	defer cancel()
	err := func() error {
		a, err := s.leaseFirst(ctx, k)
		if err != nil {
			return err
		}
		defer a.release(ctx)
		a.shown = sh
		if err := a.mapped(ctx); err != nil {
			return err
		}
		return fn(ctx, a)
	}()
	if err != nil {
		s.cfg.Log.WarnContext(ctx, "A subtitle download or upload failed", "file", k.File, "want", k.Want, "error", err)
		s.held.attempts.drop(sh.id)
		s.moved(ctx, sh)
	}
}

func (a *attempt) picked(ctx context.Context, o subtitling.Offer) error {
	j, err := a.fetch(ctx, o)
	if p, paused := errors.AsType[*pausedError](err); paused {
		a.finish(ctx, Attempt{Ended: Unfetched, Answer: p.answer})
		return nil
	}
	if err != nil {
		return err
	}
	if j.checked != nil {
		a.s.hold(a, o, j.members)
		if err := a.pass(ctx, *j.checked, ending{}); err != nil {
			return err
		}
		a.finish(ctx, Attempt{Ended: Added, Match: j.match})
		return nil
	}
	if j.unfetched != nil {
		a.finish(ctx, Attempt{Ended: Unfetched, Answer: j.unfetched})
		return nil
	}
	if err := a.note(ctx, []try{*j.try}); err != nil {
		return err
	}
	a.finish(ctx, Attempt{Ended: endedOf[j.try.outcome], Match: j.match, Threshold: percent(a.gate.Match)})
	return nil
}

// Upload times b against the file as k's subtitle, taking k's lease ahead
// of Subtitle search. A pass becomes the want's Sidecar, final. A failure
// is held for Timings.UploadHold so the Admin can keep it anyway, as is
// any upload to a file with no audio track. It refuses bytes over
// Timings.UploadCap with *TooLargeError and image subtitles with
// ErrImage.
func (s *Subtitles) Upload(ctx context.Context, _ identity.Admin, k WantKey, name string, b []byte) (storage.UUID, error) {
	if err := s.uploadShape(b); err != nil {
		return storage.UUID{}, err
	}
	row, err := s.fileRow(ctx, k)
	if err != nil {
		return storage.UUID{}, err
	}
	_, err = parseUpload(b, k.Want, s.cfg.Timings.UploadCap)
	_, unreadable := errors.AsType[cues.Unreadable](err)
	switch {
	case errors.Is(err, cues.ErrImage):
		return storage.UUID{}, ErrImage
	case unreadable:
		return s.show(k, Attempt{Step: Finished, Ended: Unreadable}).id, nil
	case err != nil:
		return storage.UUID{}, err
	case row.MapState == "failed" && deref(row.MapFault) == "no_audio":
		return s.show(k, Attempt{Step: Finished, Ended: FailedTiming, Upload: s.refuse(k, b)}).id, nil
	case row.MapState != "built":
		return storage.UUID{}, ErrNotScanned
	}
	sh := s.show(k, Attempt{Step: Checking})
	go s.manual(context.WithoutCancel(ctx), k, sh, func(ctx context.Context, a *attempt) error { return a.uploaded(ctx, name, b) })
	return sh.id, nil
}

// refuse holds b for Timings.UploadHold, for KeepUpload.
func (s *Subtitles) refuse(k WantKey, b []byte) storage.UUID {
	id, now := storage.NewUUID(), s.cfg.Clock()
	s.held.uploads.put(id, refusedUpload{key: k, bytes: b}, now, now.Add(s.cfg.Timings.UploadHold))
	return id
}

func (a *attempt) uploaded(ctx context.Context, name string, b []byte) error {
	sha := sha256.Sum256(b)
	template := Sidecar{Tail: Tail{Language: a.key.Language, Forced: a.key.Forced}, Source: FromUpload, Final: true}
	j, err := a.sync(ctx, hex.EncodeToString(sha[:]), name, b, nil, template)
	if err != nil {
		return err
	}
	if j.checked != nil {
		if err := a.pass(ctx, *j.checked, ending{}); err != nil {
			return err
		}
		a.finish(ctx, Attempt{Ended: Added, Match: j.match})
		return nil
	}
	x := Attempt{Ended: Unreadable, Match: j.match, Threshold: percent(a.gate.Match)}
	if j.try != nil {
		x.Ended = endedOf[j.try.outcome]
	}
	if x.Ended == FailedTiming {
		x.Upload = a.s.refuse(a.key, b)
	}
	a.finish(ctx, x)
	return nil
}

// KeepUpload keeps a refused upload untimed and final, as its want's
// Sidecar.
func (s *Subtitles) KeepUpload(ctx context.Context, _ identity.Admin, f files.ID, upload storage.UUID) error {
	u, ok := s.held.uploads.get(upload, s.cfg.Clock())
	if !ok || u.key.File != f {
		return ErrUploadExpired
	}
	c, err := parseUpload(u.bytes, u.key.Want, s.cfg.Timings.UploadCap)
	if err != nil {
		return err
	}
	a, err := s.leaseFirst(ctx, u.key)
	if err != nil {
		return err
	}
	defer a.release(ctx)
	if err := a.pass(ctx, c, ending{}); err != nil {
		return err
	}
	s.held.uploads.drop(upload)
	return nil
}

// AttemptOf is a pick's or an upload's step, and its outcome once
// finished.
func (s *Subtitles) AttemptOf(_ context.Context, _ identity.Admin, f files.ID, id storage.UUID) (Attempt, error) {
	sh, ok := s.held.attempts.get(id, s.cfg.Clock())
	if !ok || sh.file != f {
		return Attempt{}, ErrAttemptExpired
	}
	return sh.read(), nil
}

// MarkBadNow marks k's Sidecar bad for the Admin.
func (s *Subtitles) MarkBadNow(ctx context.Context, by identity.Admin, k WantKey) error {
	var work works.WorkID
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		if work, err = s.workOf(ctx, tx, k.File); err != nil {
			return err
		}
		return s.MarkBad(ctx, tx, []files.ID{k.File}, k.Want, actorOf(by))
	})
	if err != nil {
		return err
	}
	s.cfg.Changed.Publish(ctx, changedOf([]works.WorkID{work}, []files.ID{k.File}))
	return nil
}

func actorOf(by identity.Admin) entry.Actor {
	a, ok := by.Account()
	if !ok {
		return entry.Actor{APIKey: true}
	}
	return entry.Actor{Name: a.Profile().DisplayName}
}
