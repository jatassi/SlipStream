package subtitlestest

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/calls"
	subtitlesv1 "github.com/jatassi/slipstream/internal/pb/slipstream/subtitles/v1"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// Specs is the World's Subtitle Providers: SubDL, keyed, for features and
// episodes, then Gestdown, keyless, for episodes.
var Specs = []subtitling.Spec{
	{ID: "subdl", Label: "SubDL", Kinds: []subtitling.Kind{subtitling.Feature, subtitling.Episode}, Keyed: true, Position: 1},
	{ID: "gestdown", Label: "Gestdown", Kinds: []subtitling.Kind{subtitling.Episode}, Position: 2},
}

// Catalog is the World holder's Subtitle Providers. Each one lists the
// offers a test put, answers OK unless a test scripted its next answers,
// and serves an offer's bytes, which the holder unpacks and times as it
// does a real download.
type Catalog struct {
	*subtitling.Holder

	mu     sync.Mutex
	listed map[subtitling.ProviderID][]Listing
	next   map[subtitling.ProviderID][]calls.Outcome
	twice  map[subtitling.ProviderID]bool
	asked  []string
	gate   chan struct{}
}

// Listing is one offer and the bytes its download serves.
type Listing struct {
	Offer subtitling.Offer
	Name  string
	Bytes []byte
}

func newCatalog(ht subtitling.Timings) *Catalog {
	c := &Catalog{listed: map[subtitling.ProviderID][]Listing{}, next: map[subtitling.ProviderID][]calls.Outcome{}, twice: map[subtitling.ProviderID]bool{}}
	ps := make([]subtitling.Provider, len(Specs))
	for i, s := range Specs {
		ps[i] = listed{c: c, spec: s}
	}
	c.Holder = subtitling.Serve(ps, ht)
	return c
}

// List puts offers on their providers.
func (c *Catalog) List(ls ...Listing) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, l := range ls {
		c.listed[l.Offer.Provider] = append(c.listed[l.Offer.Provider], l)
	}
}

// Answer scripts p's next calls' answers, one each, in order.
func (c *Catalog) Answer(p subtitling.ProviderID, os ...calls.Outcome) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, o := range os {
		o.Provider = p
		c.next[p] = append(c.next[p], o)
	}
}

// Lookups makes each search of p two calls, a lookup of the title then
// its list, as SubSource's does. Each call takes its own answer.
func (c *Catalog) Lookups(p subtitling.ProviderID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.twice[p] = true
}

// Hold makes every Fetch wait for Release or its call's end.
func (c *Catalog) Hold() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gate = make(chan struct{})
}

// Release lets the held Fetch calls go on, and holds no more.
func (c *Catalog) Release() {
	c.mu.Lock()
	defer c.mu.Unlock()
	close(c.gate)
	c.gate = nil
}

// Asked is each call so far, as "search provider" or "fetch provider key".
func (c *Catalog) Asked() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.asked)
}

// answer is p's scripted next answer, else OK.
func (c *Catalog) answer(p subtitling.ProviderID, call string) (calls.Outcome, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.asked = append(c.asked, call)
	o := calls.Outcome{Provider: p, Kind: calls.OK, At: time.Now()}
	if q := c.next[p]; len(q) > 0 {
		o, c.next[p] = q[0], q[1:]
		if o.At.IsZero() {
			o.At = time.Now()
		}
	}
	return o, o.Kind == calls.OK || o.Kind == calls.NotFound
}

func (c *Catalog) Search(ctx context.Context, req *subtitlesv1.SearchRequest) (*subtitlesv1.SearchResponse, error) {
	res := &subtitlesv1.SearchResponse{}
	for _, id := range req.GetProviders() {
		if !c.searched(subtitling.ProviderID(id), res) {
			continue
		}
		got, err := c.Holder.Search(ctx, &subtitlesv1.SearchRequest{Subject: req.GetSubject(), Query: req.GetQuery(), Keys: req.GetKeys(), Providers: []string{id}})
		if err != nil {
			return nil, err
		}
		res.Offers = append(res.Offers, got.GetOffers()...)
	}
	return res, nil
}

// searched makes a search's calls to p, each with its answer, and is
// whether every call answered.
func (c *Catalog) searched(p subtitling.ProviderID, res *subtitlesv1.SearchResponse) bool {
	c.mu.Lock()
	n := 1
	if c.twice[p] {
		n = 2
	}
	c.mu.Unlock()
	for range n {
		o, ok := c.answer(p, "search "+string(p))
		res.Outcomes = append(res.Outcomes, wire(o))
		if !ok {
			return false
		}
	}
	return true
}

