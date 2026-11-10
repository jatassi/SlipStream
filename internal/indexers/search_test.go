package indexers_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

// planned is Indexers with saved Torznab Indexers by name, all homed on
// home and answered by node.
type planned struct {
	t    *testing.T
	lib  *storage.DB[storage.Library]
	x    *indexers.Indexers
	node *indexerstest.Node
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
	p := &planned{t: t, lib: lib, node: &indexerstest.Node{}, ids: map[string]indexers.ID{}}
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
		Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{},
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
	if _, err := p.x.Update(p.t.Context(), identity.Admin{}, p.ids[name], e, indexers.Typed{}); err != nil {
		p.t.Fatal(err)
	}
}

func (p *planned) plan() []indexers.Line {
	p.t.Helper()
	return p.planOf(indexers.AutoSearch)
}

func (p *planned) planOf(k indexers.SearchKind) []indexers.Line {
	p.t.Helper()
	var lines []indexers.Line
	err := p.lib.Write(p.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		lines, err = p.x.Plan(ctx, tx, indexerstest.Type, k)
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
	answers, err := p.x.Search(p.t.Context(), indexers.Search{Type: indexerstest.Type, Terms: indexers.Terms{Mode: "search", Query: "orchard"}, Asked: asked}, nil)
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
	p.node.Answer(p.ids["Birch"], 0, searching.Answer{Releases: []searching.Release{{Title: "Orchard.1080p"}}})
	p.node.Answer(p.ids["Gale"], 0, searching.Answer{Outcome: &searching.Stopped{Cause: searching.AccountRefused, Text: "Account suspended"}})

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
	for _, j := range p.node.Jobs() {
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
	p.node.Answer(p.ids["Birch"], 0, searching.Answer{Outcome: &searching.Stopped{Cause: searching.AccountRefused, Text: "Account suspended"}})
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
	p.node.Answer(p.ids["Alder"], 0, searching.Answer{Got: searching.Torrent{Raw: []byte(raw)}})
	if got, err := p.x.Fetch(t.Context(), tk, link); err != nil || !reflect.DeepEqual(got, indexers.Torrent{Raw: []byte(raw), Node: self}) {
		t.Fatalf("Fetch = %+v, %v; want the .torrent fetched on home", got, err)
	}

	p.node.Answer(p.ids["Alder"], 0, searching.Answer{Outcome: &searching.Failing{Err: errors.New("the Indexer responded with status 500")}})
	_, err = p.x.Fetch(t.Context(), tk, link)
	if wait, ok := errors.AsType[*indexers.WaitError](err); !ok || wait.Outcome.Kind != indexers.IsFailing || wait.Outcome.Until.IsZero() {
		t.Fatalf("a failing fetch = %v, want a wait until its retry", err)
	}
	waits, err := fetchWaits(p, "Alder")
	if want := "Waiting to fetch the .torrent from Alder. SlipStream retries at "; err != nil || !strings.HasPrefix(waits.Sentence, want) {
		t.Fatalf("FetchWaits = %+v, %v; want %q and its time", waits, err, want)
	}

	p.node.Answer(p.ids["Birch"], 0, searching.Answer{Outcome: &searching.Stopped{Cause: searching.AccountRefused, Text: "Account suspended"}})
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
	p.node.Answer(p.ids["Alder"], 0, searching.Answer{Outcome: &searching.Stopped{Cause: searching.ChallengeUnsolved}})
	answers := p.search(p.plan())
	if len(answers) != 1 || answers[0].Outcome != (indexers.Outcome{Kind: indexers.Waiting, Node: box}) {
		t.Fatalf("a search whose failover Node met a challenge = %+v, want Alder waiting for seedbox", answers)
	}
	if line := p.x.Line(indexerstest.Type, &answers[0].Outcome); line != "Waiting for seedbox" {
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
		{&indexers.Outcome{Kind: indexers.IsFailing, Until: at}, "Isn't responding. SlipStream retries at 14:05."},
		{&indexers.Outcome{Kind: indexers.IsStopped, Cause: searching.CredentialsRejected}, "Stopped because the username or password was rejected."},
		{&indexers.Outcome{Kind: indexers.IsStopped, Cause: searching.CredentialsRejected, Text: "Wrong code"}, "Stopped because the sign-in was refused with 'Wrong code'."},
		{&indexers.Outcome{Kind: indexers.IsStopped, Cause: searching.CaptchaNeeded}, "Stopped because signing in again needs a captcha."},
		{&indexers.Outcome{Kind: indexers.IsStopped, Cause: searching.ChallengeUnsolved, Node: self}, "Stopped at a browser challenge because home has no Challenge solver."},
		{&indexers.Outcome{Kind: indexers.IsStopped, Cause: searching.CookieExpired}, "Stopped because the browser cookie is no longer accepted."},
		{&indexers.Outcome{Kind: indexers.IsStopped, Cause: searching.DefinitionUnloadable}, "Stopped because its definition doesn't load."},
		{&indexers.Outcome{Kind: indexers.IsStopped, Cause: searching.AccountRefused, Text: "Account suspended"}, "Stopped because the API key was refused with 'Account suspended'."},
		{&indexers.Outcome{Kind: indexers.IsStopped, Cause: searching.RequestRejected, Text: "Incorrect parameter"}, "Stopped because a search was refused with 'Incorrect parameter'."},
		{&indexers.Outcome{Kind: indexers.NoCategories}, "Skipped because it has no shelf categories"},
		{&indexers.Outcome{Kind: indexers.Late}, "Didn't finish in time"},
		{&indexers.Outcome{Kind: indexers.Removed}, "Deleted since this search"},
		{&indexers.Outcome{Kind: indexers.RSSOnly}, "Skipped because it's an RSS feed and can't search"},
	} {
		if got := p.x.Line(contenttypetest.ID, tc.o); got != tc.want {
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
			}, indexers.Typed{})
			if err != nil {
				t.Fatal(err)
			}
			if got, want := spelled(p.plan()), []string{"Alder asked"}; !slices.Equal(got, want) {
				t.Fatalf("the plan = %q, want only the Torznab feed asked", got)
			}
		})
	}
}

