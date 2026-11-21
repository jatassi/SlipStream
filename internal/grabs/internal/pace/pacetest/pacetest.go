// Package pacetest is the queue and room simulator: a synthetic queue of
// search groups and a set of Indexers run minute by minute on a fake clock
// through pace.Merge, pace.Room.Starts, pacing.Gap and indexers.FitOf, the
// rules grabs and indexers run in production. It reports what an Admin
// would see over time: queries per Indexer per window, open runs, drain
// time and RSS sync's slots.
package pacetest

import (
	"cmp"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/pacing"
	"github.com/jatassi/slipstream/internal/searches"
)

// Indexer is one simulated Indexer.
type Indexer struct {
	Name   string
	AtOnce int
	Limit  indexers.Limit
	// Answer is how long each search takes to answer.
	Answer time.Duration
	// RSSEvery is RSS sync's cadence for it, zero while off. Each fetch
	// reads RSSPages pages, at least one.
	RSSEvery time.Duration
	RSSPages int
}

// Arrival is a trigger landing on group Group at At after the start.
// Entry.At is filled in.
type Arrival struct {
	At    time.Duration
	Group string
	Entry pace.Entry
}

// Sim is one simulation. Each minute, in order: answered queries complete,
// Manual searches reserve, RSS sync fetches, arrivals and due Backlog rows
// merge, runs start lane by lane, lines dispatch, and finished runs
// decide.
type Sim struct {
	Start    time.Time
	Indexers []Indexer
	Arrivals []Arrival
	// Manuals is when Manual searches run, after the start. Each reserves
	// one press query per Indexer, as Plan does.
	Manuals []time.Duration
	// Wait is the search wait.
	Wait              time.Duration
	Interval, Longest time.Duration
	For               time.Duration
}

// Report is what a Sim saw.
type Report struct {
	// Queries is each Indexer's search and recent queries per window of its
	// limit, counted from the start.
	Queries map[string][]int
	// Lines is each lane's lines by how they ended: "searched", or the
	// outcome kind that skipped them. A line still waiting isn't counted.
	Lines map[pace.Lane]map[string]int
	// Manual is each Indexer's Manual search lines the same way.
	Manual map[string]map[string]int
	// Starved is the minutes a background line waited Starved.
	Starved  int
	PeakOpen int
	// Drained is when no row was queued and no run open, every arrival
	// landed, zero when that never happened.
	Drained time.Duration
	RSS     map[string]RSSUse
	// Skips is the time each skip carried, in order, for limit_reached.
	Skips []time.Time
}

// RSSUse is one Indexer's RSS fetches due and made, and the slots they
// used.
type RSSUse struct {
	Due, Made, Slots int
}

type query struct {
	claim indexers.Claim
	at    time.Time
	done  time.Time
	line  *line
}

type line struct {
	run     *run
	indexer int
	lane    pace.Lane
	sent    bool
	settled bool
	heard   bool
}

type run struct {
	group   string
	started time.Time
	lines   []*line
}

type sim struct {
	Sim
	now      time.Time
	rows     map[string]pace.Row
	runs     []*run
	inflight [][]*query
	windows  [][]time.Time
	report   Report
}

// Run runs s and reports what it saw.
func (s Sim) Run() Report {
	m := &sim{
		Sim: s, rows: map[string]pace.Row{}, inflight: make([][]*query, len(s.Indexers)), windows: make([][]time.Time, len(s.Indexers)),
		report: Report{
			Queries: map[string][]int{}, Lines: map[pace.Lane]map[string]int{}, Manual: map[string]map[string]int{}, RSS: map[string]RSSUse{},
		},
	}
	for _, idx := range s.Indexers {
		m.report.Manual[idx.Name] = map[string]int{}
	}
	for _, l := range pace.Lanes {
		m.report.Lines[l] = map[string]int{}
	}
	last := time.Duration(0)
	for _, a := range s.Arrivals {
		last = max(last, a.At)
	}
	for t := time.Duration(0); t <= s.For; t += time.Minute {
		m.now = s.Start.Add(t)
		m.complete()
		if slices.Contains(s.Manuals, t) {
			m.manual()
		}
		m.rss(t)
		for _, a := range s.Arrivals {
			if a.At == t {
				m.merge(a.Group, a.Entry)
			}
		}
		m.backlog()
		m.start()
		m.dispatch()
		m.decide()
		m.report.PeakOpen = max(m.report.PeakOpen, len(m.runs))
		if m.report.Drained == 0 && t >= last && len(m.runs) == 0 && !m.queued() {
			m.report.Drained = t
		}
	}
	for i, idx := range s.Indexers {
		per := idx.Limit.Per.Duration()
		counts := make([]int, int(s.For/per)+1)
		for _, at := range m.windows[i] {
			counts[at.Sub(s.Start)/per]++
		}
		m.report.Queries[idx.Name] = counts
	}
	return m.report
}

func (m *sim) complete() {
	for i, qs := range m.inflight {
		m.inflight[i] = slices.DeleteFunc(qs, func(q *query) bool {
			if m.now.Before(q.done) {
				return false
			}
			if q.line != nil {
				q.line.settled, q.line.heard = true, true
			}
			return true
		})
	}
}

func (m *sim) reserve(i int, c indexers.Claim, l *line) {
	m.windows[i] = append(m.windows[i], m.now)
	m.inflight[i] = append(m.inflight[i], &query{claim: c, at: m.now, done: m.now.Add(m.Indexers[i].Answer), line: l})
}

// under is whether Indexer i's limit has a free slot, the rule Plan and
// RSS sync's fetches reserve by.
func (m *sim) under(i int) bool {
	l := m.Indexers[i].Limit
	if l.Count <= 0 {
		return true
	}
	n := 0
	for _, at := range m.windows[i] {
		if m.now.Sub(at) < l.Per.Duration() {
			n++
		}
	}
	return n < l.Count
}

