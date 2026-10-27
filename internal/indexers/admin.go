package indexers

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Draft is a Torznab or RSS feed as an Admin enters it.
type Draft struct {
	// Kind is Torznab when empty.
	Kind Kind
	Name string
	// Home is zero for the default home.
	Home role.NodeID
	// Address is a Torznab feed's base, to which the Node adds /api, or an
	// RSS feed's whole address.
	Address string
	APIKey  string
	// Tracker names the Tracker the feed reaches when its items name no
	// source, "" for the Indexer's name.
	Tracker string
	// Media is the Content Types an RSS feed carries.
	Media       []works.TypeID
	AllowNoSize bool
	// Cookie is an RSS feed's, nil for none.
	Cookie        *PastedCookie
	On            bool
	AutoSearch    bool
	RSS           bool
	AllowFailover bool
	Priority      int
	Categories    []newznab.ID
	Searches      Limit
	Grabs         Limit
}

// Edit is a change to a saved Indexer: nil leaves a field. Categories
// set to a nil slice means every category.
type Edit struct {
	Name                               *string
	Home                               *role.NodeID
	Address, APIKey                    *string
	On, AutoSearch, RSS, AllowFailover *bool
	Priority                           *int
	Categories                         *[]newznab.ID
	Searches, Grabs                    *Limit
	Media                              *[]works.TypeID
	AllowNoSize                        *bool
	Cookie                             *PastedCookie
}

// tested reports whether e changes a field the Test reads.
func (e Edit) tested() bool {
	return e.Home != nil || e.Address != nil || e.APIKey != nil || e.AllowNoSize != nil || e.Cookie != nil
}

// Report is what a passing Test learned.
type Report struct {
	Caps  Caps
	Items int
	// Sources are the different source names a Torznab feed's items
	// named.
	Sources []string
}

// Create runs the Test on the Node the Indexer will search from, then
// saves in one Write the Indexer at revision 1, its status, the session
// the Test earned, the Test's query row and, when the Test's items named
// no source, the Tracker the feed reaches. It fails with
// *TestFailedError, ErrNameTaken, ErrNoSearchingNode, ErrNotSearching,
// *WaitError, *UnavailableError or ErrLate.
func (x *Indexers) Create(ctx context.Context, by identity.Admin, d Draft) (View, error) {
	if err := x.nameFree(ctx, x.cfg.Library, d.Name, ID{}); err != nil {
		return View{}, err
	}
	home, err := x.homeOf(d.Home)
	if err != nil {
		return View{}, err
	}
	created := storage.Now()
	idx := Indexer{
		ID: ID{UUID: storage.NewUUID()}, Name: d.Name, Kind: cmp.Or(d.Kind, Torznab), Home: home, On: d.On, AutoSearch: d.AutoSearch, RSS: d.RSS,
		AllowFailover: d.AllowFailover, Priority: cmp.Or(d.Priority, DefaultPriority), Categories: d.Categories, Media: d.Media,
		Searches: d.Searches, Grabs: d.Grabs, Source: Feed{Tracker: cmp.Or(d.Tracker, d.Name), AllowNoSize: d.AllowNoSize}, FeedHost: hostOf(d.Address),
		Revision: 1, Created: created.Time(),
	}
	if err := checkMedia(idx, x.cfg.Types); err != nil {
		return View{}, err
	}
	session, err := pasted(d.Cookie, idx)
	if err != nil {
		return View{}, err
	}
	if idx.Secrets, err = x.seal(idx.ID, Secrets{Address: d.Address, APIKey: d.APIKey}); err != nil {
		return View{}, err
	}
	q, node, a, err := x.test(ctx, idx, session)
	if err != nil {
		return View{}, err
	}
	idx = read(idx, a)
	var named TrackerID
	err = x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if err := x.nameFree(ctx, tx, idx.Name, ID{}); err != nil {
			return err
		}
		p, err := row(idx, created)
		if err != nil {
			return err
		}
		if err := db.New(tx).InsertIndexer(ctx, p); err != nil {
			return err
		}
		if len(reportOf(a.Report).Sources) == 0 {
			if named, _, err = reachTracker(ctx, tx, idx.ID, reachedBy(idx, Release{})); err != nil {
				return err
			}
		}
		return x.recordTest(ctx, tx, q, node, idx, a, session, created)
	})
	if err != nil {
		return View{}, err
	}
	if !named.IsZero() {
		x.cfg.Trackers.Publish(ctx, TrackerChanged{TrackerID: named.String()})
	}
	x.saved(ctx, idx.ID)
	return x.Get(ctx, by, idx.ID)
}

