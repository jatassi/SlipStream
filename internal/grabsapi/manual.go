package grabsapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type manualCell struct {
	ItemID   string `json:"itemId" format:"uuid"`
	Edition  string `json:"edition" minLength:"1" doc:"The cell's Edition: any, or a cut key"`
	TargetID string `json:"targetId" format:"uuid"`
}

type startManual struct {
	LevelIDs []string    `json:"levelIds,omitempty" format:"uuid" maxItems:"20000" nullable:"false" doc:"Each level stands for every Item under it"`
	ItemIDs  []string    `json:"itemIds,omitempty" format:"uuid" maxItems:"20000" nullable:"false"`
	Cell     *manualCell `json:"cell,omitempty" doc:"One cell to plan for; send it without levelIds and itemIds"`
	Query    string      `json:"query,omitempty" maxLength:"200" doc:"Text every Indexer searches for in place of the Item's terms; only for a scope of one Item"`
	Again    bool        `json:"again,omitempty" doc:"Start a new search even when the Work's Manual search has this scope and query"`
}

type startManualInput struct {
	WorkID string `path:"workId" format:"uuid"`
	Body   *startManual
}

type manualStarted struct {
	ManualSearchID string `json:"manualSearchId" format:"uuid"`
}

type manualStartedOutput struct {
	Status int
	Body   manualStarted
}

type indexerLine struct {
	IndexerID *string `json:"indexerId" format:"uuid" doc:"Null once the Indexer is deleted"`
	Name      string  `json:"name"`
	Sentence  string  `json:"sentence" doc:"Its outcome in words, such as Limit reached until 14:05, or TorrentLeech searched 6 of 10 episodes. Its limit resets at 14:05."`
	Done      bool    `json:"done" doc:"It has an outcome for every Item it was asked about"`
	Automatic bool    `json:"automatic" doc:"Automatic search uses it"`
	Failed    bool    `json:"failed" doc:"A fault at the site, as opposed to a limit or no Releases"`
}

type releaseDetails struct {
	ID          string               `json:"id" doc:"Send it back as a pick's releaseId"`
	Title       string               `json:"title"`
	Words       string               `json:"words" doc:"What SlipStream reads from the title, in words"`
	Indexers    []string             `json:"indexers" nullable:"false" doc:"The Indexers that listed it, by priority"`
	Tracker     string               `json:"tracker,omitempty" doc:"The Tracker its link reaches, absent when none is known"`
	Size        int64                `json:"size" minimum:"0" doc:"Bytes"`
	PublishedAt *time.Time           `json:"publishedAt" doc:"Null when the Indexer sent no date"`
	Seeders     int                  `json:"seeders" minimum:"0"`
	Freeleech   bool                 `json:"freeleech"`
	Formats     []string             `json:"formatIds" format:"uuid" nullable:"false" doc:"The Custom Formats it matched of those the Work's Targets score"`
	Verdicts    []string             `json:"verdicts" nullable:"false" doc:"Its verdict at each Target of its Item, such as It would fill 1080p."`
	LostAt      api.Nullable[lostAt] `json:"lostAt" doc:"Where it lost to the Release that fills its first cell; null when it didn't"`
	Blocked     *string              `json:"blocked" doc:"Since when and why it is on the Blocklist; null when it isn't"`
}

type planPick struct {
	Release releaseDetails `json:"release"`
	Lifts   []wordedReason `json:"lifts" nullable:"false" doc:"The soft refusals it is grabbed past; send their codes and Targets back as the pick's lifts"`
	Edition *string        `json:"edition" doc:"The Edition it is grabbed as; null unless that is another Edition than its cut's"`
	Routed  string         `json:"routed" doc:"Where Tracker routing sends it, such as Where Tracker routing sends it (qBittorrent on Den)"`
}

type choice struct {
	Pin     string         `json:"pin" doc:"Send it in pins to swap this Release into its row"`
	Release releaseDetails `json:"release"`
	Lifts   []wordedReason `json:"lifts" nullable:"false"`
	Edition *string        `json:"edition" doc:"The Edition it is grabbed as; null unless that is another Edition than its cut's"`
	Note    string         `json:"note,omitempty" doc:"Such as From TorrentLeech, which automatic search doesn't use."`
}

