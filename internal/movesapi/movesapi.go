// Package movesapi is S48 (Moves)'s operations: preview a Move, start it,
// follow its batches and units, cancel, retry, switch now, preview the
// way back, and its settings. Every operation translates HTTP; moves decides.
package movesapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Mover is the Node's Moves.
type Mover interface {
	// Moves is the Moves, false unless the Node is a Controller.
	Moves() (*moves.Moves, bool)
}

func open(n Mover) *moves.Moves {
	m, ok := n.Moves()
	if !ok {
		panic("movesapi: an operation ran on a Node that is no Controller")
	}
	return m
}

var (
	planChanged      = api.NewCode(http.StatusConflict, "move_plan_changed", "The Move changed since its preview. Open the preview again.")
	planExpired      = api.NewCode(http.StatusGone, "move_plan_expired", "The preview expired. Open it again.")
	nothingToMove    = api.NewCode(http.StatusConflict, "nothing_to_move", "Nothing in the preview can move now. Open the preview again.")
	unitSwitched     = api.NewCode(http.StatusConflict, "unit_switched", "This Move has switched. Move it back instead.")
	switchNotWaiting = api.NewCode(http.StatusConflict, "switch_not_waiting", "This Move isn't waiting for playback or a Media Server")
	unitNotFailed    = api.NewCode(http.StatusConflict, "unit_not_failed", "This Move hasn't failed")
	moveNotFound     = api.NewCode(http.StatusNotFound, "move_not_found", "SlipStream has no Move with this id")
	workNotFound     = api.NewCode(http.StatusNotFound, "work_not_found", "No Work has that id")
	rootNotFound     = api.NewCode(http.StatusNotFound, "root_not_found", "No root has that id")
	torrentNotFound  = api.NewCode(http.StatusNotFound, "torrent_not_found", "No torrent has that id")
	clientNotFound   = api.NewCode(http.StatusNotFound, "download_client_not_found", "No Download Client has this id")
	emptyAsk         = api.NewCode(http.StatusUnprocessableEntity, "move_ask_empty", "Pick a Work, a root, a torrent or a Download Client to move.")
)

// answer is err as its problem.
func answer(err error) error {
	var changed *moves.PlanChangedError
	switch {
	case errors.As(err, &changed):
		return planChanged
	case errors.Is(err, moves.ErrPlanExpired):
		return planExpired
	case errors.Is(err, moves.ErrNothingToMove):
		return nothingToMove
	case errors.Is(err, moves.ErrUnitSwitched):
		return unitSwitched
	case errors.Is(err, moves.ErrSwitchNotWaiting):
		return switchNotWaiting
	case errors.Is(err, moves.ErrUnitNotFailed):
		return unitNotFailed
	case errors.Is(err, moves.ErrNotFound):
		return moveNotFound
	case errors.Is(err, works.ErrNoWork):
		return workNotFound
	case errors.Is(err, moves.ErrNoRoot):
		return rootNotFound
	case errors.Is(err, moves.ErrNoTorrent):
		return torrentNotFound
	case errors.Is(err, moves.ErrReconciling):
		return api.Reconciling
	}
	return err
}

type place struct {
	NodeID   string `json:"nodeId" format:"uuid"`
	NodeName string `json:"nodeName"`
	RootID   string `json:"rootId" format:"uuid"`
	Root     string `json:"root" doc:"The library root in words, such as Movies library root on den; empty for a torrent"`
	Client   string `json:"client,omitempty" doc:"A torrent's Download Client, such as qBittorrent or the built-in client"`
}

func placeOf(p moves.Place) place {
	return place{NodeID: p.Node.String(), NodeName: p.NodeName, RootID: p.End.Root.String(), Root: p.Root, Client: p.Client}
}

type reason struct {
	Check string `json:"check" enum:"space,names,folder,destination,media_servers,waits,moving,probe,torrent,hit_and_run"`
	Color string `json:"color" enum:"amber,red"`
	Code  string `json:"code" doc:"Stable per reason, such as space_error or mapping_missing"`
	Words string `json:"words"`
}