func (m *sim) manual() {
	for i, idx := range m.Indexers {
		if !m.under(i) {
			m.report.Manual[idx.Name][string(indexers.LimitReached)]++
			continue
		}
		m.reserve(i, indexers.ClaimPress, nil)
		m.report.Manual[idx.Name]["searched"]++
	}
}

func (m *sim) rss(t time.Duration) {
	for i, idx := range m.Indexers {
		if idx.RSSEvery <= 0 || t%idx.RSSEvery != 0 {
			continue
		}
		use := m.report.RSS[idx.Name]
		use.Due++
		for page := range max(1, idx.RSSPages) {
			if !m.under(i) {
				break
			}
			if page == 0 {
				use.Made++
			}
			use.Slots++
			m.windows[i] = append(m.windows[i], m.now)
		}
		m.report.RSS[idx.Name] = use
	}
}

func (m *sim) merge(group string, e pace.Entry) {
	e.At = m.now
	r, _, promote := pace.Merge(m.rows[group], e, m.Interval, pacing.Spread("work", group, m.Interval))
	if promote {
		for _, ru := range m.runs {
			if ru.group == group {
				for _, l := range ru.lines {
					l.lane = e.Lane
				}
			}
		}
	}
	m.rows[group] = r
}

func (m *sim) backlog() {
	for _, g := range slices.Sorted(maps.Keys(m.rows)) {
		r := m.rows[g]
		if r.Lane == 0 && r.Searching == nil && !r.Searched.IsZero() && !m.now.Before(r.Due) {
			m.merge(g, pace.Entry{Lane: pace.Backlog, Trigger: searches.Backlog, Holds: true})
		}
	}
}

func (m *sim) queued() bool {
	for _, r := range m.rows {
		if r.Lane != 0 {
			return true
		}
	}
	return false
}

func (m *sim) start() {
	for _, lane := range pace.Lanes {
		var queued []string
		for g, r := range m.rows {
			if r.Lane == lane && r.Searching == nil {
				queued = append(queued, g)
			}
		}
		slices.SortFunc(queued, func(a, b string) int {
			return cmp.Or(m.rows[a].Queued.Compare(m.rows[b].Queued), cmp.Compare(a, b))
		})
		for _, g := range queued {
			if !m.starts(lane.Press()) {
				return
			}
			r := m.rows[g]
			ru := &run{group: g, started: m.now}
			for i := range m.Indexers {
				ru.lines = append(ru.lines, &line{run: ru, indexer: i, lane: lane})
			}
			r.Searching = &pace.Run{Lane: r.Lane, Holds: r.Holds, Whole: r.Items == nil, Items: r.Items}
			r.Lane = 0
			m.rows[g] = r
			m.runs = append(m.runs, ru)
		}
	}
}

func (m *sim) starts(press bool) bool {
	for i, idx := range m.Indexers {
		waiting := 0
		for _, ru := range m.runs {
			if l := ru.lines[i]; !l.sent && !l.settled {
				waiting++
			}
		}
		if !(pace.Room{AtOnce: idx.AtOnce, Waiting: waiting}).Starts(press) {
			return false
		}
	}
	return true
}

func (m *sim) dispatch() {
	for i := range m.Indexers {
		var pending []*line
		for _, ru := range m.runs {
			if l := ru.lines[i]; !l.sent && !l.settled {
				pending = append(pending, l)
			}
		}
		slices.SortStableFunc(pending, func(a, b *line) int {
			return cmp.Or(cmp.Compare(a.lane, b.lane), a.run.started.Compare(b.run.started))
		})
		for _, l := range pending {
			c := indexers.ClaimBackground
			if l.lane.Press() {
				c = indexers.ClaimPress
			}
			fit := indexers.FitOf(m.facts(i), c, m.now)
			switch {
			case fit.Reserve:
				l.sent = true
				m.reserve(i, c, l)
				m.report.Lines[l.lane]["searched"]++
				continue
			case fit.Skip != nil:
				l.settled = true
				m.report.Lines[l.lane][string(fit.Skip.Kind)]++
				if fit.Skip.Kind == indexers.LimitReached {
					m.report.Skips = append(m.report.Skips, fit.Skip.Until)
				}
				continue
			case fit.Wait.Starved:
				m.report.Starved++
			}
			break
		}
	}
}

func (m *sim) facts(i int) indexers.Facts {
	idx := m.Indexers[i]
	f := indexers.Facts{
		AtOnce: idx.AtOnce, InFlight: map[indexers.Claim]int{}, Stuck: len(m.inflight[i]) > 0, Limit: idx.Limit, Window: m.windows[i],
		RSS: indexers.RSSPlan{On: idx.RSSEvery > 0, Every: idx.RSSEvery, LastPages: max(1, idx.RSSPages)},
	}
	for _, q := range m.inflight[i] {
		f.InFlight[q.claim]++
		if m.now.Before(q.at.Add(m.Wait)) {
			f.Stuck = false
		}
	}
	return f
}

func (m *sim) decide() {
	m.runs = slices.DeleteFunc(m.runs, func(ru *run) bool {
		heard := false
		for _, l := range ru.lines {
			if !l.settled {
				return false
			}
			heard = heard || l.heard
		}
		result := pacing.Unheard
		if heard {
			result = pacing.Answered
		}
		r := m.rows[ru.group]
		r.Searching, r.Searched = nil, m.now
		r.Gap = pacing.Gap(r.Gap, m.Interval, m.Longest, result, false, 0)
		r.Due = m.now.Add(r.Gap)
		m.rows[ru.group] = r
		return true
	})
}
