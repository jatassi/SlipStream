package playback

import (
	"sync"
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
	File                                 files.ID
	PlayerKey, Player, Platform, Product string
	Decision                             link.Decision
	Streams                              link.Streams
	// Reasons, ItemKeys and Received are read for a transcode only.
	// ItemKeys are the items that held the played file and the versions
	// its player could choose instead, frozen at play time.
	Reasons, ItemKeys []string
	Received          link.Received
	Snapshot          *Snapshot
	Started, LastSeen time.Time
}

// judgedPlay is a transcoded play with a snapshot, its blocking
// dimensions and its outcome.
type judgedPlay struct {
	play    Play
	dims    []Blocked
	outcome Outcome
}

// Compute is the Playback report of in's plays. Each player's plays
// prove and judge only that player's, so players are judged concurrently.
func Compute(in Input) Report {
	pls := playersOf(in.Plays)
	rep := Report{
		Since: in.Window.Since, Window: int64(in.Window.Keep / time.Second), Reachable: in.Reachable,
		Plex: in.Kind == "plex", Players: make([]Player, len(pls)),
	}
	var wg sync.WaitGroup
	for i, pl := range pls {
		wg.Go(func() { rep.Players[i] = pl.judge(in) })
	}
	wg.Wait()
	for _, p := range rep.Players {
		rep.Plays += p.Plays
		rep.Transcoded += p.Transcoded
		rep.Unmatched += p.Unmatched
	}
	sortWorstFirst(rep.Players)
	return rep
}

// judge is the player with its transcodes grouped by what blocked them.
func (pl playerPlays) judge(in Input) Player {
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
	return player
}

// ItemsToJudge is the Items of the transcoded plays with a snapshot,
// whose files a version could be.
func ItemsToJudge(ps []Play) []works.ItemID {
	seen := map[works.ItemID]bool{}
	var out []works.ItemID
	for _, p := range ps {
		if p.Decision == link.DecisionTranscode && p.Snapshot != nil && !seen[p.Snapshot.Item] {
			seen[p.Snapshot.Item] = true
			out = append(out, p.Snapshot.Item)
		}
	}
	return out
}

// WorksOf is the Works of the plays' snapshots and of the siblings.
func WorksOf(ps []Play, s Siblings) []works.WorkID {
	seen := map[works.WorkID]bool{}
	var out []works.WorkID
	add := func(w works.WorkID) {
		if !seen[w] {
			seen[w] = true
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
