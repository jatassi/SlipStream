package subtitles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// keeper settles every row that names a file's Sidecar bytes, one file at
// a time. It is the only caller of SidecarService's List, Read, Write and
// Remove. A key is a file id, live or doomed: a doomed file's parked and
// removal rows carry its id, so they settle behind any call still in
// flight for it. The queue holds keys, not work: settle reads the rows
// each time, so a wake for nothing costs one Read.
type keeper struct {
	s *Subtitles

	mu   sync.Mutex
	due  map[files.ID]bool
	busy map[files.ID]bool
	ring chan struct{}
	// rand draws removal backoffs, from several settles at once.
	rand  sync.Mutex
	bytes intentBytes
}

func newKeeper(s *Subtitles) *keeper {
	return &keeper{s: s, due: map[files.ID]bool{}, busy: map[files.ID]bool{}, ring: make(chan struct{}, 1)}
}

// soon settles ids after Timings.WakeDelay, by when the Write that named
// them has committed or rolled back. The sweep catches a wake a rollback
// wasted.
func (k *keeper) soon(ids ...files.ID) {
	if len(ids) == 0 {
		return
	}
	time.AfterFunc(k.s.cfg.Timings.WakeDelay, func() { k.wake(ids...) })
}

func (k *keeper) wake(ids ...files.ID) {
	k.mu.Lock()
	for _, id := range ids {
		k.due[id] = true
	}
	k.mu.Unlock()
	select {
	case k.ring <- struct{}{}:
	default:
	}
}

// take is every due key no settle runs for, now marked running.
func (k *keeper) take() []files.ID {
	k.mu.Lock()
	defer k.mu.Unlock()
	var out []files.ID
	for id := range k.due {
		if !k.busy[id] {
			out = append(out, id)
			k.busy[id] = true
			delete(k.due, id)
		}
	}
	slices.SortFunc(out, func(a, b files.ID) int { return a.Compare(b.UUID) })
	return out
}

func (k *keeper) done(id files.ID) {
	k.mu.Lock()
	delete(k.busy, id)
	again := k.due[id]
	k.mu.Unlock()
	if again {
		k.wake()
	}
}

// run settles woken keys, at most Timings.DrainWidth at once, and sweeps
// every Timings.KeeperPoll, first at once.
func (k *keeper) run(ctx context.Context) error {
	slots := make(chan struct{}, k.s.cfg.Timings.DrainWidth)
	var wg sync.WaitGroup
	defer wg.Wait()
	poll := time.NewTicker(k.s.cfg.Timings.KeeperPoll)
	defer poll.Stop()
	k.sweep(ctx)
	for {
		for _, id := range k.take() {
			select {
			case slots <- struct{}{}:
			case <-ctx.Done():
				return nil
			}
			wg.Go(func() {
				defer func() { <-slots }()
				defer k.done(id)
				if err := k.settle(ctx, id); err != nil && ctx.Err() == nil {
					k.s.cfg.Log.WarnContext(ctx, "Updating a file's subtitle files failed", "file", id, "error", err)
				}
			})
		}
		select {
		case <-ctx.Done():
			return nil
		case <-k.ring:
		case <-poll.C:
			k.sweep(ctx)
		}
	}
}

// sweep wakes the keys with work the rows hold: surveys due, intents,
// Sidecars to retime or bring home, parked rows and removals due. It
// turns parked rows whose replacer is gone into removals first.
func (k *keeper) sweep(ctx context.Context) {
	var ids []files.ID
	err := k.s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		now := storage.FromTime(k.s.cfg.Clock())
		orphans, err := q.ListOrphanParked(ctx)
		if err != nil {
			return err
		}
		for _, p := range orphans {
			if err := unpark(ctx, q, p, now); err != nil {
				return err
			}
		}
		n := int64(k.s.cfg.Timings.ReconcileChunk)
		for _, list := range []func() ([]files.ID, error){
			func() ([]files.ID, error) { return q.ListSurveyDue(ctx, n) },
			func() ([]files.ID, error) { return q.ListIntentFiles(ctx, n) },
			func() ([]files.ID, error) { return q.ListPendingFiles(ctx, n) },
			func() ([]files.ID, error) { return q.ListParkedFiles(ctx, n) },
			func() ([]files.ID, error) { return q.ListRemovalFiles(ctx, db.ListRemovalFilesParams{Now: now, N: n}) },
		} {
			got, err := list()
			if err != nil {
				return err
			}
			ids = append(ids, got...)
		}
		return nil
	})
	if err != nil {
		if ctx.Err() == nil {
			k.s.cfg.Log.WarnContext(ctx, "Finding the files whose subtitle files need updating failed", "error", err)
		}
		return
	}
	k.wake(ids...)
}

