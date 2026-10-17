package node

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/node/internal/control"
	"github.com/jatassi/slipstream/internal/node/internal/lock"
	"github.com/jatassi/slipstream/internal/node/op"
	"github.com/jatassi/slipstream/internal/nodesystem"
)

type ConfigRequest struct {
	Changes []ConfigChange `json:"changes"`
}

// ConfigChange is one key's new value as config.yaml spells it. A nil
// Value unsets the key.
type ConfigChange struct {
	Key   string  `json:"key"`
	Value *string `json:"value"`
}

// ConfigReply holds one of: every key resolved after the change, or why
// nothing changed.
type ConfigReply struct {
	Keys    []Key               `json:"keys,omitempty"`
	Refused *nodesystem.Refusal `json:"refused,omitempty"`
}

// ConfigOp changes Node settings table keys in the running Node's
// config.yaml. A live key applies at once.
var ConfigOp = op.Op[ConfigRequest, ConfigReply]{Name: "config"}

func (c ConfigChange) change() nodesystem.Change {
	if c.Value == nil {
		return nodesystem.Change{Key: c.Key, Value: nodesystem.Reset()}
	}
	return nodesystem.Change{Key: c.Key, Value: nodesystem.Text(*c.Value)}
}

func changesOf(cs []ConfigChange) []nodesystem.Change {
	out := make([]nodesystem.Change, len(cs))
	for i, c := range cs {
		out[i] = c.change()
	}
	return out
}

// setConfig writes changes to config.yaml and applies each live row whose
// saved value now differs from the running one, then reports the new
// settings revision. A restart row keeps its running value until the Node
// restarts.
func (n *Node) setConfig(changes []nodesystem.Change) ([]Key, error) {
	keys, err := n.saveConfig(changes)
	if err != nil {
		return nil, err
	}
	n.reportHost()
	return keys, nil
}

func (n *Node) saveConfig(changes []nodesystem.Change) ([]Key, error) {
	n.cfgMu.Lock()
	defer n.cfgMu.Unlock()
	saved, keys, err := n.env.configFile().set(changes)
	if err != nil {
		return nil, err
	}
	var changed []settingRow
	var appliers []*liveApply
	for _, r := range nodeSettings {
		if !r.live() || r.same(&saved, &n.cfg) {
			continue
		}
		changed = append(changed, r)
		if !slices.Contains(appliers, r.applier()) {
			appliers = append(appliers, r.applier())
		}
	}
	for _, a := range appliers {
		if err := a.apply(n, saved); err != nil {
			return nil, err
		}
	}
	for _, r := range changed {
		r.adopt(&n.cfg, &saved)
	}
	return keys, nil
}

func (n *Node) serveConfig(_ context.Context, body json.RawMessage, send func(any) error) error {
	var req ConfigRequest
	if err := control.Decode(body, &req); err != nil {
		return err
	}
	keys, err := n.setConfig(changesOf(req.Changes))
	if refused, ok := errors.AsType[*nodesystem.Refusal](err); ok {
		return send(ConfigReply{Refused: refused})
	}
	if err != nil {
		return err
	}
	return send(ConfigReply{Keys: keys})
}

// SetConfig changes keys of the Node on env's data dir, every one or none.
// While no Node holds the data dir it writes config.yaml under the lock;
// otherwise the running Node writes it and applies live keys, and online
// is true. A refused change is a *nodesystem.Refusal.
func SetConfig(ctx context.Context, env Env, changes []ConfigChange) (keys []Key, online bool, err error) {
	lk, err := lock.Acquire(env.DataDir.RunDir())
	if _, held := errors.AsType[*lock.HeldError](err); held {
		var reply ConfigReply
		reply, err = callWhenReady(ctx, env.Net, env.DataDir, ConfigOp, ConfigRequest{Changes: changes})
		if err == nil && reply.Refused != nil {
			err = reply.Refused
		}
		return reply.Keys, true, err
	}
	if err != nil {
		return nil, false, err
	}
	_, keys, err = env.configFile().set(changesOf(changes))
	return keys, false, errors.Join(err, lk.Release())
}

// notReadyPoll is how often callWhenReady asks a starting Node again.
const notReadyPoll = 200 * time.Millisecond

// callWhenReady is Call, asked again while the Node is starting, until ctx
// ends.
func callWhenReady[Req, Resp any](ctx context.Context, network link.Network, dir DataDir, o op.Op[Req, Resp], req Req) (Resp, error) {
	for {
		resp, err := Call(ctx, network, dir, o, req)
		if notReady, ok := errors.AsType[*control.Error](err); !ok || notReady.Code != control.CodeNotReady {
			return resp, err
		}
		select {
		case <-ctx.Done():
			return resp, err
		case <-time.After(notReadyPoll):
		}
	}
}
