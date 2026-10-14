// Package events is a run's event log, events.jsonl: one Record per line,
// in the order its one writer, Log, appended them. Readers, fault triggers
// and the journal are all reads of that file, so they work the same on a
// live run, a stopped run and a downloaded bundle.
package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"
	"time"
)

// File is the event log's name in a run dir.
const File = "events.jsonl"

// The kinds the world writes.
const (
	RunLaunched  = "run.launched"
	RunReady     = "run.ready"
	RunStopping  = "run.stopping"
	RunStopped   = "run.stopped"
	NodeSpawned  = "node.spawned"
	NodeReady    = "node.ready"
	NodeExited   = "node.exited"
	NodeError    = "node.error"
	FaultSet     = "fault.set"
	FaultCleared = "fault.cleared"
	FaultFired   = "fault.fired"
	StepStart    = "step.start"
	StepEnd      = "step.end"
)

// World is the Node of a record the harness itself wrote.
const World = "world"

type Record struct {
	// Seq is 1, 2, 3 … with no gaps within a run.
	Seq uint64 `json:"seq"`
	// At is for humans; no trigger reads it.
	At time.Time `json:"at"`
	// Kind is "<noun>.<what>".
	Kind string `json:"kind"`
	// Node is the Node concerned or the dialing Node, "world" for the
	// harness.
	Node string          `json:"node,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
	// Files maps a name to a path relative to the run dir.
	Files map[string]string `json:"files,omitempty"`
}

type Query struct {
	// Kind is a path.Match glob such as "mail.*". Empty matches every kind.
	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`
	// Node is empty for every Node.
	Node string `json:"node,omitempty" yaml:"node,omitempty"`
	// Match compares top-level Data fields as JSON text: a string field by
	// its value, any other field by its compact JSON.
	Match map[string]string `json:"match,omitempty" yaml:"match,omitempty"`
	// After keeps only records with a greater Seq.
	After uint64 `json:"after,omitempty" yaml:"after,omitempty"`
}

func (q Query) Matches(r Record) bool {
	if r.Seq <= q.After || (q.Node != "" && q.Node != r.Node) {
		return false
	}
	if q.Kind != "" {
		if ok, err := path.Match(q.Kind, r.Kind); err != nil || !ok {
			return false
		}
	}
	if len(q.Match) == 0 {
		return true
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(r.Data, &fields) != nil {
		return false
	}
	for k, want := range q.Match {
		raw, ok := fields[k]
		if !ok || text(raw) != want {
			return false
		}
	}
	return true
}

func text(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var b bytes.Buffer
	_ = json.Compact(&b, raw) // raw is a field of an object that decoded, so it is valid JSON
	return b.String()
}

// String is q as a reader's flags would write it, for messages.
func (q Query) String() string {
	var parts []string
	if q.Kind != "" {
		parts = append(parts, "kind "+q.Kind)
	}
	if q.Node != "" {
		parts = append(parts, "node "+q.Node)
	}
	for _, k := range slices.Sorted(maps.Keys(q.Match)) {
		parts = append(parts, k+"="+q.Match[k])
	}
	if q.After > 0 {
		parts = append(parts, fmt.Sprintf("after seq %d", q.After))
	}
	if len(parts) == 0 {
		return "any record"
	}
	return strings.Join(parts, ", ")
}
