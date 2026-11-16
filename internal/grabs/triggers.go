package grabs

import (
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/searches"
)

// laneRule is what a trigger does to the queue. laneRules is the only
// place a trigger's lane, gap reset, gone cells and Holds are decided.
type laneRule struct {
	// single is the lane of a call that asks for one Work, several of one
	// that asks for more. Zero queues nothing: a gap reset only.
	single, several pace.Lane
	// resets resets the gap to Backlog search's interval.
	resets bool
	// gone opens gone cells.
	gone bool
	// holds is whether a Target's Hold applies to its runs' grabs.
	holds bool
	// paced opens stopped cells too, which only Backlog search searches.
	paced bool
}

var laneRules = map[searches.Trigger]laneRule{
	searches.Search:      {single: pace.Now, several: pace.Now, resets: true, gone: true},
	searches.SearchAll:   {single: pace.Bulk, several: pace.Bulk, resets: true},
	searches.Add:         {single: pace.Now, several: pace.Now, resets: true, holds: true},
	searches.Monitor:     {single: pace.Now, several: pace.Backlog, resets: true, holds: true},
	searches.Profile:     {single: pace.Now, several: pace.Backlog, resets: true, holds: true},
	searches.Editions:    {single: pace.Now, several: pace.Backlog, resets: true, holds: true},
	searches.Retarget:    {resets: true},
	searches.ReleaseDate: {single: pace.Backlog, several: pace.Backlog, holds: true},
	searches.Refresh:     {single: pace.Backlog, several: pace.Backlog, holds: true},
	searches.Backlog:     {single: pace.Backlog, several: pace.Backlog, holds: true, paced: true},
	// A failure's lane comes from its Code's failure rule.
	searches.FailedGrab: {holds: true},
	searches.RSS:        {holds: true},
}
