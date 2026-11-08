package seeding

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

const gib = 1 << 30

var (
	now    = time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	day    = 24 * time.Hour
	margin = 2 * time.Hour
	opened = now.Add(-20 * day)

	tenDays     = TimeRule{Seed: 10 * day}
	defaultGoal = Finite{Combine: First, Ratio: 200, Seed: 14 * day, Then: RemoveDelete}

	// owed is a quarter of the way to ratio 1 with six days credited.
	owed = Totals{Uploaded: gib, Downloaded: 4 * gib, Size: 4 * gib, Announced: 6 * day, Seeding: 6 * day}
	// past is past the rule and the default goal.
	past = Totals{Uploaded: 9 * gib, Downloaded: 4 * gib, Size: 4 * gib, Announced: 15 * day, Seeding: 15 * day}
)

func seeder(t Totals, change ...func(*Copy)) Copy {
	c := Copy{Totals: t, AddedAt: opened, Complete: true, Seen: Announcing}
	for _, f := range change {
		f(&c)
	}
	return c
}

func quiet(c *Copy)        { c.Seen = Quiet }
func unseen(c *Copy)       { c.Seen = Unseen }
func incomplete(c *Copy)   { c.Complete = false }
func partial(c *Copy)      { c.Partial = true }
func paused(c *Copy)       { c.Paused = true }
func removing(c *Copy)     { c.Removing = true }
func draining(c *Copy)     { c.Draining = true }
func unregistered(c *Copy) { c.Unregistered = true }
func unimported(c *Copy)   { c.Grabs.Unimported = true }
func ran(r Ran) func(*Copy) {
	return func(c *Copy) { c.Ran = r }
}
func added(at time.Time) func(*Copy) {
	return func(c *Copy) { c.AddedAt = at }
}

func private(rule Rule) Privacy { return Private{Opening: rule} }

func owing(rule Rule) *Window { return &Window{State: Owing, Rule: rule, OpenedAt: opened} }

func met() *Window {
	return &Window{State: Met, Rule: tenDays, OpenedAt: opened, MetAt: now.Add(-5 * day)}
}

func hash(p Privacy, stored *Window, copies ...Copy) Hash {
	return Hash{Now: now, Privacy: p, Goal: defaultGoal, Margin: margin, Stored: stored, Copies: copies}
}

func withGoal(g Goal, h Hash) Hash {
	h.Goal = g
	return h
}

func withLast(r Removed, h Hash) Hash {
	h.Last = &r
	return h
}

func reconciling(h Hash) Hash {
	h.Reconciling = true
	return h
}

func withAdmin(rule Rule, h Hash) Hash {
	p, _ := h.Privacy.(Private)
	p.Admin = rule
	h.Privacy = p
	return h
}

func withGrabs(g Grabs) func(*Copy) {
	return func(c *Copy) { c.Grabs = g }
}

func pause() Finite  { g := defaultGoal; g.Then = Pause; return g }
func remove() Finite { g := defaultGoal; g.Then = Remove; return g }
func both() Finite   { g := defaultGoal; g.Combine = Both; return g }

type outcome struct {
	Window  *Window
	Write   bool
	Owes    *Owes
	Risk    Risk
	GoalMet bool
	Plans   []Plan
}

func outcomeOf(v Verdict) outcome {
	o := outcome{Window: v.View.Window, Write: v.Write, Owes: v.View.Owes, Risk: v.View.Risk, GoalMet: v.View.GoalMet, Plans: v.Plans}
	if len(o.Plans) == 0 {
		o.Plans = nil
	}
	return o
}

func owesTime(d time.Duration) *Owes { return &Owes{Time: d, Clears: ClearsTime} }

func waits(w Why) []Plan { return []Plan{Wait{Why: w}} }

