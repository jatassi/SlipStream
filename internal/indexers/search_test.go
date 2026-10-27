package indexers_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

// scripted is a Node that answers each job at once: with its Indexer's
// answer, else no Releases.
type scripted struct {
	mu      sync.Mutex
	answers map[string]searching.Answer
	jobs    []searching.Job
}

func (n *scripted) Run(_ context.Context, j searching.Job) (searching.Poll, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.jobs = append(n.jobs, j)
	a := n.answers[j.Indexer]
	a.Started = time.Now()
	return searching.Poll{Started: a.Started, Answer: &a}, nil
}

func (n *scripted) answer(id indexers.ID, a searching.Answer) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.answers[id.String()] = a
}

// planned is Indexers with saved Torznab Indexers by name, all homed on
// home and answered by node.
type planned struct {
	t    *testing.T
	lib  *storage.DB[storage.Library]
	x    *indexers.Indexers
	node *scripted
	ids  map[string]indexers.ID
	// repointed is each merge's from and into, as the Repoints saw them.
	repointed [][2]indexers.TrackerID
}

var self = role.NodeID{UUID: storage.NewUUID()}

func plannedWith(t *testing.T, names ...string) *planned {
	t.Helper()
	return plannedIn(t, indexerstest.Alone(self), self, names...)
}