func (c *Catalog) Fetch(ctx context.Context, req *subtitlesv1.FetchRequest) (*subtitlesv1.FetchResponse, error) {
	id := req.GetOffer().GetProvider()
	o, ok := c.answer(subtitling.ProviderID(id), "fetch "+id+" "+req.GetOffer().GetKey())
	c.mu.Lock()
	gate := c.gate
	c.mu.Unlock()
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return nil, fmt.Errorf("the fetch: %w", context.Cause(ctx))
		}
	}
	if !ok {
		return &subtitlesv1.FetchResponse{Key: req.GetOffer().GetKey(), Outcomes: []*subtitlesv1.Outcome{wire(o)}, Verdict: subtitlesv1.Verdict_VERDICT_UNFETCHED}, nil
	}
	res, err := c.Holder.Fetch(ctx, req)
	if err != nil {
		return nil, err
	}
	res.Outcomes = append(res.Outcomes, wire(o))
	return res, nil
}

var outcomeKinds = map[calls.OutcomeKind]subtitlesv1.OutcomeKind{
	calls.OK: subtitlesv1.OutcomeKind_OUTCOME_KIND_OK, calls.NotFound: subtitlesv1.OutcomeKind_OUTCOME_KIND_NOT_FOUND,
	calls.Refused: subtitlesv1.OutcomeKind_OUTCOME_KIND_REFUSED, calls.Limited: subtitlesv1.OutcomeKind_OUTCOME_KIND_LIMITED,
	calls.Capped: subtitlesv1.OutcomeKind_OUTCOME_KIND_CAPPED, calls.Unreachable: subtitlesv1.OutcomeKind_OUTCOME_KIND_UNREACHABLE,
}

func wire(o calls.Outcome) *subtitlesv1.Outcome {
	w := &subtitlesv1.Outcome{Provider: string(o.Provider), Kind: outcomeKinds[o.Kind], Words: o.Words, AtUnixMicro: o.At.UnixMicro()}
	if !o.Until.IsZero() {
		w.UntilUnixMicro = o.Until.UnixMicro()
	}
	if a := o.Allowance; a != nil && a.Downloads != nil {
		d := a.Downloads
		w.Allowance = &subtitlesv1.Allowance{Downloads: &subtitlesv1.Window{
			Used: int64(d.Used), Limit: int64(d.Limit), Remaining: int64(d.Remaining), ResetsUnixMicro: d.Resets.UnixMicro(), Exact: d.Exact,
		}}
	}
	return w
}

// listed is one provider of a Catalog.
type listed struct {
	c    *Catalog
	spec subtitling.Spec
}

func (l listed) Spec() subtitling.Spec { return l.spec }

func (l listed) CheckKey(context.Context, calls.Credentials) (subtitling.Account, error) {
	return subtitling.Account{}, nil
}

func (l listed) Search(context.Context, calls.Credentials, subtitling.Query) ([]subtitling.Offer, error) {
	l.c.mu.Lock()
	defer l.c.mu.Unlock()
	var out []subtitling.Offer
	for _, x := range l.c.listed[l.spec.ID] {
		out = append(out, x.Offer)
	}
	return out, nil
}

func (l listed) Download(_ context.Context, _ calls.Credentials, o subtitling.Offer) (subtitling.Download, error) {
	l.c.mu.Lock()
	defer l.c.mu.Unlock()
	for _, x := range l.c.listed[l.spec.ID] {
		if x.Offer.Key == o.Key {
			return subtitling.Download{Name: x.Name, Bytes: x.Bytes}, nil
		}
	}
	return subtitling.Download{}, fmt.Errorf("%s lists no offer %q", l.spec.Label, o.Key)
}

// Cadences is the tasks' cadences: each its default until a test or the
// settings set it.
type Cadences struct {
	mu    sync.Mutex
	every map[tasks.TaskID]int64
}

func (c *Cadences) EveryIn(_ context.Context, _ storage.Reader[storage.Library], id tasks.TaskID) (*int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.every[id]
	if !ok {
		return nil, nil
	}
	return &v, nil
}

func (c *Cadences) SetCadence(_ context.Context, id tasks.TaskID, v setting.Stored[int64]) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.every == nil {
		c.every = map[tasks.TaskID]int64{}
	}
	if o := v.Override(); o != nil {
		c.every[id] = *o
	} else {
		delete(c.every, id)
	}
	return nil
}

// episodes is the World's facet once Episodes is on: each file holds the
// episodes its Items air as, in season 1.
type episodes struct{ w *World }

func (e episodes) Subtitled() []subtitling.Kind {
	if e.w.shows.Load() {
		return []subtitling.Kind{subtitling.Episode}
	}
	return e.w.Fixture.Subtitled()
}

func (e episodes) SubtitleSubject(ctx context.Context, r storage.Reader[storage.Library], work works.WorkID, items []works.ItemID) (subtitling.Subject, error) {
	if !e.w.shows.Load() {
		return e.w.Fixture.SubtitleSubject(ctx, r, work, items)
	}
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	s := subtitling.Subject{Kind: subtitling.Episode, Title: "Show", Season: 1}
	for _, it := range items {
		n, ok := e.w.aired[it]
		if !ok {
			return subtitling.Subject{}, subtitles.ErrNoAiredNumber
		}
		s.Episodes = append(s.Episodes, n)
	}
	return s, nil
}
