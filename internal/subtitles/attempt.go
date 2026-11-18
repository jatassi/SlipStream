package subtitles

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/pacing"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/subtitles/timing"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

var (
	// ErrBusy is a want another attempt, an intent or a carried Sidecar
	// holds.
	ErrBusy = errors.New("another search or write holds the subtitle")
	// ErrNotWanted is a want the file has no search state for: nothing
	// wants it, or an embedded track or the audio meets it.
	ErrNotWanted = errors.New("the file doesn't search for that subtitle")
)

// attempt is one try at a want under its lease: the automatic search,
// a pick, an upload or a kept upload. Every Write it makes compares its
// lease, so one whose want a doom, a language drop or a restart took ends
// with errStale and no effect.
type attempt struct {
	s   *Subtitles
	id  storage.UUID
	key WantKey

	sub       files.Subject
	file      db.SubtitlesFile
	row       db.SubtitlesWant
	wanted    targets.WantedLanguage
	current   *Sidecar
	tried     []db.SubtitlesTried
	settings  db.SubtitlesSetting
	gate      timing.Gate
	providers []Provider
	interval  time.Duration
	capped    time.Duration
	subject   subtitling.Subject
	noAired   bool

	ref timing.Reference
	m   []byte
	// shown is the attempt's state a pick or an upload shows, nil for
	// Subtitle search's.
	shown *shownAttempt
}

// lease takes k's lease in a Write and reads, in the same Write,
// everything the attempt decides from.
func (s *Subtitles) lease(ctx context.Context, k WantKey) (*attempt, error) {
	a := &attempt{s: s, id: storage.NewUUID(), key: k}
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		n, err := q.TakeLease(ctx, db.TakeLeaseParams{AttemptID: a.id, FileID: k.File, Language: k.Language.String(), Forced: k.Forced})
		if err != nil {
			return err
		}
		if n == 0 {
			if _, err := q.GetWant(ctx, wantParams(k)); errors.Is(err, storage.ErrNoRows) {
				return ErrNotWanted
			}
			return ErrBusy
		}
		return a.read(ctx, tx)
	})
	if err != nil {
		return nil, err
	}
	return a, nil
}

func wantParams(k WantKey) db.GetWantParams {
	return db.GetWantParams{FileID: k.File, Language: k.Language.String(), Forced: k.Forced}
}

func (a *attempt) read(ctx context.Context, r storage.Reader[storage.Library]) error {
	s, k, q := a.s, a.key, db.New(r)
	subjects, err := s.cfg.Files.SubjectsOf(ctx, r, []files.ID{k.File})
	if err != nil {
		return err
	}
	rows, err := q.ListFilesIn(ctx, []files.ID{k.File})
	if err != nil {
		return err
	}
	if len(subjects) == 0 || len(rows) == 0 {
		return ErrNotWanted
	}
	a.sub, a.file = subjects[0], rows[0]
	if a.row, err = q.GetWant(ctx, wantParams(k)); err != nil {
		return err
	}
	langs, err := s.cfg.Targets.LanguagesOf(ctx, r)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(langs[a.sub.Profile], func(l targets.WantedLanguage) bool { return l.Language == k.Language && l.Forced == k.Forced })
	if i < 0 {
		return ErrNotWanted
	}
	a.wanted = langs[a.sub.Profile][i]
	scs, err := sidecarsIn(ctx, q, []files.ID{k.File})
	if err != nil {
		return err
	}
	a.current = sidecarOfWant(scs[k.File], k.Want)
	if a.tried, err = q.ListTried(ctx, db.ListTriedParams{FileID: k.File, Language: k.Language.String(), Forced: k.Forced}); err != nil {
		return err
	}
	if a.settings, err = q.GetSettings(ctx); err != nil {
		return err
	}
	if a.gate, err = s.gate(ctx, r); err != nil {
		return err
	}
	if a.providers, err = s.providersIn(ctx, r); err != nil {
		return err
	}
	every, err := s.cfg.Tasks.EveryIn(ctx, r, TaskID)
	if err != nil {
		return err
	}
	a.interval = Every.Value(every)
	if c, ok := s.cfg.GapCap(ctx, r, k); ok {
		a.capped = c
	}
	a.subject, err = s.cfg.Types[a.sub.Type].SubtitleSubject(ctx, r, a.sub.Work, a.sub.Items)
	a.noAired = errors.Is(err, ErrNoAiredNumber)
	if a.noAired {
		return nil
	}
	return err
}

