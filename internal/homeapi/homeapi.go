// Package homeapi is Home's read and its one setting: the moving Grabs,
// what needs an Admin, and the newest finished Imports.
package homeapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/downloadclientsapi"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/filesapi"
	"github.com/jatassi/slipstream/internal/grabsapi"
	"github.com/jatassi/slipstream/internal/home"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/works"
)

type Home interface {
	// Home is Home's service, false unless the Node is a Controller.
	Home() (*home.Service, bool)
	// Minter signs the artwork URLs the answer carries, false unless the
	// Node is a Controller.
	Minter() (artwork.Minter, bool)
}

func open(n Home) (*home.Service, artwork.Minter) {
	h, ok := n.Home()
	m, minted := n.Minter()
	if !ok || !minted {
		panic("homeapi: an operation ran on a Node that is no Controller")
	}
	return h, m
}

type summary struct {
	Jobs     jobs     `json:"jobs"`
	NeedsYou needsYou `json:"needsYou"`
	Recent   recent   `json:"recent"`
}

type jobs struct {
	Rows   []job `json:"rows" nullable:"false" doc:"The moving Grabs, nearest Available first, at most Rows on Home"`
	Moving int   `json:"moving" minimum:"0" doc:"Every moving Grab"`
}

type job struct {
	Grab    grabsapi.Grab                               `json:"grab"`
	Work    work                                        `json:"work"`
	Torrent api.Nullable[downloadclientsapi.TorrentRow] `json:"torrent" doc:"Its Torrent as the Torrents view lists it, without queuePosition; null until the .torrent is fetched"`
}

type needsYou struct {
	Rows    []need     `json:"rows" nullable:"false" doc:"Newest first, at most Rows on Home"`
	Moves   []moveNeed `json:"moves" nullable:"false" doc:"Moves that failed, or waited past the moves.needsYouAfter setting, newest first, at most Rows on Home"`
	Failed  int        `json:"failed" minimum:"0" maximum:"100" doc:"Failed Grabs that need attention, counted up to 100; 100 means 100 or more"`
	Waiting int        `json:"waiting" minimum:"0" doc:"Grabs that wait on an Admin"`
}

type need struct {
	Grab grabsapi.Grab                      `json:"grab"`
	Work work                               `json:"work"`
	Wait api.Nullable[filesapi.WaitingNeed] `json:"wait" doc:"What it waits on, as the Work page words it; null for a failed Grab"`
}

type moveNeed struct {
	UnitID    string     `json:"unitId" format:"uuid"`
	BatchID   string     `json:"batchId" format:"uuid"`
	Kind      string     `json:"kind" enum:"failed,waiting"`
	Work      work       `json:"work"`
	Words     string     `json:"words" doc:"What happened and what to do, such as Heat (1995) has waited 1 hour to switch to den. Someone is playing it on a Media Server."`
	Since     *time.Time `json:"since,omitempty" doc:"When it began to wait; absent for a failed Move"`
	SwitchNow bool       `json:"switchNow" doc:"Whether Switch now can end its wait"`
}

type recent struct {
	Rows  []arrival `json:"rows" nullable:"false" doc:"Finished Imports, newest first, at most Rows on Home"`
	Today today     `json:"today"`
}

type arrival struct {
	ImportID string    `json:"importId" format:"uuid"`
	Work     work      `json:"work"`
	At       time.Time `json:"at" doc:"When SlipStream imported its last file"`
	Words    string    `json:"words" doc:"What it did, such as Imported 10 episodes · 2 upgrades"`
}

type today struct {
	Imported int `json:"imported" minimum:"0" doc:"Finished Imports, other than upgrades, that imported a file at or after the time in the since parameter"`
	Upgraded int `json:"upgraded" minimum:"0" doc:"Finished Imports that imported a file at or after the time in the since parameter and whose every file replaced one"`
}

type work struct {
	ID          string  `json:"id" format:"uuid"`
	ContentType string  `json:"contentType"`
	Label       string  `json:"label"`
	PosterURL   *string `json:"posterUrl" doc:"The artwork route without a size; null when it has none"`
	BackdropURL *string `json:"backdropUrl" doc:"The artwork route without a size; null when it has none"`
	Failed      bool    `json:"failed" doc:"Its status is failed, so it never tints Home"`
}

type summaryInput struct {
	Since time.Time `query:"since" required:"true" doc:"The browser's local midnight, from which today's totals count"`
}

type summaryOutput struct{ Body summary }

type settings struct {
	Settings []api.OneOf[settingapi.Field] `json:"settings" nullable:"false" doc:"Rows on Home"`
}

type settingsOutput struct{ Body settings }

type settingsInput struct {
	Body struct {
		Values map[string]json.RawMessage `json:"values" doc:"Each setting's new value by key, all or none; null resets one to its default"`
	}
}