// TestDraft tests d without saving anything.
func (x *Indexers) TestDraft(ctx context.Context, _ identity.Admin, d Draft) (Report, error) {
	home, err := x.homeOf(d.Home)
	if err != nil {
		return Report{}, err
	}
	idx := Indexer{
		ID: ID{UUID: storage.NewUUID()}, Name: d.Name, Kind: cmp.Or(d.Kind, Torznab), Home: home, AllowFailover: d.AllowFailover, Media: d.Media,
		Source: Feed{Tracker: d.Tracker, AllowNoSize: d.AllowNoSize}, FeedHost: hostOf(d.Address), Revision: 1,
	}
	if err := checkMedia(idx, x.cfg.Types); err != nil {
		return Report{}, err
	}
	session, err := pasted(d.Cookie, idx)
	if err != nil {
		return Report{}, err
	}
	if idx.Secrets, err = x.seal(idx.ID, Secrets{Address: d.Address, APIKey: d.APIKey}); err != nil {
		return Report{}, err
	}
	_, _, a, err := x.test(ctx, idx, session)
	if err != nil {
		return Report{}, err
	}
	return reportOf(a.Report), nil
}

// Update saves e in one Write that bumps the revision, running the Test
// first when e changes a field the Test reads. It fails as Create does,
// and with ErrNoIndexer.
func (x *Indexers) Update(ctx context.Context, by identity.Admin, id ID, e Edit) (View, error) {
	idx, err := Get(ctx, x.cfg.Library, id)
	if err != nil {
		return View{}, err
	}
	if e.Name != nil {
		if err := x.nameFree(ctx, x.cfg.Library, *e.Name, id); err != nil {
			return View{}, err
		}
	}
	if idx, err = x.edited(idx, e); err != nil {
		return View{}, err
	}
	session, err := pasted(e.Cookie, idx)
	if err != nil {
		return View{}, err
	}
	var q QueryID
	var node role.NodeID
	var a searching.Answer
	if e.tested() {
		if q, node, a, err = x.test(ctx, idx, session); err != nil {
			return View{}, err
		}
		idx = read(idx, a)
	}
	err = x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if e.Name != nil {
			if err := x.nameFree(ctx, tx, *e.Name, id); err != nil {
				return err
			}
		}
		now := storage.Now()
		p, err := row(idx, now)
		if err != nil {
			return err
		}
		n, err := db.New(tx).UpdateIndexer(ctx, db.UpdateIndexerParams{
			Name: p.Name, HomeNodeID: p.HomeNodeID, IsOn: p.IsOn, AutoSearch: p.AutoSearch, Rss: p.Rss, AllowFailover: p.AllowFailover,
			Priority: p.Priority, Categories: p.Categories, Media: p.Media, SearchLimit: p.SearchLimit, SearchWindow: p.SearchWindow,
			GrabLimit: p.GrabLimit, GrabWindow: p.GrabWindow, BaseAddress: p.BaseAddress, Plain: p.Plain, SecretsSealed: p.SecretsSealed,
			FeedHost: p.FeedHost, TrackerName: p.TrackerName, AllowNoSize: p.AllowNoSize, RssReading: p.RssReading, Caps: p.Caps, Revision: p.Revision,
			UpdatedAt: now, ID: id,
		})
		switch {
		case err != nil:
			return err
		case n == 0:
			return ErrNoIndexer
		case !e.tested():
			return nil
		}
		return x.recordTest(ctx, tx, q, node, idx, a, session, now)
	})
	if err != nil {
		return View{}, err
	}
	x.saved(ctx, id)
	return x.Get(ctx, by, id)
}

// edited is idx with e applied at the next revision.
func (x *Indexers) edited(idx Indexer, e Edit) (Indexer, error) {
	if e.Home != nil {
		if !slices.Contains(x.cfg.Roster.Holding(role.Searching), *e.Home) {
			return Indexer{}, ErrNotSearching
		}
		idx.Home = *e.Home
	}
	if e.Address != nil || e.APIKey != nil {
		s, err := x.open(idx)
		if err != nil {
			return Indexer{}, err
		}
		if e.Address != nil {
			s.Address, idx.FeedHost = *e.Address, hostOf(*e.Address)
		}
		if e.APIKey != nil {
			s.APIKey = *e.APIKey
		}
		if idx.Secrets, err = x.seal(idx.ID, s); err != nil {
			return Indexer{}, err
		}
	}
	set(&idx.Name, e.Name)
	set(&idx.On, e.On)
	set(&idx.AutoSearch, e.AutoSearch)
	set(&idx.RSS, e.RSS)
	set(&idx.AllowFailover, e.AllowFailover)
	set(&idx.Priority, e.Priority)
	set(&idx.Categories, e.Categories)
	set(&idx.Searches, e.Searches)
	set(&idx.Grabs, e.Grabs)
	set(&idx.Media, e.Media)
	if f, ok := idx.Source.(Feed); ok {
		set(&f.AllowNoSize, e.AllowNoSize)
		idx.Source = f
	}
	idx.Revision++
	return idx, checkMedia(idx, x.cfg.Types)
}