// searchEpisode searches Alder, whose caps are caps, for the episode at
// season and episode of TVDB series 1001.
func searchEpisode(t *testing.T, caps string, season int, episode string) searching.Query {
	t.Helper()
	p := plannedWith(t, "Alder")
	err := p.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return indexerstest.SetCaps(ctx, tx, p.ids["Alder"], caps)
	})
	if err != nil {
		t.Fatal(err)
	}
	terms := indexers.Terms{
		Expect: indexers.Expect{Claim: release.Claim{IDs: map[string]string{"tvdbid": "1001"}}},
		Mode:   "tvsearch", Query: "The Lantern Keepers", Season: &season, Episode: episode,
	}
	if _, err := p.x.Search(t.Context(), indexers.Search{Type: indexerstest.Type, Terms: terms, Asked: p.plan()}, nil); err != nil {
		t.Fatal(err)
	}
	sent := p.node.Searches()
	if len(sent) != 1 {
		t.Fatalf("the Node ran %d searches, want 1", len(sent))
	}
	return sent[0]
}

func TestQuerySendsSeasonZero(t *testing.T) {
	got := searchEpisode(t, indexerstest.TVCaps, 0, "2")
	want := searching.Query{Mode: "tvsearch", IDs: map[string]string{"tvdbid": "1001"}, Season: new(0), Episode: "2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("the special's query = %+v (season %v), want %+v with season 0", got, got.Season, want)
	}
}

func TestQueryFallbackNamesTheEpisode(t *testing.T) {
	for _, tc := range []struct {
		name, caps, want string
		season           int
		episode          string
	}{
		{"an episode", indexerstest.Caps, "The Lantern Keepers S01E03", 1, "3"},
		{"a special", indexerstest.Caps, "The Lantern Keepers S00E02", 0, "2"},
		{"a three-digit episode", indexerstest.Caps, "The Lantern Keepers S02E103", 2, "103"},
		{"tvsearch without tvdbid", `{"modes":{"search":["q"],"tvsearch":["q","season","ep"]},"categories":[2000]}`, "The Lantern Keepers S01E03", 1, "3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := searchEpisode(t, tc.caps, tc.season, tc.episode)
			if want := (searching.Query{Mode: "search", Text: tc.want}); !reflect.DeepEqual(got, want) {
				t.Fatalf("the query = %+v, want the free-text search %+v", got, want)
			}
		})
	}
}

// heard is an each that notes every Answer as its Indexer's name, outcome
// and when it landed after began, and fails once it has noted stopAfter.
func (p *planned) heard(began time.Time, stopAfter int) (each func(indexers.Answer) error, got *[]string) {
	got = new([]string)
	return func(a indexers.Answer) error {
		for name, id := range p.ids {
			if id == a.Indexer {
				*got = append(*got, fmt.Sprintf("%s %s at %s", name, a.Outcome.Kind, time.Since(began)))
			}
		}
		if len(*got) == stopAfter {
			return errHeardEnough
		}
		return nil
	}, got
}

var errHeardEnough = errors.New("heard enough")