type planRow struct {
	Key     string                 `json:"key" doc:"The row's first cell, stable while that cell leads it. A pin chosen from its choices replaces the pin with this key."`
	Pick    api.Nullable[planPick] `json:"pick" doc:"Null for a cell nothing fills"`
	Pinned  bool                   `json:"pinned" doc:"Its Release was swapped in by a pin"`
	Outcome string                 `json:"outcome,omitempty" doc:"What a Grab of it gets, such as It gets you 1080p and 720p for E05."`
	Missing string                 `json:"missing,omitempty" doc:"For a cell nothing fills, such as Missing: 720p for E05."`
	Cause   string                 `json:"cause,omitempty" doc:"For a cell nothing fills, its plain cause, such as No Indexer listed a 720p Release."`
	Failure string                 `json:"failure,omitempty" doc:"For a failed cell, its last Grab's failure"`
	Choices []choice               `json:"choices" nullable:"false" doc:"Every other Release that could fill its first cell, ranked"`
}

type heldCell struct {
	GrabID string `json:"grabId" format:"uuid"`
	Words  string `json:"words" doc:"Such as A Grab is downloading Heat.1995.1080p.WEB for 1080p."`
	Stage  string `json:"stage" enum:"fetching,downloading,probing,transferring,importing"`
}

type plan struct {
	Headline string     `json:"headline"`
	Rows     []planRow  `json:"rows" nullable:"false" doc:"Each Grab and each cell nothing fills, in Want order"`
	Held     []heldCell `json:"held" nullable:"false" doc:"The cells a Grab downloads, which the plan leaves out"`
}

type foldGroup struct {
	Words string `json:"words" doc:"Such as 8 are 4K, which no Target here takes."`
	Fault bool   `json:"fault" doc:"A fault in the Releases, as opposed to a refusal"`
}

type folded struct {
	Line   string      `json:"line" doc:"Such as SlipStream wouldn't grab 14 more Releases."`
	Groups []foldGroup `json:"groups" nullable:"false"`
}

type pinned struct {
	Key string `json:"key" doc:"The key of the row it fills"`
	Pin string `json:"pin"`
}

type manualSearch struct {
	ID           string        `json:"id" format:"uuid"`
	Title        string        `json:"title" doc:"The scope in words"`
	Query        *string       `json:"query" doc:"The typed query; null when each run sent its Item's terms"`
	StartedAt    time.Time     `json:"startedAt"`
	StartedBy    string        `json:"startedBy" doc:"A name, or the API key"`
	Searching    bool          `json:"searching" doc:"An Indexer asked hasn't responded yet"`
	Queryable    bool          `json:"queryable" doc:"The scope is one Item, so a typed query can search it"`
	Line         string        `json:"line" doc:"Such as Searching 6 Indexers. 4 have responded."`
	Indexers     []indexerLine `json:"indexers" nullable:"false"`
	Plan         plan          `json:"plan"`
	Folded       folded        `json:"folded" doc:"The Releases SlipStream wouldn't grab, in groups"`
	ReleaseCount int           `json:"releaseCount" minimum:"0" doc:"How many Releases listManualSearchReleases lists"`
	Pins         []pinned      `json:"pins" nullable:"false" doc:"The pins still in effect; send these next"`
}

type getManualInput struct {
	WorkID string   `path:"workId" format:"uuid"`
	Pins   []string `query:"pins" maxItems:"2000" doc:"The pins from the last response, with any swap applied. A pin that no longer applies is dropped."`
}

type manualSearchOutput struct{ Body manualSearch }

type grabOffer struct {
	Grabbable bool           `json:"grabbable" doc:"It fills some cell and its Indexer would take it now. One on the Blocklist is grabbed only with unblock."`
	Outcome   string         `json:"outcome,omitempty" doc:"What a Grab of it alone gets"`
	Lifts     []wordedReason `json:"lifts" nullable:"false"`
	Edition   *string        `json:"edition" doc:"The Edition it is grabbed as; null unless that is another Edition than its cut's"`
	Cancels   []heldCell     `json:"cancels" nullable:"false" doc:"The Grabs downloading a cell it would also fill"`
	Waits     string         `json:"waits,omitempty" doc:"Why its Indexer won't take it now, such as TorrentLeech is at its grab limit until 14:05."`
	Refusal   string         `json:"refusal,omitempty" doc:"Why it can't be grabbed"`
}

type offer struct {
	Release releaseDetails `json:"release"`
	Rank    *int           `json:"rank" minimum:"1" doc:"Its place in the ranking from 1; null when no cell's Target takes it"`
	Score   int            `json:"score" doc:"Its score at its best Target"`
	Quality int            `json:"quality" minimum:"0" doc:"Its quality's place in its best Target's tiers, for sorting; higher is better"`
	Grab    grabOffer      `json:"grab"`
}

type offerList struct {
	Releases []offer `json:"releases" nullable:"false" doc:"One per group of copies, in rank order"`
}

