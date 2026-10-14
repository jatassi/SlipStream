package world

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/world/internal/control"
)

// Controls is every write a harness makes to a run. *Run implements it in
// process (worldtest, and the supervisor behind the control socket);
// *Remote implements it over the socket.
type Controls interface {
	SetFault(ctx context.Context, r fault.Rule) (events.Record, error)
	// ClearFault removes the rule with id, or every rule for "".
	ClearFault(ctx context.Context, id string) (events.Record, error)
	// Cue plays the cue name a fake declares, with args decoded strictly
	// into the cue's own arguments.
	Cue(ctx context.Context, name string, args json.RawMessage) (events.Record, error)
	StartStep(ctx context.Context, s StepStart) (Step, error)
	EndStep(ctx context.Context, n int, e StepEnd) (events.Record, error)
	Stop(ctx context.Context) (StopReport, error)
}

var (
	_ Controls = (*Run)(nil)
	_ Controls = (*Remote)(nil)
)

// StepStart is one journaled step: a fakeworld verb with its arguments
// after the run dir, unexpanded, or a note a Go test makes.
type StepStart struct {
	Verb string   `json:"verb,omitempty"`
	Args []string `json:"args,omitempty"`
	Note string   `json:"note,omitempty"`
}

// Step is a started step: its number in the run's journal and the seq of
// its step.start record.
type Step struct {
	N   int    `json:"n"`
	Seq uint64 `json:"seq"`
}

type StepEnd struct {
	Exit int `json:"exit"`
	// Files maps a name such as stdout to a path relative to the run dir.
	Files map[string]string `json:"files,omitempty"`
}

type stepStarted struct {
	N int `json:"n"`
	StepStart
}

type stepEnded struct {
	N    int `json:"n"`
	Exit int `json:"exit"`
}

type cleared struct {
	ID      string `json:"id,omitempty"`
	Removed bool   `json:"removed"`
}

type stepKey struct{}

// InStep is ctx for a control that step n sends; the control's record
// names the step.
func InStep(ctx context.Context, n int) context.Context {
	return context.WithValue(ctx, stepKey{}, n)
}

func stepOf(ctx context.Context) int {
	n, _ := ctx.Value(stepKey{}).(int)
	return n
}

// stepData is v's JSON object with the step ctx carries, if any.
func stepData(ctx context.Context, v any) json.RawMessage {
	if n := stepOf(ctx); n > 0 {
		return withStep(data(v), n)
	}
	return data(v)
}

// withStep is the JSON object obj with its step field set to n.
func withStep(obj json.RawMessage, n int) json.RawMessage {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(obj, &fields); err != nil {
		panic("world: " + err.Error())
	}
	fields["step"] = data(n)
	return data(fields)
}

// allRules is the id DELETE /v1/faults/{id} takes for every rule.
const allRules = "_"

func (r *Run) SetFault(ctx context.Context, rule fault.Rule) (events.Record, error) {
	if rule.ID == "" || rule.ID == allRules {
		return events.Record{}, fmt.Errorf("a rule needs an id, and %q clears every rule", allRules)
	}
	return r.events.Apply(func() (events.Record, []events.Record, error) {
		fired, err := r.faults.Set(rule)
		if err != nil {
			return events.Record{}, nil, err
		}
		return events.Record{Kind: events.FaultSet, Node: events.World, Data: stepData(ctx, rule)}, firedRecords(fired), nil
	})
}

func (r *Run) ClearFault(ctx context.Context, id string) (events.Record, error) {
	return r.events.Apply(func() (events.Record, []events.Record, error) {
		removed := r.faults.Clear(id)
		return events.Record{Kind: events.FaultCleared, Node: events.World, Data: stepData(ctx, cleared{ID: id, Removed: removed})}, nil, nil
	})
}

type cued struct {
	Name   string          `json:"name"`
	Fake   string          `json:"fake"`
	Args   json.RawMessage `json:"args,omitempty"`
	Result any             `json:"result,omitempty"`
}