// settled is what one settle reads of its key, in one Read.
type settled struct {
	reconciling bool
	// subject is the live file, nil once doomed.
	subject  *files.Subject
	row      *db.SubtitlesFile
	wants    []db.SubtitlesWant
	sidecars []Sidecar
	intents  []db.SubtitlesWrite
	// held is the bytes of intents this process checked, by attempt.
	held     map[storage.UUID][]byte
	copies   map[storage.UUID]sibling
	parked   []db.SubtitlesParked
	removals []db.SubtitlesRemoval
	langs    []targets.WantedLanguage
	videos   media.Videos
}

func (k *keeper) read(ctx context.Context, f files.ID) (settled, error) {
	var in settled
	err := k.s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		if in.reconciling, err = k.s.cfg.Reconciling(ctx, r); err != nil || in.reconciling {
			return err
		}
		ids := []files.ID{f}
		subjects, err := k.s.cfg.Files.SubjectsOf(ctx, r, ids)
		if err != nil {
			return err
		}
		q := db.New(r)
		rows, err := q.ListFilesIn(ctx, ids)
		if err != nil {
			return err
		}
		if len(subjects) == 1 && len(rows) == 1 {
			if err := in.live(ctx, r, k.s.cfg.Targets, subjects[0], rows[0]); err != nil {
				return err
			}
		}
		if err := in.rows(ctx, q, f, storage.FromTime(k.s.cfg.Clock())); err != nil {
			return err
		}
		in.videos, err = k.s.cfg.Videos(ctx, r)
		return err
	})
	return in, err
}

// rows is the key's rows: wants, Sidecars, intents, the rows parked from
// it and its removals due.
func (in *settled) rows(ctx context.Context, q *db.Queries, f files.ID, now storage.Time) error {
	ids := []files.ID{f}
	var err error
	if in.wants, err = q.ListWantsIn(ctx, ids); err != nil {
		return err
	}
	scs, err := sidecarsIn(ctx, q, ids)
	if err != nil {
		return err
	}
	in.sidecars = scs[f]
	if in.intents, err = q.ListIntentsIn(ctx, ids); err != nil {
		return err
	}
	if in.parked, err = q.ListParkedFrom(ctx, f); err != nil {
		return err
	}
	in.removals, err = q.ListRemovalsDue(ctx, db.ListRemovalsDueParams{FileID: f, Now: now})
	return err
}

// live is the read of a key whose file is still in the library.
func (in *settled) live(ctx context.Context, r storage.Reader[storage.Library], t Targets, sub files.Subject, row db.SubtitlesFile) error {
	langs, err := t.LanguagesOf(ctx, r)
	if err != nil {
		return err
	}
	in.subject, in.row, in.langs = &sub, &row, langs[sub.Profile]
	return nil
}

// commit runs fn in one Library Write that changes what sub's subtitles
// read as and, once it commits, tells live screens of sub's Work.
func (k *keeper) commit(ctx context.Context, sub files.Subject, fn func(context.Context, *storage.Tx[storage.Library]) error) error {
	if err := k.s.cfg.Library.Write(ctx, fn); err != nil {
		return err
	}
	k.s.cfg.Changed.Publish(ctx, changedOf([]works.WorkID{sub.Work}, []files.ID{sub.ID}))
	return nil
}

// errNotReady ends a settle whose Storage Node can't take the call now:
// the rows stay for the next wake.
var errNotReady = errors.New("the root's Storage Node isn't ready")