type previewUnit struct {
	Key       string   `json:"key" doc:"Stable per unit inside the preview, such as work:<id> or torrent:<id>"`
	Kind      string   `json:"kind" enum:"work,torrent"`
	Name      string   `json:"name"`
	WorkID    string   `json:"workId" format:"uuid" doc:"A Work unit's; the zero id for a torrent"`
	TorrentID string   `json:"torrentId,omitempty" format:"uuid" doc:"A torrent unit's"`
	From      place    `json:"from"`
	To        place    `json:"to"`
	Verdict   string   `json:"verdict" enum:"green,amber,red" doc:"green moves, amber moves with a note, red stays"`
	Reasons   []reason `json:"reasons" nullable:"false" doc:"Each check worse than green"`
	Bytes     int64    `json:"bytes" minimum:"0" doc:"Bytes that copy; a hard link or a clone copies none"`
}

type movePreview struct {
	Token          string        `json:"token" doc:"Names the preview for its Start"`
	ExpiresAt      time.Time     `json:"expiresAt"`
	Recommendation string        `json:"recommendation" doc:"The preview in a sentence, such as Move 41 of 43 movies to den. 2 can't move: The Office and Heat."`
	Button         string        `json:"button" doc:"The Start button's label, such as Move 41 of 43"`
	Movable        int           `json:"movable" minimum:"0" doc:"The units that aren't red, which a Start confirms"`
	Total          int           `json:"total" minimum:"0"`
	Homed          int           `json:"homed" minimum:"0" doc:"Works without files the Start homes in their new root"`
	Units          []previewUnit `json:"units" nullable:"false"`
}

// previewOf is p on the wire, its places named by names.
func previewOf(p moves.Plan, named moves.Named) movePreview {
	out := movePreview{
		Token: string(p.Token), ExpiresAt: p.ExpiresAt, Recommendation: p.Recommendation, Button: p.Button,
		Movable: p.Movable(), Total: len(p.Units), Homed: len(p.Homed), Units: make([]previewUnit, len(p.Units)),
	}
	for i, u := range p.Units {
		pu := previewUnit{
			Key: "work:" + u.Subject.Work.String(), Kind: "work", Name: u.Label, WorkID: u.Subject.Work.String(),
			From: placeOf(named.Place(u.Subject.From)), To: placeOf(named.Place(u.Subject.To)),
			Verdict: u.Verdict.Color.String(), Reasons: make([]reason, len(u.Verdict.Reasons)), Bytes: u.Bytes,
		}
		for j, r := range u.Verdict.Reasons {
			pu.Reasons[j] = reason{Check: string(r.Check), Color: r.Color.String(), Code: r.Code, Words: r.Words}
		}
		if t := u.Torrent; t != nil {
			pu.Key, pu.Kind, pu.TorrentID = "torrent:"+t.Torrent.String(), "torrent", t.Torrent.String()
			pu.From, pu.To = torrentPlace(t.From, named), torrentPlace(t.To, named)
		}
		out.Units[i] = pu
	}
	return out
}

// torrentPlace is a torrent's side on the wire.
func torrentPlace(s moves.Side, named moves.Named) place {
	p := placeOf(named.Place(moves.End{Node: s.Node}))
	p.Client = named.ClientWords(s)
	return p
}

type workTo struct {
	WorkID string `json:"workId" format:"uuid"`
	RootID string `json:"rootId" format:"uuid" doc:"The library root it moves to"`
}

type torrentTo struct {
	TorrentID string `json:"torrentId" format:"uuid"`
	NodeID    string `json:"nodeId" format:"uuid" doc:"The Node of the Download Client it moves to"`
	ClientID  string `json:"clientId,omitempty" format:"uuid" doc:"The external Download Client it moves to; absent for the Node's built-in client"`
}

type rootTo struct {
	RootID string `json:"rootId" format:"uuid" doc:"The library root whose every Work moves"`
	To     string `json:"to" format:"uuid" doc:"The library root they move to"`
}

type side struct {
	NodeID   string `json:"nodeId" format:"uuid"`
	ClientID string `json:"clientId,omitempty" format:"uuid" doc:"An external Download Client; absent for the Node's built-in client"`
}