func TestEvaluate(t *testing.T) {
	for _, c := range []struct {
		name string
		hash Hash
		want outcome
	}{
		{"a time rule owes its time plus the margin",
			hash(private(tenDays), owing(tenDays), seeder(owed)),
			outcome{Window: owing(tenDays), Owes: owesTime(4*day + margin), Risk: Safe, Plans: waits(StillOwes)}},
		{"a time rule reached without the margin still owes the margin",
			hash(private(tenDays), owing(tenDays), seeder(Totals{Downloaded: gib, Size: gib, Announced: 10 * day})),
			outcome{Window: owing(tenDays), Owes: owesTime(margin), Risk: Safe, Plans: waits(StillOwes)}},
		{"a time rule is met at the margin",
			hash(private(tenDays), owing(tenDays), seeder(Totals{Downloaded: gib, Size: gib, Announced: 10*day + margin})),
			outcome{Window: &Window{State: Met, Rule: tenDays, OpenedAt: opened, MetAt: now}, Write: true, Risk: Safe, Plans: waits(NotMet)}},
		{"a ratio rule owes upload on real bytes",
			hash(private(RatioRule{Ratio: 100}), owing(RatioRule{Ratio: 100}), seeder(owed)),
			outcome{Window: owing(RatioRule{Ratio: 100}), Owes: &Owes{Bytes: 3 * gib, Clears: ClearsRatio}, Risk: Safe, Plans: waits(StillOwes)}},
		{"a ratio rule is met at the ratio",
			hash(private(RatioRule{Ratio: 100}), owing(RatioRule{Ratio: 100}), seeder(Totals{Uploaded: 4 * gib, Downloaded: 4 * gib, Size: 4 * gib})),
			outcome{Window: &Window{State: Met, Rule: RatioRule{Ratio: 100}, OpenedAt: opened, MetAt: now}, Write: true, Risk: Safe, Plans: waits(NotMet)}},
		{"an either rule owes both parts until one is reached",
			hash(private(EitherRule{Ratio: 100, Seed: 10 * day}), owing(EitherRule{Ratio: 100, Seed: 10 * day}), seeder(owed)),
			outcome{Window: owing(EitherRule{Ratio: 100, Seed: 10 * day}), Owes: &Owes{Time: 4*day + margin, Bytes: 3 * gib, Clears: ClearsEither}, Risk: Safe, Plans: waits(StillOwes)}},
		{"an either rule is met by its ratio alone",
			hash(private(EitherRule{Ratio: 100, Seed: 10 * day}), owing(EitherRule{Ratio: 100, Seed: 10 * day}), seeder(Totals{Uploaded: 4 * gib, Downloaded: 4 * gib, Size: 4 * gib, Announced: day})),
			outcome{Window: &Window{State: Met, Rule: EitherRule{Ratio: 100, Seed: 10 * day}, OpenedAt: opened, MetAt: now}, Write: true, Risk: Safe, Plans: waits(NotMet)}},
		{"a both rule owes the part not yet reached",
			hash(private(BothRule{Ratio: 100, Seed: 10 * day}), owing(BothRule{Ratio: 100, Seed: 10 * day}), seeder(Totals{Uploaded: gib, Downloaded: 4 * gib, Size: 4 * gib, Announced: 11 * day})),
			outcome{Window: owing(BothRule{Ratio: 100, Seed: 10 * day}), Owes: &Owes{Bytes: 3 * gib, Clears: ClearsBoth}, Risk: Safe, Plans: waits(StillOwes)}},
		{"a both rule is met by both parts",
			hash(private(BothRule{Ratio: 100, Seed: 10 * day}), owing(BothRule{Ratio: 100, Seed: 10 * day}), seeder(Totals{Uploaded: 4 * gib, Downloaded: 4 * gib, Size: 4 * gib, Announced: 11 * day})),
			outcome{Window: &Window{State: Met, Rule: BothRule{Ratio: 100, Seed: 10 * day}, OpenedAt: opened, MetAt: now}, Write: true, Risk: Safe, Plans: waits(NotMet)}},

		{"owing starts at the first downloaded byte",
			hash(private(tenDays), nil, seeder(Totals{Downloaded: 1, Size: 4 * gib}, incomplete)),
			outcome{Window: &Window{State: Owing, Rule: tenDays, OpenedAt: now}, Write: true, Owes: owesTime(10*day + margin), Risk: Safe, Plans: waits(StillOwes)}},
		{"a torrent with nothing downloaded owes nothing",
			hash(private(tenDays), nil, seeder(Totals{Size: 4 * gib})),
			outcome{Risk: Safe, Plans: waits(NotMet)}},
		{"a partial download owing a time-only rule has no end",
			hash(private(tenDays), owing(tenDays), seeder(Totals{Downloaded: gib, Size: 4 * gib}, partial)),
			outcome{Window: owing(tenDays), Owes: owesTime(10*day + margin), Risk: Safe, Plans: waits(PartialOwesTime)}},
		{"a partial download can meet a ratio rule and its goal",
			hash(private(RatioRule{Ratio: 100}), owing(RatioRule{Ratio: 100}), seeder(Totals{Uploaded: 9 * gib, Downloaded: gib, Size: 4 * gib}, partial)),
			outcome{Window: &Window{State: Met, Rule: RatioRule{Ratio: 100}, OpenedAt: opened, MetAt: now}, Write: true, Risk: Safe, GoalMet: true, Plans: []Plan{Act{Do: RemoveDelete}}}},
		{"a private Tracker with no rule owes with no countdown and its goal never acts",
			hash(Private{}, owing(nil), seeder(past)),
			outcome{Window: owing(nil), Risk: Safe, Plans: waits(NoRule)}},
		{"a private torrent with no known Tracker owes with no countdown and its goal never acts",
			hash(Untracked{}, nil, seeder(past)),
			outcome{Window: &Window{State: Owing, OpenedAt: now}, Write: true, Risk: Safe, Plans: waits(NoTracker)}},
		{"a public torrent opens no window and its goal reads libtorrent's seeding time",
			hash(Public{}, nil, seeder(Totals{Uploaded: gib, Downloaded: 4 * gib, Size: 4 * gib, Seeding: 15 * day})),
			outcome{Risk: Safe, GoalMet: true, Plans: []Plan{Act{Do: RemoveDelete}}}},
		{"a private torrent's goal reads the time its Tracker credits",
			hash(private(tenDays), &Window{State: Released, OpenedAt: opened, ClosedAt: now.Add(-day)}, seeder(Totals{Uploaded: gib, Downloaded: 4 * gib, Size: 4 * gib, Announced: 6 * day, Seeding: 15 * day})),
			outcome{Window: &Window{State: Released, OpenedAt: opened, ClosedAt: now.Add(-day)}, Risk: Safe, Plans: waits(NotMet)}},
		{"a tracker outage leaves libtorrent's time past the rule and the Tracker's short of it",
			hash(private(tenDays), owing(tenDays), seeder(Totals{Uploaded: gib, Downloaded: 4 * gib, Size: 4 * gib, Announced: 9 * day, Seeding: 12 * day}, quiet)),
			outcome{Window: &Window{State: Owing, Rule: tenDays, OpenedAt: opened, QuietSince: now}, Write: true, Owes: owesTime(day + margin), Risk: AtRisk, Plans: waits(StillOwes)}},

		{"the Admin's rule wins over the window's snapshot",
			withAdmin(TimeRule{Seed: 8 * day}, hash(private(tenDays), owing(tenDays), seeder(Totals{Downloaded: gib, Size: gib, Announced: 8*day + margin}))),
			outcome{Window: &Window{State: Met, Rule: tenDays, OpenedAt: opened, MetAt: now}, Write: true, Risk: Safe, Plans: waits(NotMet)}},
		{"a rule from Releases changing never touches an open window",
			hash(private(TimeRule{Seed: 20 * day}), owing(tenDays), seeder(owed)),
			outcome{Window: owing(tenDays), Owes: owesTime(4*day + margin), Risk: Safe, Plans: waits(StillOwes)}},
		{"a window with no rule takes the first one known",
			hash(private(tenDays), owing(nil), seeder(owed)),
			outcome{Window: owing(tenDays), Write: true, Owes: owesTime(4*day + margin), Risk: Safe, Plans: waits(StillOwes)}},
		{"a met window stays met when the Admin raises the rule",
			withAdmin(TimeRule{Seed: 30 * day}, hash(private(tenDays), met(), seeder(past))),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: []Plan{Act{Do: RemoveDelete}}}},

		{"the default goal removes and deletes once met",
			hash(private(tenDays), met(), seeder(past)),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: []Plan{Act{Do: RemoveDelete}}}},
		{"whichever first meets on time alone",
			hash(private(tenDays), met(), seeder(Totals{Uploaded: gib, Downloaded: 4 * gib, Size: 4 * gib, Announced: 15 * day})),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: []Plan{Act{Do: RemoveDelete}}}},
		{"both waits for its ratio",
			withGoal(both(), hash(private(tenDays), met(), seeder(Totals{Uploaded: gib, Downloaded: 4 * gib, Size: 4 * gib, Announced: 15 * day}))),
			outcome{Window: met(), Risk: Safe, Plans: waits(NotMet)}},
		{"both meets on both parts",
			withGoal(both(), hash(private(tenDays), met(), seeder(past))),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: []Plan{Act{Do: RemoveDelete}}}},
		{"a pause goal pauses",
			withGoal(pause(), hash(private(tenDays), met(), seeder(past))),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: []Plan{Act{Do: Pause}}}},
		{"a remove goal removes",
			withGoal(remove(), hash(private(tenDays), met(), seeder(past))),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: []Plan{Act{Do: Remove}}}},
		{"a forever goal seeds with no end",
			withGoal(Forever{}, hash(private(tenDays), met(), seeder(past))),
			outcome{Window: met(), Risk: Safe, Plans: waits(SeedsForever)}},
		{"a goal never acts before the rule is met",
			hash(private(tenDays), owing(tenDays), seeder(Totals{Uploaded: 9 * gib, Downloaded: 4 * gib, Size: 4 * gib, Announced: 6 * day})),
			outcome{Window: owing(tenDays), Owes: owesTime(4*day + margin), Risk: Safe, Plans: waits(StillOwes)}},

		{"goals wait while the Controller is Reconciling",
			reconciling(hash(private(tenDays), met(), seeder(past))),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: waits(Reconciling)}},
		{"a goal waits while a Grab downloads from the torrent",
			hash(private(tenDays), met(), seeder(past, withGrabs(Grabs{Downloading: true}))),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: waits(GrabDownloading)}},
		{"a goal waits while a Grab imports from the torrent",
			hash(private(tenDays), met(), seeder(past, withGrabs(Grabs{Importing: true}))),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: waits(GrabImporting)}},
		{"a goal waits while the torrent is removed",
			hash(private(tenDays), met(), seeder(past, removing, ran(RanRemove))),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: waits(BeingRemoved)}},
		{"a goal acts only on a complete torrent",
			hash(Public{}, nil, seeder(Totals{Uploaded: 9 * gib, Downloaded: 2 * gib, Size: 4 * gib}, incomplete)),
			outcome{Risk: Safe, GoalMet: true, Plans: waits(Incomplete)}},
		{"a goal acts once on a Torrent",
			hash(private(tenDays), met(), seeder(past, paused, ran(RanPause))),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: waits(AlreadyRan)}},
		{"remove and delete keeps the copy when a Grab didn't import",
			hash(private(tenDays), met(), seeder(past, unimported)),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: []Plan{Keep{Why: KeptNotImported}}}},
		{"remove keeps the copy anyway, so a Grab that didn't import changes nothing",
			withGoal(remove(), hash(private(tenDays), met(), seeder(past, unimported))),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: []Plan{Act{Do: Remove}}}},
		{"a pause on a draining Node removes and keeps the copy",
			withGoal(pause(), hash(private(tenDays), met(), seeder(past, draining))),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: []Plan{Keep{Why: KeptDraining}}}},
		{"a draining Node removes a torrent a goal paused before the drain",
			withGoal(Forever{}, hash(private(tenDays), met(), seeder(past, paused, draining, ran(RanPause)))),
			outcome{Window: met(), Risk: Safe, Plans: []Plan{Keep{Why: KeptDraining}}}},
		{"a draining Node still waits out Reconciling for a goal pause",
			reconciling(hash(private(tenDays), met(), seeder(past, paused, draining, ran(RanPause)))),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: waits(Reconciling)}},
		{"a goal pause the Admin resumed seeds with no end",
			hash(private(tenDays), met(), seeder(past, ran(RanPause))),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: waits(ResumedAfterPause)}},
		{"a draining Node keeps seeding a resumed goal pause",
			hash(private(tenDays), met(), seeder(past, draining, ran(RanPause))),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: waits(ResumedAfterPause)}},

		{"an owing torrent that stops announcing is at risk at once",
			hash(private(tenDays), owing(tenDays), seeder(owed, quiet)),
			outcome{Window: &Window{State: Owing, Rule: tenDays, OpenedAt: opened, QuietSince: now}, Write: true, Owes: owesTime(4*day + margin), Risk: AtRisk, Plans: waits(StillOwes)}},
		{"at risk keeps when it first went quiet",
			hash(private(tenDays), &Window{State: Owing, Rule: tenDays, OpenedAt: opened, QuietSince: now.Add(-3 * time.Hour)}, seeder(owed, quiet)),
			outcome{Window: &Window{State: Owing, Rule: tenDays, OpenedAt: opened, QuietSince: now.Add(-3 * time.Hour)}, Owes: owesTime(4*day + margin), Risk: AtRisk, Plans: waits(StillOwes)}},
		{"announcing again is no longer at risk",
			hash(private(tenDays), &Window{State: Owing, Rule: tenDays, OpenedAt: opened, QuietSince: now.Add(-3 * time.Hour)}, seeder(owed)),
			outcome{Window: owing(tenDays), Write: true, Owes: owesTime(4*day + margin), Risk: Safe, Plans: waits(StillOwes)}},
		{"a torrent on a Node out of step reads unknown and keeps when it went quiet",
			hash(private(tenDays), &Window{State: Owing, Rule: tenDays, OpenedAt: opened, QuietSince: now.Add(-3 * time.Hour)}, seeder(owed, unseen)),
			outcome{Window: &Window{State: Owing, Rule: tenDays, OpenedAt: opened, QuietSince: now.Add(-3 * time.Hour)}, Owes: owesTime(4*day + margin), Risk: RiskUnknown, Plans: waits(StillOwes)}},
		{"a met window is never at risk",
			hash(private(tenDays), met(), seeder(owed, quiet)),
			outcome{Window: met(), Risk: Safe, Plans: waits(NotMet)}},
		{"meeting the rule ends at risk",
			hash(private(tenDays), &Window{State: Owing, Rule: tenDays, OpenedAt: opened, QuietSince: now.Add(-3 * time.Hour)}, seeder(Totals{Downloaded: gib, Size: gib, Announced: 11 * day}, quiet)),
			outcome{Window: &Window{State: Met, Rule: tenDays, OpenedAt: opened, MetAt: now}, Write: true, Risk: Safe, Plans: waits(NotMet)}},

		{"two live Torrents of one hash share its maximum totals and announce for each other",
			hash(private(tenDays), owing(tenDays),
				seeder(Totals{Uploaded: 9 * gib, Downloaded: 4 * gib, Size: 4 * gib, Announced: 6 * day}, quiet),
				seeder(Totals{Uploaded: gib, Downloaded: 4 * gib, Size: 4 * gib, Announced: 11 * day}, incomplete)),
			outcome{Window: &Window{State: Met, Rule: tenDays, OpenedAt: opened, MetAt: now}, Write: true, Risk: Safe, GoalMet: true, Plans: []Plan{Act{Do: RemoveDelete}, Wait{Why: Incomplete}}}},
		{"two live Torrents of one hash are at risk only when neither announces",
			hash(private(tenDays), owing(tenDays), seeder(owed, quiet), seeder(owed, unseen)),
			outcome{Window: owing(tenDays), Owes: owesTime(4*day + margin), Risk: RiskUnknown, Plans: []Plan{Wait{Why: StillOwes}, Wait{Why: StillOwes}}}},

		{"an unregistered torrent releases its window and its goal acts",
			hash(private(tenDays), owing(tenDays), seeder(Totals{Uploaded: 9 * gib, Downloaded: 4 * gib, Size: 4 * gib, Announced: 6 * day}, unregistered)),
			outcome{Window: &Window{State: Released, Rule: tenDays, OpenedAt: opened, ClosedAt: now}, Write: true, Risk: Safe, GoalMet: true, Plans: []Plan{Act{Do: RemoveDelete}}}},
		{"removing the last copy while owing misses with what it still owed",
			withLast(Removed{At: now.Add(-time.Hour), Totals: owed}, hash(private(tenDays), owing(tenDays))),
			outcome{Window: &Window{State: Missed, Rule: tenDays, OpenedAt: opened, ClosedAt: now.Add(-time.Hour), Owed: Owes{Time: 4*day + margin, Clears: ClearsTime}}, Write: true, Risk: Safe}},
		{"a copy removed after it met the rule is met",
			withLast(Removed{At: now.Add(-time.Hour), Totals: past}, hash(private(tenDays), owing(tenDays))),
			outcome{Window: &Window{State: Met, Rule: tenDays, OpenedAt: opened, MetAt: now}, Write: true, Risk: Safe}},
		{"a removal whose Grab continues in an external client releases the window",
			withLast(Removed{At: now.Add(-time.Hour), Totals: owed, Continues: true}, hash(private(tenDays), owing(tenDays))),
			outcome{Window: &Window{State: Released, Rule: tenDays, OpenedAt: opened, ClosedAt: now}, Write: true, Risk: Safe}},
		{"an unregistered torrent removed while owing releases the window",
			withLast(Removed{At: now.Add(-time.Hour), Totals: owed, Unregistered: true}, hash(private(tenDays), owing(tenDays))),
			outcome{Window: &Window{State: Released, Rule: tenDays, OpenedAt: opened, ClosedAt: now}, Write: true, Risk: Safe}},
		{"a missed window with no live copy stays missed",
			withLast(Removed{At: now.Add(-2 * day), Totals: owed}, hash(private(tenDays), &Window{State: Missed, Rule: tenDays, OpenedAt: opened, ClosedAt: now.Add(-2 * day), Owed: Owes{Time: 4*day + margin, Clears: ClearsTime}})),
			outcome{Window: &Window{State: Missed, Rule: tenDays, OpenedAt: opened, ClosedAt: now.Add(-2 * day), Owed: Owes{Time: 4*day + margin, Clears: ClearsTime}}, Risk: Safe}},
		{"a regrab after a miss owes again and keeps the miss in its history",
			withLast(Removed{At: now.Add(-2 * day), Totals: owed}, hash(private(tenDays), &Window{State: Missed, Rule: tenDays, OpenedAt: opened, ClosedAt: now.Add(-2 * day), Owed: Owes{Time: 4*day + margin, Clears: ClearsTime}}, seeder(owed, added(now.Add(-time.Hour))))),
			outcome{Window: &Window{State: Owing, Rule: tenDays, OpenedAt: opened, ClosedAt: now.Add(-2 * day), RegrabbedAt: now.Add(-time.Hour), Owed: Owes{Time: 4*day + margin, Clears: ClearsTime}}, Write: true, Owes: owesTime(4*day + margin), Risk: Safe, Plans: waits(StillOwes)}},
		{"a removal and a regrab between two runs read as the miss they were",
			withLast(Removed{At: now.Add(-2 * time.Hour), Totals: owed}, hash(private(tenDays), owing(tenDays), seeder(owed, added(now.Add(-time.Hour))))),
			outcome{Window: &Window{State: Owing, Rule: tenDays, OpenedAt: opened, ClosedAt: now.Add(-2 * time.Hour), RegrabbedAt: now.Add(-time.Hour), Owed: Owes{Time: 4*day + margin, Clears: ClearsTime}}, Write: true, Owes: owesTime(4*day + margin), Risk: Safe, Plans: waits(StillOwes)}},
		{"removing one copy while another seeds misses nothing",
			withLast(Removed{At: now.Add(-2 * time.Hour), Totals: owed}, hash(private(tenDays), owing(tenDays), seeder(owed))),
			outcome{Window: owing(tenDays), Owes: owesTime(4*day + margin), Risk: Safe, Plans: waits(StillOwes)}},
		{"a regrab after a goal removed it meets its goal again on the inherited totals",
			withLast(Removed{At: now.Add(-2 * time.Hour), Totals: past}, hash(private(tenDays), met(), seeder(past, added(now.Add(-time.Hour))))),
			outcome{Window: met(), Risk: Safe, GoalMet: true, Plans: []Plan{Act{Do: RemoveDelete}}}},
		{"a public regrab after a goal removed it meets its goal again",
			withLast(Removed{At: now.Add(-2 * time.Hour), Totals: past}, hash(Public{}, nil, seeder(past, added(now.Add(-time.Hour))))),
			outcome{Risk: Safe, GoalMet: true, Plans: []Plan{Act{Do: RemoveDelete}}}},
		{"a removal from before any window was written leaves none",
			withLast(Removed{At: now.Add(-time.Hour), Totals: owed}, hash(private(tenDays), nil)),
			outcome{Risk: Safe}},
	} {
		got := Evaluate(c.hash)
		if g := outcomeOf(got); !reflect.DeepEqual(g, c.want) {
			t.Errorf("%s:\n got %+v\nwant %+v", c.name, show(g), show(c.want))
		}
		again := c.hash
		again.Stored = got.View.Window
		if second := Evaluate(again); second.Write || !reflect.DeepEqual(second.Plans, got.Plans) {
			t.Errorf("%s: a second evaluation over what the first wrote wrote %v and planned %v", c.name, second.Write, second.Plans)
		}
	}
}

