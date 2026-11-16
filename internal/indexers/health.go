package indexers

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// HealthAbout follows tasks.ChangedEvent too, since the category also
// shows the Indexer tasks' warnings, nodes.ChangedEvent for a home that
// leaves or returns, and works.ChangedEvent for a Content Type's first
// Work.
var HealthAbout = health.About{
	ID:     health.IndexersID,
	Label:  "Indexers",
	Phrase: "Indexers",
	Events: []live.Event{
		ChangedEvent, StatusChangedEvent, CatalogChangedEvent, SettingsChangedEvent, RSSChangedEvent,
		tasks.ChangedEvent, nodes.ChangedEvent, works.ChangedEvent,
	},
}

// HealthSource is the indexers category's items, each linking to the
// Indexer page: a Fail for each stopped Indexer that is on, a Warn for
// each one failing longer than LongBackoff, a Warn for each login-session
// Indexer waiting for a home that lapsed, a Fail for each Content Type
// with Works that no usable Indexer serves, and a Warn for each catalog
// Indexer whose definition left the catalog or whose new version doesn't
// load, and a Warn for each Indexer with RSS on whose last RSS fetches
// didn't reach its feed's position. An Indexer whose calls wait for its home is unsure. Next is the
// soonest long-backoff edge or retry.
func (x *Indexers) HealthSource() health.Source {
	return health.Source{Answer: func(ctx context.Context) (health.Answer, error) {
		var a health.Answer
		err := x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
			var err error
			a, err = x.healthAnswer(ctx, r, time.Now())
			return err
		})
		return a, err
	}}
}

func (x *Indexers) healthAnswer(ctx context.Context, r storage.Reader[file], now time.Time) (health.Answer, error) {
	q := db.New(r)
	s, err := q.GetSettings(ctx)
	if err != nil {
		return health.Answer{}, err
	}
	long := LongBackoff.Value(s.LongBackoffSeconds)
	all, err := x.all(ctx, r)
	if err != nil {
		return health.Answer{}, err
	}
	statuses, err := statusesIn(ctx, r)
	if err != nil {
		return health.Answer{}, err
	}
	held, err := works.Held(ctx, r)
	if err != nil {
		return health.Answer{}, err
	}
	feeds, err := feedFacts(ctx, r)
	if err != nil {
		return health.Answer{}, err
	}
	plan, err := x.rssPlan(ctx, r)
	if err != nil {
		return health.Answer{}, err
	}
	var a health.Answer
	for _, idx := range all {
		if idx.On {
			x.judge(&a, idx, statuses[idx.ID], long, now)
		}
		if f, ok := feeds[idx.ID]; ok && idx.On && idx.RSS && f.Gap != GapNone {
			a.Items = append(a.Items, x.gapItem(idx, f))
		}
	}
	a.Items = append(a.Items, x.unserved(held, all, statuses, plan.on)...)
	pins, err := x.pinItems(ctx, q)
	a.Items = append(a.Items, pins...)
	return a, err
}

// judge adds what Health says of idx standing at st to a: its stop, or
// its failing past long. While its calls wait for its home, those items
// are unsure, and once that home lapsed a login-session Indexer whose
// fix is Allow failover warns.
func (x *Indexers) judge(a *health.Answer, idx Indexer, st Status, long time.Duration, now time.Time) {
	var items []health.Item
	switch {
	case st.State == Stopped:
		items = append(items, x.stopItem(idx, st))
	case st.State == Failing && now.Before(st.Since.Add(long)):
		a.Next = soonest(a.Next, st.Since.Add(long))
	case st.State == Failing:
		items = append(items, x.failingItem(idx, st, now))
		if now.Before(st.Until) {
			a.Next = soonest(a.Next, st.Until)
		}
	}
	_, wait := route(x.cfg.Roster, idx, st)
	if wait == nil {
		a.Items = append(a.Items, items...)
		return
	}
	waiting, fix := x.waitWords(View{Indexer: idx, Status: st, SignsIn: holdsSession(idx)}, *wait)
	if fix != "" {
		a.Items = append(a.Items, waitingItem(idx, *wait, waiting+" "+fix))
	}
	for _, it := range items {
		a.Unsure = append(a.Unsure, health.Unsure{Key: it.Key, Says: health.Says{Subject: idx.Name, Text: waiting}})
	}
}

func soonest(a, b time.Time) time.Time {
	if a.IsZero() || b.Before(a) {
		return b
	}
	return a
}

// stopItem links a challenge met with no Challenge solver to the
// Searching section of the Node it was met on, where the solver is set,
// and a sign-in that needs a two-factor code to the Indexer's code step.
func (x *Indexers) stopItem(idx Indexer, st Status) health.Item {
	action := indexerLink(idx)
	switch {
	case st.Cause == searching.ChallengeUnsolved && !st.SolverTried:
		action = health.LinkAction("/nodes/"+st.Node.String()+"/searching", "Open Searching on "+x.cfg.Roster.Name(st.Node))
	case st.Cause == searching.CodeNeeded:
		action = health.LinkAction("/settings/indexers/"+idx.ID.String()+"#code", "Type code")
	}
	return health.Item{Key: health.KeyOf("stopped", idx.ID.String()), Level: health.Fail, Says: health.Says{
		Subject: idx.Name,
		Text:    x.stopItemWords(idx, st),
		Detail:  []health.Fact{{Label: "Stopped since", At: st.Since}},
		Action:  action,
	}}
}