func TestEachAnswerLandsAsItsIndexerAnswers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := plannedWith(t, "Alder", "Birch")
		p.node.Answer(p.ids["Alder"], 90*time.Second, searching.Answer{Releases: []searching.Release{{Title: "Orchard.2160p"}}})
		p.node.Answer(p.ids["Birch"], 5*time.Second, searching.Answer{Releases: []searching.Release{{Title: "Orchard.1080p"}}})
		began := time.Now()
		each, heard := p.heard(began, 0)

		answers, err := p.x.Search(t.Context(), indexers.Search{Type: indexerstest.Type, Terms: indexers.Terms{Mode: "search", Query: "orchard"}, Asked: p.plan()}, each)
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"Birch found at 5s", "Alder found at 1m30s"}; !slices.Equal(*heard, want) {
			t.Fatalf("each heard %q, want Birch's answer at 5s before Alder's at 90s", *heard)
		}
		if len(answers) != 2 || answers[0].Indexer != p.ids["Alder"] || answers[1].Indexer != p.ids["Birch"] {
			t.Fatalf("Search returned %+v, want every answer in line order, Alder then Birch", answers)
		}
	})
}

func TestAnErrorFromEachStopsTheJobsStillOut(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := plannedWith(t, "Alder", "Birch")
		p.node.Answer(p.ids["Alder"], 90*time.Second, searching.Answer{Releases: []searching.Release{{Title: "Orchard.2160p"}}})
		p.node.Answer(p.ids["Birch"], 5*time.Second, searching.Answer{Releases: []searching.Release{{Title: "Orchard.1080p"}}})
		began := time.Now()
		each, heard := p.heard(began, 1)

		answers, err := p.x.Search(t.Context(), indexers.Search{Type: indexerstest.Type, Terms: indexers.Terms{Mode: "search", Query: "orchard"}, Asked: p.plan()}, each)
		if !errors.Is(err, errHeardEnough) || answers != nil {
			t.Fatalf("Search = %+v, %v; want each's error", answers, err)
		}
		if took := time.Since(began); took != 5*time.Second {
			t.Fatalf("Search returned after %s, want 5s, when Birch answered", took)
		}
		if want := []string{"Birch found at 5s"}; !slices.Equal(*heard, want) {
			t.Fatalf("each heard %q, want only Birch's answer", *heard)
		}
		if got := p.node.Canceled(); !slices.Equal(got, []string{p.ids["Alder"].String()}) {
			t.Fatalf("the Node saw %q canceled, want Alder's job", got)
		}
	})
}

func TestManualSearchAsksIndexersAutomaticSearchSkipsAndListsAnRSSFeed(t *testing.T) {
	p := plannedWith(t, "Alder", "Cedar", "Dune")
	off := false
	p.edit("Cedar", indexers.Edit{On: &off})
	p.edit("Dune", indexers.Edit{AutoSearch: &off})
	for _, f := range []struct {
		name  string
		media works.TypeID
	}{{"Brook", indexerstest.Type}, {"Creek", contenttypetest.ID}} {
		_, err := p.x.Create(t.Context(), identity.Admin{}, indexers.Draft{
			Kind: indexers.RSS, Name: f.name, Address: "https://" + strings.ToLower(f.name) + ".test/rss", Tracker: f.name, Media: []works.TypeID{f.media},
			On: true, AutoSearch: true, RSS: true,
		}, indexers.Typed{})
		if err != nil {
			t.Fatal(err)
		}
	}

	if got, want := spelled(p.planOf(indexers.AutoSearch)), []string{"Alder asked"}; !slices.Equal(got, want) {
		t.Fatalf("the automatic plan = %q, want %q", got, want)
	}
	manual := p.planOf(indexers.ManualSearch)
	if got, want := spelled(manual), []string{"Alder asked", "Brook rss_only", "Dune asked"}; !slices.Equal(got, want) {
		t.Fatalf("the Manual plan = %q, want %q: Dune though its Automatic search is off, Brook skipped as an RSS feed of the type, and neither Cedar, which is off, nor Creek, a feed of another type", got, want)
	}
	answers := p.search(manual)
	if len(answers) != 2 || answers[1].Indexer != p.ids["Dune"] || answers[1].Outcome.Kind != indexers.NoReleases {
		t.Fatalf("the Manual search's answers = %+v, want Alder's and Dune's", answers)
	}
}

