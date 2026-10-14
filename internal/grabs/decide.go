package grabs

import (
	"cmp"
	"regexp"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/targets"
)

var resolutionToken = regexp.MustCompile(`(?i)\b(2160p|4k|uhd|1080p|720p|576p|480p)\b`)

var resolutionOf = map[string]targets.Resolution{
	"2160p": targets.R2160p, "4k": targets.R2160p, "uhd": targets.R2160p,
	"1080p": targets.R1080p, "720p": targets.R720p, "576p": targets.R576p, "480p": targets.R480p,
}

// resolution reads a release title's resolution from its first token,
// false when it holds none.
func resolution(title string) (targets.Resolution, bool) {
	m := resolutionToken.FindString(title)
	if m == "" {
		return 0, false
	}
	return resolutionOf[strings.ToLower(m)], true
}

// Pick is the candidate a search grabs and every open cell's Target it
// meets, in Profile order.
type Pick struct {
	Item    indexers.Item
	Targets []targets.TargetID
}

type Reason string

const (
	ResolutionUnknown Reason = "resolution_unknown"
	WrongWork         Reason = "wrong_work"
	MeetsNothing      Reason = "meets_nothing"
)

type Rejection struct {
	Title  string
	Reason Reason
}

// decide picks the best candidate for open, one Item's cells a search may
// fill, in rank order. A candidate is rejected when its title holds no
// resolution, when an id it carries differs from the same id in terms, or
// when it meets no open cell's Target. The rest rank by the first open
// cell met, then the resolution's place in that Target, then seeders and
// size descending, then title, so ties replay.
func decide(open []files.Cell, terms indexers.Terms, candidates []indexers.Item) (*Pick, []Rejection) {
	type ranked struct {
		item             indexers.Item
		cell, resolution int
		targets          []targets.TargetID
	}
	var keep []ranked
	var rejected []Rejection
	for _, it := range candidates {
		res, ok := resolution(it.Title)
		if !ok {
			rejected = append(rejected, Rejection{it.Title, ResolutionUnknown})
			continue
		}
		if wrongWork(it, terms) {
			rejected = append(rejected, Rejection{it.Title, WrongWork})
			continue
		}
		r := ranked{item: it, cell: -1}
		for i, c := range open {
			at := slices.Index(c.Target.Resolutions, res)
			if at < 0 {
				continue
			}
			if r.cell < 0 {
				r.cell, r.resolution = i, at
			}
			r.targets = append(r.targets, c.Target.ID)
		}
		if r.cell < 0 {
			rejected = append(rejected, Rejection{it.Title, MeetsNothing})
			continue
		}
		keep = append(keep, r)
	}
	if len(keep) == 0 {
		return nil, rejected
	}
	best := slices.MinFunc(keep, func(a, b ranked) int {
		return cmp.Or(
			cmp.Compare(a.cell, b.cell),
			cmp.Compare(a.resolution, b.resolution),
			cmp.Compare(b.item.Seeders, a.item.Seeders),
			cmp.Compare(b.item.Size, a.item.Size),
			strings.Compare(a.item.Title, b.item.Title),
		)
	})
	return &Pick{Item: best.item, Targets: best.targets}, rejected
}

func wrongWork(it indexers.Item, terms indexers.Terms) bool {
	for k, v := range it.IDs {
		if want, ok := terms.IDs[k]; ok && want != v {
			return true
		}
	}
	return false
}
