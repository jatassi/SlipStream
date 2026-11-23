package moves

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Place is one side of a unit, with its Node's name and its root in
// words, such as "Movies library root on den".
type Place struct {
	End
	NodeName string
	Root     string
	// Client is a torrent unit's Download Client in words, Root "".
	Client string
}

// Named is what a reader shows beside a unit's ids, read in one Reader.
type Named struct {
	works    map[works.WorkID]works.Summary
	nodes    map[role.NodeID]string
	roots    map[roots.ID]string
	torrents map[downloadclients.ID]downloadclients.MoveSource
	sides    sides
}

func (m *Moves) namesOf(ctx context.Context, r storage.Reader[file], ws []works.WorkID) (Named, error) {
	var n Named
	var err error
	if n.works, err = works.SummariesOf(ctx, r, ws); err != nil {
		return n, err
	}
	if n.roots, err = m.cfg.Roots.WordsIn(ctx, r); err != nil {
		return n, err
	}
	n.nodes, err = nodeNamesIn(ctx, r)
	return n, err
}

// Place is e with its Node and root named.
// sides is each Node's and external client's name, for torrent units'
// Download Clients.
type sides struct {
	nodes   map[role.NodeID]string
	clients map[downloadclients.ClientID]string
}

// sidesIn reads the names of us' Download Clients in r. A deleted
// client has none.
func sidesIn(ctx context.Context, r storage.Reader[file], us []Unit) (sides, error) {
	n := sides{clients: map[downloadclients.ClientID]string{}}
	var err error
	if n.nodes, err = nodeNamesIn(ctx, r); err != nil {
		return n, err
	}
	for _, u := range us {
		if u.Torrent == nil {
			continue
		}
		for _, c := range []*downloadclients.ClientID{u.Torrent.From.Client, u.Torrent.To.Client} {
			if c == nil {
				continue
			}
			if _, ok := n.clients[*c]; ok {
				continue
			}
			e, err := downloadclients.ExternalOf(ctx, r, *c)
			if err != nil && !errors.Is(err, downloadclients.ErrNoClient) {
				return n, err
			}
			n.clients[*c] = e.Name
		}
	}
	return n, nil
}

// client is s's client in words: its name, or the built-in client.
func (n sides) client(s Side) string {
	if s.Client == nil {
		return "the built-in client"
	}
	return n.clients[*s.Client]
}

// words is s in words, such as "qBittorrent on den".
func (n sides) words(s Side) string { return n.client(s) + " on " + n.nodes[s.Node] }

func (n Named) Place(e End) Place {
	return Place{End: e, NodeName: n.nodes[e.Node], Root: n.roots[e.Root]}
}

// noun is the Content Types' noun for n of them: the type's when every
// one is of t, "Works" when they mix.
func (m *Moves) noun(types []works.TypeID, n int) string {
	if len(types) != 1 {
		return plural(n, "Work", "Works")
	}
	d, _ := m.cfg.Works.Describe(types[0])
	return plural(n, strings.ToLower(d.Work.One), strings.ToLower(d.Work.Many))
}

// kinds is a count of Work units and of torrent units.
type kinds struct{ works, torrents int }

// counted is k in words, its Works of types, such as "1 movie and 1
// torrent".
func (m *Moves) counted(types []works.TypeID, k kinds) string {
	ts := fmt.Sprintf("%d %s", k.torrents, plural(k.torrents, "torrent", "torrents"))
	ws := fmt.Sprintf("%d %s", k.works, m.noun(types, k.works))
	switch {
	case k.torrents == 0:
		return ws
	case k.works == 0:
		return ts
	}
	return ws + " and " + ts
}