// plannedIn is planned with roster's Nodes, its Indexers homed on home.
func plannedIn(t *testing.T, roster indexerstest.Roster, home role.NodeID, names ...string) *planned {
	t.Helper()
	sealer := secrettest.Sealer(t)
	lib := storagetest.Library(t, secret.Schema(), nodes.Schema(), works.Schema(), indexers.Schema())
	p := &planned{t: t, lib: lib, node: &scripted{answers: map[string]searching.Answer{}}, ids: map[string]indexers.ID{}}
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, n := range roster.Holders {
			if err := nodestest.Insert(ctx, tx, n, roster.Name(n)); err != nil {
				return err
			}
		}
		for _, name := range names {
			p.ids[name] = indexers.ID{UUID: storage.NewUUID()}
			if err := indexerstest.Torznab(ctx, tx, sealer, p.ids[name], name, home, "https://site.test/torznab", "k"); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	fx := contenttypetest.New()
	w, err := works.Open(t.Context(), works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	p.x = indexers.New(indexers.Config{
		Library: lib, Sealer: sealer, Roster: roster,
		Searching: func(role.NodeID) searching.Remote { return p.node },
		Types:     map[works.TypeID]indexers.Facet{indexerstest.Type: indexerstest.Facet{}, contenttypetest.ID: fx},
		Works:     w,
		Changed:   discard[indexers.Changed]{}, Status: discard[indexers.StatusChanged]{}, Trackers: discard[indexers.TrackerChanged]{},
		Repoints: []indexers.Repoint{func(_ context.Context, _ *storage.Tx[storage.Library], from, into indexers.TrackerID) error {
			p.repointed = append(p.repointed, [2]indexers.TrackerID{from, into})
			return nil
		}},
		Timings: indexers.DefaultTimings(), Zone: time.UTC,
		Log: slog.New(slog.DiscardHandler),
	})
	return p
}

func (p *planned) edit(name string, e indexers.Edit) {
	p.t.Helper()
	if _, err := p.x.Update(p.t.Context(), identity.Admin{}, p.ids[name], e, nil); err != nil {
		p.t.Fatal(err)
	}
}

func (p *planned) plan() []indexers.Line {
	p.t.Helper()
	var lines []indexers.Line
	err := p.lib.Write(p.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		lines, err = p.x.Plan(ctx, tx, indexerstest.Type)
		return err
	})
	if err != nil {
		p.t.Fatal(err)
	}
	return lines
}

// search asks the lines that plan asks.
func (p *planned) search(lines []indexers.Line) []indexers.Answer {
	p.t.Helper()
	asked := slices.DeleteFunc(slices.Clone(lines), func(l indexers.Line) bool { return l.Skipped != nil })
	answers, err := p.x.Search(p.t.Context(), indexers.Search{Type: indexerstest.Type, Terms: indexers.Terms{Mode: "search", Query: "orchard"}, Asked: asked})
	if err != nil {
		p.t.Fatal(err)
	}
	return answers
}

// spelled is each line as its name and asked, or skipped and why.
func spelled(lines []indexers.Line) []string {
	var out []string
	for _, l := range lines {
		switch {
		case l.Skipped == nil && !l.Query.IsZero():
			out = append(out, l.Name+" asked")
		case l.Skipped != nil:
			out = append(out, l.Name+" "+string(l.Skipped.Kind))
		default:
			out = append(out, l.Name+" neither asked nor skipped")
		}
	}
	return out
}

func TestPlanAsksEachIndexerByPriorityAndSaysWhyItSkipsOne(t *testing.T) {
	p := plannedWith(t, "Alder", "Birch", "Cedar", "Dune", "Elm", "Fir", "Gale")
	off, priority, tvOnly := false, 10, []newznab.ID{5000}
	p.edit("Alder", indexers.Edit{Priority: new(30)})
	p.edit("Birch", indexers.Edit{Priority: &priority})
	p.edit("Cedar", indexers.Edit{On: &off})
	p.edit("Dune", indexers.Edit{AutoSearch: &off})
	p.edit("Elm", indexers.Edit{Categories: &tvOnly})
	p.edit("Fir", indexers.Edit{Searches: &indexers.Limit{Count: 1, Per: indexers.Hour}})
	p.node.answer(p.ids["Birch"], searching.Answer{Releases: []searching.Release{{Title: "Orchard.1080p"}}})
	p.node.answer(p.ids["Gale"], searching.Answer{Outcome: &searching.Stopped{Cause: searching.AccountRefused, Text: "Account suspended"}})

	first := p.plan()
	if got, want := spelled(first), []string{"Birch asked", "Elm no_categories", "Fir asked", "Gale asked", "Alder asked"}; !slices.Equal(got, want) {
		t.Fatalf("the first plan = %q, want %q", got, want)
	}
	var answered []string
	for _, a := range p.search(first) {
		answered = append(answered, fmt.Sprintf("%s %d", a.Outcome.Kind, len(a.Releases)))
	}
	if want := []string{"found 1", "no_releases 0", "stopped 0", "no_releases 0"}; !slices.Equal(answered, want) {
		t.Fatalf("the answers = %q, want %q", answered, want)
	}
	for _, j := range p.node.jobs {
		if !slices.Equal(j.Categories, []int{2000}) {
			t.Fatalf("Indexer %s was asked for categories %v, want the categories it has in family 2000, [2000]", j.Indexer, j.Categories)
		}
	}

	second := p.plan()
	if got, want := spelled(second), []string{"Birch asked", "Elm no_categories", "Fir limit_reached", "Gale stopped", "Alder asked"}; !slices.Equal(got, want) {
		t.Fatalf("the second plan = %q, want %q", got, want)
	}
	if until := second[2].Skipped.Until; until.Before(time.Now().Add(59*time.Minute)) || until.After(time.Now().Add(time.Hour)) {
		t.Fatalf("Fir's search limit frees at %s, want an hour after its first search", until)
	}
}

func TestAnIndexerDeletedSinceThePlanAnswersRemoved(t *testing.T) {
	p := plannedWith(t, "Alder", "Birch")
	lines := p.plan()
	if err := p.x.Delete(t.Context(), identity.Admin{}, p.ids["Alder"]); err != nil {
		t.Fatal(err)
	}
	answers := p.search(lines)
	if answers[0].Outcome.Kind != indexers.Removed || answers[1].Outcome.Kind != indexers.NoReleases {
		t.Fatalf("the answers = %+v, want Alder removed and Birch with no Releases", answers)
	}
}

func (p *planned) take(name string, r searching.Release) (indexers.Ticket, error) {
	p.t.Helper()
	var t indexers.Ticket
	err := p.lib.Write(p.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		t, err = p.x.Take(ctx, tx, indexers.Release{Indexer: p.ids[name], Release: r})
		return err
	})
	return t, err
}

func notTaken(err error, kind indexers.OutcomeKind) bool {
	e, ok := errors.AsType[*indexers.NotTakenError](err)
	return ok && e.Outcome.Kind == kind
}