type clientTo struct {
	NodeID   string `json:"nodeId" format:"uuid" doc:"The Node of the Download Client whose every torrent moves"`
	ClientID string `json:"clientId,omitempty" format:"uuid" doc:"The external Download Client; absent for the Node's built-in client"`
	To       side   `json:"to" doc:"The Download Client they move to"`
}

type moveOptions struct {
	IncludeLinkedTorrents bool `json:"includeLinkedTorrents,omitempty" doc:"Also move the seeding torrent a Work's files are linked to"`
	RepointRouting        bool `json:"repointRouting,omitempty" doc:"Have each Tracker's new Grabs go to the client its torrents move to"`
	DeleteEmptiedRoots    bool `json:"deleteEmptiedRoots,omitempty" doc:"Delete each root the Move empties, once every unit ends"`
}

type moveAsk struct {
	Origin   string      `json:"origin" enum:"work,bulk,root,drain,client,torrent" doc:"Where the Admin starts it: a Work's page, the Library's bulk bar, a root's row, a Node's drain, a Download Client, or the Torrent table"`
	Works    []workTo    `json:"works" maxItems:"10000" nullable:"false" doc:"At least one Work, root, torrent or client"`
	Roots    []rootTo    `json:"roots,omitempty" maxItems:"100"`
	Torrents []torrentTo `json:"torrents,omitempty" maxItems:"10000"`
	Clients  []clientTo  `json:"clients,omitempty" maxItems:"100"`
	Options  moveOptions `json:"options,omitempty"`
}

type previewInput struct{ Body moveAsk }

type previewOutput struct{ Body movePreview }

type startAsk struct {
	Token string `json:"token" minLength:"1"`
}

type startInput struct {
	Confirm int `query:"confirm" required:"true" minimum:"0" doc:"The movable count the preview showed"`
	Body    startAsk
}

type dropped struct {
	WorkID    string `json:"workId" format:"uuid" doc:"A Work unit's; the zero id for a torrent"`
	TorrentID string `json:"torrentId,omitempty" format:"uuid"`
	Name      string `json:"name"`
	Words     string `json:"words" doc:"Why it stays"`
}

type startedMoves struct {
	BatchID string    `json:"batchId" format:"uuid"`
	Started int       `json:"started" minimum:"0" doc:"The units the batch moves"`
	Dropped []dropped `json:"dropped" nullable:"false" doc:"The preview's units that can't move now; empty for a Start sent again"`
}

type startOutput struct{ Body startedMoves }

type failure struct {
	Code  string `json:"code" enum:"source_gone,source_failed,damaged,source_written_off,destination_gone,destination_check_failed,switch_refused,destination_unreachable"`
	Words string `json:"words"`
}

type moveUnit struct {
	ID           string                `json:"id" format:"uuid"`
	BatchID      string                `json:"batchId" format:"uuid"`
	Kind         string                `json:"kind" enum:"work,torrent"`
	WorkID       string                `json:"workId" format:"uuid" doc:"A Work unit's; the zero id for a torrent"`
	TorrentID    string                `json:"torrentId,omitempty" format:"uuid" doc:"A torrent unit's"`
	Name         string                `json:"name"`
	ContentType  string                `json:"contentType"`
	Phase        string                `json:"phase" enum:"queued,transfer,place,switch,cleanup,done,failed,canceled"`
	Waiting      string                `json:"waiting,omitempty" enum:"pair_busy,low_priority,node_offline,no_route,space,renames,downloading,importing,open_files,playing,media_servers,guard,linked_torrent,reconciling" doc:"What it waits for; absent while it waits for nothing"`
	WaitingSince *time.Time            `json:"waitingSince,omitempty"`
	From         place                 `json:"from"`
	To           place                 `json:"to"`
	Bytes        int64                 `json:"bytes" minimum:"0"`
	Failure      api.Nullable[failure] `json:"failure" doc:"Why it failed; null unless it failed"`
	Undone       bool                  `json:"undone" doc:"A failed or canceled Move removed what it placed"`
	SwitchNow    bool                  `json:"switchNow" doc:"Whether Switch now can end its wait"`
	Notes        []string              `json:"notes" nullable:"false" doc:"What it recorded for the Admin, as sentences"`
	Words        string                `json:"words" doc:"What it does now, such as Copying to den"`
}

