package indexersapi

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
)

type Registry interface {
	// Indexers is the Indexers, false unless the Node is a Controller.
	Indexers() (*indexers.Indexers, bool)
}

func open(r Registry) *indexers.Indexers {
	x, ok := r.Indexers()
	if !ok {
		panic("indexersapi: an operation ran on a Node that is no Controller")
	}
	return x
}

var (
	indexerNameTaken   = api.NewCode(http.StatusConflict, "indexer_name_taken", "Another Indexer has that name")
	noSearchingNode    = api.NewCode(http.StatusConflict, "no_searching_node", "No Node holds Searching. Give one the Searching Role first.")
	nodeNotSearching   = api.NewCode(http.StatusConflict, "node_not_searching", "That Node doesn't stand in Searching")
	indexerTestFailed  = api.NewCode(http.StatusUnprocessableEntity, "indexer_test_failed", "The Indexer's Test failed")
	indexerUnavailable = api.NewCode(http.StatusBadGateway, "indexer_unavailable", "The Test couldn't run")
	indexerNotFound    = api.NewCode(http.StatusNotFound, "indexer_not_found", "No Indexer has that id")
)

func answer(x *indexers.Indexers, err error) error {
	var failed *indexers.TestFailedError
	var wait *indexers.WaitError
	var unavailable *indexers.UnavailableError
	switch {
	case errors.As(err, &failed):
		return indexerTestFailed.Because(failed.Sentence)
	case errors.As(err, &wait), errors.As(err, &unavailable), errors.Is(err, indexers.ErrLate):
		return indexerUnavailable.Because(x.Unreached(err))
	case errors.Is(err, indexers.ErrNameTaken):
		return indexerNameTaken
	case errors.Is(err, indexers.ErrNoSearchingNode):
		return noSearchingNode
	case errors.Is(err, indexers.ErrNotSearching):
		return nodeNotSearching
	case errors.Is(err, indexers.ErrNoIndexer):
		return indexerNotFound
	}
	return err
}

type searchMode struct {
	Mode   string   `json:"mode" doc:"The Torznab t= value"`
	Params []string `json:"params" nullable:"false" doc:"In the Indexer's order"`
}

type limit struct {
	Count int    `json:"count" minimum:"0" maximum:"100000" doc:"0 is no limit"`
	Per   string `json:"per" enum:"hour,day" doc:"The sliding window it counts in"`
}

type limitUse struct {
	Count  int        `json:"count" minimum:"0" doc:"0 is no limit"`
	Per    string     `json:"per" enum:"hour,day"`
	Used   int        `json:"used" minimum:"0" doc:"The calls counted in the window now"`
	FreeAt *time.Time `json:"freeAt" doc:"When the oldest counted call leaves the window; null while under the limit"`
}

type failure struct {
	At   time.Time `json:"at"`
	Text string    `json:"text"`
}

type status struct {
	State       string                `json:"state" enum:"ok,limited,failing,stopped,home_only" doc:"home_only: a stop met on a failover Node, which waits for home"`
	Cause       *string               `json:"cause" enum:"credentials_rejected,captcha_needed,challenge_unsolved,cookie_expired,definition_unloadable,account_refused,request_rejected" doc:"Null unless stopped or home_only"`
	Text        *string               `json:"text" doc:"The site's or feed's own words"`
	Until       *time.Time            `json:"until" doc:"When a limited or failing Indexer is asked again"`
	Since       *time.Time            `json:"since" doc:"When it began failing, stopped or began waiting for home"`
	NodeID      *string               `json:"nodeId" format:"uuid" doc:"The Node the stop was met on"`
	SolverTried bool                  `json:"solverTried" doc:"A challenge_unsolved that the Node's Challenge solver couldn't clear"`
	Rung        int                   `json:"rung" minimum:"0" doc:"The backoff ladder's position, stepping down one on each success"`
	LastFailure api.Nullable[failure] `json:"lastFailure"`
	SuccessAt   *time.Time            `json:"successAt" doc:"The last answer the site responded to"`
}

