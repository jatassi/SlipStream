package indexers

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Claim is whose line a reservation is for.
type Claim uint8

const (
	// ClaimPress is Manual search and the now lane: one slot above
	// Searches at once, counted over every claim in flight, and any free
	// slot of the search limit, its last ones included. A press at the
	// limit is skipped limit_reached.
	ClaimPress Claim = iota + 1
	// ClaimBackground is the bulk and backlog lanes: Searches at once,
	// counted over background claims in flight, and only the slots beyond
	// RSS sync's reserve. A background line at the limit waits.
	ClaimBackground
)

func (c Claim) word() *string {
	switch c {
	case ClaimPress:
		return new("press")
	case ClaimBackground:
		return new("background")
	}
	return nil
}

func claimNamed(word *string) Claim {
	if word != nil && *word == "background" {
		return ClaimBackground
	}
	return ClaimPress
}

// RoomError is a line that waits for its Indexer.
type RoomError struct {
	// Busy is Searches at once in flight for the claim.
	Busy bool
	// Until is when a slot of the search limit the claim may use frees,
	// zero when Busy or Starved.
	Until time.Time
	// Starved is RSS sync's reserve alone filling the search limit: no
	// background slot fits under today's settings.
	Starved bool
}

func (e *RoomError) Error() string {
	switch {
	case e.Busy:
		return "the Indexer has as many searches in flight as its Searches at once allows"
	case e.Starved:
		return "the slots RSS sync keeps fill the Indexer's search limit"
	}
	return "the Indexer's search limit has no free slot until " + e.Until.UTC().Format(time.RFC3339)
}

// Facts is everything the room rule reads about one Indexer.
type Facts struct {
	// AtOnce is the Indexer's Searches at once.
	AtOnce int
	// InFlight is the search queries reserved within searching's JobCap
	// and not completed, per claim.
	InFlight map[Claim]int
	// Stuck is every in-flight query past its reservation plus the search
	// wait.
	Stuck  bool
	Limit  Limit
	Window []time.Time
	RSS    RSSPlan
}

// RSSPlan is what the reserve needs from RSS sync.
type RSSPlan struct {
	// On is RSS sync on, the Indexer on and its RSS switch on.
	On    bool
	Every time.Duration
	// LastPages is how many pages the Indexer's last fetch used, 1 when it
	// never fetched.
	LastPages int
}

// Fit is FitOf's answer: exactly one of Reserve, Skip or Wait is set.
type Fit struct {
	Reserve bool
	Skip    *Outcome
	Wait    *RoomError
}

// FitOf is the room rule, pure. Reserve, Room and the queue simulator call
// it, so there is one rule.
//
// Busy is the claim's cap in flight: background claims at AtOnce for a
// background line, every claim at AtOnce plus one for a press. A busy
// Indexer whose in-flight queries are all Stuck skips the line late, so a
// wedged Indexer holds back neither the queue nor the start rule. Under
// the search limit a press takes any free slot and is skipped
// limit_reached when none is. A background line needs a free slot beyond
// RSS sync's reserve and else waits for one, or waits Starved when the
// reserve is the whole limit.
func FitOf(f Facts, c Claim, now time.Time) Fit {
	if f.busy(c) {
		if f.Stuck {
			return Fit{Skip: &Outcome{Kind: Late}}
		}
		return Fit{Wait: &RoomError{Busy: true}}
	}
	if f.Limit.Count <= 0 {
		return Fit{Reserve: true}
	}
	in := windowAt(f.Limit, f.Window, now)
	room := f.Limit.Count
	if c == ClaimBackground {
		kept := f.reserved(in, now)
		if kept >= room {
			return Fit{Wait: &RoomError{Starved: true}}
		}
		room -= kept
	}
	if len(in) < room {
		return Fit{Reserve: true}
	}
	until := in[len(in)-room].Add(f.Limit.Per.Duration())
	if c == ClaimPress {
		return Fit{Skip: &Outcome{Kind: LimitReached, Until: until}}
	}
	return Fit{Wait: &RoomError{Until: until}}
}

func (f Facts) busy(c Claim) bool {
	if c == ClaimPress {
		return f.InFlight[ClaimPress]+f.InFlight[ClaimBackground] > f.AtOnce
	}
	return f.InFlight[ClaimBackground] >= f.AtOnce
}

