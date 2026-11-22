// Package historyapi is History: what happened to the library, newest
// first, with links to what each entry is about while it still exists.
package historyapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type History interface {
	// History is the log, false unless the Node is a Controller.
	History() (*history.Service, bool)
	// Works reads the live labels and posters of the Works entries are
	// about, false unless the Node is a Controller.
	Works() (*works.Works, bool)
	// Health is which Health items are still on Health, false unless the
	// Node is a Controller.
	Health() (*health.Health, bool)
	// Minter signs the poster URLs the answers carry, false unless the
	// Node is a Controller.
	Minter() (artwork.Minter, bool)
}

type owners struct {
	history *history.Service
	works   *works.Works
	health  *health.Health
	minter  artwork.Minter
}

func open(n History) owners {
	h, ok := n.History()
	w, worksOK := n.Works()
	hl, healthOK := n.Health()
	m, minted := n.Minter()
	if !ok || !worksOK || !healthOK || !minted {
		panic("historyapi: an operation ran on a Node that is no Controller")
	}
	return owners{history: h, works: w, health: hl, minter: m}
}

var entryGone = api.NewCode(http.StatusNotFound, "history_entry_gone", "This History entry was deleted")

var (
	listHistoryOp = api.Op{
		ID: "listHistory", Method: http.MethodGet, Path: "/history",
		Summary: "List History entries, newest first", Status: http.StatusOK,
	}
	getHistoryEntryOp = api.Op{
		ID: "getHistoryEntry", Method: http.MethodGet, Path: "/history/{id}",
		Summary: "Read one History entry in full", Status: http.StatusOK, Errors: []*api.Code{entryGone},
	}
	getHistorySummaryOp = api.Op{
		ID: "getHistorySummary", Method: http.MethodGet, Path: "/history/summary",
		Summary: "Count what happened since an instant", Status: http.StatusOK,
	}
	listHistoryKindsOp = api.Op{
		ID: "listHistoryKinds", Method: http.MethodGet, Path: "/history/kinds",
		Summary: "List History's groups and kinds", Status: http.StatusOK,
	}
	clearHistoryOp = api.Op{
		ID: "clearHistory", Method: http.MethodDelete, Path: "/history",
		Summary: "Delete every History entry", Status: http.StatusAccepted,
	}
)

func Routes(r *api.Router, n History) {
	o := r.Owner("history", api.Admin)
	api.Register(o, api.Admin, listHistoryOp, func(ctx context.Context, by identity.Admin, in *listInput) (*listOutput, error) {
		return list(ctx, open(n), by, in)
	})
	api.Register(o, api.Admin, getHistorySummaryOp, func(ctx context.Context, by identity.Admin, in *summaryInput) (*summaryOutput, error) {
		return summary(ctx, open(n), by, in)
	})
	api.Register(o, api.Admin, listHistoryKindsOp, func(_ context.Context, _ identity.Admin, _ *struct{}) (*kindsOutput, error) {
		return kinds(open(n)), nil
	})
	api.Register(o, api.Admin, getHistoryEntryOp, func(ctx context.Context, by identity.Admin, in *oneInput) (*oneOutput, error) {
		return one(ctx, open(n), by, in)
	})
	api.Register(o, api.Admin, clearHistoryOp, func(ctx context.Context, by identity.Admin, _ *struct{}) (*struct{}, error) {
		return nil, open(n).history.Clear(ctx, by)
	})
}

func refused(location, message string) error {
	return api.Invalid.Fields(api.FieldError{Location: "query." + location, Message: message})
}

type listInput struct {
	Before      int64     `query:"before" minimum:"1" doc:"Only entries with a smaller id: the next member of the previous page; left out, the newest"`
	Limit       int       `query:"limit" minimum:"1" maximum:"100" default:"50"`
	ContentType string    `query:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$" doc:"Only entries about this Content Type's Works; left out, every entry, Health included"`
	Groups      []string  `query:"groups" enum:"grabs,imports,library,subtitles,health,backups,moves" doc:"Only these groups; left out, all seven"`
	Kinds       []string  `query:"kinds" doc:"A comma list of the kinds GET /history/kinds lists. Each kind narrows its own group, and groups, when given, must include that group"`
	Since       time.Time `query:"since" doc:"Only entries captured at or after this instant"`
	Work        string    `query:"work" format:"uuid" doc:"Only entries about this Work"`
}