// release ends the lease, on a context of its own, and lets a pick or an
// upload waiting for the want go.
func (a *attempt) release(ctx context.Context) {
	ctx = context.WithoutCancel(ctx)
	err := a.s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		k := a.key
		return db.New(tx).ReleaseLease(ctx, db.ReleaseLeaseParams{FileID: k.File, Language: k.Language.String(), Forced: k.Forced, AttemptID: a.id})
	})
	if err != nil {
		a.s.cfg.Log.WarnContext(ctx, "A subtitle search's lease wasn't released", "file", a.key.File, "want", a.key.Want, "error", err)
	}
	a.s.leases.free(a.key)
}

// write runs fn in one Write that holds only while the attempt still
// holds its lease.
func (a *attempt) write(ctx context.Context, fn func(ctx context.Context, tx *storage.Tx[storage.Library], q *db.Queries) error) error {
	return a.s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		k := a.key
		held, err := q.HoldsLease(ctx, db.HoldsLeaseParams{FileID: k.File, Language: k.Language.String(), Forced: k.Forced, AttemptID: a.id})
		if err != nil {
			return err
		}
		if held == 0 {
			return errStale
		}
		return fn(ctx, tx, q)
	})
}

// told folds a holder's outcomes into the providers' rows, under the
// lease when the attempt holds one. A Manual subtitle search holds none.
func (a *attempt) told(ctx context.Context, node role.NodeID, os []calls.Outcome) error {
	var changed bool
	fold := func(ctx context.Context, tx *storage.Tx[storage.Library], _ *db.Queries) error {
		var err error
		changed, err = a.s.told(ctx, tx, node, os)
		return err
	}
	var err error
	if a.id.IsZero() {
		err = a.s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error { return fold(ctx, tx, nil) })
	} else {
		err = a.write(ctx, fold)
	}
	if changed {
		a.s.cfg.ProvidersChanged.Publish(ctx, ProvidersChanged{})
	}
	return err
}

// mapped reads the file's map, which every timing runs against.
func (a *attempt) mapped(ctx context.Context) error {
	var err error
	a.ref, a.m, err = a.s.speechMap(ctx, a.file, a.sub)
	return err
}

func (a *attempt) query() subtitling.Query {
	return subtitling.Query{Subject: a.subject, Language: a.key.Language, Forced: a.key.Forced}
}

func (a *attempt) kind() subtitling.Kind { return kindOf(a.file.Kind) }

// searchers is the providers a search asks, in order: ready ones that
// search the file's kind.
func (a *attempt) searchers() []Provider {
	now := a.s.cfg.Clock()
	var out []Provider
	for _, p := range a.providers {
		if p.ready(now) && slices.Contains(p.Spec.Kinds, a.kind()) {
			out = append(out, p)
		}
	}
	return out
}

func (a *attempt) ranking() ranking {
	order := make([]subtitling.ProviderID, len(a.providers))
	for i, p := range a.providers {
		order[i] = p.Spec.ID
	}
	return rankingOf(a.wanted, a.subject, a.sub.Path.String(), a.sub.Group, order)
}

// skips is whether a search passes over the candidate because Tried
// before holds it. A pass skips only while the want has a Sidecar: one
// whose Sidecar went, as a language dropped and added again leaves, is
// fetched again.
func (a *attempt) skips(provider subtitling.ProviderID, key string) bool {
	for _, t := range a.tried {
		if t.CandidateKey == key && (t.Provider == nil || *t.Provider == string(provider)) {
			return TriedOutcome(t.Outcome) != TriedPassed || a.current != nil
		}
	}
	return false
}

// try is one candidate that didn't pass, as Tried before keeps it.
type try struct {
	key      string
	provider subtitling.ProviderID
	release  string
	match    *int64
	outcome  TriedOutcome
}