// reserved is the slots RSS sync keeps from background lines: one per
// fetch due before the window's oldest reservation leaves, a line taken
// now being the oldest of an empty window, plus the pages the last fetch
// used. It keeps at most every slot but one, so a background line gets a
// slot each window on an Indexer whose limit RSS sync alone would use up.
// A limit of one keeps its slot for RSS sync.
func (f Facts) reserved(in []time.Time, now time.Time) int {
	if !f.RSS.On || f.RSS.Every <= 0 {
		return 0
	}
	oldest := now
	if len(in) > 0 {
		oldest = in[0]
	}
	untilFree := oldest.Add(f.Limit.Per.Duration()).Sub(now)
	fetches := int((untilFree + f.RSS.Every - 1) / f.RSS.Every)
	return min(fetches+max(1, f.RSS.LastPages), max(1, f.Limit.Count-1))
}

// windowAt is the reservations inside l's window at now, oldest first.
func windowAt(l Limit, reserved []time.Time, now time.Time) []time.Time {
	in := slices.DeleteFunc(slices.Clone(reserved), func(at time.Time) bool { return now.Sub(at) >= l.Per.Duration() })
	slices.SortFunc(in, time.Time.Compare)
	return in
}

// Reserve reserves one search query on Indexer id for a line of
// typ and claim c in tx, the Write that dispatches it, and writes the
// claim on the query row. It answers:
//   - the QueryID: reserved;
//   - *WaitError: settle the line now with its Outcome. The gate
//     (stopped, limited or failing inside its backoff, waiting on its
//     home Node, removed, no categories for typ), a press at the limit
//     (limit_reached with Until), or a stuck Indexer (late);
//   - *RoomError: the line waits.
func (x *Indexers) Reserve(ctx context.Context, tx *storage.Tx[file], id ID, typ works.TypeID, c Claim) (QueryID, error) {
	idx, err := x.get(ctx, tx, id)
	if errors.Is(err, ErrNoIndexer) {
		return QueryID{}, &WaitError{Outcome: Outcome{Kind: Removed}}
	}
	if err != nil {
		return QueryID{}, err
	}
	if len(served(idx, x.categories(typ))) == 0 {
		return QueryID{}, &WaitError{Outcome: Outcome{Kind: NoCategories}}
	}
	st, err := status(ctx, tx, id)
	if err != nil {
		return QueryID{}, err
	}
	now := storage.Now()
	if _, wait := gate(idx, st, Usage{}, x.cfg.Roster, UseSearch, now.Time()); wait != nil {
		return QueryID{}, &WaitError{Outcome: *wait}
	}
	f, err := x.factsOf(ctx, tx, idx, now.Time())
	if err != nil {
		return QueryID{}, err
	}
	fit := FitOf(f, c, now.Time())
	switch {
	case fit.Skip != nil:
		return QueryID{}, &WaitError{Outcome: *fit.Skip}
	case fit.Wait != nil:
		return QueryID{}, fit.Wait
	}
	q := QueryID{UUID: storage.NewUUID()}
	return q, insertQuery(ctx, tx, q, idx, UseSearch, c, now)
}

func (x *Indexers) factsOf(ctx context.Context, r storage.Reader[file], idx Indexer, now time.Time) (Facts, error) {
	q := db.New(r)
	rows, err := q.InFlightSearchesOf(ctx, db.InFlightSearchesOfParams{IndexerID: idx.ID, Since: storage.FromTime(now.Add(-x.cfg.JobCap))})
	if err != nil {
		return Facts{}, err
	}
	flights := make([]flight, len(rows))
	for i, row := range rows {
		flights[i] = flight{claim: claimNamed(row.Claim), at: row.ReservedAt.Time()}
	}
	window, err := reservedFor(ctx, r, idx.ID, UseSearch, now.Add(-idx.Searches.Per.Duration()))
	if err != nil {
		return Facts{}, err
	}
	pages := 1
	switch row, err := q.GetRSS(ctx, idx.ID); {
	case err == nil:
		pages = int(row.LastPages)
	case !errors.Is(err, storage.ErrNoRows):
		return Facts{}, err
	}
	plan, err := x.rssPlan(ctx, r)
	if err != nil {
		return Facts{}, err
	}
	wait, err := searchWait(ctx, r)
	if err != nil {
		return Facts{}, err
	}
	return factsFrom(idx, flights, window, plan, pages, wait, now), nil
}

// flight is one in-flight search query.
type flight struct {
	claim Claim
	at    time.Time
}