func (x *Indexers) failingItem(idx Indexer, st Status, now time.Time) health.Item {
	detail := []health.Fact{{Label: "Failing since", At: st.Since}}
	if st.Last != nil {
		detail = append(detail, health.Fact{Label: "Last error", Text: st.Last.Text})
	}
	return health.Item{Key: health.KeyOf("failing", idx.ID.String()), Level: health.Warn, Says: health.Says{
		Subject: idx.Name,
		Text:    x.failingItemWords(idx.Name, st, now),
		Detail:  detail,
		Action:  indexerLink(idx),
	}}
}

// waitingItem is the Warn of idx waiting at wait for a home that lapsed,
// saying text and linking to its Allow failover switch.
func waitingItem(idx Indexer, wait Outcome, text string) health.Item {
	return health.Item{Key: health.KeyOf("waiting", idx.ID.String()), Level: health.Warn, Says: health.Says{
		Subject: idx.Name,
		Text:    text,
		Detail:  []health.Fact{{Label: "Offline since", At: wait.Since}},
		Action:  health.LinkAction("/settings/indexers/"+idx.ID.String()+"#allow-failover", "Open "+idx.Name),
	}}
}

func indexerLink(idx Indexer) health.Action {
	return health.LinkAction("/settings/indexers/"+idx.ID.String(), "Open "+idx.Name)
}

// gapItem is the Warn of idx's RSS fetches missing uploads since f,
// linking to Wanted for the Content Types its feed may match.
func (x *Indexers) gapItem(idx Indexer, f FeedFact) health.Item {
	types, _ := x.feedTypes(idx)
	return health.Item{Key: health.KeyOf("rss-gap", idx.ID.String()), Level: health.Warn, Says: health.Says{
		Subject: idx.Name,
		Text:    x.gapWords(idx.Name, f.Gap),
		Detail:  []health.Fact{{Label: "Last RSS sync", At: f.Fetched}},
		Action:  health.LinkAction(wantedHref(types), "Open Wanted"),
	}}
}

// wantedHref is Wanted filtered to missing and failed files: the one
// Content Type's page when types holds one, every type's otherwise.
func wantedHref(types []works.TypeID) string {
	path := "/library/wanted"
	if len(types) == 1 {
		path += "/" + url.PathEscape(string(types[0]))
	}
	return path + "?" + url.Values{"reason": {"missing,failed"}}.Encode()
}

// unserved is a Fail for each Content Type of held, those holding a Work,
// that has categories and that no usable Indexer serves: one that is on,
// isn't an RSS feed, isn't stopped and serves one of its categories. While
// RSS sync is on it says how many RSS feeds that are on still read the
// type's new uploads.
func (x *Indexers) unserved(held []works.TypeID, all []Indexer, statuses map[ID]Status, rssOn bool) []health.Item {
	var items []health.Item
	for _, typ := range slices.Sorted(slices.Values(held)) {
		cats := x.categories(typ)
		if len(cats) == 0 {
			continue
		}
		if slices.ContainsFunc(all, func(idx Indexer) bool {
			return idx.On && idx.Kind != RSS && statuses[idx.ID].State != Stopped && len(served(idx, cats)) > 0
		}) {
			continue
		}
		d, _ := x.cfg.Works.Describe(typ)
		items = append(items, health.Item{Key: health.KeyOf("unserved", string(typ)), Level: health.Fail, Says: health.Says{
			Subject: d.Work.Many,
			Text:    unservedWords(d.Work.Many, feedsOf(all, typ, rssOn)),
			Action:  health.LinkAction("/settings/indexers", "Open Indexers"),
		}})
	}
	return items
}

// feedsOf is how many RSS feeds that are on with RSS on carry typ, none
// while RSS sync is off.
func feedsOf(all []Indexer, typ works.TypeID, rssOn bool) int {
	if !rssOn {
		return 0
	}
	n := 0
	for _, idx := range all {
		if idx.On && idx.RSS && idx.Kind == RSS && slices.Contains(idx.Media, typ) {
			n++
		}
	}
	return n
}

func (x *Indexers) pinItems(ctx context.Context, q *db.Queries) ([]health.Item, error) {
	pins, err := q.ListPins(ctx)
	if err != nil {
		return nil, err
	}
	var items []health.Item
	for _, p := range pins {
		if Origin(p.Origin) != FromCatalog {
			continue
		}
		entry, err := q.GetCatalogEntry(ctx, db.DefinitionID(*p.DefinitionID))
		var text string
		switch {
		case errors.Is(err, storage.ErrNoRows):
			text = p.Name + " is no longer in the Indexer catalog."
		case err != nil:
			return nil, err
		case entry.Sha == *p.DefinitionSha:
			continue
		default:
			newer, err := q.GetDefinition(ctx, db.GetDefinitionParams{ID: entry.ID, Sha: entry.Sha})
			if err != nil {
				return nil, err
			}
			if newer.Refusal == nil {
				continue
			}
			text = "The catalog's new version of " + p.Name + " doesn't load."
		}
		items = append(items, health.Item{Key: health.KeyOf("definition", p.ID.String()), Level: health.Warn, Says: health.Says{
			Subject: p.Name,
			Text:    text + " The Indexer keeps running the version from " + x.day(p.FetchedAt.Time()) + ".",
			Action:  health.LinkAction("/settings/indexers/"+p.ID.String(), "Open "+p.Name),
		}})
	}
	return items, nil
}