func (r *Run) Cue(ctx context.Context, name string, args json.RawMessage) (events.Record, error) {
	i := slices.IndexFunc(r.fakes, func(f *fakeRun) bool {
		return slices.ContainsFunc(f.spec.Cues, func(c fake.CueSpec) bool { return c.Name == name })
	})
	if i < 0 {
		return events.Record{}, fmt.Errorf("no fake declares cue %s", name)
	}
	f := r.fakes[i]
	result, err := f.env.Cue(ctx, name, args)
	if err != nil {
		return events.Record{}, err
	}
	if string(args) == "null" {
		args = nil
	}
	return r.events.Append(events.Record{Kind: events.Cue, Node: events.World, Data: stepData(ctx, cued{Name: name, Fake: f.spec.Name, Args: args, Result: result})})
}

func (r *Run) StartStep(_ context.Context, s StepStart) (Step, error) {
	if (s.Verb == "") == (s.Note == "") || (s.Note != "" && len(s.Args) > 0) {
		return Step{}, errors.New("a step is a verb with its arguments or a note")
	}
	var n int
	rec, err := r.events.Apply(func() (events.Record, []events.Record, error) {
		r.lastStep++
		n = r.lastStep
		r.openSteps[n] = true
		return events.Record{Kind: events.StepStart, Node: events.World, Data: data(stepStarted{N: n, StepStart: s})}, nil, nil
	})
	if err != nil {
		return Step{}, err
	}
	return Step{N: n, Seq: rec.Seq}, nil
}

func (r *Run) EndStep(_ context.Context, n int, e StepEnd) (events.Record, error) {
	for name, path := range e.Files {
		if !filepath.IsLocal(path) {
			return events.Record{}, fmt.Errorf("step %d's %s file %s is not inside the run dir", n, name, path)
		}
	}
	return r.events.Apply(func() (events.Record, []events.Record, error) {
		if !r.openSteps[n] {
			return events.Record{}, nil, fmt.Errorf("step %d is not open", n)
		}
		delete(r.openSteps, n)
		return events.Record{Kind: events.StepEnd, Node: events.World, Data: data(stepEnded{N: n, Exit: e.Exit}), Files: e.Files}, nil, nil
	})
}

// controlHandler serves c's controls on the control API's fixed routes.
// A control's record names the step in the Fakeworld-Step header.
func controlHandler(c Controls) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("POST /v1/stop", route(func(ctx context.Context, _ *http.Request, _ struct{}) (StopReport, error) {
		return c.Stop(ctx)
	}))
	mux.Handle("POST /v1/faults", route(func(ctx context.Context, _ *http.Request, rule fault.Rule) (events.Record, error) {
		return c.SetFault(ctx, rule)
	}))
	mux.Handle("DELETE /v1/faults/{id}", route(func(ctx context.Context, r *http.Request, _ struct{}) (events.Record, error) {
		id := r.PathValue("id")
		if id == allRules {
			id = ""
		}
		return c.ClearFault(ctx, id)
	}))
	mux.Handle("POST /v1/cues/{name}", route(func(ctx context.Context, r *http.Request, args json.RawMessage) (events.Record, error) {
		return c.Cue(ctx, r.PathValue("name"), args)
	}))
	mux.Handle("POST /v1/steps", route(func(ctx context.Context, _ *http.Request, s StepStart) (Step, error) {
		return c.StartStep(ctx, s)
	}))
	mux.Handle("POST /v1/steps/{n}/end", route(func(ctx context.Context, r *http.Request, e StepEnd) (events.Record, error) {
		n, err := strconv.Atoi(r.PathValue("n"))
		if err != nil {
			return events.Record{}, fmt.Errorf("step %q is not a number", r.PathValue("n"))
		}
		return c.EndStep(ctx, n, e)
	}))
	return mux
}

const stepHeader = "Fakeworld-Step"

type controlError struct {
	Error string `json:"error"`
}

// route decodes a request's body strictly into In and answers the
// control's result. A client half-closes after its request, which
// cancels the request's context, so the control runs without it.
func route[In, Out any](do func(context.Context, *http.Request, In) (Out, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		answer := func(status int, v any) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write(marshal(v))
		}
		var in In
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			answer(http.StatusBadRequest, controlError{err.Error()})
			return
		}
		ctx := context.WithoutCancel(r.Context())
		if h := r.Header.Get(stepHeader); h != "" {
			n, err := strconv.Atoi(h)
			if err != nil || n < 1 {
				answer(http.StatusBadRequest, controlError{stepHeader + " " + strconv.Quote(h) + " is not a step number"})
				return
			}
			ctx = InStep(ctx, n)
		}
		out, err := do(ctx, r, in)
		if err != nil {
			answer(http.StatusConflict, controlError{err.Error()})
			return
		}
		answer(http.StatusOK, out)
	})
}