func Routes(r *api.Router, n Home) {
	o := r.Owner("home", api.Admin)

	api.Register(o, api.Admin, api.Op{
		ID:      "getHome",
		Method:  http.MethodGet,
		Path:    "/home",
		Summary: "Get Home: the moving Grabs, what needs attention and the newest finished Imports",
		Status:  http.StatusOK,
	}, func(ctx context.Context, by identity.Admin, in *summaryInput) (*summaryOutput, error) {
		h, m := open(n)
		s, err := h.Read(ctx, by, in.Since)
		if err != nil {
			return nil, err
		}
		return &summaryOutput{Body: summaryOf(m, s)}, nil
	})

	api.Register(o, api.Admin, api.Op{
		ID:      "getHomeSettings",
		Method:  http.MethodGet,
		Path:    "/home-settings",
		Summary: "Get Home's settings",
		Status:  http.StatusOK,
	}, func(ctx context.Context, _ identity.Admin, _ *struct{}) (*settingsOutput, error) {
		h, _ := open(n)
		return settingsOf(ctx, h)
	})

	api.Register(o, api.Admin, api.Op{
		ID:      "updateHomeSettings",
		Method:  http.MethodPatch,
		Path:    "/home-settings",
		Summary: "Change Home's settings by key, all or none",
		Status:  http.StatusOK,
		Errors:  []*api.Code{settingapi.ErrOutOfRange, settingapi.ErrUnknown},
	}, func(ctx context.Context, _ identity.Admin, in *settingsInput) (*settingsOutput, error) {
		h, _ := open(n)
		if err := h.Apply(ctx, in.Body.Values); err != nil {
			return nil, settingapi.Refused(err)
		}
		return settingsOf(ctx, h)
	})
}

func settingsOf(ctx context.Context, h *home.Service) (*settingsOutput, error) {
	fields, err := h.Fields(ctx)
	if err != nil {
		return nil, err
	}
	return &settingsOutput{Body: settings{Settings: settingapi.FieldsOf(fields)}}, nil
}

func summaryOf(m artwork.Minter, s home.Summary) summary {
	out := summary{
		Jobs:     jobs{Rows: make([]job, len(s.Jobs)), Moving: s.Moving},
		NeedsYou: needsYou{Rows: make([]need, len(s.Needs)), Failed: s.Failed, Waiting: s.Waiting},
		Recent:   recent{Rows: make([]arrival, len(s.Recent.Arrivals)), Today: today{Imported: s.Recent.Imported, Upgraded: s.Recent.Upgraded}},
	}
	for i, v := range s.Jobs {
		out.Jobs.Rows[i] = job{Grab: grabsapi.GrabOf(v), Work: workOf(m, v.WorkID, s.Works)}
		if v.Torrent == nil {
			continue
		}
		if row, ok := s.Torrents[v.Torrent.ID]; ok {
			t := downloadclientsapi.RowOf(row)
			// A read of a few Torrents counts only those, so its queue
			// places would be wrong.
			t.InLine = 0
			out.Jobs.Rows[i].Torrent = api.Some(t)
		}
	}
	for i, n := range s.Needs {
		out.NeedsYou.Rows[i] = need{Grab: grabsapi.GrabOf(n.View), Work: workOf(m, n.WorkID, s.Works)}
		if w := n.Wait; w != nil {
			out.NeedsYou.Rows[i].Wait = api.Some(filesapi.WaitingOf(files.Need{
				Kind: files.NeedWaiting, Grab: n.ID.UUID, Code: w.Code, ClientError: w.ClientError, Hold: w.Hold,
			}))
		}
	}
	out.NeedsYou.Moves = make([]moveNeed, len(s.Moves))
	for i, n := range s.Moves {
		row := moveNeed{UnitID: n.Unit.String(), BatchID: n.Batch.String(), Kind: "waiting", Work: workOf(m, n.Work, s.Works), Words: n.Words, SwitchNow: n.SwitchNow}
		if n.Failure != nil {
			row.Kind = "failed"
		} else {
			row.Since = n.Since
		}
		out.NeedsYou.Moves[i] = row
	}
	for i, a := range s.Recent.Arrivals {
		out.Recent.Rows[i] = arrival{ImportID: a.Import.String(), Work: workOf(m, a.Work, s.Works), At: a.At, Words: a.Words}
	}
	return out
}

func workOf(m artwork.Minter, id works.WorkID, ws map[works.WorkID]home.Work) work {
	w := ws[id]
	out := work{ID: id.String(), ContentType: string(w.ContentType), Label: w.Label, Failed: w.Failed}
	if !w.Poster.IsZero() {
		out.PosterURL = new(m.URL(w.Poster))
	}
	if !w.Backdrop.IsZero() {
		out.BackdropURL = new(m.URL(w.Backdrop))
	}
	return out
}
