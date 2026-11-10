package grabs

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// routedNowhere is where Tracker routing sends a Grab while no Node may
// take it.
const routedNowhere = "Where Tracker routing sends it"

// manualView is in worked out around pins and written in words. It is
// pure: the GET and the grab Write both call it.
func manualView(in manualIn, pins []Pin) (ManualView, error) {
	s, err := worked(in, pins)
	if err != nil {
		return ManualView{}, err
	}
	m := in.manual
	v := ManualView{
		ID: m.ID, Title: s.title(), Query: deref(m.Query), Started: m.StartedAt.Time(), StartedBy: Actor{Name: deref(m.StartedByName)},
		Queryable: len(in.runs) == 1, Folded: s.folded(),
	}
	if !m.StartedBy.IsZero() {
		v.StartedBy.Account = &m.StartedBy
	}
	v.Indexers, v.Searching, v.Line = s.indexerLines()
	if v.Plan, v.Pins, err = s.plan(); err != nil {
		return ManualView{}, err
	}
	for _, rs := range s.runs {
		v.Releases += len(rs.as)
	}
	return v, nil
}

// itemNames is each Item of the tree in words: the Work's label when its
// type's Works hold one Item, else the Item noun's initial and the Item's
// place in its level, "E05".
func (s sheet) itemNames() map[works.ItemID]string {
	out := map[works.ItemID]string{}
	place := map[works.LevelID]int{}
	initial := strings.ToUpper(s.in.nouns.Item.One[:1])
	for _, it := range s.in.tree.Items {
		if s.in.nouns.Items == works.ItemsOne {
			out[it.ID] = s.in.tree.Label
			continue
		}
		place[it.LevelID]++
		out[it.ID] = fmt.Sprintf("%s%02d", initial, place[it.LevelID])
	}
	return out
}

// title is the scope in words: "Heat (1995)", "Season 2 of Severance".
func (s sheet) title() string {
	t := s.in.tree
	if s.in.nouns.Items == works.ItemsOne || len(s.runs) == len(liveItems(t)) {
		return t.Label
	}
	var items []works.ItemID
	for _, rs := range s.runs {
		items = append(items, rs.run.ItemID)
	}
	return s.itemsWords(items, true) + " of " + t.Label
}

func liveItems(t works.Tree) []works.Item {
	var out []works.Item
	for _, it := range t.Items {
		if it.Live() {
			out = append(out, it)
		}
	}
	return out
}

// itemsWords is items in words, a run of three or more in one level
// written "E08 to E10": "E05, E06 and E08 to E10". Each level's Items say
// which level they are in when the sheet spans more than one, or when
// named is set. A level whose every live Item is among them is its label.
func (s sheet) itemsWords(items []works.ItemID, named bool) string {
	names := s.itemNames()
	t := s.in.tree
	levels := map[works.LevelID]string{}
	for _, l := range t.Levels {
		levels[l.ID] = l.Label
	}
	var order []works.LevelID
	byLevel := map[works.LevelID][]int{}
	for i, it := range t.Items {
		if !slices.Contains(items, it.ID) {
			continue
		}
		if _, ok := byLevel[it.LevelID]; !ok {
			order = append(order, it.LevelID)
		}
		byLevel[it.LevelID] = append(byLevel[it.LevelID], i)
	}
	spans := named || len(order) > 1
	var parts []string
	for _, l := range order {
		at := byLevel[l]
		if named && levels[l] != "" && len(at) == s.liveIn(l) {
			parts = append(parts, levels[l])
			continue
		}
		var words []string
		for i := 0; i < len(at); {
			k := i
			for k+1 < len(at) && at[k+1] == at[k]+1 {
				k++
			}
			if k-i >= 2 {
				words = append(words, names[t.Items[at[i]].ID]+" to "+names[t.Items[at[k]].ID])
			} else {
				for _, a := range at[i : k+1] {
					words = append(words, names[t.Items[a].ID])
				}
			}
			i = k + 1
		}
		part := joinWords(words)
		if spans && levels[l] != "" {
			part += " of " + levels[l]
		}
		parts = append(parts, part)
	}
	return joinWords(parts)
}