func unitOf(v moves.View) moveUnit {
	out := moveUnit{
		ID: v.Unit.String(), BatchID: v.Batch.String(), Kind: "work", WorkID: v.Work.String(), Name: v.Label, ContentType: string(v.ContentType),
		Phase: string(v.Phase), Waiting: string(v.Waiting), WaitingSince: v.Since, From: placeOf(v.From), To: placeOf(v.To),
		Bytes: v.Bytes, Undone: v.Undone, SwitchNow: v.SwitchNow, Notes: moves.NotesWords(v), Words: v.Words,
	}
	if f := v.Failure; f != nil {
		out.Failure = api.Some(failure{Code: string(f.Code), Words: f.Words})
	}
	if v.Torrent != nil {
		out.Kind, out.TorrentID = "torrent", v.Torrent.String()
	}
	return out
}

type unitOutput struct{ Body moveUnit }

type unitInput struct {
	UnitID string `path:"unitId" format:"uuid"`
}

type batchRow struct {
	ID          string         `json:"id" format:"uuid"`
	Origin      string         `json:"origin" enum:"work,bulk,root,drain,client,torrent"`
	CreatedAt   time.Time      `json:"createdAt"`
	FinishedAt  *time.Time     `json:"finishedAt,omitempty" doc:"When its last unit ended; absent while one runs"`
	CanceledAt  *time.Time     `json:"canceledAt,omitempty"`
	Counts      map[string]int `json:"counts" doc:"Units per phase"`
	Total       int            `json:"total" minimum:"0"`
	Words       string         `json:"words" doc:"The batch in a sentence, such as 12 of 43 movies switched"`
	CanRetry    bool           `json:"canRetry" doc:"A unit failed"`
	CanCancel   bool           `json:"canCancel" doc:"A unit hasn't switched yet"`
	CanMoveBack bool           `json:"canMoveBack" doc:"The batch finished, and a unit it moved hasn't moved again since"`
}

func batchOf(b moves.BatchView) batchRow {
	out := batchRow{
		ID: b.ID.String(), Origin: string(b.Origin), CreatedAt: b.CreatedAt, FinishedAt: b.FinishedAt, CanceledAt: b.CanceledAt,
		Counts: make(map[string]int, len(b.Counts)), Total: b.Total, Words: b.Words, CanRetry: b.CanRetry, CanCancel: b.CanCancel, CanMoveBack: b.CanBack,
	}
	for p, n := range b.Counts {
		out.Counts[string(p)] = n
	}
	return out
}

type listInput struct {
	After string `query:"after" format:"uuid" doc:"The last batch of the page before"`
	Limit int    `query:"limit" minimum:"1" maximum:"100" default:"50"`
}

type batchList struct {
	Batches []batchRow `json:"batches" nullable:"false" doc:"Open ones first, then newest first"`
	Next    string     `json:"next,omitempty" doc:"The after of the next page; absent on the last"`
}

type listOutput struct{ Body batchList }

type batchInput struct {
	BatchID string `path:"batchId" format:"uuid"`
}

type getInput struct {
	BatchID string `path:"batchId" format:"uuid"`
	After   int    `query:"after" minimum:"-1" default:"-1" doc:"The last unit's position on the page before; -1 for the first page"`
	Limit   int    `query:"limit" minimum:"1" maximum:"500" default:"100"`
}

type batchPage struct {
	Batch batchRow   `json:"batch"`
	Units []moveUnit `json:"units" nullable:"false" doc:"In the batch's order"`
	Next  *int       `json:"next,omitempty" doc:"The after of the next page; absent on the last"`
}

type getOutput struct{ Body batchPage }

type canceledMoves struct {
	Canceled int    `json:"canceled" minimum:"0" doc:"Units canceled before their switch"`
	Switched int    `json:"switched" minimum:"0" doc:"Units that had already switched, which stay"`
	Words    string `json:"words" doc:"The toast, such as Canceled 31 Moves. 12 movies already switched stay on den."`
}

type cancelOutput struct{ Body canceledMoves }

func uuidOf(s string) (storage.UUID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return storage.UUID{}, moveNotFound
	}
	return u, nil
}

type handlers struct{ n Mover }

