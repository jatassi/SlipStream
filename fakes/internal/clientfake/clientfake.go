// Package clientfake is what the four fake Download Clients share past
// the engine: the cues that play row 16's faults on one Node's client.
package clientfake

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/internal/torrentclient"
	"github.com/jatassi/slipstream/internal/bencode"
)

const (
	remove  = "remove"
	stall   = "stall"
	failCue = "error"
	tracker = "tracker"
	empty   = "empty"
	dropAdd = "drop-add"
	lateAdd = "late-add"
	restart = "restart"
)

var cueNames = []string{remove, stall, failCue, tracker, empty, dropAdd, lateAdd, restart}

// Cues are the cues the fake named kind plays, each <kind>-<cue>:
//
//   - remove {node, release, data}: the client removes the release's
//     torrent, as its own seeding rule or a person in its web UI does,
//     deleting its files with data.
//   - stall {node, release, on}: no more data arrives, or it arrives again.
//   - error {node, release, message}: a local error such as a full disk
//     stops the torrent; "" clears it.
//   - tracker {node, release, message, error}: the tracker's message, a
//     warning or with error an error; "" clears it.
//   - empty {node}: the client lists no torrent, keeping their files.
//   - drop-add {node}: the next add answers as the client does and keeps
//     nothing.
//   - late-add {node, after}: the next added torrent is missing from the
//     client's lists for after, such as "30s".
//   - restart {node}: the client restarts, keeping its torrents.
func Cues(kind string) []fake.CueSpec {
	out := make([]fake.CueSpec, len(cueNames))
	for i, name := range cueNames {
		out[i] = fake.CueSpec{Name: kind + "-" + name}
	}
	return out
}

type onNode struct {
	Node string `json:"node"`
}

type onRelease struct {
	Node    string `json:"node"`
	Release string `json:"release"`
}

type removeArgs struct {
	onRelease
	Data bool `json:"data,omitempty"`
}

type stallArgs struct {
	onRelease
	On bool `json:"on,omitempty"`
}

type failArgs struct {
	onRelease
	Message string `json:"message,omitempty"`
}

type trackerArgs struct {
	onRelease
	Message string `json:"message,omitempty"`
	Error   bool   `json:"error,omitempty"`
}

type lateArgs struct {
	onNode
	After string `json:"after"`
}

// OnCues has the fake named kind play its Cues on the engine of the Node
// a cue names, as engineOf finds it.
func OnCues(env fake.Env, kind string, engineOf func(node string) (*torrentclient.Engine, bool)) error {
	c := cues{env: env, kind: kind, engineOf: engineOf}
	return firstErr(
		fake.OnCue(env, kind+"-"+remove, onReleaseCue(c, func(e *torrentclient.Engine, h bencode.InfoHash, a removeArgs) error {
			return e.Remove(h, a.Data)
		})),
		fake.OnCue(env, kind+"-"+stall, onReleaseCue(c, func(e *torrentclient.Engine, h bencode.InfoHash, a stallArgs) error {
			return e.Stall(h, a.On)
		})),
		fake.OnCue(env, kind+"-"+failCue, onReleaseCue(c, func(e *torrentclient.Engine, h bencode.InfoHash, a failArgs) error {
			return e.Fail(h, a.Message)
		})),
		fake.OnCue(env, kind+"-"+tracker, onReleaseCue(c, func(e *torrentclient.Engine, h bencode.InfoHash, a trackerArgs) error {
			return e.Announced(h, torrentclient.Tracker{Message: a.Message, Error: a.Error})
		})),
		fake.OnCue(env, kind+"-"+empty, onNodeCue(c, func(e *torrentclient.Engine, _ onNode) error { e.Empty(); return nil })),
		fake.OnCue(env, kind+"-"+dropAdd, onNodeCue(c, func(e *torrentclient.Engine, _ onNode) error { e.DropNextAdd(); return nil })),
		fake.OnCue(env, kind+"-"+lateAdd, onNodeCue(c, func(e *torrentclient.Engine, a lateArgs) error {
			d, err := time.ParseDuration(a.After)
			if err != nil || d <= 0 {
				return fmt.Errorf("after %q isn't a positive duration such as 30s", a.After)
			}
			e.ListNextAddAfter(d)
			return nil
		})),
		fake.OnCue(env, kind+"-"+restart, onNodeCue(c, func(e *torrentclient.Engine, _ onNode) error { e.Restart(); return nil })),
	)
}

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

type cues struct {
	env      fake.Env
	kind     string
	engineOf func(node string) (*torrentclient.Engine, bool)
}

type nodeArgs interface{ node() string }

type releaseArgs interface {
	nodeArgs
	release() string
}

func (a onNode) node() string       { return a.Node }
func (a onRelease) node() string    { return a.Node }
func (a onRelease) release() string { return a.Release }

func (c cues) engine(a nodeArgs) (*torrentclient.Engine, error) {
	e, ok := c.engineOf(a.node())
	if !ok {
		return nil, fmt.Errorf("fake %s plays no client on Node %q", c.kind, a.node())
	}
	return e, nil
}

func onNodeCue[A nodeArgs](c cues, play func(*torrentclient.Engine, A) error) func(context.Context, A) (any, error) {
	return func(_ context.Context, a A) (any, error) {
		e, err := c.engine(a)
		if err != nil {
			return nil, err
		}
		return a, play(e, a)
	}
}

func onReleaseCue[A releaseArgs](c cues, play func(*torrentclient.Engine, bencode.InfoHash, A) error) func(context.Context, A) (any, error) {
	return onNodeCue(c, func(e *torrentclient.Engine, a A) error {
		i := slices.IndexFunc(c.env.Releases.Releases, func(r releases.Release) bool { return r.Name == a.release() })
		if i < 0 {
			return fmt.Errorf("the Scenario has no release %q", a.release())
		}
		return play(e, c.env.Releases.Releases[i].InfoHash, a)
	})
}