type outcome struct {
	Kind     string     `json:"kind" enum:"found,no_releases,limited,failing,stopped,waiting,late,limit_reached,no_categories,removed"`
	Releases int        `json:"releases" minimum:"0"`
	Until    *time.Time `json:"until" doc:"Set for limited, failing and limit_reached"`
	Since    *time.Time `json:"since" doc:"Set for waiting: since when the home Node is away; null when it holds no Role to search with"`
	NodeID   *string    `json:"nodeId" format:"uuid" doc:"Set for waiting: the Node waited for; for stopped: the Node the stop was met on"`
	Lapsed   bool       `json:"lapsed" doc:"Waiting past the offline grace"`
	Cause    *string    `json:"cause" doc:"Set for stopped"`
	Text     *string    `json:"text" doc:"Set for stopped: the site's own words"`
}

type session struct {
	NodeID   string    `json:"nodeId" format:"uuid"`
	Pasted   bool      `json:"pasted" doc:"Seeded from a browser cookie an Admin pasted"`
	EarnedAt time.Time `json:"earnedAt"`
}

type indexer struct {
	ID              string                `json:"id" format:"uuid"`
	Name            string                `json:"name"`
	Kind            string                `json:"kind" enum:"catalog,torznab,rss"`
	HomeNodeID      string                `json:"homeNodeId" format:"uuid" doc:"The Node it searches from"`
	On              bool                  `json:"on"`
	AutomaticSearch bool                  `json:"automaticSearch"`
	RSS             bool                  `json:"rss"`
	AllowFailover   bool                  `json:"allowFailover" doc:"Read only for an Indexer that signs in"`
	Priority        int                   `json:"priority" minimum:"1" maximum:"50" doc:"Lower wins a tie"`
	Categories      []int                 `json:"categories" doc:"Newznab category ids; null for every category it advertises"`
	Searches        limitUse              `json:"searches"`
	Grabs           limitUse              `json:"grabs"`
	Status          status                `json:"status"`
	Sentence        string                `json:"sentence" doc:"Its status in words"`
	Fix             *string               `json:"fix" doc:"What the Admin can do about its status, in words; null when nothing"`
	SignsIn         bool                  `json:"signsIn" doc:"It holds a login session, so it waits for its home rather than failing over unless allowFailover"`
	TrackerIDs      []string              `json:"trackerIds" nullable:"false" doc:"The Trackers it reaches"`
	Waits           api.Nullable[outcome] `json:"waits" doc:"Why its searches wait now, null when they don't"`
	Session         api.Nullable[session] `json:"session" doc:"Its newest session on any Node, null for none"`
	FeedHost        *string               `json:"feedHost" doc:"The host of a feed's address, which is never returned"`
	SearchModes     []searchMode          `json:"searchModes" nullable:"false" doc:"By mode"`
	Revision        int64                 `json:"revision" minimum:"1"`
	CreatedAt       time.Time             `json:"createdAt"`
}