// settle converges one key: the gate, a survey of a live file at its
// name, its copies' bytes once its map is built, its intents, retimes and
// strays, the rows parked from it, then the removals of the key's bytes.
// Each step is derived from rows and listings, so a settle that stops
// anywhere is finished by the next.
func (k *keeper) settle(ctx context.Context, f files.ID) error {
	in, err := k.read(ctx, f)
	if err != nil || in.reconciling {
		return err
	}
	placed := in.subject != nil && in.subject.AtName
	err = k.hold(ctx, f, &in)
	if err == nil && placed && in.surveys() {
		if err = k.survey(ctx, in); err == nil {
			if in, err = k.read(ctx, f); err == nil {
				err = k.hold(ctx, f, &in)
			}
		}
	}
	if err == nil && placed && in.row != nil {
		if err = k.fetch(ctx, in); err == nil {
			err = k.converge(ctx, in)
		}
	}
	if err == nil && len(in.parked) > 0 {
		err = k.claimAll(ctx, in)
	}
	if err == nil {
		err = k.removeAll(ctx, in)
	}
	if errors.Is(err, errNotReady) {
		return nil
	}
	return err
}

// surveys is whether a settle lists the file's companions: when a survey
// is due, when it never had one while it has a want or a Sidecar, or
// when an intent's bytes aren't in this process and copy no other file's.
func (in *settled) surveys() bool {
	never := in.row.SurveyedAt.IsZero() && (len(in.wants) > 0 || len(in.sidecars) > 0)
	unheld := slices.ContainsFunc(in.intents, func(w db.SubtitlesWrite) bool { return in.held[w.AttemptID] == nil && !in.copying(w) })
	return in.row.SurveyDue || never || unheld
}

// calls is SidecarService on the Node that holds root.
func (k *keeper) calls(ctx context.Context, root roots.ID) (storing.SidecarCalls, error) {
	r, _, err := k.s.cfg.Roots.StateOf(ctx, root)
	if err != nil {
		return nil, err
	}
	return k.s.cfg.Storage(r.Node), nil
}

func callError(err error) error {
	if _, ok := errors.AsType[*storing.NotReady](err); ok {
		return errNotReady
	}
	return err
}

// errSurveyStale rolls back a survey's finish whose file moved or was
// asked for another survey since it was listed.
var errSurveyStale = errors.New("the file changed since its survey")

func (k *keeper) survey(ctx context.Context, in settled) error {
	sub := in.subject
	calls, err := k.calls(ctx, sub.Root)
	if err != nil {
		return err
	}
	l := storing.SidecarListing{Video: sub.Path}
	if from := in.row.SurveyFrom; from != nil {
		also, err := storing.ParseRel(*from)
		if err != nil {
			return err
		}
		l.Also = []storing.Rel{also}
	}
	if l.Also, err = strays(l.Also, in.sidecars); err != nil {
		return err
	}
	listed, err := calls.List(ctx, sub.Root.UUID, in.videos, []storing.SidecarListing{l})
	if err != nil {
		return callError(err)
	}
	intents := slices.DeleteFunc(slices.Clone(in.intents), in.copying)
	plan := match(sub.ID, sub.Path.String(), in.sidecars, intents, listed[0].Companions, in.langs)
	err = k.commit(ctx, *sub, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return k.finishSurvey(ctx, tx, in, plan)
	})
	if errors.Is(err, errSurveyStale) {
		k.soon(sub.ID)
		return nil
	}
	return err
}