func (s sheet) liveIn(l works.LevelID) int {
	n := 0
	for _, it := range liveItems(s.in.tree) {
		if it.LevelID == l {
			n++
		}
	}
	return n
}

// cellName is c's Target in words, with its Edition unless it takes any
// cut: "1080p", "1080p (Director's Cut)".
func cellName(c files.Cell) string {
	if c.Edition == targets.AnyEdition {
		return c.Target.Name
	}
	return c.Target.Name + " (" + string(c.Edition) + ")"
}

// namesOf is the distinct names of cs, in order.
func namesOf(cs []files.Cell) []string {
	var out []string
	for _, c := range cs {
		if n := cellName(c); !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out
}

// gets is what a Grab filling cs, cells of one Item, gets: "It gets you
// 1080p and 720p for E05.", "It gets you 1080p for Heat (1995) as an
// upgrade.", "It gets you 1080p for E05, and 720p as an upgrade."
func (s sheet) gets(cs []files.Cell) string {
	var fresh, better []files.Cell
	for _, c := range cs {
		if c.State == files.Upgradable {
			better = append(better, c)
		} else {
			fresh = append(fresh, c)
		}
	}
	item := s.itemsWords([]works.ItemID{cs[0].Item.ID}, false)
	switch {
	case len(fresh) == 0:
		return "It gets you " + joinWords(namesOf(better)) + " for " + item + " as an upgrade."
	case len(better) == 0:
		return "It gets you " + joinWords(namesOf(fresh)) + " for " + item + "."
	}
	return "It gets you " + joinWords(namesOf(fresh)) + " for " + item + ", and " + joinWords(namesOf(better)) + " as an upgrade."
}

// gapWords is a gap's cells in words, all of one state and one Target:
// "Missing: 720p for E05, E06 and E08 to E10.", "Could be better: 1080p
// SDR.", "Failed: 1080p for E10."
func (s sheet) gapWords(cs []files.Cell) string {
	label := "Missing"
	switch cs[0].State {
	case files.Upgradable:
		label = "Could be better"
	case files.Failed:
		label = "Failed"
	case files.Unreleased, files.Missing, files.Downloading, files.Met:
	}
	if s.in.nouns.Items == works.ItemsOne {
		return label + ": " + cellName(cs[0]) + "."
	}
	items := make([]works.ItemID, len(cs))
	for i, c := range cs {
		items[i] = c.Item.ID
	}
	return label + ": " + cellName(cs[0]) + " for " + s.itemsWords(items, false) + "."
}

// causeAt is the plain cause nothing fills cell w of rs: "No Indexer
// listed a 720p Release.", "Every 1080p Release is on the Blocklist."
func (rs runSheet) causeAt(w int, gap planned) string {
	if gap.waits != nil {
		return gap.waits.Sentence
	}
	name := cellName(rs.item[w])
	var blocked, lifted bool
	for _, st := range rs.st {
		blocked = blocked || st[w].kind == standsBlocked
		lifted = lifted || st[w].kind == standsLifted || st[w].kind == standsBlocked && len(st[w].lifts) > 0
	}
	switch {
	case lifted:
		return "Every " + name + " Release needs an override. Other choices lists them."
	case blocked:
		return "Every " + name + " Release is on the Blocklist."
	case rs.run.Finished.IsZero():
		return "No Indexer has listed a " + name + " Release yet."
	}
	return "No Indexer listed a " + name + " Release."
}

// plan is the sheet's plan in words, and the pins still in effect.
func (s sheet) plan() (ManualPlan, []Pin, error) {
	var p ManualPlan
	var pins []Pin
	walked, filled, grabs := 0, 0, 0
	var unfound []string
	gaps := map[string]int{}
	for _, rs := range s.runs {
		walked += len(rs.walk)
		for k, g := range rs.live {
			pins = append(pins, Pin{Cell: keyOf(rs.item[k]), Release: ReleaseRef{Run: rs.run.ID, Position: rs.as[g].at}})
		}
		for k, g := range rs.held {
			if s.in.cell == nil || *s.in.cell == keyOf(rs.item[k]) {
				p.Held = append(p.Held, heldCell(rs.item[k], g))
			}
		}
		for _, r := range rs.rows {
			lead := r.cells[0]
			row := Row{Pinned: r.pinned, Failure: rs.failure[lead]}
			for _, k := range r.cells {
				row.Cells = append(row.Cells, keyOf(rs.item[k]))
			}
			except := r.group
			if r.group < 0 {
				cause := rs.causeAt(lead, r)
				key := rs.item[lead].State.String() + "/" + cellName(rs.item[lead]) + "/" + cause
				if i, ok := gaps[key]; ok {
					p.Rows[i].Cells = append(p.Rows[i].Cells, row.Cells...)
					continue
				}
				gaps[key] = len(p.Rows)
				row.Cause = cause
				if n := cellName(rs.item[lead]); !slices.Contains(unfound, n) {
					unfound = append(unfound, n)
				}
			} else {
				grabs++
				filled += len(r.cells)
				cs := make([]files.Cell, len(r.cells))
				for i, k := range r.cells {
					cs[i] = rs.item[k]
				}
				row.Outcome = s.gets(cs)
				row.Pick = &PlanPick{
					Release: s.details(rs, r.group), Lifts: rs.worded(r.stance.lifts, r.group), Edition: r.stance.edition,
					Routed: s.in.routed[r.pick.Release.Indexer],
				}
			}
			choices, err := rs.choicesAt(lead, except)
			if err != nil {
				return ManualPlan{}, nil, err
			}
			for _, g := range choices {
				row.Choices = append(row.Choices, s.choice(rs, lead, g))
			}
			p.Rows = append(p.Rows, row)
		}
	}
	for i, r := range p.Rows {
		if r.Pick == nil {
			cs := make([]files.Cell, len(r.Cells))
			for k, key := range r.Cells {
				cs[k] = s.cellOf(key)
			}
			p.Rows[i].Missing = s.gapWords(cs)
		}
	}
	p.Headline = headline(walked, filled, grabs, len(p.Held), unfound)
	return p, pins, nil
}

func (s sheet) cellOf(key files.CellKey) files.Cell {
	return s.in.cells[slices.IndexFunc(s.in.cells, func(c files.Cell) bool { return keyOf(c) == key })]
}

// headline counts a plan's Grabs and cells: "2 Grabs get you
// everything.", "1 Grab gets you 2 of the 3 files you're missing.
// SlipStream hasn't found a Release for 4K yet."
func headline(walked, filled, grabs, held int, unfound []string) string {
	missing := func() string { return "SlipStream hasn't found a Release for " + joinWords(unfound) + " yet." }
	switch {
	case walked == 0 && held == 0:
		return "Every Target here already has its file."
	case walked == 0:
		return "Every Target here has its file or a Grab downloading one."
	case grabs == 0:
		return missing()
	}
	n := plural(grabs, "1 Grab gets", "%d Grabs get")
	if filled == walked {
		return n + " you everything."
	}
	return fmt.Sprintf("%s you %d of the %d files you're missing. %s", n, filled, walked, missing())
}

// plural is one when n is 1, else many with n.
func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return fmt.Sprintf(many, n)
}