// judged is a holder's answer to one Fetch or Sync: bytes the check
// passed, or the try that failed, or the provider's answer when it
// served nothing.
type judged struct {
	checked   *Checked
	try       *try
	unfetched *calls.Outcome
	members   []subtitling.Member
	match     int
}

var verdictOutcomes = map[subtitling.Verdict]TriedOutcome{
	subtitling.FailedTiming: TriedFailedTiming, subtitling.Unreadable: TriedUnreadable, subtitling.Image: TriedImage,
	subtitling.WrongEpisode: TriedWrongEpisode, subtitling.TooLarge: TriedTooLarge,
}

// judge checks ans, the holder at node's answer about the candidate key,
// whose pass becomes template. offer is the candidate a Fetch or a pack
// held, nil for bytes of SlipStream's own.
func (a *attempt) judge(ctx context.Context, node role.NodeID, key string, offer *subtitling.Offer, template Sidecar, ans subtitling.Fetched) judged {
	t := &try{key: key}
	if offer != nil {
		t.provider, t.release = offer.Provider, first(offer.Releases)
	}
	switch ans.Verdict {
	case subtitling.Passed:
		if offer != nil {
			ans.Key = offer.Key
		}
		bound := boundOf(a.sub.Duration, a.ref.Frames(), a.s.cfg.Timings.CueMargin)
		c, err := check(asked{sidecar: template, offer: offer, group: a.sub.Group, hi: a.wanted.HI, bound: bound}, a.ref, ans, a.gate, a.s.cfg.Timings.Sync, a.s.cfg.FetchCap)
		if err == nil {
			return judged{checked: &c, members: ans.Members, match: c.sidecar.Fit.Match}
		}
		a.s.cfg.Log.WarnContext(ctx, "A subtitle the Subtitles Node timed failed the Controller's check", "node", node, "file", a.key.File, "candidate", key, "error", err)
		t.outcome = TriedUnreadable
	case subtitling.Unfetched:
		return judged{}
	case subtitling.FailedTiming, subtitling.Unreadable, subtitling.Image, subtitling.WrongEpisode, subtitling.TooLarge:
		t.outcome = verdictOutcomes[ans.Verdict]
		if ans.Verdict == subtitling.FailedTiming {
			t.match = new(int64(percent(ans.Claimed.Match)))
		}
	}
	j := judged{try: t}
	if t.match != nil {
		j.match = int(*t.match)
	}
	return j
}

// errPaused is a provider that paused while the attempt waited for its
// slot. The want stays queued until one is ready again.
var errPaused = errors.New("the Subtitle Provider paused")

// stillReady is ids less those that paused since the attempt read them.
func (a *attempt) stillReady(ctx context.Context, ids []subtitling.ProviderID) ([]subtitling.ProviderID, error) {
	var ps []Provider
	err := a.s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		ps, err = a.s.providersIn(ctx, r)
		return err
	})
	now := a.s.cfg.Clock()
	return slices.DeleteFunc(ids, func(id subtitling.ProviderID) bool {
		i := slices.IndexFunc(ps, func(p Provider) bool { return p.Spec.ID == id })
		return i < 0 || !ps[i].ready(now)
	}), err
}

// fetch downloads o on a holder and checks its answer.
func (a *attempt) fetch(ctx context.Context, o subtitling.Offer) (judged, error) {
	done, err := a.s.slots.take(ctx, o.Provider)
	if err != nil {
		return judged{}, err
	}
	defer done()
	if ready, err := a.stillReady(ctx, []subtitling.ProviderID{o.Provider}); err != nil {
		return judged{}, err
	} else if len(ready) == 0 {
		return judged{}, errPaused
	}
	node, holder, ok := a.s.holderFor(o.Provider)
	if !ok {
		return judged{}, ErrNoHolder
	}
	keys, err := a.s.keysOf(ctx, []subtitling.ProviderID{o.Provider})
	if err != nil {
		return judged{}, err
	}
	ans, outcomes, err := holder.Fetch(ctx, subtitling.FetchAsk{Offer: o, Key: keys[o.Provider], Want: a.query(), Map: a.m, Gate: a.gate, Constants: a.s.cfg.Timings.Sync})
	if err != nil {
		return judged{}, err
	}
	if err := a.told(ctx, node, outcomes); err != nil {
		return judged{}, err
	}
	if ans.Verdict == subtitling.Unfetched {
		return judged{unfetched: new(unfetched(o.Provider, outcomes))}, nil
	}
	a.step(ctx, Checking)
	template := Sidecar{Tail: Tail{Language: a.key.Language, Forced: a.key.Forced}, Source: FromProvider}
	return a.judge(ctx, node, o.Key, &o, template, ans), nil
}

