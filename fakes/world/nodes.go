package world

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/jatassi/slipstream/fakes/world/internal/lifecycle"
)

// NodeAction is what fakeworld node does to a Node.
type NodeAction = lifecycle.Action

const (
	StartNode = lifecycle.Start
	StopNode  = lifecycle.Stop
	KillNode  = lifecycle.Kill
	WipeNode  = lifecycle.Wipe
)

// ParseNodeAction reads an action's name, as fakeworld node and the
// control socket take it.
func ParseNodeAction(s string) (NodeAction, error) { return lifecycle.ParseAction(s) }

// Node runs action a on the Node name and answers its manifest entry,
// rewritten with the Node's new state and PID.
func (r *Run) Node(ctx context.Context, name string, a NodeAction) (Node, error) {
	i := slices.IndexFunc(r.nodes, func(n *nodeRun) bool { return n.spec.Name == name })
	if i < 0 {
		return Node{}, fmt.Errorf("the run has no node %s", name)
	}
	if err := r.nodes[i].life.Do(ctx, a); err != nil {
		return Node{}, err
	}
	return r.rewriteNode(i)
}

// nodeExited rewrites the manifest for Node i, which exited unasked. A
// Node that exits during launch fails the launch, which writes none.
func (r *Run) nodeExited(i int) {
	if _, err := r.rewriteNode(i); err != nil && !errors.Is(err, errUnpublished) {
		r.log.Error("manifest", "node", r.nodes[i].spec.Name, "err", err.Error())
	}
}

var errUnpublished = errors.New("the run has no manifest: it is launching or stopping")

// rewriteNode writes the manifest again, atomically, with Node i's state
// and PID as they are now.
func (r *Run) rewriteNode(i int) (Node, error) {
	r.manifestMu.Lock()
	defer r.manifestMu.Unlock()
	if !r.published {
		return Node{}, errUnpublished
	}
	m := r.manifest
	m.Nodes = slices.Clone(m.Nodes)
	n := r.nodes[i]
	m.Nodes[i].State, m.Nodes[i].PID = n.life.State(), n.life.PID()
	if err := writeAtomically(filepath.Join(r.runDir, manifestFile), marshal(m)); err != nil {
		return Node{}, err
	}
	r.manifest = m
	return m.Nodes[i], nil
}