// heldCell is cell, which unfinished Grab g holds, in words: "A Grab is
// downloading Heat.1995.1080p.WEB for 1080p."
func heldCell(c files.Cell, g db.ListCellGrabsOfWorkRow) HeldCell {
	verb := "downloading"
	if g.Stage == Fetching {
		verb = "fetching"
	}
	return HeldCell{
		Cell: keyOf(c), Grab: g.ID, Stage: g.Stage,
		Words: "A Grab is " + verb + " " + g.ReleaseTitle + " for " + cellName(c) + ".",
	}
}

// worded is rs, reasons about as[g], in words.
func (rs runSheet) worded(reasons []Reason, g int) []Worded {
	out := make([]Worded, len(reasons))
	for i, r := range reasons {
		out[i] = Worded{Reason: r, Words: reasonWords(r, rs.targetOf(r.Target), rs.as[g].read.Facts)}
	}
	return out
}

// targetOf is the Target of id among the run's cells, nil for an own
// reason or a Target the Item has no cell of.
func (rs runSheet) targetOf(id targets.TargetID) *targets.Target {
	for i := range rs.item {
		if rs.item[i].Target.ID == id {
			return &rs.item[i].Target
		}
	}
	return nil
}

// choice is as[g] offered at cell w of rs.
func (s sheet) choice(rs runSheet, w, g int) Choice {
	st := rs.st[g][w]
	c := Choice{
		Pin:     Pin{Cell: keyOf(rs.item[w]), Release: ReleaseRef{Run: rs.run.ID, Position: rs.as[g].at}},
		Release: s.details(rs, g), Lifts: rs.worded(st.lifts, g), Edition: st.edition,
	}
	p, _, _ := rs.offer(g, w)
	if st, ok := s.in.standings[p.Release.Indexer]; ok && !st.Automatic {
		c.Note = "From " + rs.heard[p.at] + ", which automatic search doesn't use."
	}
	return c
}

