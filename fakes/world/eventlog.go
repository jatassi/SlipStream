package world

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/world/internal/lifecycle"
)

// world.log keeps only diagnostics: the world's slog lines and a crashed
// supervisor's stderr. What a run did is in its event log.
const worldLogFile = "world.log"

func openWorldLog(runDir string) (*slog.Logger, *os.File, error) {
	f, err := os.OpenFile(filepath.Join(runDir, worldLogFile), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644) //nolint:gosec // world.log is evidence anyone on the machine may read
	if err != nil {
		return nil, nil, fmt.Errorf("open world.log: %w", err)
	}
	return slog.New(slog.NewJSONHandler(f, nil)), f, nil
}

// Origin is where a run's bundle came from, which says how it replays.
type Origin struct {
	Kind OriginKind `json:"kind"`
	// Command is, for a test, the command that runs it again.
	Command string `json:"command,omitempty"`
}

type OriginKind string

const (
	// FromSteps is a run fakeworld start launched; it replays by its
	// journaled steps.
	FromSteps OriginKind = "steps"
	// FromTest is a run a Go test launched; it replays by its Command.
	FromTest OriginKind = "test"
)

type launched struct {
	RunID      string `json:"runId"`
	PID        int    `json:"pid"`
	Control    string `json:"control"`
	ScratchDir string `json:"scratchDir"`
	Scenario   string `json:"scenario"`
	Origin     Origin `json:"origin"`
}

type stopped struct {
	Clean bool   `json:"clean"`
	Error string `json:"error,omitempty"`
}

type fired struct {
	Rule   string       `json:"rule"`
	Target fault.Target `json:"target"`
	Fault  string       `json:"fault"`
	Match  int          `json:"match"`
}

// Spawn is a Node process the world started, as its node.spawned record
// says the moment it starts.
type Spawn struct {
	Node    string
	PID     int
	DataDir string
}

// readLaunch reads the supervisor and the Nodes it spawned from the event
// log of the run in runDir, each Node's latest spawn in the order the
// Nodes first spawned.
func readLaunch(runDir string) (Supervisor, []Spawn, error) {
	recs, err := events.Read(runDir)
	if err != nil {
		return Supervisor{}, nil, err
	}
	if len(recs) == 0 || recs[0].Kind != events.RunLaunched {
		return Supervisor{}, nil, errors.New(filepath.Join(runDir, events.File) + " does not start with " + events.RunLaunched)
	}
	var l launched
	if err := json.Unmarshal(recs[0].Data, &l); err != nil {
		return Supervisor{}, nil, fmt.Errorf("%s: %w", events.RunLaunched, err)
	}
	var spawns []Spawn
	for _, rec := range recs {
		if rec.Kind != events.NodeSpawned {
			continue
		}
		var s lifecycle.Spawned
		if err := json.Unmarshal(rec.Data, &s); err != nil {
			return Supervisor{}, nil, fmt.Errorf("%s: %w", events.NodeSpawned, err)
		}
		spawn := Spawn{Node: rec.Node, PID: s.PID, DataDir: s.DataDir}
		if i := slices.IndexFunc(spawns, func(s Spawn) bool { return s.Node == rec.Node }); i >= 0 {
			spawns[i] = spawn
		} else {
			spawns = append(spawns, spawn)
		}
	}
	return Supervisor{PID: l.PID, Log: filepath.Join(runDir, worldLogFile), Control: l.Control}, spawns, nil
}

// record appends a record of kind about node.
func (r *Run) record(kind, node string, v any) {
	r.appendRecord(events.Record{Kind: kind, Node: node, Data: data(v)})
}

// appendRecord appends rec. A run whose log fails keeps running;
// world.log says why its record is missing.
func (r *Run) appendRecord(rec events.Record) {
	if _, err := r.events.Append(rec); err != nil {
		r.log.Error("event log", "kind", rec.Kind, "err", err.Error())
	}
}

// recordFor appends rec, naming in its data the step ctx carries.
func (r *Run) recordFor(ctx context.Context, rec events.Record) {
	if n := stepOf(ctx); n > 0 {
		rec.Data = withStep(rec.Data, n)
	}
	r.appendRecord(rec)
}

// observe arms the rules waiting on rec. No rule exists before bindFakes
// makes the Plan.
func (r *Run) observe(rec events.Record) []events.Record {
	if r.faults == nil {
		return nil
	}
	return firedRecords(r.faults.Observe(rec))
}

func (r *Run) fired(f fault.Firing) { r.appendRecord(firedRecords([]fault.Firing{f})[0]) }

func firedRecords(fs []fault.Firing) []events.Record {
	recs := make([]events.Record, len(fs))
	for i, f := range fs {
		recs[i] = events.Record{
			Kind: events.FaultFired,
			Node: f.Front.Origin,
			Data: data(fired{Rule: f.Rule, Target: f.Front.Target, Fault: f.Fault.String(), Match: f.Match}),
		}
	}
	return recs
}

func data(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic("world: " + err.Error())
	}
	return b
}