type offerListOutput struct{ Body offerList }

type liftKey struct {
	Code     string `json:"code" enum:"wrong_title,size_below,size_above,score_below_minimum,repack_other_group"`
	TargetID string `json:"targetId,omitempty" format:"uuid" doc:"Absent for wrong_title"`
}

type clientChoice struct {
	ClientID  *string `json:"clientId,omitempty" format:"uuid" doc:"An external client; send this or builtInOn"`
	BuiltInOn *string `json:"builtInOn,omitempty" format:"uuid" doc:"A downloading Node, for its built-in client; send this or clientId"`
}

type grabPick struct {
	ReleaseID    string        `json:"releaseId"`
	Lifts        []liftKey     `json:"lifts" maxItems:"100" nullable:"false" doc:"The lifts the row showed"`
	Edition      *string       `json:"edition,omitempty" minLength:"1" doc:"The Edition the row showed"`
	Client       *clientChoice `json:"client,omitempty" doc:"Absent to go where Tracker routing sends it"`
	Unblock      bool          `json:"unblock,omitempty" doc:"Remove the Release from the Blocklist and grab it"`
	CancelGrabID *string       `json:"cancelGrabId,omitempty" format:"uuid" doc:"The Grab downloading a cell it fills, canceled first"`
}

type grabManualInput struct {
	WorkID string `path:"workId" format:"uuid"`
	Body   struct {
		ManualSearchID string     `json:"manualSearchId" format:"uuid"`
		Picks          []grabPick `json:"picks" minItems:"1" maxItems:"2000" nullable:"false"`
	}
}

type pickProblem struct {
	Code   string  `json:"code" enum:"pick_conflict,cell_held,pick_changed,not_taken,not_grabbable,grab_busy,download_client_field_refused,node_not_downloading"`
	Detail string  `json:"detail"`
	GrabID *string `json:"grabId,omitempty" format:"uuid" doc:"cell_held: the Grab downloading the cell"`
}

type pickResult struct {
	GrabID  *string                   `json:"grabId" format:"uuid" doc:"Null when the pick was refused"`
	Problem api.Nullable[pickProblem] `json:"problem" doc:"Null when the pick made a Grab"`
}

type pickResults struct {
	Results []pickResult `json:"results" nullable:"false" doc:"One for each pick, in the order sent"`
}

type pickResultsOutput struct{ Body pickResults }

func workOf(s string) (works.WorkID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return works.WorkID{}, workNotFound
	}
	return works.WorkID{UUID: u}, nil
}

func (h handlers) startManualSearch(ctx context.Context, by identity.Admin, in *startManualInput) (*manualStartedOutput, error) {
	work, err := workOf(in.WorkID)
	if err != nil {
		return nil, err
	}
	scope, err := manualScopeOf(in.Body)
	if err != nil {
		return nil, err
	}
	id, started, err := open(h.p).StartManualSearch(ctx, by, work, scope)
	switch {
	case errors.Is(err, grabs.ErrReconciling):
		return nil, reconciling.Because("Open Manual search again once Reconciling finishes.")
	case errors.Is(err, grabs.ErrNoCell):
		return nil, api.Invalid.On("cell", "The Item has no cell of that Edition and Target.")
	}
	if e, ok := errors.AsType[*works.NotInWorkError](err); ok && scope.Cell != nil {
		return nil, api.Invalid.On("cell.itemId", "The Work has no Item with that id.")
	} else if ok {
		return nil, notInWork(e)
	}
	if err != nil {
		return nil, answer(err)
	}
	status := http.StatusOK
	if started {
		status = http.StatusAccepted
	}
	return &manualStartedOutput{Status: status, Body: manualStarted{ManualSearchID: id.String()}}, nil
}

// manualScopeOf is the Manual search scope body names, the whole Work
// when it names no part.
func manualScopeOf(body *startManual) (grabs.ManualScope, error) {
	if body == nil {
		return grabs.ManualScope{}, nil
	}
	s := grabs.ManualScope{Query: body.Query, Again: body.Again}
	if c := body.Cell; c != nil {
		if len(body.LevelIDs) > 0 || len(body.ItemIDs) > 0 {
			return grabs.ManualScope{}, api.Invalid.On("cell", "Send a cell alone, without levelIds and itemIds.")
		}
		item, err := storage.ParseUUID(c.ItemID)
		if err != nil {
			return grabs.ManualScope{}, api.Invalid
		}
		target, err := storage.ParseUUID(c.TargetID)
		if err != nil {
			return grabs.ManualScope{}, api.Invalid
		}
		key := files.CellKey{Item: works.ItemID{UUID: item}, Edition: targets.Edition(c.Edition), Target: targets.TargetID{UUID: target}}
		s.Part, s.Cell = grabs.Within(works.Within{Items: []works.ItemID{key.Item}}), &key
		return s, nil
	}
	if len(body.LevelIDs) == 0 && len(body.ItemIDs) == 0 {
		return s, nil
	}
	part, err := withinOf(body.LevelIDs, body.ItemIDs)
	if err != nil {
		return grabs.ManualScope{}, err
	}
	s.Part = grabs.Within(part)
	return s, nil
}