// Remote is a run's controls over its control socket.
type Remote struct {
	socket string
}

// ErrNotLive is a run dir with no manifest.json: a run stopped, not yet
// ready, or a downloaded bundle.
var ErrNotLive = errors.New("no run is live")

// Dial reaches the live run in runDir through the socket its manifest
// names.
func Dial(runDir string) (*Remote, error) {
	m, err := ReadManifest(runDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w in %s: it has no manifest.json", ErrNotLive, runDir)
	}
	if err != nil {
		return nil, err
	}
	return &Remote{socket: m.Supervisor.Control}, nil
}

func (c *Remote) SetFault(ctx context.Context, r fault.Rule) (events.Record, error) {
	return send[events.Record](ctx, c, http.MethodPost, "/v1/faults", r)
}

func (c *Remote) ClearFault(ctx context.Context, id string) (events.Record, error) {
	if id == "" {
		id = allRules
	}
	return send[events.Record](ctx, c, http.MethodDelete, "/v1/faults/"+url.PathEscape(id), struct{}{})
}

func (c *Remote) Cue(ctx context.Context, name string, args json.RawMessage) (events.Record, error) {
	return send[events.Record](ctx, c, http.MethodPost, "/v1/cues/"+url.PathEscape(name), args)
}

func (c *Remote) StartStep(ctx context.Context, s StepStart) (Step, error) {
	return send[Step](ctx, c, http.MethodPost, "/v1/steps", s)
}

func (c *Remote) EndStep(ctx context.Context, n int, e StepEnd) (events.Record, error) {
	return send[events.Record](ctx, c, http.MethodPost, "/v1/steps/"+strconv.Itoa(n)+"/end", e)
}

func (c *Remote) Stop(ctx context.Context) (StopReport, error) {
	report, err := send[StopReport](ctx, c, http.MethodPost, "/v1/stop", struct{}{})
	if err != nil {
		return StopReport{}, err
	}
	return report, report.Err()
}

func send[Out any](ctx context.Context, c *Remote, method, path string, in any) (Out, error) {
	var out Out
	err := c.call(ctx, method, path, in, &out)
	return out, err
}

func (c *Remote) call(ctx context.Context, method, path string, in, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, "http://fakeworld"+path, bytes.NewReader(marshal(in)))
	if err != nil {
		return fmt.Errorf("control %s %s: %w", method, path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if n := stepOf(ctx); n > 0 {
		req.Header.Set(stepHeader, strconv.Itoa(n))
	}
	status, body, err := control.Do(ctx, c.socket, req)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		var e controlError
		if json.Unmarshal(body, &e) != nil || e.Error == "" {
			return fmt.Errorf("control %s %s: %d %s", method, path, status, body)
		}
		return errors.New(e.Error)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("control %s %s: %w", method, path, err)
	}
	return nil
}

// LiveFaults reads the run in runDir's event log for the fault.set
// records of the rules still set, in the order they were set. A rule
// that finished its Times stays set until it is cleared.
func LiveFaults(runDir string) ([]events.Record, error) {
	recs, err := events.Read(runDir)
	if err != nil {
		return nil, err
	}
	type set struct {
		id  string
		rec events.Record
	}
	var live []set
	for _, rec := range recs {
		if rec.Kind != events.FaultSet && rec.Kind != events.FaultCleared {
			continue
		}
		var d struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(rec.Data, &d); err != nil {
			return nil, fmt.Errorf("%s %d: %w", rec.Kind, rec.Seq, err)
		}
		all := rec.Kind == events.FaultCleared && d.ID == ""
		live = slices.DeleteFunc(live, func(s set) bool { return all || s.id == d.ID })
		if rec.Kind == events.FaultSet {
			live = append(live, set{d.ID, rec})
		}
	}
	out := make([]events.Record, len(live))
	for i, s := range live {
		out[i] = s.rec
	}
	return out, nil
}