// rss is RSS sync as the reserve reads it, before the Indexer's own
// switches and pages.
type rss struct {
	on    bool
	every time.Duration
}

func (x *Indexers) rssPlan(ctx context.Context, r storage.Reader[file]) (rss, error) {
	if x.cfg.RSS == nil {
		return rss{}, nil
	}
	every, on, err := x.cfg.RSS(ctx, r)
	return rss{on: on, every: every}, err
}

func factsFrom(idx Indexer, flights []flight, window []time.Time, plan rss, pages int, wait time.Duration, now time.Time) Facts {
	f := Facts{
		AtOnce: idx.SearchesAtOnce, InFlight: map[Claim]int{}, Stuck: len(flights) > 0, Limit: idx.Searches, Window: window,
		RSS: RSSPlan{On: plan.on && idx.On && idx.RSS, Every: plan.every, LastPages: pages},
	}
	for _, q := range flights {
		f.InFlight[q.claim]++
		if now.Before(q.at.Add(wait)) {
			f.Stuck = false
		}
	}
	return f
}

// Room is one Indexer's room now, per claim.
type Room struct {
	Name              string
	Press, Background RoomFor
}

// RoomFor is how many lines of one claim an Indexer would take now.
type RoomFor struct {
	Free int
	// Until is when the search limit next has a slot for the claim, zero
	// while it isn't what holds the claim back.
	Until   time.Time
	Starved bool
}

// Room is the room of each Indexer that is on and can search, per claim,
// read in r: the read twin of Reserve, for the status sentence ("Waiting
// for TorrentLeech's search limit until 14:00.") and the hero's Behind. It
// reads no gate and no grabs row.
func (x *Indexers) Room(ctx context.Context, r storage.Reader[file]) (map[ID]Room, error) {
	all, err := x.all(ctx, r)
	if err != nil {
		return nil, err
	}
	now := storage.Now().Time()
	q := db.New(r)
	inflight, err := q.InFlightSearches(ctx, storage.FromTime(now.Add(-x.cfg.JobCap)))
	if err != nil {
		return nil, err
	}
	flights := map[ID][]flight{}
	for _, row := range inflight {
		flights[row.IndexerID] = append(flights[row.IndexerID], flight{claim: claimNamed(row.Claim), at: row.ReservedAt.Time()})
	}
	reserved, err := q.ReservedSince(ctx, storage.FromTime(now.Add(-Day.Duration())))
	if err != nil {
		return nil, err
	}
	windows := map[ID][]time.Time{}
	for _, row := range reserved {
		if Use(row.Use) == UseSearch || Use(row.Use) == UseRecent {
			windows[row.IndexerID] = append(windows[row.IndexerID], row.ReservedAt.Time())
		}
	}
	stored, err := q.ListRSSPages(ctx)
	if err != nil {
		return nil, err
	}
	pages := map[ID]int{}
	for _, row := range stored {
		pages[row.IndexerID] = int(row.LastPages)
	}
	plan, err := x.rssPlan(ctx, r)
	if err != nil {
		return nil, err
	}
	wait, err := searchWait(ctx, r)
	if err != nil {
		return nil, err
	}
	out := map[ID]Room{}
	for _, idx := range all {
		if !idx.On || idx.Kind == RSS {
			continue
		}
		f := factsFrom(idx, flights[idx.ID], windows[idx.ID], plan, max(1, pages[idx.ID]), wait, now)
		out[idx.ID] = Room{Name: idx.Name, Press: roomFor(f, ClaimPress, now), Background: roomFor(f, ClaimBackground, now)}
	}
	return out, nil
}

// roomFor is how many lines of claim c FitOf takes in a row from f at
// now, and what stops the next one.
func roomFor(f Facts, c Claim, now time.Time) RoomFor {
	var out RoomFor
	f.InFlight = map[Claim]int{ClaimPress: f.InFlight[ClaimPress], ClaimBackground: f.InFlight[ClaimBackground]}
	f.Window = slices.Clone(f.Window)
	for {
		fit := FitOf(f, c, now)
		switch {
		case fit.Reserve:
			out.Free++
			f.InFlight[c]++
			f.Stuck = false
			f.Window = append(f.Window, now)
			continue
		case fit.Wait != nil:
			out.Until, out.Starved = fit.Wait.Until, fit.Wait.Starved
		case fit.Skip.Kind == LimitReached:
			out.Until = fit.Skip.Until
		}
		return out
	}
}
