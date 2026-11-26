package playback

import (
	"cmp"
	"slices"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// playerPlays is one player's plays, oldest first.
type playerPlays struct {
	key   string
	plays []Play
}

// playersOf is ps by player key, in the order each player first played.
func playersOf(ps []Play) []playerPlays {
	var out []playerPlays
	at := map[string]int{}
	for _, p := range ps {
		i, ok := at[p.PlayerKey]
		if !ok {
			i = len(out)
			at[p.PlayerKey] = i
			out = append(out, playerPlays{key: p.PlayerKey})
		}
		out[i].plays = append(out[i].plays, p)
	}
	return out
}

// view is the player as its latest play names it, with its counts.
func (pl playerPlays) view() Player {
	last := pl.plays[len(pl.plays)-1]
	v := Player{
		Key: pl.key, Name: last.Player, Product: last.Product, Platform: last.Platform,
		LastPlayedAt: last.Started, Plays: len(pl.plays), Groups: []Group{},
	}
	for _, p := range pl.plays {
		switch p.Decision {
		case link.DecisionDirectPlay:
			v.DirectPlays++
		case link.DecisionDirectStream:
			v.DirectStreams++
		case link.DecisionTranscode:
			v.Transcoded++
			if p.Snapshot == nil {
				v.Unmatched++
			}
		}
	}
	return v
}

// sortWorstFirst orders players by transcoded plays, then by the share of
// their plays transcoded, then by name.
func sortWorstFirst(ps []Player) {
	slices.SortStableFunc(ps, func(a, b Player) int {
		return cmp.Or(
			cmp.Compare(b.Transcoded, a.Transcoded),
			cmp.Compare(b.Transcoded*a.Plays, a.Transcoded*b.Plays),
			cmp.Compare(a.Name, b.Name),
		)
	})
}