func when(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func text[T ~string](s T) *string {
	if s == "" {
		return nil
	}
	v := string(s)
	return &v
}

func node(n role.NodeID) *string {
	if n.IsZero() {
		return nil
	}
	s := n.String()
	return &s
}

func indexerOf(v indexers.View) indexer {
	out := indexer{
		ID: v.ID.String(), Name: v.Name, Kind: string(v.Kind), HomeNodeID: v.Home.String(), On: v.On, AutomaticSearch: v.AutoSearch,
		RSS: v.RSS, AllowFailover: v.AllowFailover, Priority: v.Priority,
		Searches:    limitUseOf(v.Searches, v.Usage.Searches, v.Usage.SearchesFree),
		Grabs:       limitUseOf(v.Grabs, v.Usage.Grabs, v.Usage.GrabsFree),
		Status:      statusOf(v.Status),
		Sentence:    v.Sentence,
		Fix:         text(v.Fix),
		SignsIn:     v.SignsIn,
		TrackerIDs:  []string{},
		FeedHost:    text(v.FeedHost),
		SearchModes: []searchMode{},
		Revision:    v.Revision,
		CreatedAt:   v.Created,
	}
	if v.Categories != nil {
		out.Categories = []int{}
		for _, c := range v.Categories {
			out.Categories = append(out.Categories, int(c))
		}
	}
	for _, t := range v.Trackers {
		out.TrackerIDs = append(out.TrackerIDs, t.String())
	}
	if v.Waits != nil {
		out.Waits = api.Some(outcomeOf(*v.Waits))
	}
	if v.Session != nil {
		out.Session = api.Some(session{NodeID: v.Session.Node.String(), Pasted: v.Session.Pasted, EarnedAt: v.Session.Earned})
	}
	for _, m := range slices.Sorted(maps.Keys(v.Caps.Modes)) {
		out.SearchModes = append(out.SearchModes, searchMode{Mode: m, Params: v.Caps.Modes[m]})
	}
	return out
}

func limitUseOf(l indexers.Limit, used int, free time.Time) limitUse {
	if l.Per == "" {
		l.Per = indexers.Hour
	}
	return limitUse{Count: l.Count, Per: string(l.Per), Used: used, FreeAt: when(free)}
}

func statusOf(s indexers.Status) status {
	out := status{
		State: string(s.State), Cause: text(s.Cause), Text: text(s.Text), Until: when(s.Until), Since: when(s.Since), NodeID: node(s.Node),
		SolverTried: s.SolverTried, Rung: s.Rung, SuccessAt: when(s.Success),
	}
	if s.Last != nil {
		out.LastFailure = api.Some(failure{At: s.Last.At, Text: s.Last.Text})
	}
	return out
}

func outcomeOf(o indexers.Outcome) outcome {
	return outcome{
		Kind: string(o.Kind), Releases: o.Releases, Until: when(o.Until), Since: when(o.Since), NodeID: node(o.Node), Lapsed: o.Lapsed,
		Cause: text(o.Cause), Text: text(o.Text),
	}
}

type newIndexer struct {
	Kind            string `json:"kind" enum:"torznab" doc:"The kinds SlipStream can add so far"`
	Name            string `json:"name" minLength:"1" maxLength:"100"`
	Address         string `json:"address" maxLength:"2048" pattern:"^https?://[^/?#]+(/[^?#]*)?$" doc:"The Torznab feed's base; SlipStream adds /api. Write-only"`
	APIKey          string `json:"apiKey" maxLength:"256" doc:"Write-only"`
	HomeNodeID      string `json:"homeNodeId,omitempty" format:"uuid" doc:"A Node that holds Searching; absent for the default home"`
	On              bool   `json:"on,omitempty" default:"true"`
	AutomaticSearch bool   `json:"automaticSearch,omitempty" default:"true"`
	RSS             bool   `json:"rss,omitempty" default:"true"`
	AllowFailover   bool   `json:"allowFailover,omitempty"`
	Priority        int    `json:"priority,omitempty" minimum:"1" maximum:"50" default:"25"`
	Categories      []int  `json:"categories,omitempty" doc:"Newznab category ids; absent for every category it advertises"`
	Searches        *limit `json:"searches,omitempty"`
	Grabs           *limit `json:"grabs,omitempty"`
}

func (n newIndexer) draft() (indexers.Draft, error) {
	d := indexers.Draft{
		Name: n.Name, Address: n.Address, APIKey: n.APIKey, On: n.On, AutoSearch: n.AutomaticSearch, RSS: n.RSS,
		AllowFailover: n.AllowFailover, Priority: n.Priority, Categories: indexers.CategoriesOf(n.Categories),
	}
	if n.Searches != nil {
		d.Searches = limitIn(*n.Searches)
	}
	if n.Grabs != nil {
		d.Grabs = limitIn(*n.Grabs)
	}
	if n.HomeNodeID != "" {
		id, err := storage.ParseUUID(n.HomeNodeID)
		if err != nil {
			return indexers.Draft{}, nodeNotSearching
		}
		d.Home = role.NodeID{UUID: id}
	}
	return d, nil
}

func limitIn(l limit) indexers.Limit {
	return indexers.Limit{Count: l.Count, Per: indexers.Window(l.Per)}
}

type indexerEdit struct {
	Name            *string          `json:"name,omitempty" minLength:"1" maxLength:"100"`
	HomeNodeID      *string          `json:"homeNodeId,omitempty" format:"uuid" doc:"A Node that holds Searching"`
	Address         *string          `json:"address,omitempty" maxLength:"2048" pattern:"^https?://[^/?#]+(/[^?#]*)?$" doc:"Write-only"`
	APIKey          *string          `json:"apiKey,omitempty" maxLength:"256" doc:"Write-only"`
	On              api.Patch[bool]  `json:"on,omitempty" doc:"Null resets it to on"`
	AutomaticSearch api.Patch[bool]  `json:"automaticSearch,omitempty" doc:"Null resets it to on"`
	RSS             api.Patch[bool]  `json:"rss,omitempty" doc:"Null resets it to on"`
	AllowFailover   api.Patch[bool]  `json:"allowFailover,omitempty" doc:"Null resets it to off"`
	Priority        api.Patch[int]   `json:"priority,omitempty" minimum:"1" maximum:"50" doc:"Null resets it to 25"`
	Categories      api.Patch[[]int] `json:"categories,omitempty" doc:"Null for every category it advertises"`
	Searches        api.Patch[limit] `json:"searches,omitempty" doc:"Null turns the limit off"`
	Grabs           api.Patch[limit] `json:"grabs,omitempty" doc:"Null turns the limit off"`
}

// patched is p's value, def for null, nil when absent.
func patched[T any](p api.Patch[T], def T) *T {
	switch {
	case !p.Sent():
		return nil
	case p.Value() == nil:
		return &def
	}
	return p.Value()
}

func (e indexerEdit) edit() (indexers.Edit, error) {
	out := indexers.Edit{
		Name: e.Name, Address: e.Address, APIKey: e.APIKey,
		On: patched(e.On, true), AutoSearch: patched(e.AutomaticSearch, true), RSS: patched(e.RSS, true),
		AllowFailover: patched(e.AllowFailover, false), Priority: patched(e.Priority, indexers.DefaultPriority),
	}
	if e.Categories.Sent() {
		var c []newznab.ID
		if v := e.Categories.Value(); v != nil {
			c = indexers.CategoriesOf(*v)
		}
		out.Categories = &c
	}
	for _, l := range []struct {
		p  api.Patch[limit]
		to **indexers.Limit
	}{{e.Searches, &out.Searches}, {e.Grabs, &out.Grabs}} {
		if v := patched(l.p, limit{Per: string(indexers.Hour)}); v != nil {
			in := limitIn(*v)
			*l.to = &in
		}
	}
	if e.HomeNodeID != nil {
		id, err := storage.ParseUUID(*e.HomeNodeID)
		if err != nil {
			return indexers.Edit{}, nodeNotSearching
		}
		out.Home = &role.NodeID{UUID: id}
	}
	return out, nil
}

type report struct {
	SearchModes []searchMode `json:"searchModes" nullable:"false" doc:"By mode"`
	Items       int          `json:"items" minimum:"0" doc:"The newest items it listed"`
	Sources     []string     `json:"sources" nullable:"false" doc:"The different sources a Torznab feed's items named: more than one is an aggregate feed"`
}

func reportOf(r indexers.Report) report {
	out := report{SearchModes: []searchMode{}, Items: r.Items, Sources: append([]string{}, r.Sources...)}
	for _, m := range slices.Sorted(maps.Keys(r.Caps.Modes)) {
		out.SearchModes = append(out.SearchModes, searchMode{Mode: m, Params: r.Caps.Modes[m]})
	}
	return out
}

type query struct {
	ID          string     `json:"id" format:"uuid"`
	Use         string     `json:"use" enum:"search,recent,fetch,test"`
	ReservedAt  time.Time  `json:"reservedAt"`
	StartedAt   *time.Time `json:"startedAt" doc:"When its Node sent its first request; null before, or when it sent none"`
	CompletedAt *time.Time `json:"completedAt" doc:"Null while it runs"`
	Outcome     *string    `json:"outcome" enum:"ok,limited,failing,stopped,lost" doc:"Null while it runs; lost when its Node never answered"`
	Releases    int        `json:"releases" minimum:"0"`
}

const queriesPage = 50

type queryList struct {
	Queries    []query `json:"queries" nullable:"false" doc:"Newest first"`
	NextOffset *int    `json:"nextOffset" doc:"The offset of the next page; null on the last"`
}

type idInput struct {
	ID string `path:"id" format:"uuid"`
}

type queriesInput struct {
	ID     string `path:"id" format:"uuid"`
	Offset int    `query:"offset" minimum:"0" doc:"How many of the newest to skip"`
}

type createIndexerInput struct{ Body newIndexer }

type updateIndexerInput struct {
	ID   string `path:"id" format:"uuid"`
	Body indexerEdit
}

type indexerOutput struct{ Body indexer }

type reportOutput struct{ Body report }

type queriesOutput struct{ Body queryList }

type indexerList struct {
	Indexers []indexer `json:"indexers" nullable:"false" doc:"By name"`
}

type indexerListOutput struct{ Body indexerList }

func Routes(r *api.Router, registry Registry) {
	o := r.Owner("indexers", api.Admin)
	h := handlers{registry}
	tested := []*api.Code{indexerTestFailed, indexerUnavailable}

	api.Register(o, api.Admin, api.Op{
		ID: "listIndexers", Method: http.MethodGet, Path: "/indexers", Summary: "List the Indexers", Status: http.StatusOK,
	}, h.listIndexers)

	api.Register(o, api.Admin, api.Op{
		ID: "getIndexer", Method: http.MethodGet, Path: "/indexers/{id}", Summary: "Get an Indexer", Status: http.StatusOK,
		Errors: []*api.Code{indexerNotFound},
	}, h.getIndexer)

	api.Register(o, api.Admin, api.Op{
		ID:      "createIndexer",
		Method:  http.MethodPost,
		Path:    "/indexers",
		Summary: "Add an Indexer once its Test passed on the Node it searches from",
		Status:  http.StatusCreated,
		Errors:  append([]*api.Code{indexerNameTaken, noSearchingNode, nodeNotSearching}, tested...),
	}, h.createIndexer)

	api.Register(o, api.Admin, api.Op{
		ID:      "updateIndexer",
		Method:  http.MethodPatch,
		Path:    "/indexers/{id}",
		Summary: "Change an Indexer, testing it first when the change is to its home, address or API key",
		Status:  http.StatusOK,
		Errors:  append([]*api.Code{indexerNotFound, indexerNameTaken, nodeNotSearching}, tested...),
	}, h.updateIndexer)

	api.Register(o, api.Admin, api.Op{
		ID:      "deleteIndexer",
		Method:  http.MethodDelete,
		Path:    "/indexers/{id}",
		Summary: "Delete an Indexer with its status, sessions and queries. Its Grabs keep their history.",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{indexerNotFound},
	}, h.deleteIndexer)

	api.Register(o, api.Admin, api.Op{
		ID:      "testIndexer",
		Method:  http.MethodPost,
		Path:    "/indexers/{id}/test",
		Summary: "Test an Indexer from the Node it searches from. A pass clears its backoff and any stop.",
		Status:  http.StatusOK,
		Errors:  append([]*api.Code{indexerNotFound}, tested...),
	}, h.testIndexer)

	api.Register(o, api.Admin, api.Op{
		ID:      "testIndexerDraft",
		Method:  http.MethodPost,
		Path:    "/indexers/test",
		Summary: "Test an Indexer without saving it",
		Status:  http.StatusOK,
		Errors:  append([]*api.Code{noSearchingNode, nodeNotSearching}, tested...),
	}, h.testIndexerDraft)

	api.Register(o, api.Admin, api.Op{
		ID:      "listIndexerQueries",
		Method:  http.MethodGet,
		Path:    "/indexers/{id}/queries",
		Summary: "List an Indexer's searches, fetches and Tests, 50 at a time",
		Status:  http.StatusOK,
		Errors:  []*api.Code{indexerNotFound},
	}, h.listIndexerQueries)

	api.Register(o, api.Admin, api.Op{
		ID:      "getIndexerCatalog",
		Method:  http.MethodGet,
		Path:    "/indexer-catalog",
		Summary: "List the Indexer catalog's definitions and the custom ones, with when the catalog last refreshed",
		Status:  http.StatusOK,
	}, h.getCatalog)

	api.Register(o, api.Admin, api.Op{
		ID:      "getCatalogDefinition",
		Method:  http.MethodGet,
		Path:    "/indexer-catalog/{id}",
		Summary: "Get the fields of a definition's add form, from the custom definition with the id or else the catalog's",
		Status:  http.StatusOK,
		Errors:  []*api.Code{definitionNotFound, catalogUnavailable},
	}, h.getDefinition)

	api.Register(o, api.Admin, api.Op{
		ID:      "setCatalogAddress",
		Method:  http.MethodPut,
		Path:    "/indexer-catalog/address",
		Summary: "Change the Indexer catalog's address once SlipStream reads a schema 12 index there. The next refresh reads it in full.",
		Status:  http.StatusOK,
		Errors:  []*api.Code{catalogAddressUnreachable},
	}, h.setAddress)

	api.Register(o, api.Admin, api.Op{
		ID:      "addCustomDefinition",
		Method:  http.MethodPost,
		Path:    "/indexer-catalog/custom",
		Summary: "Add a custom definition, or replace the custom definition with its id and move that definition's Indexers to it",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{definitionUnloadable, definitionIDTaken},
	}, h.addCustom)

	api.Register(o, api.Admin, api.Op{
		ID:      "deleteCustomDefinition",
		Method:  http.MethodDelete,
		Path:    "/indexer-catalog/custom/{id}",
		Summary: "Delete a custom definition that no Indexer runs",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{definitionNotFound, definitionInUse},
	}, h.deleteCustom)

	api.Register(o, api.Admin, api.Op{
		ID: "listTrackers", Method: http.MethodGet, Path: "/trackers", Summary: "List the Trackers and the Indexers that reach each", Status: http.StatusOK,
	}, h.listTrackers)

	api.Register(o, api.Admin, api.Op{
		ID: "getIndexerSettings", Method: http.MethodGet, Path: "/indexer-settings", Summary: "Get the Indexer settings", Status: http.StatusOK,
	}, h.getSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "updateIndexerSettings",
		Method:  http.MethodPatch,
		Path:    "/indexer-settings",
		Summary: "Change Indexer settings by key, all or none",
		Status:  http.StatusOK,
		Errors:  []*api.Code{settingapi.ErrOutOfRange, settingapi.ErrUnknown},
	}, h.updateSettings)
}

