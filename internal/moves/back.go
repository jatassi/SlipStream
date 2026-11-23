package moves

import (
	"context"
	"fmt"
	"strings"

	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Back previews moving each Work batch moved back to the root it left,
// and each torrent back to the Download Client it left. A Work whose old
// root is gone is a red unit, so the Admin picks a root for it.
func (m *Moves) Back(ctx context.Context, batch BatchID) (Plan, error) {
	var done []Unit
	var gone []PlanUnit
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		if _, err := q.GetBatch(ctx, batch); err != nil {
			return notFound(err)
		}
		rows, err := q.ListDoneUnitsOfBatch(ctx, batch)
		if err != nil {
			return err
		}
		if done, err = unitsOf(ctx, q, rows); err != nil {
			return err
		}
		gone, err = m.goneRoots(ctx, r, done)
		return err
	})
	if err != nil {
		return Plan{}, err
	}
	// A torrent a Work placed its files from goes back with that Work
	// again, linked, so its payload crosses once.
	linked := map[UnitID]bool{}
	for _, u := range done {
		if u.PairsWith != nil && !gonePlanned(gone, u.Subject.Work) {
			linked[*u.PairsWith] = true
		}
	}
	ask := Ask{Origin: OriginBulk, Options: Options{IncludeLinkedTorrents: len(linked) > 0}}
	for _, u := range done {
		switch {
		case linked[u.ID]:
		case u.Torrent != nil:
			ask.Torrents = append(ask.Torrents, TorrentTo{Torrent: *u.Torrent.Arriving, To: u.Torrent.From})
		case !gonePlanned(gone, u.Subject.Work):
			ask.Works = append(ask.Works, WorkTo{Work: u.Subject.Work, To: u.Subject.From.Root})
		}
	}
	switch {
	case len(ask.Works) == 0 && len(gone) == 0:
		ask.Origin = OriginTorrent
	case len(done) == 1:
		ask.Origin = OriginWork
	}
	p, err := m.Preview(ctx, ask)
	if err != nil {
		return Plan{}, err
	}
	p.Units = append(p.Units, gone...)
	return p, nil
}

// goneRoots is the units of us whose source root is gone, each a red
// planned unit.
func (m *Moves) goneRoots(ctx context.Context, r storage.Reader[file], us []Unit) ([]PlanUnit, error) {
	var ids []roots.ID
	var ws []works.WorkID
	for _, u := range us {
		if u.Torrent == nil {
			ids = appendOnce(ids, u.Subject.From.Root)
			ws = append(ws, u.Subject.Work)
		}
	}
	dests, err := m.cfg.Destinations(ctx, r, ids)
	if err != nil {
		return nil, err
	}
	sums, err := works.SummariesOf(ctx, r, ws)
	if err != nil {
		return nil, err
	}
	var out []PlanUnit
	for _, u := range us {
		if _, ok := dests[u.Subject.From.Root]; ok || u.Torrent != nil {
			continue
		}
		reason := Reason{Check: CheckDestination, Color: Red, Code: "root_gone", Words: "The root it moved from is gone. Pick another root to move it to."}
		out = append(out, PlanUnit{Subject: u.Subject, Label: sums[u.Subject.Work].Label, Verdict: Verdict{Color: Red, Reasons: []Reason{reason}}})
	}
	return out, nil
}

func gonePlanned(gone []PlanUnit, w works.WorkID) bool {
	for _, u := range gone {
		if u.Subject.Work == w {
			return true
		}
	}
	return false
}

// recommend words p's recommendation and its Start button, its
// destinations' Nodes named by nodeNames.
func (m *Moves) recommend(p *Plan, nodeNames map[role.NodeID]string) {
	total, movable := len(p.Units), p.Movable()
	var types []works.TypeID
	var to []role.NodeID
	var red []string
	torrents := 0
	for _, u := range p.Units {
		if u.Torrent != nil {
			torrents++
			to = appendOnce(to, u.Torrent.To.Node)
		} else {
			types = appendOnce(types, u.Subject.Type)
			to = appendOnce(to, u.Subject.To.Node)
		}
		if u.Verdict.Color == Red {
			red = append(red, u.Label)
		}
	}
	where := ""
	if len(to) == 1 {
		where = " to " + nodeNames[to[0]]
	}
	counted := m.counted(types, kinds{works: total - torrents, torrents: torrents})
	switch {
	case total == 0:
		p.Recommendation, p.Button = "Nothing here has files to move.", "Move"
	case total == 1 && movable == 1:
		p.Recommendation, p.Button = fmt.Sprintf("Move %s%s.", p.Units[0].Label, where), "Move"
	case total == 1:
		p.Recommendation, p.Button = p.Units[0].Label+" can't move.", "Move"
	case movable == total:
		p.Recommendation, p.Button = fmt.Sprintf("Move %s%s.", counted, where), fmt.Sprintf("Move %d", total)
	case movable == 0:
		p.Recommendation, p.Button = fmt.Sprintf("None of these %s can move.", counted), fmt.Sprintf("Move %d of %d", movable, total)
	default:
		p.Recommendation = fmt.Sprintf("Move %d of %s%s. %d can't move: %s.", movable, counted, where, len(red), listWords(red))
		p.Button = fmt.Sprintf("Move %d of %d", movable, total)
	}
}

// listWords joins names as a person lists them: "A, B and C", the first
// three, then how many more.
func listWords(names []string) string {
	const shown = 3
	if len(names) > shown+1 {
		return strings.Join(names[:shown], ", ") + fmt.Sprintf(" and %d more", len(names)-shown)
	}
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// nodeNamesIn is every live Node's name, read in r.
func nodeNamesIn(ctx context.Context, r storage.Reader[file]) (map[role.NodeID]string, error) {
	live, err := nodes.Names(ctx, r)
	out := make(map[role.NodeID]string, len(live))
	for _, n := range live {
		out[n.ID] = n.Name
	}
	return out, err
}