// unfetched is how provider answered a download it didn't serve: its
// last call's outcome. A holder that reports none, or one that answered,
// reads as unreachable.
func unfetched(provider subtitling.ProviderID, os []calls.Outcome) calls.Outcome {
	o, ok := lastOutcome(os, provider)
	if !ok || o.Kind == calls.OK {
		return calls.Outcome{Provider: provider, Kind: calls.Unreachable}
	}
	return o
}

// sync times bytes the Controller holds on a holder and checks the
// answer.
func (a *attempt) sync(ctx context.Context, key, name string, b []byte, offer *subtitling.Offer, template Sidecar) (judged, error) {
	node, holder, ok := a.s.holderFor()
	if !ok {
		return judged{}, ErrNoHolder
	}
	ans, err := holder.Sync(ctx, subtitling.SyncAsk{Name: name, Bytes: b, Language: a.key.Language, Map: a.m, Gate: a.gate, Constants: a.s.cfg.Timings.Sync})
	if err != nil {
		return judged{}, err
	}
	a.step(ctx, Checking)
	return a.judge(ctx, node, key, offer, template, ans), nil
}

// ending is how an attempt ends: the search's result, how it moves the
// gap, what it found and the tries that failed.
type ending struct {
	result string
	pace   pacing.Result
	found  int
	best   *int64
	tries  []try
	final  bool
}

func (e *ending) failed(t try) {
	e.tries = append(e.tries, t)
	e.matched(t.match)
}

func (e *ending) matched(m *int64) {
	if m != nil && (e.best == nil || *m > *e.best) {
		e.best = m
	}
}

// settle ends the attempt in one Write under its lease: its tries, its
// want's last search and next due time, off the queue. A fresh import's
// first search that heard a provider records Subtitles missing or No
// subtitle matched the timing.
func (a *attempt) settle(ctx context.Context, e ending) error {
	now := a.s.cfg.Clock()
	gap := pacing.Gap(time.Duration(a.row.GapSeconds)*time.Second, a.interval, LongestGap.Value(a.settings.LongestGapSeconds), e.pace, false, a.capped)
	var due storage.Time
	if e.result != "passed" || !e.final {
		due = storage.FromTime(now.Add(gap))
	}
	err := a.write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library], q *db.Queries) error {
		k := a.key
		for _, t := range e.tries {
			if err := putTry(ctx, q, k, t, now); err != nil {
				return err
			}
		}
		n, err := q.Searched(ctx, db.SearchedParams{
			Now: storage.FromTime(now), Found: new(int64(e.found)), BestMatch: e.best, Result: &e.result, GapSeconds: int64(gap / time.Second), DueAt: due,
			FileID: k.File, Language: k.Language.String(), Forced: k.Forced, AttemptID: a.id,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return errStale
		}
		if !a.file.Fresh || e.pace == pacing.Unheard {
			return nil
		}
		if err := q.ClearFresh(ctx, k.File); err != nil {
			return err
		}
		return a.firstSearch(ctx, tx, e)
	})
	if err != nil {
		return err
	}
	a.s.cfg.Changed.Publish(ctx, changedOf([]works.WorkID{a.sub.Work}, []files.ID{a.key.File}))
	return nil
}

func (a *attempt) firstSearch(ctx context.Context, tx *storage.Tx[storage.Library], e ending) error {
	row := missingRow
	switch e.result {
	case "nothing_found":
	case "none_passed":
		row = timingFailedRow
	default:
		return nil
	}
	w, err := entry.WorkIn(ctx, tx, a.s.cfg.Works, a.sub.Work)
	if err != nil {
		return err
	}
	return row.Record(ctx, tx, a.s.cfg.History, SearchFacts{Work: w, File: a.sub.Path.String(), Language: a.key.Language.Name(), Forced: a.key.Forced, Found: e.found})
}