type handlers struct{ registry Registry }

func parseID(raw string) (indexers.ID, error) {
	id, err := storage.ParseUUID(raw)
	if err != nil {
		return indexers.ID{}, indexerNotFound
	}
	return indexers.ID{UUID: id}, nil
}

func (h handlers) listIndexers(ctx context.Context, by identity.Admin, _ *struct{}) (*indexerListOutput, error) {
	list, err := open(h.registry).List(ctx, by)
	if err != nil {
		return nil, err
	}
	out := indexerList{Indexers: []indexer{}}
	for _, v := range list {
		out.Indexers = append(out.Indexers, indexerOf(v))
	}
	return &indexerListOutput{Body: out}, nil
}

func (h handlers) getIndexer(ctx context.Context, by identity.Admin, in *idInput) (*indexerOutput, error) {
	id, err := parseID(in.ID)
	if err != nil {
		return nil, err
	}
	xs := open(h.registry)
	v, err := xs.Get(ctx, by, id)
	if err != nil {
		return nil, answer(xs, err)
	}
	return &indexerOutput{Body: indexerOf(v)}, nil
}

func (h handlers) createIndexer(ctx context.Context, by identity.Admin, in *createIndexerInput) (*indexerOutput, error) {
	d, err := in.Body.draft()
	if err != nil {
		return nil, err
	}
	xs := open(h.registry)
	v, err := xs.Create(ctx, by, d)
	if err != nil {
		return nil, answer(xs, err)
	}
	return &indexerOutput{Body: indexerOf(v)}, nil
}

