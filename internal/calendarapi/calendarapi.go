// Package calendarapi is the Calendar's one read: the release dates in a
// span of days, with whether the library has each.
package calendarapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/calendar"
	"github.com/jatassi/slipstream/internal/filesapi"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/works"
)

type Calendar interface {
	// Calendar is the Calendar, false unless the Node is a Controller.
	Calendar() (*calendar.Calendar, bool)
	// Minter signs the poster URLs the answer carries, false unless the
	// Node is a Controller.
	Minter() (artwork.Minter, bool)
}

func open(n Calendar) (*calendar.Calendar, artwork.Minter) {
	c, ok := n.Calendar()
	m, minted := n.Minter()
	if !ok || !minted {
		panic("calendarapi: an operation ran on a Node that is no Controller")
	}
	return c, m
}

type calendarInput struct {
	From        string   `query:"from" required:"true" format:"date" doc:"The first day"`
	To          string   `query:"to" required:"true" format:"date" doc:"The day after the last day, at most 42 days after from"`
	ContentType string   `query:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$" doc:"Left out, every Content Type that is on"`
	Kinds       []string `query:"kinds" doc:"A comma list of the dated kinds' keys GET /content-types lists; left out, every kind"`
	Unmonitored bool     `query:"unmonitored" doc:"Also list unmonitored Items, each in a row of their own"`
}

type answer struct {
	Today     string                 `json:"today" format:"date" doc:"The Controller's day"`
	NextDayAt time.Time              `json:"nextDayAt" doc:"When the Controller's next day starts"`
	Sentence  api.Nullable[sentence] `json:"sentence" doc:"What needs attention, else what comes out in the next 7 days; null when neither has anything"`
	Days      []day                  `json:"days" nullable:"false" doc:"Only the days with rows, in order"`
}

type sentence struct {
	NeedsYou bool   `json:"needsYou" doc:"It names Items that are out and still missing"`
	Words    string `json:"words"`
}

type day struct {
	Day  string `json:"day" format:"date"`
	Rows []row  `json:"rows" nullable:"false"`
}

type work struct {
	ID          string  `json:"id" format:"uuid"`
	ContentType string  `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Label       string  `json:"label"`
	PosterURL   *string `json:"posterUrl" doc:"The artwork route for its poster, without a size; null when it has none"`
}

type item struct {
	ID    string `json:"id" format:"uuid"`
	Label string `json:"label" doc:"Such as Episode 1 · Hello, Ms. Cobel"`
}

type row struct {
	Work       work                                   `json:"work"`
	Kind       string                                 `json:"kind" doc:"The dated kind's key"`
	KindLabel  string                                 `json:"kindLabel"`
	Items      []item                                 `json:"items" nullable:"false" doc:"In tree order"`
	ItemsWords string                                 `json:"itemsWords" doc:"The Items with their level, such as Season 2 · Episodes 1 to 10; empty for one Item without a level"`
	State      api.Nullable[api.Enum[filesapi.State]] `json:"state" doc:"The most urgent state among its Items, in the order failed, missing, downloading, upgradable, unreleased, met; null while none of them has a Target"`
	Met        int                                    `json:"met" minimum:"0" doc:"Its Items with a Target that are met"`
	Of         int                                    `json:"of" minimum:"0" doc:"Its Items with a Target"`
	Monitored  bool                                   `json:"monitored" doc:"false only for a row of unmonitored Items, which unmonitored asks for"`
}

type answerOutput struct{ Body answer }

var getCalendarOp = api.Op{
	ID:      "getCalendar",
	Method:  http.MethodGet,
	Path:    "/calendar",
	Summary: "Read release dates in a span of days",
	Status:  http.StatusOK,
}

func Routes(r *api.Router, n Calendar) {
	o := r.Owner("calendar", api.Admin)
	api.Register(o, api.Admin, getCalendarOp, func(ctx context.Context, by identity.Admin, in *calendarInput) (*answerOutput, error) {
		return getCalendar(ctx, n, by, in)
	})
}

func refused(location, message string) error {
	return api.Invalid.Fields(api.FieldError{Location: "query." + location, Message: message})
}

func getCalendar(ctx context.Context, n Calendar, by identity.Admin, in *calendarInput) (*answerOutput, error) {
	q, err := queryOf(in)
	if err != nil {
		return nil, err
	}
	c, m := open(n)
	a, err := c.Read(ctx, by, q)
	var off *calendar.TypeOffError
	var unknown *calendar.UnknownKindError
	switch {
	case errors.Is(err, calendar.ErrUnknownType):
		return nil, refused("contentType", "There's no kind of media called "+in.ContentType+".")
	case errors.As(err, &off):
		return nil, refused("contentType", off.Name+" is turned off. Turn it on under Kinds of media first.")
	case errors.As(err, &unknown):
		return nil, refused("kinds", "There's no release date kind called "+string(unknown.Key)+". Choose one from the Calendar's filter.")
	case err != nil:
		return nil, err
	}
	return &answerOutput{Body: answerOf(m, a)}, nil
}

func queryOf(in *calendarInput) (calendar.Query, error) {
	from, err := calendar.ParseDay(in.From)
	if err != nil {
		return calendar.Query{}, err
	}
	to, err := calendar.ParseDay(in.To)
	if err != nil {
		return calendar.Query{}, err
	}
	span, err := calendar.NewSpan(from, to)
	switch {
	case errors.Is(err, calendar.ErrSpanTooLong):
		return calendar.Query{}, refused("to", "A calendar request can span at most 42 days. Ask for a shorter span.")
	case errors.Is(err, calendar.ErrSpanEmpty):
		return calendar.Query{}, refused("to", "The span has no days. Ask for an end day after the start day.")
	}
	q := calendar.Query{Span: span, Type: works.TypeID(in.ContentType), Unmonitored: in.Unmonitored}
	for _, k := range in.Kinds {
		q.Kinds = append(q.Kinds, works.DatedKey(k))
	}
	return q, nil
}

func answerOf(m artwork.Minter, a calendar.Answer) answer {
	out := answer{Today: a.Today.String(), NextDayAt: a.NextDayAt, Days: make([]day, len(a.Days))}
	if a.Sentence != nil {
		out.Sentence = api.Some(sentence{NeedsYou: a.Sentence.NeedsYou, Words: a.Sentence.Words})
	}
	for i, d := range a.Days {
		out.Days[i] = day{Day: d.Day.String(), Rows: make([]row, len(d.Rows))}
		for j, r := range d.Rows {
			out.Days[i].Rows[j] = rowOf(m, r)
		}
	}
	return out
}

func rowOf(m artwork.Minter, r calendar.Row) row {
	out := row{
		Work: work{ID: r.Work.String(), ContentType: string(r.Type), Label: r.Label},
		Kind: string(r.Kind.Key), KindLabel: r.Kind.Label, Items: make([]item, len(r.Items)), ItemsWords: r.ItemsWords,
		State: filesapi.StatusOf(r.State), Met: r.Met, Of: r.Of, Monitored: r.Monitored,
	}
	if !r.Poster.IsZero() {
		out.Work.PosterURL = new(m.URL(r.Poster))
	}
	for i, it := range r.Items {
		out.Items[i] = item{ID: it.ID.String(), Label: it.Label}
	}
	return out
}