func putTry(ctx context.Context, q *db.Queries, k WantKey, t try, now time.Time) error {
	return q.PutTried(ctx, db.PutTriedParams{
		FileID: k.File, Language: k.Language.String(), Forced: k.Forced, CandidateKey: t.key, Provider: some(string(t.provider)), Release: some(t.release),
		Match: t.match, Outcome: string(t.outcome), TriedAt: storage.FromTime(now),
	})
}

// pass hands c to the Keeper as the want's intent, then ends the
// attempt.
func (a *attempt) pass(ctx context.Context, c Checked, e ending) error {
	if err := a.s.intend(ctx, a.key, a.id, c); err != nil {
		return err
	}
	e.result, e.pace, e.final = "passed", pacing.Took, c.sidecar.Final
	e.matched(matchOf(c.sidecar))
	return a.settle(ctx, e)
}

// run is Subtitle search for one queued want, bounded by
// Timings.AttemptBound: its siblings and held pack members first when it
// has no Sidecar, then every ready provider, fetching down the ranked
// list until one passes or the tries per search run out. A want whose
// file's map isn't current leaves the queue until it is.
func (s *Subtitles) run(ctx context.Context, k WantKey) error {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Timings.AttemptBound)
	defer cancel()
	a, err := s.lease(ctx, k)
	if err != nil {
		return err
	}
	defer a.release(ctx)
	if err := a.mapped(ctx); errors.Is(err, errMapDue) {
		return s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return db.New(tx).Unqueue(ctx, db.UnqueueParams(wantParams(k)))
		})
	} else if err != nil {
		return err
	}
	if a.current == nil && !a.noAired {
		if ok, err := a.trySiblings(ctx); ok || err != nil {
			return err
		}
		if ok, err := a.tryHeld(ctx); ok || err != nil {
			return err
		}
	}
	return a.search(ctx)
}

// search asks every ready provider that searches the file's kind, then
// fetches down the ranked candidates.
func (a *attempt) search(ctx context.Context) error {
	ps := a.searchers()
	switch {
	case a.noAired:
		return a.settle(ctx, ending{result: "no_provider", pace: pacing.Unheard})
	case len(ps) == 0:
		result := "no_provider"
		if slices.ContainsFunc(a.providers, func(p Provider) bool { return p.searches() && slices.Contains(p.Spec.Kinds, a.kind()) }) {
			result = "paused"
		}
		return a.settle(ctx, ending{result: result, pace: pacing.Unheard})
	}
	offers, outcomes, err := a.ask(ctx, ps)
	switch {
	case errors.Is(err, errPaused):
		return nil
	case errors.Is(err, errStale):
		return err
	}
	if err != nil {
		a.s.cfg.Log.WarnContext(ctx, "A subtitle search didn't reach the Subtitles Node", "file", a.key.File, "error", err)
		return a.settle(ctx, ending{result: "unreachable", pace: pacing.Unheard})
	}
	e := ending{result: "unreachable", pace: pacing.Unheard}
	if slices.ContainsFunc(outcomes, func(o calls.Outcome) bool { return o.Kind == calls.OK || o.Kind == calls.NotFound }) {
		e.result, e.pace = "nothing_found", pacing.Answered
	} else if ready, err := a.stillReady(ctx, idsOf(ps)); err != nil || len(ready) == 0 {
		return err
	}
	rk := a.ranking()
	all := rk.rank(offers, false)
	e.found = len(all)
	if e.found > 0 {
		e.result = "none_passed"
	}
	cur, has := rk.current(a.current)
	return a.fetchDown(ctx, all, func(x ranked) bool { return !has || outranks(x, cur) }, e)
}