func (h handlers) updateIndexer(ctx context.Context, by identity.Admin, in *updateIndexerInput) (*indexerOutput, error) {
	id, err := parseID(in.ID)
	if err != nil {
		return nil, err
	}
	e, err := in.Body.edit()
	if err != nil {
		return nil, err
	}
	xs := open(h.registry)
	v, err := xs.Update(ctx, by, id, e)
	if err != nil {
		return nil, answer(xs, err)
	}
	return &indexerOutput{Body: indexerOf(v)}, nil
}

func (h handlers) deleteIndexer(ctx context.Context, by identity.Admin, in *idInput) (*struct{}, error) {
	id, err := parseID(in.ID)
	if err != nil {
		return nil, err
	}
	xs := open(h.registry)
	return nil, answer(xs, xs.Delete(ctx, by, id))
}

func (h handlers) testIndexer(ctx context.Context, by identity.Admin, in *idInput) (*reportOutput, error) {
	id, err := parseID(in.ID)
	if err != nil {
		return nil, err
	}
	xs := open(h.registry)
	r, err := xs.Test(ctx, by, id)
	if err != nil {
		return nil, answer(xs, err)
	}
	return &reportOutput{Body: reportOf(r)}, nil
}

func (h handlers) testIndexerDraft(ctx context.Context, by identity.Admin, in *createIndexerInput) (*reportOutput, error) {
	d, err := in.Body.draft()
	if err != nil {
		return nil, err
	}
	xs := open(h.registry)
	r, err := xs.TestDraft(ctx, by, d)
	if err != nil {
		return nil, answer(xs, err)
	}
	return &reportOutput{Body: reportOf(r)}, nil
}

func (h handlers) listIndexerQueries(ctx context.Context, by identity.Admin, in *queriesInput) (*queriesOutput, error) {
	id, err := parseID(in.ID)
	if err != nil {
		return nil, err
	}
	xs := open(h.registry)
	rows, err := xs.Queries(ctx, by, id, indexers.Page{Offset: in.Offset, Limit: queriesPage + 1})
	if err != nil {
		return nil, answer(xs, err)
	}
	out := queryList{Queries: []query{}}
	if len(rows) > queriesPage {
		rows, out.NextOffset = rows[:queriesPage], new(in.Offset+queriesPage)
	}
	for _, q := range rows {
		out.Queries = append(out.Queries, query{
			ID: q.ID.String(), Use: string(q.Use), ReservedAt: q.Reserved, StartedAt: when(q.Started), CompletedAt: when(q.Completed),
			Outcome: text(q.Outcome), Releases: q.Releases,
		})
	}
	return &queriesOutput{Body: out}, nil
}