// details is as[g] of rs with its facts and its verdicts in words.
func (s sheet) details(rs runSheet, g int) ReleaseDetails {
	a := rs.as[g]
	r := a.Release
	d := ReleaseDetails{
		Ref: ReleaseRef{Run: rs.run.ID, Position: a.at}, Title: r.Title, Words: release.Describe(a.read), Size: r.Size,
		Published: r.Published, Seeders: r.Seeders, Freeleech: r.Factors.Download == 0, Formats: a.formats.Matched,
		Indexers: []string{rs.heard[a.at]}, Verdicts: rs.verdicts(g), LostAt: rs.lostAt(g),
	}
	for _, c := range a.copies {
		if n := rs.heard[c.at]; !slices.Contains(d.Indexers, n) {
			d.Indexers = append(d.Indexers, n)
		}
	}
	if src := r.Source; src != nil {
		d.Tracker = src.Name
	}
	if e, ok := s.in.blocks.of(a); ok {
		d.Blocked = "It's been on the Blocklist since " + s.dayOf(e.Added) + " because " + becauseClause(e)
	}
	return d
}

// dayOf is t as a day: "2 Nov", with its year when it isn't this year's.
func (s sheet) dayOf(t time.Time) string {
	if t.Year() == s.in.now.Year() {
		return t.Format("2 Jan")
	}
	return t.Format("2 Jan 2006")
}

// becauseClause is e's cause as a clause after "because": "TorrentLeech
// reported it unregistered.", "it never downloaded any data."
func becauseClause(e BlockEntry) string {
	switch {
	case e.Cause == CauseUnregistered && e.IndexerName == "":
		return "its tracker reported it unregistered."
	case e.Cause == CauseNoPayload:
		return "it never downloaded any data."
	case e.Cause == CauseMarkedFailed && e.By.Name == "":
		return "it was marked failed through the API key."
	case e.Cause == CauseByHand && e.By.Name == "":
		return "it was blocklisted through the API key when its Grab was canceled."
	}
	return e.because()
}

// verdicts is as[g]'s verdict at each Target of its Item, at the Target's
// first cell whose Edition takes its cut: "It would fill 1080p.", "It
// meets 720p. It isn't better than the file you have." A Release with a
// hard refusal of its own has the one verdict of that refusal.
func (rs runSheet) verdicts(g int) []string {
	j := rs.js[g]
	if len(j.verdicts) == 0 {
		var hard []Reason
		for _, r := range j.refusals {
			if r.Code.Hard() {
				hard = append(hard, r)
			}
		}
		return []string{wordsOf(rs.worded(hard, g))}
	}
	var seen []targets.TargetID
	var out []string
	for k, c := range rs.item {
		if slices.Contains(seen, c.Target.ID) {
			continue
		}
		at := slices.IndexFunc(rs.item, func(o files.Cell) bool { return o.Target.ID == c.Target.ID && j.verdicts[keyIndex(rs.item, o)].wanted })
		if at < 0 {
			at = k
		}
		seen = append(seen, c.Target.ID)
		out = append(out, rs.verdictAt(g, at))
	}
	return out
}

func keyIndex(cs []files.Cell, c files.Cell) int {
	return slices.IndexFunc(cs, func(o files.Cell) bool { return keyOf(o) == keyOf(c) })
}