// tally is done of all in words, each kind as "12 of 43 movies", or as
// "43 movies" once all of it is done.
func (m *Moves) tally(types []works.TypeID, done, all kinds) string {
	if done == all {
		return m.counted(types, all)
	}
	ws := fmt.Sprintf("%d of %d %s", done.works, all.works, m.noun(types, all.works))
	ts := fmt.Sprintf("%d of %d %s", done.torrents, all.torrents, plural(all.torrents, "torrent", "torrents"))
	switch {
	case all.torrents == 0:
		return ws
	case all.works == 0:
		return ts
	}
	return ws + " and " + ts
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// wordsOf is the band's sentence for v: what the Move does now.
func wordsOf(v View) string {
	from, to := v.From.NodeName, v.To.NodeName
	switch v.Phase {
	case PhaseQueued, PhaseTransfer, PhasePlace:
		if v.Phase == PhaseQueued && v.Waiting == WaitDownloading {
			return from + " is finishing its work. This torrent moves once it finishes downloading."
		}
		if w := waitWords(v.Waiting, from, to); w != "" {
			return w
		}
		if v.Phase == PhaseQueued {
			return "Moving to " + to
		}
		return "Copying to " + to
	case PhaseSwitch:
		if w := waitWords(v.Waiting, from, to); w != "" {
			return w
		}
		return "Switching to " + to
	case PhaseCleanup:
		return "Removing the files from " + from
	case PhaseDone:
		return "Moved to " + to
	case PhaseFailed:
		return v.Failure.Words
	case PhaseCanceled:
	}
	return "Move canceled"
}

// waitWords is a wait's sentence, empty for none.
func waitWords(w Wait, from, to string) string {
	switch w {
	case WaitPairBusy:
		return "Waiting for another Move from " + from + " to " + to
	case WaitLowPriority:
		return "Waiting until the downloads between " + from + " and " + to + " finish copying"
	case WaitNodeOffline:
		return "Waiting until " + from + " and " + to + " are connected"
	case WaitNoRoute:
		return from + " and " + to + " can't connect to each other yet. The copy waits until they can."
	case WaitSpace:
		return "Waiting for free space on " + to
	case WaitRenames:
		return "Waiting until its files' renames finish"
	case WaitImporting:
		return "Waiting until its Import finishes"
	case WaitOpenFiles:
		return "Waiting until SlipStream finishes writing or sending its files"
	case WaitPlaying:
		return "Waiting until playback ends"
	case WaitMediaServers:
		return "Saving watched state, resume positions and ratings on the Media Servers"
	case WaitReconciling:
		return "Waiting until Reconciling finishes"
	case WaitLinkedTorrent:
		return "Waiting for its torrent to finish copying"
	case WaitDownloading, WaitGuard, "":
	}
	return ""
}

// needWords is Needs you's sentence for v, at now.
func needWords(v View, now time.Time) string {
	if v.Failure != nil {
		return fmt.Sprintf("%s couldn't move to %s. %s", v.Label, v.To.NodeName, v.Failure.Words)
	}
	waited := fmt.Sprintf("%s has waited %s to switch to %s.", v.Label, setting.Words(now.Sub(*v.Since).Truncate(time.Minute)), v.To.NodeName)
	switch v.Waiting {
	case WaitPlaying:
		return waited + " Someone is playing it on a Media Server."
	case WaitMediaServers:
		return waited + " SlipStream can't connect to a Media Server to save who watched it."
	case WaitImporting:
		if v.Torrent != nil {
			return waited + " An Import from it hasn't finished."
		}
		return waited + " An Import into it hasn't finished."
	case WaitOpenFiles:
		return waited + " SlipStream is still writing or sending its files."
	case WaitPairBusy, WaitLowPriority, WaitNodeOffline, WaitNoRoute, WaitSpace, WaitRenames, WaitDownloading, WaitGuard, WaitLinkedTorrent, WaitReconciling, "":
	}
	return waited
}

// Names is every Node's name and every root in words, for a preview's
// places.
func (m *Moves) Names(ctx context.Context) (Named, error) {
	var n Named
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		n, err = m.namesOf(ctx, r, nil)
		return err
	})
	return n, err
}

// PlanNames is Names with the Download Clients of p's torrent units.
func (m *Moves) PlanNames(ctx context.Context, p Plan) (Named, error) {
	var us []Unit
	for _, u := range p.Units {
		if u.Torrent != nil {
			us = append(us, Unit{Torrent: u.Torrent})
		}
	}
	var n Named
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if n, err = m.namesOf(ctx, r, nil); err != nil {
			return err
		}
		n.sides, err = sidesIn(ctx, r, us)
		return err
	})
	return n, err
}

// ClientWords is s's Download Client in words, read by PlanNames.
func (n Named) ClientWords(s Side) string { return n.sides.client(s) }

// NotesWords is what v's unit recorded for the Admin, as sentences: the
// preview's amber reasons, and the source files cleanup left in place.
func NotesWords(v View) []string {
	out := []string{}
	for _, r := range v.Notes.Amber {
		out = append(out, r.Words)
	}
	if n := len(v.Notes.Skipped); n > 0 {
		out = append(out, fmt.Sprintf("%d %s on %s changed after this Move was planned. SlipStream left %s in place, and the next Library scan finds %s.",
			n, plural(n, "file", "files"), v.From.NodeName, plural(n, "it", "them"), plural(n, "it", "them")))
	}
	return out
}

// CanceledWords is the toast of a cancel of batch that canceled canceled
// units, naming the units that had switched.
func (m *Moves) CanceledWords(ctx context.Context, batch BatchID, canceled int) (string, error) {
	var types []works.TypeID
	var to []string
	var switched kinds
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		of, err := typesOf(ctx, q, []BatchID{batch})
		if err != nil {
			return err
		}
		types = of[batch]
		counts, err := q.CountBatchPhases(ctx, batch)
		if err != nil {
			return err
		}
		switched = switchedIn(counts)
		nodes, err := nodeNamesIn(ctx, r)
		if err != nil {
			return err
		}
		dests, err := q.ListBatchDestinations(ctx, batch)
		for _, d := range dests {
			to = append(to, nodes[d])
		}
		return err
	})
	if err != nil {
		return "", err
	}
	words := fmt.Sprintf("Canceled %d %s.", canceled, plural(canceled, "Move", "Moves"))
	n := switched.works + switched.torrents
	if n == 0 {
		return words, nil
	}
	stay := fmt.Sprintf(" %s already switched %s.", m.counted(types, switched), plural(n, "stays", "stay"))
	if len(to) == 1 {
		stay = fmt.Sprintf(" %s already switched %s on %s.", m.counted(types, switched), plural(n, "stays", "stay"), to[0])
	}
	return words + stay, nil
}
