package series

import (
	"cmp"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/works"
)

// episodes are a series' stored episodes as releases name them: by season
// and number in the order searchedAs reads, by TVDB's absolute number,
// and by air date in season and number order.
type episodes struct {
	at       map[episodeAt]works.ItemID
	seasons  map[int][]works.ItemID
	absolute map[int]works.ItemID
	aired    map[string][]dated
}

// dated is a stored episode an air date names.
type dated struct {
	at episodeAt
	id works.ItemID
}

func episodesOf(order tvdb.Order, eps []db.SeriesEpisode) episodes {
	e := episodes{at: map[episodeAt]works.ItemID{}, seasons: map[int][]works.ItemID{}, absolute: map[int]works.ItemID{}, aired: map[string][]dated{}}
	for _, ep := range eps {
		at := searchedAs(order, ep)
		e.at[at] = ep.ItemID
		e.seasons[at.season] = append(e.seasons[at.season], ep.ItemID)
		if ep.AbsoluteNumber != nil {
			e.absolute[int(*ep.AbsoluteNumber)] = ep.ItemID
		}
		if ep.AirDate != nil {
			e.aired[*ep.AirDate] = append(e.aired[*ep.AirDate], dated{at, ep.ItemID})
		}
	}
	for _, on := range e.aired {
		slices.SortFunc(on, func(a, b dated) int {
			return cmp.Or(cmp.Compare(a.at.season, b.at.season), cmp.Compare(a.at.episode, b.at.episode))
		})
	}
	return e
}

// covered is the stored episodes n's marks name, as Sonarr's GetEpisodes
// maps them: absolute numbers when the name gives any, else a season
// pack's every episode, each season's of a multi-season pack, each
// episode first to last, or the episode an air date names. A partial
// season and an absolute special cover nothing.
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
		case markAired:
			if id, ok := e.onAirDate(m); ok {
				out = append(out, id)
			}
		case markPart, markAbsolute, markSpecial:
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

// onAirDate ports Sonarr's FindOneByAirDate: the one episode on the date,
// else the one regular episode when the name gives no part, else the
// part's place among the regular episodes in season and number order.
// Any other date names no episode, where Sonarr throws.
func (e episodes) onAirDate(m marked) (works.ItemID, bool) {
	on := e.aired[m.aired.Format(time.DateOnly)]
	if len(on) == 1 {
		return on[0].id, true
	}
	regular := slices.DeleteFunc(slices.Clone(on), func(d dated) bool { return d.at.season == 0 })
	switch {
	case len(regular) == 1 && m.first == 0:
		return regular[0].id, true
	case m.first > 0 && m.first <= len(regular):
		return regular[m.first-1].id, true
	}
	return works.ItemID{}, false
}