// verdictAt is as[g]'s verdict at cell k in a sentence.
func (rs runSheet) verdictAt(g, k int) string {
	c, st, v := rs.item[k], rs.st[g][k], rs.js[g].verdicts[k]
	name := cellName(c)
	switch {
	case st.kind == standsAuto || st.kind == standsBlocked && len(st.lifts) == 0 && st.edition == "":
		if c.State == files.Upgradable {
			return "It would fill " + name + " as an upgrade."
		}
		return "It would fill " + name + "."
	case !v.v.Meets():
		return wordsOf(rs.worded(st.why, g))
	}
	var says []string
	for _, w := range rs.worded(slices.Concat(st.why, st.lifts), g) {
		if w.Code == targets.NotAnUpgrade {
			says = append(says, "It isn't better than the file you have.")
		} else {
			says = append(says, w.Words)
		}
	}
	if st.edition != "" {
		says = append(says, reasonWords(Reason{Reason: targets.Reason{Code: targets.CutNotWanted, Value: string(v.cut)}, Target: c.Target.ID}, &c.Target, rs.as[g].read.Facts))
	}
	return "It meets " + name + ". " + strings.Join(says, " ")
}

func wordsOf(ws []Worded) string {
	out := make([]string, len(ws))
	for i, w := range ws {
		out[i] = w.Words
	}
	return strings.Join(out, " ")
}

// standsFirst is the first cell of rs as[g] stands at, -1 for none.
func (rs runSheet) standsFirst(g int) int {
	return slices.IndexFunc(rs.st[g], func(s stance) bool { return s.kind != standsRefused })
}

// lostAt is where as[g] lost to the Release the plan fills its first
// cell with, nil when no other Release fills it.
func (rs runSheet) lostAt(g int) *Loss {
	f := rs.standsFirst(g)
	if f < 0 {
		return nil
	}
	for _, r := range rs.rows {
		if r.group >= 0 && r.group != g && slices.Contains(r.cells, f) {
			ref, own := rs.js[r.group], rs.js[g]
			ref.first, own.first = f, f
			return lostAt(cells{open: rs.item}, &ref, &own)
		}
	}
	return nil
}

// folded is the Releases no cell in scope takes, auto or lifted, in
// groups by why: "8 are Bluray-2160p, which no Target here takes."
func (s sheet) folded() Folded {
	counts := map[string]int{}
	fault := map[string]bool{}
	var order []string
	add := func(key string, isFault bool) {
		if _, ok := counts[key]; !ok {
			order = append(order, key)
		}
		counts[key]++
		fault[key] = isFault
	}
	n := 0
	for _, rs := range s.runs {
		for g := range rs.as {
			why, blocked, ok := rs.foldedWhy(g)
			if !ok {
				continue
			}
			n++
			add(why, blocked)
		}
	}
	if n == 0 {
		return Folded{}
	}
	slices.SortStableFunc(order, func(a, b string) int { return cmp.Compare(counts[b], counts[a]) })
	f := Folded{Line: "SlipStream wouldn't grab " + plural(n, "1 more Release.", "%d more Releases.")}
	for _, key := range order {
		f.Groups = append(f.Groups, FoldGroup{Words: s.foldWords(key, counts[key]), Fault: fault[key]})
	}
	return f
}

// foldedWhy is why the plan's cells take as[g] neither auto nor lifted,
// as a fold key, and whether the Blocklist holds it. ok is false when a
// cell in scope takes it.
func (rs runSheet) foldedWhy(g int) (why string, blocked bool, ok bool) {
	a := rs.as[g]
	cellsIn := rs.walk
	if len(cellsIn) == 0 {
		cellsIn = make([]int, len(rs.item))
		for k := range cellsIn {
			cellsIn[k] = k
		}
	}
	for k, st := range rs.st[g] {
		if rs.fill[k] && st.grabbable() {
			return "", false, false
		}
		blocked = blocked || st.kind == standsBlocked
	}
	if blocked || a.blocked() {
		return string(targets.Blocklisted), true, true
	}
	for _, r := range a.refusals {
		if r.Code.Hard() || r.Code == targets.CoversMore {
			return string(r.Code), false, true
		}
	}
	if len(cellsIn) == 0 {
		return string(targets.NotAnUpgrade), false, true
	}
	st := rs.st[g][cellsIn[0]]
	for _, r := range st.why {
		if r.Code == targets.QualityNotInTarget {
			if q, ok := a.read.Facts.Quality.Get(); ok {
				return "quality:" + q.Info().Name, false, true
			}
			return "quality:", false, true
		}
	}
	if len(st.why) > 0 {
		return string(st.why[0].Code), false, true
	}
	return string(targets.NotAnUpgrade), false, true
}

