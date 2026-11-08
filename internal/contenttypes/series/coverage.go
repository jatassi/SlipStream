package series

import (
	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/works"
)

// episodes are a series' stored episodes as releases name them: by season
// and number in the order searchedAs reads, and by TVDB's absolute number.
type episodes struct {
	at       map[episodeAt]works.ItemID
	seasons  map[int][]works.ItemID
	absolute map[int]works.ItemID
}

func episodesOf(order tvdb.Order, eps []db.SeriesEpisode) episodes {
	e := episodes{at: map[episodeAt]works.ItemID{}, seasons: map[int][]works.ItemID{}, absolute: map[int]works.ItemID{}}
	for _, ep := range eps {
		at := searchedAs(order, ep)
		e.at[at] = ep.ItemID
		e.seasons[at.season] = append(e.seasons[at.season], ep.ItemID)
		if ep.AbsoluteNumber != nil {
			e.absolute[int(*ep.AbsoluteNumber)] = ep.ItemID
		}
	}
	return e
}

// covered is the stored episodes n's marks name, as Sonarr's GetEpisodes
// maps them: absolute numbers when the name gives any, else a season
// pack's every episode, each season's of a multi-season pack, or each
// episode first to last. A partial season and an absolute special cover
// nothing.
func (e episodes) covered(n release.Name) []works.ItemID {
	ms := make([]marked, len(n.Marks))
	for i, mk := range n.Marks {
		ms[i] = markedBy(mk.Key)
	}
	for _, m := range ms {
		if m.kind == markAbsolute {
			return e.absolutes(m)
		}
	}
	var out []works.ItemID
	for _, m := range ms {
		switch m.kind {
		case markSeason:
			out = append(out, e.seasons[m.season]...)
		case markSeasons:
			for s := m.season; s <= m.last; s++ {
				out = append(out, e.seasons[s]...)
			}
		case markEpisodes:
			for ep := m.first; ep <= m.last; ep++ {
				if id, ok := e.at[episodeAt{m.season, ep}]; ok {
					out = append(out, id)
				}
			}
		case markPart, markAbsolute, markSpecial, markAired:
		}
	}
	return out
}

func (e episodes) absolutes(m marked) []works.ItemID {
	var out []works.ItemID
	for a := m.first; a <= m.last; a++ {
		if id, ok := e.absolute[a]; ok {
			out = append(out, id)
		}
	}
	return out
}