type listed struct {
	Entries []listEntry         `json:"entries" nullable:"false" doc:"Newest first"`
	Next    api.Nullable[int64] `json:"next" doc:"The before of the next page; null on the last page"`
}

type listOutput struct{ Body listed }

type listEntry struct {
	ID    int64                     `json:"id"`
	Kind  string                    `json:"kind" doc:"A kind GET /history/kinds lists"`
	Group string                    `json:"group" enum:"grabs,imports,library,subtitles,health,backups,moves"`
	At    time.Time                 `json:"at" doc:"When it was captured"`
	Words string                    `json:"words" doc:"Such as Imported 10 episodes of Severance"`
	Tone  string                    `json:"tone" enum:"info,good,warn,fail"`
	Work  api.Nullable[workSummary] `json:"work" doc:"The one Work it is about; null for none or many, and once that Work is deleted"`
	Links links                     `json:"links"`
}

type workSummary struct {
	ID          string  `json:"id" format:"uuid"`
	ContentType string  `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Label       string  `json:"label"`
	PosterURL   *string `json:"posterUrl" doc:"The artwork route for its poster, without a size; null when it has none"`
}

type links struct {
	Grab   api.Nullable[string] `json:"grab" format:"uuid" doc:"The Grab it is about; null for none, and once that Grab is deleted"`
	Import api.Nullable[string] `json:"import" format:"uuid" doc:"The Import it is about; null for none, and once that Import is deleted"`
	Health api.Nullable[string] `json:"health" doc:"The key, <category>.<key>, of a one-item Health entry's item while it is on Health; null otherwise"`
}

func list(ctx context.Context, o owners, by identity.Admin, in *listInput) (*listOutput, error) {
	f, err := filterOf(in)
	if err != nil {
		return nil, err
	}
	p, err := o.history.Page(ctx, by, f, history.Before(in.Before), in.Limit)
	if err != nil {
		return nil, kindRefused(err)
	}
	es, err := o.listed(ctx, p.Entries)
	if err != nil {
		return nil, err
	}
	out := listed{Entries: es}
	if p.Next != 0 {
		out.Next = api.Some(int64(p.Next))
	}
	return &listOutput{Body: out}, nil
}

func filterOf(in *listInput) (history.Filter, error) {
	f := history.Filter{Type: works.TypeID(in.ContentType), Since: in.Since}
	for _, g := range in.Groups {
		f.Groups = append(f.Groups, entry.Group(g))
	}
	for _, k := range in.Kinds {
		f.Kinds = append(f.Kinds, entry.Kind(k))
	}
	if in.Work != "" {
		u, err := storage.ParseUUID(in.Work)
		if err != nil {
			return history.Filter{}, err
		}
		f.Work = works.WorkID{UUID: u}
	}
	return f, nil
}

func kindRefused(err error) error {
	if unknown, ok := errors.AsType[*history.UnknownKindError](err); ok {
		return refused("kinds", "There's no History entry kind called "+string(unknown.Kind)+".")
	}
	if off, ok := errors.AsType[*history.KindOffGroupError](err); ok {
		k, g := off.Kind.Label, off.Kind.Group.Label()
		return refused("kinds", k+" is in "+g+", which isn't chosen. Choose "+g+" too, or leave out "+k+".")
	}
	return err
}

// listed is es on the wire, with the live label and poster of each Work
// and each link that still exists.
func (o owners) listed(ctx context.Context, es []history.Entry) ([]listEntry, error) {
	on, err := o.onHealth(ctx, es)
	if err != nil {
		return nil, err
	}
	k, err := o.kept(ctx, es)
	if err != nil {
		return nil, err
	}
	out := make([]listEntry, len(es))
	for i, e := range es {
		le := listEntry{
			ID: int64(e.ID), Kind: string(e.Spec.Kind), Group: string(e.Spec.Group), At: e.At,
			Words: e.Message.Title, Tone: toneOf(e.Message.Tone),
		}
		if w, ok := k.works[e.Work]; ok {
			le.Work = api.Some(o.workOf(w))
		}
		if k.grabs[grabs.GrabID{UUID: e.Grab}] {
			le.Links.Grab = api.Some(e.Grab.String())
		}
		if k.imports[imports.ImportID{UUID: e.Import}] {
			le.Links.Import = api.Some(e.Import.String())
		}
		if len(e.Health) == 1 && on[e.Health[0]] {
			le.Links.Health = api.Some(e.Health[0])
		}
		out[i] = le
	}
	return out, nil
}

// still is which of the Works, Grabs and Imports some entries name exist.
type still struct {
	grabs   map[grabs.GrabID]bool
	imports map[imports.ImportID]bool
	works   map[works.WorkID]works.Row
}

// kept is what es name that still exists, read in one Read.
func (o owners) kept(ctx context.Context, es []history.Entry) (still, error) {
	var grabIDs []grabs.GrabID
	var importIDs []imports.ImportID
	var workIDs []works.WorkID
	for _, e := range es {
		if !e.Grab.IsZero() {
			grabIDs = append(grabIDs, grabs.GrabID{UUID: e.Grab})
		}
		if !e.Import.IsZero() {
			importIDs = append(importIDs, imports.ImportID{UUID: e.Import})
		}
		if !e.Work.IsZero() {
			workIDs = append(workIDs, e.Work)
		}
	}
	k := still{works: map[works.WorkID]works.Row{}}
	err := o.history.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		if k.grabs, err = grabs.Existing(ctx, r, grabIDs); err != nil {
			return err
		}
		if k.imports, err = imports.Existing(ctx, r, importIDs); err != nil {
			return err
		}
		if len(workIDs) == 0 {
			return nil
		}
		ws, err := o.works.RowsIn(ctx, r, works.Scope{Works: workIDs})
		for _, w := range ws {
			k.works[w.ID] = w
		}
		return err
	})
	return k, err
}

// onHealth is which item keys of es's one-item Health entries are on
// Health now.
func (o owners) onHealth(ctx context.Context, es []history.Entry) (map[string]bool, error) {
	out := map[string]bool{}
	asked := false
	for _, e := range es {
		asked = asked || len(e.Health) == 1
	}
	if !asked {
		return out, nil
	}
	rep, err := o.health.Report(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range rep.Categories {
		for _, it := range c.Items {
			out[history.ItemKey(c.ID, it.Key)] = true
		}
	}
	return out, nil
}

func (o owners) workOf(w works.Row) workSummary {
	out := workSummary{ID: w.ID.String(), ContentType: string(w.ContentType), Label: w.Label}
	if !w.Poster.IsZero() {
		out.PosterURL = new(o.minter.URL(w.Poster))
	}
	return out
}

func toneOf(t entry.Tone) string { return [...]string{"info", "good", "warn", "fail"}[t] }

type oneInput struct {
	ID   int64  `path:"id" minimum:"1"`
	Zone string `query:"zone" required:"true" doc:"The IANA time zone the clock times are worded in, such as Europe/London"`
}

type oneEntry struct {
	Entry  listEntry    `json:"entry"`
	Lines  []string     `json:"lines" nullable:"false" doc:"Whole sentences, with clock times in zone"`
	Files  []file       `json:"files" nullable:"false" doc:"Each file an Import placed"`
	Stages []stageAt    `json:"stages" nullable:"false" doc:"Each stage a failed Grab reached, in order"`
	Health []healthItem `json:"health" nullable:"false" doc:"A Health entry's items"`
}

type file struct {
	Items         []string `json:"items" nullable:"false" doc:"Such as Episode 1"`
	Quality       string   `json:"quality"`
	PlacedBy      string   `json:"placedBy" doc:"How the file was placed: a placed-by value of GET /works/{workId}/files"`
	Replaced      string   `json:"replaced" doc:"The qualities it replaced, in words; empty for none"`
	ReplacedLabel string   `json:"replacedLabel" doc:"The label of the file it replaced; empty for none"`
}

type stageAt struct {
	Stage string    `json:"stage" doc:"A stage of GET /grabs"`
	At    time.Time `json:"at"`
}

type healthItem struct {
	Subject string `json:"subject"`
	Text    string `json:"text"`
	Level   string `json:"level" enum:"warn,fail"`
}

type oneOutput struct{ Body oneEntry }

func one(ctx context.Context, o owners, by identity.Admin, in *oneInput) (*oneOutput, error) {
	zone, err := time.LoadLocation(in.Zone)
	if err != nil || strings.EqualFold(in.Zone, "local") {
		return nil, refused("zone", "There's no time zone called "+in.Zone+". Use an IANA name, such as Europe/London.")
	}
	e, err := o.history.One(ctx, by, history.EventID(in.ID), zone)
	if errors.Is(err, history.ErrGone) {
		return nil, entryGone.Because("It was cleared, or it was older than Keep history for.")
	}
	if err != nil {
		return nil, err
	}
	es, err := o.listed(ctx, []history.Entry{e})
	if err != nil {
		return nil, err
	}
	m := e.Message
	out := oneEntry{Entry: es[0], Lines: m.Lines, Files: []file{}, Stages: []stageAt{}, Health: []healthItem{}}
	if out.Lines == nil {
		out.Lines = []string{}
	}
	for _, f := range m.Detail.Files {
		out.Files = append(out.Files, file{Items: f.Items, Quality: f.Quality, PlacedBy: f.PlacedBy, Replaced: f.Replaced, ReplacedLabel: f.ReplacedLabel})
	}
	for _, s := range m.Detail.Stages {
		out.Stages = append(out.Stages, stageAt(s))
	}
	for _, it := range m.Body.Health {
		out.Health = append(out.Health, healthItem{Subject: it.Subject, Text: it.Text, Level: it.Level})
	}
	return &oneOutput{Body: out}, nil
}

type summaryInput struct {
	Since time.Time `query:"since" required:"true" doc:"The instant the week starts: the browser's midnight 6 days back"`
}

type sentence struct {
	Words    string `json:"words"`
	NeedsYou int    `json:"needsYou" minimum:"0" maximum:"100" doc:"The failed Grabs that need attention, at most 100; 0 when the words count the week"`
}

type summaryOutput struct {
	Body api.Nullable[sentence] `doc:"null when History is empty and no Grab needs attention"`
}

func summary(ctx context.Context, o owners, by identity.Admin, in *summaryInput) (*summaryOutput, error) {
	var needsYou int
	err := o.history.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		needsYou, err = grabs.NeedsYou(ctx, r)
		return err
	})
	if err != nil {
		return nil, err
	}
	s, err := o.history.Summary(ctx, by, in.Since, needsYou)
	if err != nil || s == nil {
		return &summaryOutput{}, err
	}
	return &summaryOutput{Body: api.Some(sentence{Words: s.Words, NeedsYou: s.NeedsYou})}, nil
}

type kindsAnswer struct {
	Groups []group `json:"groups" nullable:"false" doc:"In filter order"`
}

type group struct {
	Group string `json:"group" enum:"grabs,imports,library,subtitles,health,backups,moves"`
	Label string `json:"label"`
	Kinds []kind `json:"kinds" nullable:"false"`
}

type kind struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

type kindsOutput struct{ Body kindsAnswer }

func kinds(o owners) *kindsOutput {
	out := kindsAnswer{Groups: []group{}}
	for _, g := range o.history.Kinds() {
		wg := group{Group: string(g.Group), Label: g.Label, Kinds: []kind{}}
		for _, k := range g.Kinds {
			wg.Kinds = append(wg.Kinds, kind{Kind: string(k.Kind), Label: k.Label})
		}
		out.Groups = append(out.Groups, wg)
	}
	return &kindsOutput{Body: out}
}