// foldWords is n Releases folded for key in a sentence.
func (s sheet) foldWords(key string, n int) string {
	item := strings.ToLower(s.in.nouns.Item.One)
	if q, ok := strings.CutPrefix(key, "quality:"); ok {
		if q == "" {
			return plural(n, "1 has", "%d have") + " no Quality SlipStream can read."
		}
		return plural(n, "1 is", "%d are") + " " + q + ", which no Target here takes."
	}
	switch targets.ReasonCode(key) {
	case targets.Blocklisted:
		return plural(n, "1 is", "%d are") + " on the Blocklist."
	case targets.WrongWork:
		return plural(n, "1 names", "%d name") + " another Work."
	case targets.WrongItem:
		return plural(n, "1 names", "%d name") + " another " + item + "."
	case targets.CoversMore:
		return plural(n, "1 is a pack", "%d are packs") + " of more than one " + item + ", and SlipStream can't import packs yet."
	case targets.FilterFailed:
		return plural(n, "1 fails", "%d fail") + " a filter of every Target here."
	case targets.NotAnUpgrade:
		return plural(n, "1 isn't", "%d aren't") + " better than the files you have."
	}
	return plural(n, "1 is", "%d are") + " refused at every Target here."
}

// indexerLines is each Indexer across the runs, whether one hasn't
// responded yet, and the Searching line: "Searching 6 Indexers. 4 have
// responded.", "SlipStream searched 6 Indexers. 2 didn't respond."
func (s sheet) indexerLines() ([]IndexerLine, bool, string) {
	type across struct {
		line     IndexerLine
		outcomes []*indexers.Outcome
	}
	var order []string
	by := map[string]*across{}
	for _, rs := range s.runs {
		for _, l := range rs.run.Indexers {
			key := l.Indexer.String() + l.Name
			a, ok := by[key]
			if !ok {
				order = append(order, key)
				a = &across{line: IndexerLine{Indexer: l.Indexer, Name: l.Name, Automatic: s.in.standings[l.Indexer].Automatic}}
				by[key] = a
			}
			a.outcomes = append(a.outcomes, l.Outcome)
		}
	}
	var out []IndexerLine
	searching, responded, late := false, 0, 0
	for _, key := range order {
		a := by[key]
		a.line.Done = !slices.Contains(a.outcomes, nil)
		a.line.Sentence = s.acrossWords(a.line.Name, a.outcomes)
		for _, o := range a.outcomes {
			a.line.Failed = a.line.Failed || o != nil && (o.Kind == indexers.IsFailing || o.Kind == indexers.IsStopped)
		}
		if a.line.Done {
			responded++
		} else {
			searching = true
		}
		if slices.ContainsFunc(a.outcomes, func(o *indexers.Outcome) bool { return o != nil && o.Kind == indexers.Late }) {
			late++
		}
		out = append(out, a.line)
	}
	return out, searching, searchingWords(out, searching, responded, late)
}

func searchingWords(lines []IndexerLine, searching bool, responded, late int) string {
	switch {
	case len(lines) == 1 && searching:
		return "Searching " + lines[0].Name + "."
	case len(lines) == 1:
		return "SlipStream searched " + lines[0].Name + "."
	case searching && responded == 0:
		return fmt.Sprintf("Searching %d Indexers. None has responded yet.", len(lines))
	case searching:
		return fmt.Sprintf("Searching %d Indexers. %s responded.", len(lines), plural(responded, "1 has", "%d have"))
	case late > 0:
		return fmt.Sprintf("SlipStream searched %d Indexers. %s.", len(lines), plural(late, "1 didn't respond", "%d didn't respond"))
	}
	return fmt.Sprintf("SlipStream searched %d Indexers.", len(lines))
}

