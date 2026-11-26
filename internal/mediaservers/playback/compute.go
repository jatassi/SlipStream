package playback

import (
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Input is everything one report reads, loaded in one Read.
type Input struct {
	Kind      link.KindName
	Window    Window
	Reachable bool
	// Plays are the window's plays, oldest first.
	Plays    []Play
	Siblings Siblings
	Targets  []targets.Target
	// Works are the titles and Content Types of the plays' and siblings'
	// Works, read now.
	Works map[works.WorkID]works.Summary
}

// Play is one play of the window as its row holds it.
type Play struct {
	ID link.PlayID
	// File is zero while the play is unfiled.
	File files.ID
	// ItemKey is the item the server played, frozen at play time.
	ItemKey                              string
	PlayerKey, Player, Platform, Product string
	User                                 string
	Decision                             link.Decision
	Streams                              link.Streams
	Reasons                              []string
	Received                             link.Received
	Snapshot                             *Snapshot
	Started, LastSeen                    time.Time
}

// judgedPlay is a transcoded play with a snapshot, its blocking
// dimensions and its outcome.
type judgedPlay struct {
	play    Play
	dims    []Blocked
	outcome Outcome
}

// Compute is the Playback report of in's plays.
func Compute(in Input) Report {
	rep := Report{
		Since: in.Window.Since, Window: int64(in.Window.Keep / time.Second), Reachable: in.Reachable,
		Plex: in.Kind == "plex", Players: []Player{},
	}
	for _, pl := range playersOf(in.Plays) {
		pr := prove(pl.plays)
		var judged []judgedPlay
		for _, p := range pl.plays {
			if p.Decision != link.DecisionTranscode || p.Snapshot == nil {
				continue
			}
			dims := blocking(p, pr)
			judged = append(judged, judgedPlay{play: p, dims: dims, outcome: outcomeOf(in, p, dims, pr)})
		}
		player := pl.view()
		player.Groups = groupsOf(judged)
		rep.Plays += player.Plays
		rep.Transcoded += player.Transcoded
		rep.Unmatched += player.Unmatched
		rep.Players = append(rep.Players, player)
	}
	sortWorstFirst(rep.Players)
	return rep
}

// ItemsToJudge is the Items of the transcoded plays with a snapshot,
// whose files a version could be.
func ItemsToJudge(ps []Play) []works.ItemID {
	var out []works.ItemID
	for _, p := range ps {
		if p.Decision == link.DecisionTranscode && p.Snapshot != nil && !slices.Contains(out, p.Snapshot.Item) {
			out = append(out, p.Snapshot.Item)
		}
	}
	return out
}

// WorksOf is the Works of the plays' snapshots and of the siblings.
func WorksOf(ps []Play, s Siblings) []works.WorkID {
	var out []works.WorkID
	add := func(w works.WorkID) {
		if !slices.Contains(out, w) {
			out = append(out, w)
		}
	}
	for _, p := range ps {
		if p.Snapshot != nil {
			add(p.Snapshot.Work)
		}
	}
	for _, sibs := range s.Files {
		for _, sib := range sibs {
			add(sib.Work)
		}
	}
	return out
}