func (h handlers) getManualSearch(ctx context.Context, by identity.Admin, in *getManualInput) (*manualSearchOutput, error) {
	work, err := workOf(in.WorkID)
	if err != nil {
		return nil, err
	}
	var pins []grabs.Pin
	for _, s := range in.Pins {
		if p, ok := parsePin(s); ok {
			pins = append(pins, p)
		}
	}
	v, err := open(h.p).ManualSearch(ctx, by, work, pins)
	if err != nil {
		return nil, answer(err)
	}
	return &manualSearchOutput{Body: manualSearchOf(v)}, nil
}

func (h handlers) listManualSearchReleases(ctx context.Context, by identity.Admin, in *workInput) (*offerListOutput, error) {
	work, err := workOf(in.WorkID)
	if err != nil {
		return nil, err
	}
	offers, err := open(h.p).ManualReleases(ctx, by, work)
	if err != nil {
		return nil, answer(err)
	}
	out := offerList{Releases: []offer{}}
	for _, o := range offers {
		out.Releases = append(out.Releases, offerOf(o))
	}
	return &offerListOutput{Body: out}, nil
}

func (h handlers) grabFromManualSearch(ctx context.Context, by identity.Admin, in *grabManualInput) (*pickResultsOutput, error) {
	work, err := workOf(in.WorkID)
	if err != nil {
		return nil, err
	}
	u, err := storage.ParseUUID(in.Body.ManualSearchID)
	if err != nil {
		return nil, manualSearchReplaced
	}
	picks := make([]grabs.ManualPick, len(in.Body.Picks))
	for i, p := range in.Body.Picks {
		if picks[i], err = manualPickOf(fmt.Sprintf("picks[%d]", i), p); err != nil {
			return nil, err
		}
	}
	results, err := open(h.p).GrabFromManual(ctx, by, work, grabs.ManualID{UUID: u}, picks)
	switch {
	case errors.Is(err, grabs.ErrManualSearchReplaced):
		return nil, manualSearchReplaced
	case errors.Is(err, grabs.ErrReconciling):
		return nil, reconciling.Because("Open Manual search again once Reconciling finishes.")
	case err != nil:
		return nil, answer(err)
	}
	out := pickResults{Results: make([]pickResult, len(results))}
	for i, r := range results {
		if e := r.Err; e != nil {
			out.Results[i].Problem = api.Some(pickProblem{Code: string(e.Code), Detail: e.Words, GrabID: idOrNil(e.Held.UUID)})
			continue
		}
		out.Results[i].GrabID = new(r.Grab.String())
	}
	return &pickResultsOutput{Body: out}, nil
}

// manualPickOf is the pick at field as grabs takes it.
func manualPickOf(field string, p grabPick) (grabs.ManualPick, error) {
	ref, ok := parseReleaseID(p.ReleaseID)
	if !ok {
		return grabs.ManualPick{}, api.Invalid.On(field+".releaseId", "Send a releaseId from this Manual search.")
	}
	out := grabs.ManualPick{Release: ref, Unblock: p.Unblock}
	for _, l := range p.Lifts {
		k := grabs.LiftKey{Code: targets.ReasonCode(l.Code)}
		if l.TargetID != "" {
			u, err := storage.ParseUUID(l.TargetID)
			if err != nil {
				return grabs.ManualPick{}, api.Invalid
			}
			k.Target = targets.TargetID{UUID: u}
		}
		out.Lifts = append(out.Lifts, k)
	}
	if p.Edition != nil {
		out.Edition = targets.Edition(*p.Edition)
	}
	if c := p.Client; c != nil {
		if (c.ClientID == nil) == (c.BuiltInOn == nil) {
			return grabs.ManualPick{}, api.Invalid.On(field+".client", "Send exactly one of clientId and builtInOn.")
		}
		choice, err := choiceOf(c.ClientID, c.BuiltInOn)
		if err != nil {
			return grabs.ManualPick{}, api.Invalid
		}
		out.Client = &choice
	}
	if p.CancelGrabID != nil {
		u, err := storage.ParseUUID(*p.CancelGrabID)
		if err != nil {
			return grabs.ManualPick{}, api.Invalid
		}
		out.Cancel = &grabs.GrabID{UUID: u}
	}
	return out, nil
}