func TestATypedQueryIsSearchedAsTextInPlaceOfTheIDs(t *testing.T) {
	p := plannedWith(t, "Alder")
	err := p.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return indexerstest.SetCaps(ctx, tx, p.ids["Alder"], indexerstest.TVCaps)
	})
	if err != nil {
		t.Fatal(err)
	}
	terms := indexers.Terms{
		Expect: indexers.Expect{Claim: release.Claim{IDs: map[string]string{"tvdbid": "1001"}}},
		Mode:   "tvsearch", Query: "The Lantern Keepers", Season: new(1), Episode: "3",
	}
	for _, text := range []string{"", "Lantern Keepers 1x03 extended"} {
		if _, err := p.x.Search(t.Context(), indexers.Search{Type: indexerstest.Type, Terms: terms, Asked: p.plan(), Text: text}, nil); err != nil {
			t.Fatal(err)
		}
	}
	want := []searching.Query{
		{Mode: "tvsearch", IDs: map[string]string{"tvdbid": "1001"}, Season: new(1), Episode: "3"},
		{Mode: "search", Text: "Lantern Keepers 1x03 extended"},
	}
	if got := p.node.Searches(); !reflect.DeepEqual(got, want) {
		t.Fatalf("the Node was sent %+v, want the ids, then the typed text alone", got)
	}
}

func (p *planned) standings() indexers.Standings {
	p.t.Helper()
	var s indexers.Standings
	err := p.lib.Read(p.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		s, err = p.x.Standings(ctx, r)
		return err
	})
	if err != nil {
		p.t.Fatal(err)
	}
	return s
}

// agreed is what Take answers name's r with, once Standings, read just
// before, answered the same.
func (p *planned) agreed(name string, r searching.Release) error {
	p.t.Helper()
	read, readErr := p.standings().Take(indexers.Release{Indexer: p.ids[name], Release: r})
	took, err := p.take(name, r)
	if !reflect.DeepEqual(readErr, err) || (err == nil && read.Indexer != took.Indexer) {
		p.t.Fatalf("for %s's %s, Standings answered %+v, %v and Take %+v, %v; want the same", name, r.Title, read, readErr, took, err)
	}
	return err
}

func sentenceOf(err error) string {
	if e, ok := errors.AsType[*indexers.NotTakenError](err); ok {
		return e.Sentence
	}
	return ""
}

func TestStandingsAnswerAsTakeDoesAndSpendNothing(t *testing.T) {
	p := plannedWith(t, "Alder", "Birch", "Cedar", "Dogwood")
	p.edit("Alder", indexers.Edit{Grabs: &indexers.Limit{Count: 1, Per: indexers.Hour}})
	p.edit("Dogwood", indexers.Edit{AutoSearch: new(false)})
	torrent := searching.Release{Title: "Orchard.1080p", Link: searching.LinkOf("https://site.test/a.torrent")}
	magnet := searching.Release{Title: "Orchard.720p", Magnet: searching.LinkOf("magnet:?xt=urn:btih:" + strings.Repeat("a", 40))}

	for range 3 {
		if _, err := p.standings().Take(indexers.Release{Indexer: p.ids["Alder"], Release: torrent}); err != nil {
			t.Fatalf("Alder's .torrent with its one grab left = %v, want it taken", err)
		}
	}
	if err := p.agreed("Alder", torrent); err != nil {
		t.Fatalf("taking Alder's .torrent after three reads of its standing = %v, want its one grab still left", err)
	}
	err := p.agreed("Alder", torrent)
	if want := "Alder is at its grab limit until "; !notTaken(err, indexers.LimitReached) || !strings.HasPrefix(sentenceOf(err), want) {
		t.Fatalf("Alder's .torrent at its grab limit = %v %q, want limit_reached saying %q and when", err, sentenceOf(err), want)
	}
	if err := p.agreed("Alder", magnet); err != nil {
		t.Fatalf("Alder's magnet link at its grab limit = %v, want it taken, since it reserves no grab", err)
	}

	p.node.Answer(p.ids["Birch"], 0, searching.Answer{Outcome: &searching.Stopped{Cause: searching.AccountRefused, Text: "Account suspended"}})
	p.search(p.plan())
	for _, r := range []searching.Release{torrent, magnet} {
		err := p.agreed("Birch", r)
		if want := "Stopped because Birch refused the API key with 'Account suspended'."; !notTaken(err, indexers.IsStopped) || sentenceOf(err) != want {
			t.Fatalf("stopped Birch's %s = %v %q, want stopped saying %q", r.Title, err, sentenceOf(err), want)
		}
	}

	if err := p.x.Delete(t.Context(), identity.Admin{}, p.ids["Cedar"]); err != nil {
		t.Fatal(err)
	}
	for _, r := range []searching.Release{torrent, magnet} {
		err := p.agreed("Cedar", r)
		if want := "The Indexer that listed it was deleted."; !notTaken(err, indexers.Removed) || sentenceOf(err) != want {
			t.Fatalf("deleted Cedar's %s = %v %q, want removed saying %q", r.Title, err, sentenceOf(err), want)
		}
	}

	if s := p.standings()[p.ids["Dogwood"]]; s.Automatic || p.agreed("Dogwood", torrent) != nil {
		t.Fatalf("Dogwood with Automatic search off stands %+v, want it taking with Automatic false", s)
	}
}