func (h handlers) preview(ctx context.Context, _ identity.Admin, in *previewInput) (*previewOutput, error) {
	ask, err := askOf(in.Body)
	if err != nil {
		return nil, err
	}
	if len(ask.Works) == 0 && len(ask.Roots) == 0 && len(ask.Torrents) == 0 && len(ask.Clients) == 0 {
		return nil, emptyAsk
	}
	m := open(h.n)
	p, err := m.Preview(ctx, ask)
	if err != nil {
		return nil, answer(err)
	}
	return h.answerPlan(ctx, m, p)
}

func askOf(b moveAsk) (moves.Ask, error) {
	o := b.Options
	ask := moves.Ask{
		Origin:  moves.Origin(b.Origin),
		Options: moves.Options{IncludeLinkedTorrents: o.IncludeLinkedTorrents, RepointRouting: o.RepointRouting, DeleteEmptiedRoots: o.DeleteEmptiedRoots},
	}
	var err error
	if ask.Works, err = each(b.Works, workToOf); err != nil {
		return moves.Ask{}, err
	}
	if ask.Roots, err = each(b.Roots, rootToOf); err != nil {
		return moves.Ask{}, err
	}
	if ask.Torrents, err = each(b.Torrents, torrentToOf); err != nil {
		return moves.Ask{}, err
	}
	if ask.Clients, err = each(b.Clients, clientToOf); err != nil {
		return moves.Ask{}, err
	}
	return ask, nil
}