func TestTakeReservesAGrabUnlessTheFetchWouldWaitForTheAdmin(t *testing.T) {
	p := plannedWith(t, "Alder", "Birch")
	p.edit("Alder", indexers.Edit{Grabs: &indexers.Limit{Count: 1, Per: indexers.Hour}})
	torrent := searching.Release{Title: "Orchard.1080p", Link: searching.LinkOf("https://site.test/a.torrent")}
	magnet := searching.Release{Title: "Orchard.720p", Magnet: searching.LinkOf("magnet:?xt=urn:btih:" + strings.Repeat("a", 40))}

	if tk, err := p.take("Alder", magnet); err != nil || !tk.Query.IsZero() {
		t.Fatalf("taking a magnet link = %+v, %v; want a Ticket with no grab reserved", tk, err)
	}
	if tk, err := p.take("Alder", torrent); err != nil || tk.Query.IsZero() {
		t.Fatalf("taking a .torrent = %+v, %v; want a grab reserved", tk, err)
	}
	if _, err := p.take("Alder", torrent); !notTaken(err, indexers.LimitReached) {
		t.Fatalf("taking a second .torrent past a grab limit of one an hour = %v, want limit_reached", err)
	}
	p.node.answer(p.ids["Birch"], searching.Answer{Outcome: &searching.Stopped{Cause: searching.AccountRefused, Text: "Account suspended"}})
	p.search(p.plan())
	if _, err := p.take("Birch", torrent); !notTaken(err, indexers.IsStopped) {
		t.Fatalf("taking a stopped Indexer's .torrent = %v, want stopped", err)
	}
	if err := p.x.Delete(t.Context(), identity.Admin{}, p.ids["Birch"]); err != nil {
		t.Fatal(err)
	}
	if _, err := p.take("Birch", torrent); !notTaken(err, indexers.Removed) {
		t.Fatalf("taking a deleted Indexer's .torrent = %v, want removed", err)
	}
}

func TestAFetchDropsOnceItsIndexerStopsAndWaitsWhileItFails(t *testing.T) {
	p := plannedWith(t, "Alder", "Birch")
	link := searching.LinkOf("https://site.test/a.torrent")
	tk, err := p.take("Alder", searching.Release{Title: "Orchard.1080p", Link: link})
	if err != nil {
		t.Fatal(err)
	}
	birch, err := p.take("Birch", searching.Release{Title: "Orchard.1080p", Link: link})
	if err != nil {
		t.Fatal(err)
	}
	raw := "d4:infod4:name1:aee"
	p.node.answer(p.ids["Alder"], searching.Answer{Got: searching.Torrent{Raw: []byte(raw)}})
	if got, err := p.x.Fetch(t.Context(), tk, link); err != nil || !reflect.DeepEqual(got, indexers.Torrent{Raw: []byte(raw), Node: self}) {
		t.Fatalf("Fetch = %+v, %v; want the .torrent fetched on home", got, err)
	}

	p.node.answer(p.ids["Alder"], searching.Answer{Outcome: &searching.Failing{Err: errors.New("the Indexer responded with status 500")}})
	_, err = p.x.Fetch(t.Context(), tk, link)
	if wait, ok := errors.AsType[*indexers.WaitError](err); !ok || wait.Outcome.Kind != indexers.IsFailing || wait.Outcome.Until.IsZero() {
		t.Fatalf("a failing fetch = %v, want a wait until its retry", err)
	}
	waits, err := fetchWaits(p, "Alder")
	if want := "Waiting to fetch the .torrent from Alder. SlipStream retries at "; err != nil || !strings.HasPrefix(waits.Sentence, want) {
		t.Fatalf("FetchWaits = %+v, %v; want %q and its time", waits, err, want)
	}

	p.node.answer(p.ids["Birch"], searching.Answer{Outcome: &searching.Stopped{Cause: searching.AccountRefused, Text: "Account suspended"}})
	if _, err := p.x.Fetch(t.Context(), birch, link); !isDrop(err, indexers.DropStopped) {
		t.Fatalf("a fetch the site refused = %v, want a drop", err)
	}
	if _, err := p.x.Fetch(t.Context(), birch, link); !isDrop(err, indexers.DropStopped) {
		t.Fatalf("a fetch from a stopped Indexer = %v, want a drop", err)
	}
	if err := p.x.Delete(t.Context(), identity.Admin{}, p.ids["Birch"]); err != nil {
		t.Fatal(err)
	}
	if _, err := p.x.Fetch(t.Context(), birch, link); !isDrop(err, indexers.DropRemoved) {
		t.Fatalf("a fetch from a deleted Indexer = %v, want a drop", err)
	}
}