type shown struct {
	outcome
	Window Window
	Owes   Owes
}

func show(o outcome) shown {
	s := shown{outcome: o}
	if o.Window != nil {
		s.Window = *o.Window
	}
	if o.Owes != nil {
		s.Owes = *o.Owes
	}
	return s
}

func TestEndless(t *testing.T) {
	for why, want := range map[Why]bool{
		ResumedAfterPause: true, SeedsForever: true, NoTracker: true, NoRule: true, PartialOwesTime: true,
		BeingRemoved: false, AlreadyRan: false, StillOwes: false, NotMet: false, Incomplete: false,
		GrabDownloading: false, GrabImporting: false, Reconciling: false,
	} {
		if got := why.Endless(); got != want {
			t.Errorf("%s: Endless = %v, want %v", why, got, want)
		}
	}
}

func ratio(r Ratio) *Ratio                    { return &r }
func duration(d time.Duration) *time.Duration { return &d }

func TestLanes(t *testing.T) {
	for _, c := range []struct {
		name        string
		hash        Hash
		ratio       Lane[Ratio]
		seedingTime Lane[time.Duration]
	}{
		{"a time rule marks the time lane past the margin and leaves the ratio lane unmarked",
			hash(private(tenDays), owing(tenDays), seeder(owed)),
			Lane[Ratio]{Value: 25, Goal: ratio(200)},
			Lane[time.Duration]{Value: 6 * day, Rule: duration(10*day + margin), Goal: duration(14 * day)}},
		{"an either rule marks both lanes",
			hash(private(EitherRule{Ratio: 100, Seed: 10 * day}), owing(EitherRule{Ratio: 100, Seed: 10 * day}), seeder(owed)),
			Lane[Ratio]{Value: 25, Rule: ratio(100), Goal: ratio(200)},
			Lane[time.Duration]{Value: 6 * day, Rule: duration(10*day + margin), Goal: duration(14 * day)}},
		{"a public torrent's time lane is libtorrent's seeding time with no rule",
			hash(Public{}, nil, seeder(Totals{Uploaded: gib, Downloaded: 4 * gib, Size: 4 * gib, Announced: day, Seeding: 3 * day})),
			Lane[Ratio]{Value: 25, Goal: ratio(200)},
			Lane[time.Duration]{Value: 3 * day, Goal: duration(14 * day)}},
		{"a forever goal marks no goal",
			withGoal(Forever{}, hash(private(tenDays), owing(tenDays), seeder(owed))),
			Lane[Ratio]{Value: 25},
			Lane[time.Duration]{Value: 6 * day, Rule: duration(10*day + margin)}},
		{"a torrent that came complete from another client reads its ratio against its size",
			hash(Public{}, nil, seeder(Totals{Uploaded: 6 * gib, Size: 4 * gib})),
			Lane[Ratio]{Value: 150, Goal: ratio(200)},
			Lane[time.Duration]{Goal: duration(14 * day)}},
	} {
		v := Evaluate(c.hash).View
		if !reflect.DeepEqual(v.Ratio, c.ratio) || !reflect.DeepEqual(v.Time, c.seedingTime) {
			t.Errorf("%s: lanes %s %s, want %s %s", c.name, lane(v.Ratio), lane(v.Time), lane(c.ratio), lane(c.seedingTime))
		}
	}
}

func lane[T any](l Lane[T]) string {
	mark := func(p *T) any {
		if p == nil {
			return "none"
		}
		return *p
	}
	return fmt.Sprintf("{%v rule %v goal %v}", l.Value, mark(l.Rule), mark(l.Goal))
}