func each[In, Out any](ins []In, of func(In) (Out, error)) ([]Out, error) {
	var out []Out
	for _, in := range ins {
		o, err := of(in)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

func workToOf(w workTo) (moves.WorkTo, error) {
	work, err := storage.ParseUUID(w.WorkID)
	if err != nil {
		return moves.WorkTo{}, workNotFound
	}
	root, err := storage.ParseUUID(w.RootID)
	if err != nil {
		return moves.WorkTo{}, rootNotFound
	}
	return moves.WorkTo{Work: works.WorkID{UUID: work}, To: roots.ID{UUID: root}}, nil
}

func rootToOf(rt rootTo) (moves.RootTo, error) {
	from, err := storage.ParseUUID(rt.RootID)
	if err != nil {
		return moves.RootTo{}, rootNotFound
	}
	to, err := storage.ParseUUID(rt.To)
	if err != nil {
		return moves.RootTo{}, rootNotFound
	}
	return moves.RootTo{Root: roots.ID{UUID: from}, To: roots.ID{UUID: to}}, nil
}

func clientToOf(c clientTo) (moves.ClientTo, error) {
	from, err := sideOf(side{NodeID: c.NodeID, ClientID: c.ClientID})
	if err != nil {
		return moves.ClientTo{}, err
	}
	to, err := sideOf(c.To)
	return moves.ClientTo{From: from, To: to}, err
}

func torrentToOf(t torrentTo) (moves.TorrentTo, error) {
	id, err := storage.ParseUUID(t.TorrentID)
	if err != nil {
		return moves.TorrentTo{}, torrentNotFound
	}
	to, err := sideOf(side{NodeID: t.NodeID, ClientID: t.ClientID})
	return moves.TorrentTo{Torrent: downloadclients.ID{UUID: id}, To: to}, err
}

func sideOf(s side) (moves.Side, error) {
	node, err := storage.ParseUUID(s.NodeID)
	if err != nil {
		return moves.Side{}, clientNotFound
	}
	out := moves.Side{Node: role.NodeID{UUID: node}}
	if s.ClientID != "" {
		c, err := storage.ParseUUID(s.ClientID)
		if err != nil {
			return moves.Side{}, clientNotFound
		}
		out.Client = &downloadclients.ClientID{UUID: c}
	}
	return out, nil
}

func (h handlers) answerPlan(ctx context.Context, m *moves.Moves, p moves.Plan) (*previewOutput, error) {
	named, err := m.PlanNames(ctx, p)
	if err != nil {
		return nil, err
	}
	return &previewOutput{Body: previewOf(p, named)}, nil
}

func (h handlers) start(ctx context.Context, by identity.Admin, in *startInput) (*startOutput, error) {
	s, err := open(h.n).Start(ctx, by, moves.Token(in.Body.Token), in.Confirm)
	if err != nil {
		return nil, answer(err)
	}
	out := &startOutput{}
	out.Body.BatchID, out.Body.Started, out.Body.Dropped = s.Batch.String(), s.Units, make([]dropped, len(s.Dropped))
	for i, d := range s.Dropped {
		out.Body.Dropped[i] = dropped{WorkID: d.Work.String(), Name: d.Label, Words: d.Words}
		if d.Torrent != nil {
			out.Body.Dropped[i].TorrentID = d.Torrent.String()
		}
	}
	return out, nil
}

func (h handlers) list(ctx context.Context, _ identity.Admin, in *listInput) (*listOutput, error) {
	var after moves.BatchID
	if in.After != "" {
		u, err := uuidOf(in.After)
		if err != nil {
			return nil, err
		}
		after = moves.BatchID{UUID: u}
	}
	bs, err := open(h.n).List(ctx, after, in.Limit)
	if err != nil {
		return nil, answer(err)
	}
	out := &listOutput{}
	out.Body.Batches = make([]batchRow, len(bs))
	for i, b := range bs {
		out.Body.Batches[i] = batchOf(b)
	}
	if len(bs) == in.Limit {
		out.Body.Next = bs[len(bs)-1].ID.String()
	}
	return out, nil
}

func (h handlers) get(ctx context.Context, _ identity.Admin, in *getInput) (*getOutput, error) {
	id, err := uuidOf(in.BatchID)
	if err != nil {
		return nil, err
	}
	b, vs, err := open(h.n).Get(ctx, moves.BatchID{UUID: id}, in.After, in.Limit)
	if err != nil {
		return nil, answer(err)
	}
	out := &getOutput{}
	out.Body.Batch, out.Body.Units = batchOf(b), make([]moveUnit, len(vs))
	for i, v := range vs {
		out.Body.Units[i] = unitOf(v)
	}
	if len(vs) == in.Limit {
		out.Body.Next = new(in.After + len(vs))
	}
	return out, nil
}

func (h handlers) cancelBatch(ctx context.Context, _ identity.Admin, in *batchInput) (*cancelOutput, error) {
	id, err := uuidOf(in.BatchID)
	if err != nil {
		return nil, err
	}
	m := open(h.n)
	canceled, switched, err := m.CancelBatch(ctx, moves.BatchID{UUID: id})
	if err != nil {
		return nil, answer(err)
	}
	words, err := m.CanceledWords(ctx, moves.BatchID{UUID: id}, canceled)
	if err != nil {
		return nil, err
	}
	out := &cancelOutput{}
	out.Body.Canceled, out.Body.Switched, out.Body.Words = canceled, switched, words
	return out, nil
}

// unitAct runs act on the unit in names, then answers its view.
func (h handlers) unitAct(ctx context.Context, in *unitInput, act func(m *moves.Moves, id moves.UnitID) error) (*unitOutput, error) {
	u, err := uuidOf(in.UnitID)
	if err != nil {
		return nil, err
	}
	m, id := open(h.n), moves.UnitID{UUID: u}
	if err := act(m, id); err != nil {
		return nil, answer(err)
	}
	v, err := m.Unit(ctx, id)
	if err != nil {
		return nil, answer(err)
	}
	return &unitOutput{Body: unitOf(v)}, nil
}

func (h handlers) back(ctx context.Context, _ identity.Admin, in *batchInput) (*previewOutput, error) {
	id, err := uuidOf(in.BatchID)
	if err != nil {
		return nil, err
	}
	m := open(h.n)
	p, err := m.Back(ctx, moves.BatchID{UUID: id})
	if err != nil {
		return nil, answer(err)
	}
	return h.answerPlan(ctx, m, p)
}

func Routes(r *api.Router, n Mover) {
	o := r.Owner("moves", api.Admin)
	h := handlers{n}

	api.Register(o, api.Admin, api.Op{
		ID: "previewMoves", Method: http.MethodPost, Path: "/moves/preview",
		Summary: "Preview moving Works to other library roots, or torrents to other Download Clients: a verdict for each, and a token to start it with",
		Status:  http.StatusOK, Errors: []*api.Code{workNotFound, rootNotFound, torrentNotFound, clientNotFound, emptyAsk}, Gzip: true,
	}, h.preview)

	api.Register(o, api.Admin, api.Op{
		ID: "startMoves", Method: http.MethodPost, Path: "/moves",
		Summary: "Start a previewed Move. A Start sent again returns the same batch",
		Status:  http.StatusAccepted, Errors: []*api.Code{planChanged, planExpired, nothingToMove, api.Reconciling},
	}, h.start)

	api.Register(o, api.Admin, api.Op{
		ID: "listMoves", Method: http.MethodGet, Path: "/moves",
		Summary: "List the Moves' batches, open ones first",
		Status:  http.StatusOK, Errors: []*api.Code{moveNotFound},
	}, h.list)

	api.Register(o, api.Admin, api.Op{
		ID: "getMove", Method: http.MethodGet, Path: "/moves/{batchId}",
		Summary: "Get a Move's batch with a page of its units",
		Status:  http.StatusOK, Errors: []*api.Code{moveNotFound},
	}, h.get)

	api.Register(o, api.Admin, api.Op{
		ID: "cancelMoves", Method: http.MethodPost, Path: "/moves/{batchId}/cancel",
		Summary: "Cancel every unit of a Move that hasn't switched. Each one removes what it placed",
		Status:  http.StatusOK, Errors: []*api.Code{moveNotFound},
	}, h.cancelBatch)

	api.Register(o, api.Admin, api.Op{
		ID: "previewMoveBack", Method: http.MethodGet, Path: "/moves/{batchId}/back",
		Summary: "Preview moving a Move's Works back to the roots they left, and its torrents back to the Download Clients they left",
		Status:  http.StatusOK, Errors: []*api.Code{moveNotFound}, Gzip: true,
	}, h.back)

	api.Register(o, api.Admin, api.Op{
		ID: "retryMoveUnit", Method: http.MethodPost, Path: "/moves/units/{unitId}/retry",
		Summary: "Queue a failed unit of a Move again",
		Status:  http.StatusOK, Errors: []*api.Code{moveNotFound, unitNotFailed},
	}, func(ctx context.Context, _ identity.Admin, in *unitInput) (*unitOutput, error) {
		return h.unitAct(ctx, in, func(m *moves.Moves, id moves.UnitID) error { return m.Retry(ctx, id) })
	})

	api.Register(o, api.Admin, api.Op{
		ID: "cancelMoveUnit", Method: http.MethodPost, Path: "/moves/units/{unitId}/cancel",
		Summary: "Cancel a unit of a Move before its switch. It removes what it placed",
		Status:  http.StatusOK, Errors: []*api.Code{moveNotFound, unitSwitched},
	}, func(ctx context.Context, _ identity.Admin, in *unitInput) (*unitOutput, error) {
		return h.unitAct(ctx, in, func(m *moves.Moves, id moves.UnitID) error { return m.Cancel(ctx, id) })
	})

	api.Register(o, api.Admin, api.Op{
		ID: "switchMoveUnitNow", Method: http.MethodPost, Path: "/moves/units/{unitId}/switch-now",
		Summary: "Switch a unit waiting for playback or a Media Server without waiting longer",
		Status:  http.StatusOK, Errors: []*api.Code{moveNotFound, switchNotWaiting},
	}, func(ctx context.Context, _ identity.Admin, in *unitInput) (*unitOutput, error) {
		return h.unitAct(ctx, in, func(m *moves.Moves, id moves.UnitID) error { return m.SwitchNow(ctx, id) })
	})

	api.Register(o, api.Admin, api.Op{
		ID: "getMoveSettings", Method: http.MethodGet, Path: "/moves/settings",
		Summary: "Get the Moves settings of Settings > Library roots", Status: http.StatusOK,
	}, h.getSettings)
	api.Register(o, api.Admin, api.Op{
		ID: "patchMoveSettings", Method: http.MethodPatch, Path: "/moves/settings",
		Summary: "Change the Moves settings by key, all or none", Status: http.StatusOK,
		Errors: []*api.Code{settingapi.ErrOutOfRange, settingapi.ErrUnknown},
	}, h.patchSettings)
}