func editionOrNil(e targets.Edition) *string {
	if e == "" {
		return nil
	}
	return new(string(e))
}

func manualSearchOf(v grabs.ManualView) manualSearch {
	out := manualSearch{
		ID: v.ID.String(), Title: v.Title, StartedAt: v.Started, StartedBy: v.StartedBy.Words(), Searching: v.Searching,
		Queryable: v.Queryable, Line: v.Line, Indexers: []indexerLine{}, Plan: planOf(v.Plan), ReleaseCount: v.Releases,
		Folded: folded{Line: v.Folded.Line, Groups: []foldGroup{}}, Pins: []pinned{},
	}
	if v.Query != "" {
		out.Query = new(v.Query)
	}
	for _, l := range v.Indexers {
		out.Indexers = append(out.Indexers, indexerLine{
			IndexerID: idOrNil(l.Indexer.UUID), Name: l.Name, Sentence: l.Sentence, Done: l.Done, Automatic: l.Automatic, Failed: l.Failed,
		})
	}
	for _, g := range v.Folded.Groups {
		out.Folded.Groups = append(out.Folded.Groups, foldGroup{Words: g.Words, Fault: g.Fault})
	}
	for _, p := range v.Pins {
		out.Pins = append(out.Pins, pinned{Key: cellKeyOf(p.Cell), Pin: pinOf(p)})
	}
	return out
}

func planOf(p grabs.ManualPlan) plan {
	out := plan{Headline: p.Headline, Rows: []planRow{}, Held: heldOf(p.Held)}
	for _, r := range p.Rows {
		row := planRow{
			Key: cellKeyOf(r.Cells[0]), Pinned: r.Pinned, Outcome: r.Outcome, Missing: r.Missing, Cause: r.Cause, Failure: r.Failure,
			Choices: []choice{},
		}
		if k := r.Pick; k != nil {
			row.Pick = api.Some(planPick{Release: releaseDetailsOf(k.Release), Lifts: WordedOf(k.Lifts), Edition: editionOrNil(k.Edition), Routed: k.Routed})
		}
		for _, c := range r.Choices {
			row.Choices = append(row.Choices, choice{
				Pin: pinOf(c.Pin), Release: releaseDetailsOf(c.Release), Lifts: WordedOf(c.Lifts), Edition: editionOrNil(c.Edition), Note: c.Note,
			})
		}
		out.Rows = append(out.Rows, row)
	}
	return out
}

func heldOf(hs []grabs.HeldCell) []heldCell {
	out := []heldCell{}
	for _, h := range hs {
		out = append(out, heldCell{GrabID: h.Grab.String(), Words: h.Words, Stage: string(h.Stage)})
	}
	return out
}

func releaseDetailsOf(d grabs.ReleaseDetails) releaseDetails {
	out := releaseDetails{
		ID: releaseIDOf(d.Ref), Title: d.Title, Words: d.Words, Indexers: append([]string{}, d.Indexers...), Tracker: d.Tracker,
		Size: d.Size, Seeders: d.Seeders, Freeleech: d.Freeleech, Formats: []string{}, Verdicts: append([]string{}, d.Verdicts...),
	}
	if !d.Published.IsZero() {
		out.PublishedAt = &d.Published
	}
	for _, f := range d.Formats {
		out.Formats = append(out.Formats, f.String())
	}
	if lost := d.LostAt; lost != nil {
		out.LostAt = api.Some(lostAt{Step: string(lost.Step), TargetID: idOrNil(lost.Target.UUID), Own: lost.Own, Ref: lost.Ref})
	}
	if d.Blocked != "" {
		out.Blocked = new(d.Blocked)
	}
	return out
}

func offerOf(o grabs.Offer) offer {
	out := offer{
		Release: releaseDetailsOf(o.Release), Score: o.Score, Quality: o.Quality,
		Grab: grabOffer{
			Grabbable: o.Grab.Grabbable, Outcome: o.Grab.Outcome, Lifts: WordedOf(o.Grab.Lifts), Edition: editionOrNil(o.Grab.Edition),
			Cancels: heldOf(o.Grab.Cancels), Waits: o.Grab.Waits, Refusal: o.Grab.Refusal,
		},
	}
	if o.Rank > 0 {
		out.Rank = new(o.Rank)
	}
	return out
}