func set[T any](to, v *T) {
	if v != nil {
		*to = *v
	}
}

// Test runs idx's Test where it searches from now and records it. A pass
// clears its backoff and any stop, even for an RSS feed whose Test found
// it can't be used. It fails with *TestFailedError, *FeedRefusedError,
// ErrNoIndexer, *WaitError, *UnavailableError or ErrLate.
func (x *Indexers) Test(ctx context.Context, _ identity.Admin, id ID) (Report, error) {
	return x.testSaved(ctx, id)
}

// testSaved is Test for an Admin or the health check.
func (x *Indexers) testSaved(ctx context.Context, id ID) (Report, error) {
	idx, err := Get(ctx, x.cfg.Library, id)
	if err != nil {
		return Report{}, err
	}
	q := QueryID{UUID: storage.NewUUID()}
	j, err := x.routed(ctx, q, idx, UseTest, searching.TestCall{})
	if err != nil {
		return Report{}, err
	}
	err = x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return insertQuery(ctx, tx, q, idx, UseTest, storage.Now())
	})
	if err != nil {
		return Report{}, err
	}
	a, st, err := x.run(ctx, j)
	if err != nil {
		return Report{}, err
	}
	if a.Outcome != nil {
		return Report{}, x.testFailed(idx, j.node, a, st)
	}
	if err := refused(a); err != nil {
		return Report{}, err
	}
	return reportOf(a.Report), nil
}

// test runs idx's Test, unsaved or at its next revision, on the Node it
// will search from, with session in place of its stored one unless nil,
// and returns the job's query id and the answer once it passed. Nothing
// is recorded: the caller saves the answer with the edit.
func (x *Indexers) test(ctx context.Context, idx Indexer, session *searching.Session) (QueryID, role.NodeID, searching.Answer, error) {
	node, wait := route(x.cfg.Roster, idx, Status{State: OK})
	if wait != nil {
		return QueryID{}, role.NodeID{}, searching.Answer{}, &WaitError{Outcome: *wait}
	}
	q := QueryID{UUID: storage.NewUUID()}
	j, err := x.prepare(ctx, q, idx, UseTest, node, searching.TestCall{})
	if err != nil {
		return QueryID{}, role.NodeID{}, searching.Answer{}, err
	}
	if session != nil {
		j.wire.Session = session
	}
	a, err := x.ask(ctx, j)
	if err != nil {
		return QueryID{}, role.NodeID{}, searching.Answer{}, err
	}
	if a.Outcome != nil {
		return QueryID{}, role.NodeID{}, searching.Answer{}, x.testFailed(idx, node, a, Status{})
	}
	if err := refused(a); err != nil {
		return QueryID{}, role.NodeID{}, searching.Answer{}, err
	}
	return q, node, a, nil
}

// recordTest saves a passing Test of idx as query row q, with its status,
// session and caps, and the session pasted for it unless nil, inside the
// Write that saves idx.
func (x *Indexers) recordTest(ctx context.Context, tx *storage.Tx[file], q QueryID, node role.NodeID, idx Indexer, a searching.Answer, session *searching.Session, now storage.Time) error {
	if err := insertQuery(ctx, tx, q, idx, UseTest, now); err != nil {
		return err
	}
	if session != nil {
		if err := x.offer(ctx, tx, idx.ID, node, *session); err != nil {
			return err
		}
	}
	j := job{q: q, idx: idx, use: UseTest, node: node}
	_, err := x.recordIn(ctx, tx, q, idx.ID, j.answer(a), now)
	return err
}

func reportOf(r *searching.Report) Report {
	if r == nil {
		return Report{}
	}
	return Report{Caps: Caps{Modes: r.Capabilities.Modes, Categories: CategoriesOf(r.Capabilities.Categories)}, Items: r.Items, Sources: r.Sources}
}