// finishSurvey applies plan in one Write, guarded by what the survey read:
// a Sidecar changes only while it holds the bytes read, and an intent
// finishes only under its own attempt.
func (k *keeper) finishSurvey(ctx context.Context, tx *storage.Tx[storage.Library], in settled, plan surveyPlan) error {
	q := db.New(tx)
	now := storage.FromTime(k.s.cfg.Clock())
	extras, err := json.Marshal(plan.extras)
	if err != nil {
		return fmt.Errorf("a survey's extras: %w", err)
	}
	var companions *string
	if len(plan.extras) > 0 {
		companions = new(string(extras))
	}
	n, err := q.FinishSurvey(ctx, db.FinishSurveyParams{
		SurveyedAt: now, Companions: companions, FileID: in.subject.ID, SurveyFrom: in.row.SurveyFrom, Video: in.subject.Path.String(),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return errSurveyStale
	}
	if err := plan.settleRows(ctx, q, now); err != nil {
		return err
	}
	finished, err := plan.add(ctx, q, in.subject.ID, now)
	if err != nil {
		return err
	}
	if err := k.announce(ctx, tx, *in.subject, finished); err != nil {
		return err
	}
	return k.s.reconcileIn(ctx, tx, []files.ID{in.subject.ID})
}

// Extra is a companion of a file that meets none of its wants, shown
// under its Details.
type Extra struct {
	Path string `json:"path"`
	Tail string `json:"tail"`
	Size int64  `json:"size"`
	// Want is the want its tail beside the video's stem names, "" when
	// none: the want it meets once wanted, which the bulk pass and Impact
	// read without listing the file again.
	Want string `json:"want,omitempty"`
}

// surveyPlan is a survey's verdict on each row and companion.
type surveyPlan struct {
	// gone are Sidecar rows whose bytes are nowhere listed.
	gone []Sidecar
	// placed are rows whose bytes moved, home or stray.
	placed []Sidecar
	// finished are intents whose bytes landed, with the Sidecar they make.
	finished []finishedIntent
	// dropped are intents whose bytes never landed.
	dropped []db.SubtitlesWrite
	// adopted are companions no row holds that meet a want: found ones,
	// and ones the Node's ledger knows SlipStream wrote.
	adopted []Sidecar
	extras  []Extra
}

type finishedIntent struct {
	intent  db.SubtitlesWrite
	sidecar Sidecar
}

// settleRows applies the verdicts on rows: gone ones go, moved ones say
// where their bytes sit, and intents whose bytes never landed drop.
func (plan *surveyPlan) settleRows(ctx context.Context, q *db.Queries, now storage.Time) error {
	for _, sc := range plan.gone {
		if err := dropSidecar(ctx, q, sc, now); err != nil {
			return err
		}
	}
	for _, sc := range plan.placed {
		if _, err := q.PlaceSidecar(ctx, db.PlaceSidecarParams{Stray: some(sc.Stray), ID: sc.ID, Sha256: sc.SHA256[:]}); err != nil {
			return err
		}
	}
	for _, w := range plan.dropped {
		if _, err := q.DeleteIntent(ctx, db.DeleteIntentParams{FileID: w.FileID, Language: w.Language, Forced: w.Forced, AttemptID: w.AttemptID}); err != nil {
			return err
		}
	}
	return nil
}

// add writes the Sidecars the survey found, each where no Sidecar meets
// its want now: a finished intent's only while its attempt still holds it.
// It answers the finished intents' Sidecars it wrote.
func (plan *surveyPlan) add(ctx context.Context, q *db.Queries, f files.ID, now storage.Time) ([]Sidecar, error) {
	current, err := sidecarsIn(ctx, q, []files.ID{f})
	if err != nil {
		return nil, err
	}
	have := current[f]
	put := func(sc Sidecar) (bool, error) {
		if sidecarOfWant(have, sc.want()) != nil {
			return false, nil
		}
		have = append(have, sc)
		return true, putSidecar(ctx, q, sc, now)
	}
	var finished []Sidecar
	for _, fin := range plan.finished {
		n, err := q.DeleteIntent(ctx, db.DeleteIntentParams{FileID: fin.intent.FileID, Language: fin.intent.Language, Forced: fin.intent.Forced, AttemptID: fin.intent.AttemptID})
		if err != nil || n == 0 {
			if err != nil {
				return nil, err
			}
			continue
		}
		ok, err := put(fin.sidecar)
		if err != nil {
			return nil, err
		}
		if ok {
			finished = append(finished, fin.sidecar)
		}
	}
	for _, sc := range plan.adopted {
		if _, err := put(sc); err != nil {
			return nil, err
		}
	}
	return finished, nil
}

// match is the pure decision of a survey of file at video. A row is home,
// stray where its bytes are listed, or gone. An intent whose bytes are
// listed finishes, else it drops and its want stays due. A companion no
// row or intent holds beside the video's own stem is adopted when its
// tail meets a Wanted Language with no Sidecar: as SlipStream's own, to
// be timed again, when the ledger knows its hash, else as found, final
// and untimed, the hearing-impaired preference picking among several of
// one language and kind. Every other companion is an extra.
func match(file files.ID, video string, rows []Sidecar, intents []db.SubtitlesWrite, listed []storing.SidecarCompanion, langs []targets.WantedLanguage) surveyPlan {
	var plan surveyPlan
	c := claims{listed: listed, held: map[string]bool{}}
	for _, sc := range rows {
		home := besideAt(video, sc.TailText)
		at, ok := c.claim(sc.SHA256, home)
		switch {
		case !ok:
			plan.gone = append(plan.gone, sc)
		case strayAt(at, home) != sc.Stray:
			sc.Stray = strayAt(at, home)
			plan.placed = append(plan.placed, sc)
		}
	}
	for _, w := range intents {
		var sha [32]byte
		copy(sha[:], w.Sha256)
		home := besideAt(video, w.Tail)
		at, ok := c.claim(sha, home)
		sc, err := decodeSidecar(w.Sidecar)
		if !ok || err != nil {
			plan.dropped = append(plan.dropped, w)
			continue
		}
		sc.ID, sc.File, sc.TailText, sc.SHA256, sc.Stray = storage.NewUUID(), file, w.Tail, sha, strayAt(at, home)
		plan.finished = append(plan.finished, finishedIntent{intent: w, sidecar: sc})
	}
	met := map[Want]bool{}
	for _, sc := range rows {
		met[sc.want()] = true
	}
	for _, f := range plan.finished {
		met[f.sidecar.want()] = true
	}
	plan.adopt(file, video, c, met, langs)
	return plan
}

// claims hands each listed companion to at most one row or intent.
type claims struct {
	listed []storing.SidecarCompanion
	held   map[string]bool
}

// claim is where bytes of sha sit, home first, unless another row or
// intent holds them.
func (c claims) claim(sha [32]byte, home string) (string, bool) {
	var at string
	for _, l := range c.listed {
		if l.SHA256 != sha || c.held[l.Path.String()] {
			continue
		}
		if l.Path.String() == home || at == "" {
			at = l.Path.String()
		}
	}
	if at != "" {
		c.held[at] = true
	}
	return at, at != ""
}

// adopt takes, for each Wanted Language met by nothing, the companion its
// preference picks among those no row holds beside the video's stem. The
// others are extras.
func (plan *surveyPlan) adopt(file files.ID, video string, c claims, met map[Want]bool, langs []targets.WantedLanguage) {
	candidates := map[Want][]storing.SidecarCompanion{}
	for _, l := range c.listed {
		if c.held[l.Path.String()] {
			continue
		}
		tail, home := homeOf(video, l.Path)
		t, named := ParseTail(tail)
		wanted := slices.ContainsFunc(langs, func(w targets.WantedLanguage) bool { return w.Language == t.Language && w.Forced == t.Forced })
		if !home || !named || !wanted || met[t.Want()] {
			plan.extras = append(plan.extras, extraOf(video, l))
			continue
		}
		candidates[t.Want()] = append(candidates[t.Want()], l)
	}
	for _, l := range langs {
		cs := candidates[Want{Language: l.Language, Forced: l.Forced}]
		if len(cs) == 0 {
			continue
		}
		slices.SortStableFunc(cs, func(a, b storing.SidecarCompanion) int {
			return preference(video, l.HI, a) - preference(video, l.HI, b)
		})
		plan.adopted = append(plan.adopted, adopt(file, video, cs[0]))
		for _, x := range cs[1:] {
			plan.extras = append(plan.extras, extraOf(video, x))
		}
	}
}

func extraOf(video string, c storing.SidecarCompanion) Extra {
	e := Extra{Path: c.Path.String(), Tail: c.Tail, Size: c.Size}
	if tail, home := homeOf(video, c.Path); home {
		if t, ok := ParseTail(tail); ok {
			e.Want = t.Want().String()
		}
	}
	return e
}

// strayAt is where bytes at at sit away from home, "" when at is home.
func strayAt(at, home string) string {
	if at == home {
		return ""
	}
	return at
}

// preference orders companions of one want for hi: the kind it prefers
// first, then a written format before .vtt, then by name.
func preference(video string, hi targets.HI, c storing.SidecarCompanion) int {
	tail, _ := homeOf(video, c.Path)
	t, _ := ParseTail(tail)
	n := 0
	if t.HI != (hi == targets.Prefer) {
		n += 2
	}
	if t.Ext == "vtt" {
		n++
	}
	return n
}

func adopt(file files.ID, video string, c storing.SidecarCompanion) Sidecar {
	tail, _ := homeOf(video, c.Path)
	t, _ := ParseTail(tail)
	sc := Sidecar{ID: storage.NewUUID(), File: file, Tail: t, TailText: tail, SHA256: c.SHA256, Size: c.Size}
	if c.Ledger {
		sc.Source, sc.TimingDue = FromProvider, true
	} else {
		sc.Source, sc.Final = Found, true
	}
	return sc
}

// removeAll removes the key's due removals by hash, each in its own
// Write. Removed bytes delete the row. Other bytes there are kept: the row
// goes and the live file at that stem is surveyed. A call that doesn't
// reach the Node backs off.
func (k *keeper) removeAll(ctx context.Context, in settled) error {
	for _, rm := range in.removals {
		calls, err := k.calls(ctx, rm.RootID)
		if err != nil {
			return err
		}
		at, err := storing.ParseRel(rm.Path)
		if err != nil {
			return err
		}
		var sha [32]byte
		copy(sha[:], rm.Sha256)
		err = callError(calls.Remove(ctx, rm.RootID.UUID, at, sha))
		_, changed := errors.AsType[*storing.ChangedError](err)
		if errors.Is(err, errNotReady) {
			return err
		}
		err = k.s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			q := db.New(tx)
			switch {
			case err == nil:
				return q.DeleteRemoval(ctx, rm.ID)
			case changed:
				if err := k.surveyAt(ctx, tx, rm, in.videos); err != nil {
					return err
				}
				return q.DeleteRemoval(ctx, rm.ID)
			default:
				return q.DeferRemoval(ctx, db.DeferRemovalParams{NextAt: storage.FromTime(k.s.cfg.Clock().Add(k.backoff(int(rm.Tries)))), ID: rm.ID})
			}
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *keeper) backoff(tries int) time.Duration {
	k.rand.Lock()
	defer k.rand.Unlock()
	return backoff.Delay(k.s.cfg.Timings.RemoveRetryMin, k.s.cfg.Timings.RemoveRetryMax, tries, k.s.cfg.Rand)
}

// surveyAt marks the live file whose companion sits at rm's path for a
// survey: rm's own file while it lives, else a file of the root whose stem
// the path extends.
func (k *keeper) surveyAt(ctx context.Context, tx *storage.Tx[storage.Library], rm db.SubtitlesRemoval, videos media.Videos) error {
	var candidates []storing.Rel
	for stem := rm.Path; strings.Contains(stem[strings.LastIndex(stem, "/")+1:], "."); {
		stem = stem[:strings.LastIndex(stem, ".")]
		for _, ext := range videos.Extensions() {
			if p, err := storing.ParseRel(stem + "." + ext); err == nil {
				candidates = append(candidates, p)
			}
		}
	}
	at, err := files.IDsAt(ctx, tx, rm.RootID, candidates)
	if err != nil {
		return err
	}
	ids := []files.ID{rm.FileID}
	for _, id := range at {
		ids = append(ids, id)
	}
	q := db.New(tx)
	for _, id := range ids {
		if err := q.MarkSurveyDue(ctx, db.MarkSurveyDueParams{FileID: id}); err != nil {
			return err
		}
	}
	k.soon(ids...)
	return nil
}