// fetchDown fetches the ranked candidates that beat the want's Sidecar and
// weren't tried before, up to Tries per search, until one passes.
func (a *attempt) fetchDown(ctx context.Context, all []ranked, beats func(ranked) bool, e ending) error {
	var tries int
	for _, x := range all {
		if tries == int(TriesPerSearch.Value(a.settings.TriesPerSearch)) {
			break
		}
		if a.skips(x.offer.Provider, x.offer.Key) || !beats(x) {
			continue
		}
		tries++
		j, err := a.fetch(ctx, x.offer)
		switch {
		case errors.Is(err, errPaused):
			return a.note(ctx, e.tries)
		case errors.Is(err, errStale):
			return err
		case err != nil:
			a.s.cfg.Log.WarnContext(ctx, "A subtitle download didn't reach the Subtitles Node", "file", a.key.File, "candidate", x.offer.Key, "error", err)
			continue
		}
		if j.checked != nil {
			a.s.hold(a, x.offer, j.members)
			return a.pass(ctx, *j.checked, e)
		}
		if j.try != nil {
			e.failed(*j.try)
		}
	}
	return a.settle(ctx, e)
}

// ask runs one Search on a holder for those of ps still ready, each
// provider through its slot. A Manual subtitle search holds no lease and
// doesn't wait in the drain's slots.
func (a *attempt) ask(ctx context.Context, ps []Provider) ([]subtitling.Offer, []calls.Outcome, error) {
	ids := idsOf(ps)
	if !a.id.IsZero() {
		done, err := a.s.slots.take(ctx, ids...)
		if err != nil {
			return nil, nil, err
		}
		defer done()
	}
	ids, err := a.stillReady(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	if len(ids) == 0 {
		return nil, nil, errPaused
	}
	node, holder, ok := a.s.holderFor(ids...)
	if !ok {
		return nil, nil, ErrNoHolder
	}
	keys, err := a.s.keysOf(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	offers, outcomes, err := holder.Search(ctx, a.query(), ids, keys)
	if err != nil {
		return nil, nil, err
	}
	return offers, outcomes, a.told(ctx, node, outcomes)
}

func idsOf(ps []Provider) []subtitling.ProviderID {
	ids := make([]subtitling.ProviderID, len(ps))
	for i, p := range ps {
		ids[i] = p.Spec.ID
	}
	return ids
}

// leases is the lease waits a process keeps: the wants a pick or an
// upload waits for, which Subtitle search leaves alone, and a signal for
// each release.
type leases struct {
	mu      sync.Mutex
	waiting map[WantKey]int
	freed   map[WantKey]chan struct{}
}

// hold marks k waited for until unhold.
func (l *leases) hold(k WantKey) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.waiting == nil {
		l.waiting, l.freed = map[WantKey]int{}, map[WantKey]chan struct{}{}
	}
	l.waiting[k]++
}

func (l *leases) unhold(k WantKey) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.waiting[k]--
	if l.waiting[k] <= 0 {
		delete(l.waiting, k)
	}
}

func (l *leases) waited(k WantKey) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.waiting[k] > 0
}

// signal is closed at k's next release.
func (l *leases) signal(k WantKey) <-chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.freed == nil {
		l.waiting, l.freed = map[WantKey]int{}, map[WantKey]chan struct{}{}
	}
	if l.freed[k] == nil {
		l.freed[k] = make(chan struct{})
	}
	return l.freed[k]
}

func (l *leases) free(k WantKey) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if c := l.freed[k]; c != nil {
		close(c)
		delete(l.freed, k)
	}
}

// leaseFirst takes k's lease for a pick or an upload, ahead of Subtitle
// search: while another attempt holds it, the want is the Admin's to
// search next and this waits for the release. An intent or a carried
// Sidecar holds it too and settles in the Keeper, so the wait also tries
// again every Timings.WakeDelay.
func (s *Subtitles) leaseFirst(ctx context.Context, k WantKey) (*attempt, error) {
	s.leases.hold(k)
	defer s.leases.unhold(k)
	for {
		freed := s.leases.signal(k)
		a, err := s.lease(ctx, k)
		if !errors.Is(err, ErrBusy) {
			return a, err
		}
		select {
		case <-freed:
		case <-time.After(s.cfg.Timings.WakeDelay):
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for the subtitle's lease: %w", context.Cause(ctx))
		}
	}
}