// Delete removes Indexer id with its status, sessions and query rows in
// one Write. The foreign keys null its history in grabs.
func (x *Indexers) Delete(ctx context.Context, _ identity.Admin, id ID) error {
	err := x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		n, err := db.New(tx).DeleteIndexer(ctx, id)
		if err == nil && n == 0 {
			return ErrNoIndexer
		}
		return err
	})
	if err != nil {
		return err
	}
	x.changed(ctx, id)
	return nil
}

func (x *Indexers) saved(ctx context.Context, id ID) {
	x.changed(ctx, id)
	x.cfg.Status.Publish(ctx, StatusChanged{IndexerID: id.String()})
}

func (x *Indexers) nameFree(ctx context.Context, r storage.Reader[file], name string, except ID) error {
	taken, err := db.New(r).NameTaken(ctx, db.NameTakenParams{Name: name, Except: except})
	if err == nil && taken {
		return ErrNameTaken
	}
	return err
}

// homeOf is home when it holds Searching, or the default home for zero.
func (x *Indexers) homeOf(home role.NodeID) (role.NodeID, error) {
	if home.IsZero() {
		return defaultHome(x.cfg.Roster)
	}
	if !slices.Contains(x.cfg.Roster.Holding(role.Searching), home) {
		return role.NodeID{}, ErrNotSearching
	}
	return home, nil
}

func hostOf(address string) string {
	u, err := url.Parse(address)
	if err != nil {
		return ""
	}
	return u.Host
}

// View is an Indexer with what its page shows.
type View struct {
	Indexer
	Status Status
	Usage  Usage
	// Session is nil while it holds none.
	Session *SessionFact
	// Waits is why its searches wait now, nil when they don't.
	Waits *Outcome
	// SignsIn is true when it holds a login session, so it waits for its
	// home rather than failing over unless AllowFailover.
	SignsIn  bool
	Trackers []TrackerID
	// Sentence is its status in words, and Fix what the Admin can do about
	// it, "" when nothing.
	Sentence, Fix string
}

// List is every Indexer by name, viewed.
func (x *Indexers) List(ctx context.Context, _ identity.Admin) ([]View, error) {
	var out []View
	err := x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		all, err := All(ctx, r)
		if err != nil {
			return err
		}
		out, err = x.views(ctx, r, all)
		return err
	})
	return out, err
}

// Get is Indexer id, viewed.
func (x *Indexers) Get(ctx context.Context, _ identity.Admin, id ID) (View, error) {
	var out []View
	err := x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		idx, err := Get(ctx, r, id)
		if err != nil {
			return err
		}
		out, err = x.views(ctx, r, []Indexer{idx})
		return err
	})
	if err != nil {
		return View{}, err
	}
	return out[0], nil
}

func (x *Indexers) views(ctx context.Context, r storage.Reader[file], all []Indexer) ([]View, error) {
	now := time.Now()
	q := db.New(r)
	statuses, err := q.ListStatuses(ctx)
	if err != nil {
		return nil, err
	}
	byID := map[ID]Status{}
	for _, row := range statuses {
		byID[row.IndexerID] = statusOf(row)
	}
	reserved, err := q.ReservedSince(ctx, storage.FromTime(now.Add(-Day.Duration())))
	if err != nil {
		return nil, err
	}
	sessions, err := sessionFacts(ctx, r)
	if err != nil {
		return nil, err
	}
	reach, err := reachOf(ctx, r)
	if err != nil {
		return nil, err
	}
	out := make([]View, len(all))
	for i, idx := range all {
		st, ok := byID[idx.ID]
		if !ok {
			st = Status{State: OK}
		}
		v := View{Indexer: idx, Status: st, Usage: usage(idx, reserved, now)}
		if s, ok := sessions[idx.ID]; ok {
			v.Session = &s
		}
		_, v.Waits = gate(idx, st, v.Usage, x.cfg.Roster, UseSearch, now)
		v.SignsIn, v.Trackers = holdsSession(idx), reach.trackers[idx.ID]
		v.Sentence, v.Fix = x.words(v)
		out[i] = v
	}
	return out, nil
}

// Queries is one page of Indexer id's query rows, newest first.
func (x *Indexers) Queries(ctx context.Context, _ identity.Admin, id ID, p Page) ([]QueryRow, error) {
	var out []QueryRow
	err := x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		if _, err := Get(ctx, r, id); err != nil {
			return err
		}
		var err error
		out, err = readQueries(ctx, r, id, p)
		return err
	})
	if err != nil && !errors.Is(err, ErrNoIndexer) {
		return nil, fmt.Errorf("read Indexer %s's queries: %w", id, err)
	}
	return out, err
}