func TestAStopMetOnAFailoverNodeWaitsForHome(t *testing.T) {
	box := role.NodeID{UUID: storage.NewUUID()}
	roster := indexerstest.Roster{
		Controller: self, Holders: []role.NodeID{self, box}, Names: map[role.NodeID]string{self: "home", box: "seedbox"},
		Absent: map[role.NodeID]nodes.Absence{box: {Since: time.Now(), Lapsed: true}},
	}
	p := plannedIn(t, roster, box, "Alder")
	link := searching.LinkOf("https://site.test/a.torrent")
	tk, err := p.take("Alder", searching.Release{Title: "Orchard.1080p", Link: link})
	if err != nil {
		t.Fatal(err)
	}
	p.node.answer(p.ids["Alder"], searching.Answer{Outcome: &searching.Stopped{Cause: searching.ChallengeUnsolved}})
	answers := p.search(p.plan())
	if len(answers) != 1 || answers[0].Outcome != (indexers.Outcome{Kind: indexers.Waiting, Node: box}) {
		t.Fatalf("a search whose failover Node met a challenge = %+v, want Alder waiting for seedbox", answers)
	}
	if line := p.x.Line("Alder", indexerstest.Type, &answers[0].Outcome); line != "Waiting for seedbox" {
		t.Errorf("its line = %q, want Waiting for seedbox", line)
	}
	if lines := spelled(p.plan()); !slices.Equal(lines, []string{"Alder waiting"}) {
		t.Errorf("the next plan = %q, want Alder waiting for seedbox rather than asked on home", lines)
	}
	if _, err := p.x.Fetch(t.Context(), tk, link); !isDrop(err, indexers.DropHomeOffline) {
		t.Errorf("a fetch while Alder waits for its lapsed home = %v, want a drop for the offline home", err)
	}
}

func isDrop(err error, why indexers.DropWhy) bool {
	drop, ok := errors.AsType[*indexers.DropError](err)
	return ok && drop.Why == why
}

func fetchWaits(p *planned, name string) (indexers.FetchWait, error) {
	var waits map[indexers.ID]indexers.FetchWait
	err := p.lib.Read(p.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		waits, err = p.x.FetchWaits(ctx, r, []indexers.ID{p.ids[name]})
		return err
	})
	return waits[p.ids[name]], err
}

func TestEachOutcomeLineSaysItInWords(t *testing.T) {
	p := plannedWith(t)
	at := time.Date(2026, 10, 17, 14, 5, 0, 0, time.UTC)
	for _, tc := range []struct {
		o    *indexers.Outcome
		want string
	}{
		{nil, "Searching"},
		{&indexers.Outcome{Kind: indexers.Found, Releases: 1}, "Found 1 Release"},
		{&indexers.Outcome{Kind: indexers.Found, Releases: 12}, "Found 12 Releases"},
		{&indexers.Outcome{Kind: indexers.NoReleases}, "No Releases"},
		{&indexers.Outcome{Kind: indexers.IsLimited, Until: at}, "Limit reached until 14:05"},
		{&indexers.Outcome{Kind: indexers.LimitReached, Until: at}, "Limit reached until 14:05"},
		{&indexers.Outcome{Kind: indexers.Waiting, Node: self}, "Waiting for home"},
		{&indexers.Outcome{Kind: indexers.IsFailing, Until: at}, "TorrentLeech isn't responding. SlipStream retries at 14:05."},
		{&indexers.Outcome{Kind: indexers.IsStopped, Cause: searching.CredentialsRejected}, "Stopped because TorrentLeech rejected the username or password."},
		{&indexers.Outcome{Kind: indexers.IsStopped, Cause: searching.AccountRefused, Text: "Account suspended"}, "Stopped because TorrentLeech refused the API key with 'Account suspended'."},
		{&indexers.Outcome{Kind: indexers.NoCategories}, "Skipped because it has no shelf categories"},
		{&indexers.Outcome{Kind: indexers.Late}, "Didn't finish in time"},
		{&indexers.Outcome{Kind: indexers.Removed}, "Deleted since this search"},
	} {
		if got := p.x.Line("TorrentLeech", contenttypetest.ID, tc.o); got != tc.want {
			t.Errorf("Line(%+v) = %q, want %q", tc.o, got, tc.want)
		}
	}
}

func TestPlanNeverAsksAnRSSFeed(t *testing.T) {
	for _, tc := range []struct {
		name          string
		on, auto, rss bool
		media         []works.TypeID
	}{
		{"on with Automatic search and RSS", true, true, true, []works.TypeID{indexerstest.Type}},
		{"on with Automatic search, RSS off", true, true, false, []works.TypeID{indexerstest.Type}},
		{"carrying another type too", true, true, true, []works.TypeID{indexerstest.Type, contenttypetest.ID}},
		{"off", false, true, true, []works.TypeID{indexerstest.Type}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := plannedWith(t, "Alder")
			_, err := p.x.Create(t.Context(), identity.Admin{}, indexers.Draft{
				Kind: indexers.RSS, Name: "Brook", Address: "https://brook.test/rss", Tracker: "Brook", Media: tc.media,
				On: tc.on, AutoSearch: tc.auto, RSS: tc.rss,
			}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := spelled(p.plan()), []string{"Alder asked"}; !slices.Equal(got, want) {
				t.Fatalf("the plan = %q, want only the Torznab feed asked", got)
			}
		})
	}
}
