package fake

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
)

// CueSpec is a cue a fake declares: a change to the world outside every
// Node, which fakeworld cue <name> plays. A cue never calls a Node, never
// writes a Node's files and never makes a state production can't reach.
type CueSpec struct {
	Name string `json:"name"`
}

type cueFunc func(ctx context.Context, args json.RawMessage) (any, error)

// cues are the plays the fake's New registered, shared by every copy of
// its Env.
type cues struct {
	mu   sync.Mutex
	play map[string]cueFunc
}

// OnCue has the fake play the cue name, which its Spec must declare, with
// the cue's arguments decoded strictly into A. What play returns is the
// cue record's result.
func OnCue[A any](env Env, name string, play func(ctx context.Context, args A) (any, error)) error {
	if !slices.ContainsFunc(env.cueSpecs, func(c CueSpec) bool { return c.Name == name }) {
		return fmt.Errorf("fake %s declares no cue %s", env.Name, name)
	}
	env.cues.mu.Lock()
	defer env.cues.mu.Unlock()
	env.cues.play[name] = func(ctx context.Context, raw json.RawMessage) (any, error) {
		var args A
		if len(bytes.TrimSpace(raw)) > 0 {
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&args); err != nil {
				return nil, fmt.Errorf("cue %s: %w", name, err)
			}
		}
		return play(ctx, args)
	}
	return nil
}

// Cue plays the cue name with args, as the fake's New registered it.
func (e Env) Cue(ctx context.Context, name string, args json.RawMessage) (any, error) {
	e.cues.mu.Lock()
	play, ok := e.cues.play[name]
	e.cues.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("fake %s plays no cue %s", e.Name, name)
	}
	return play(ctx, args)
}