// acrossWords is an Indexer's outcomes across the runs in words: its one
// outcome's line, or "TorrentLeech searched 6 of 10 episodes. Limit
// reached until 14:05."
func (s sheet) acrossWords(name string, os []*indexers.Outcome) string {
	first := s.in.line(os[0])
	if !slices.ContainsFunc(os, func(o *indexers.Outcome) bool { return s.in.line(o) != first }) {
		return first
	}
	searched := 0
	other := ""
	for _, o := range os {
		if o != nil && (o.Kind == indexers.Found || o.Kind == indexers.NoReleases) {
			searched++
		} else if other == "" {
			other = s.in.line(o)
		}
	}
	return fmt.Sprintf("%s searched %d of %d %s. %s.", name, searched, len(os), strings.ToLower(s.in.nouns.Item.Many), other)
}

// offers is every Release of the sheet, one per group of copies: each
// run's in rank order at the first cell it stands at, then those that
// stand at none, run by run.
func (s sheet) offers() []Offer {
	var ranked, rest []Offer
	for _, rs := range s.runs {
		var js []judged
		at := map[int]int{}
		for g := range rs.as {
			f := rs.standsFirst(g)
			if f < 0 {
				rest = append(rest, s.offer(rs, g, -1))
				continue
			}
			j := rs.js[g]
			j.first = f
			at[j.at] = g
			js = append(js, j)
		}
		rank(cells{open: rs.item}, js)
		for _, j := range js {
			o := s.offer(rs, at[j.at], j.first)
			o.Rank = len(ranked) + 1
			ranked = append(ranked, o)
		}
	}
	return append(ranked, rest...)
}

// offer is as[g] of rs as All Releases lists it, f the first cell it
// stands at, -1 for none.
func (s sheet) offer(rs runSheet, g, f int) Offer {
	o := Offer{Release: s.details(rs, g)}
	if f < 0 {
		o.Grab.Refusal = s.refusal(rs, g)
		return o
	}
	v := rs.js[g].verdicts[f]
	o.Score = v.v.Score()
	if rank, ok := v.v.Rank(); ok {
		o.Quality = len(rs.item[f].Target.Tiers) - int(rank)
	}
	lead := rs.st[g][f]
	o.Grab.Lifts, o.Grab.Edition = rs.worded(lead.lifts, g), lead.edition
	var fills []files.Cell
	for k, c := range rs.item {
		st := rs.st[g][k]
		if c.Edition != rs.item[f].Edition || st.kind == standsRefused || st.kind != standsAuto && st.kind != lead.kind {
			continue
		}
		if h, ok := rs.held[k]; ok {
			o.Grab.Cancels = append(o.Grab.Cancels, heldCell(c, h))
		} else if slices.Contains(walked, c.State) {
			fills = append(fills, c)
		}
	}
	if len(fills) > 0 {
		o.Grab.Outcome = s.gets(fills)
	}
	_, ok, err := rs.offer(g, f)
	if !ok {
		if _, err := rs.take(rs.as[g].Release); err != nil {
			if e, isNotTaken := errors.AsType[*indexers.NotTakenError](err); isNotTaken {
				o.Grab.Waits = e.Sentence
			}
		}
	}
	o.Grab.Grabbable = err == nil && ok && (len(fills) > 0 || len(o.Grab.Cancels) > 0)
	return o
}

// refusal is why as[g] can't be grabbed at any cell of its Item: "It's a
// pack of more than one episode, and SlipStream can't import packs yet."
func (s sheet) refusal(rs runSheet, g int) string {
	for _, r := range rs.as[g].refusals {
		if r.Code == targets.CoversMore {
			return "It's a pack of more than one " + strings.ToLower(s.in.nouns.Item.One) + ", and SlipStream can't import packs yet."
		}
		if r.Code.Hard() {
			return wordsOf(rs.worded([]Reason{r}, g))
		}
	}
	for _, st := range rs.st[g] {
		for _, r := range st.why {
			if r.Code == targets.NotAnUpgrade {
				return "It isn't better than the file you have."
			}
		}
		if len(st.why) > 0 {
			return wordsOf(rs.worded(st.why[:1], g))
		}
	}
	return ""
}
