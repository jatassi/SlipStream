package tasks

import (
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/setting"
)

type knobs struct {
	zone          *time.Location
	window        setting.Span
	jitterPercent int64
	jitterCap     time.Duration
	// overdue is the global rule, CatchUp or Wait.
	overdue Overdue
	// started is when the clock started: a slot before it was missed.
	started time.Time
}

type state struct {
	task      Task
	every     *int64
	override  *Overdue
	firstSeen time.Time
	// slot is when the last schedule or catch-up run that closed started,
	// zero before one has.
	slot      time.Time
	jitter    float64
	lastEnded time.Time
	open      bool
	// cut: a stop cut the task's slot run short, so it reruns once.
	cut bool
	// atStart: the task runs at clock start and hasn't since this start.
	atStart bool
	// returned: a unit due on return can run now.
	returned bool
}

// Plan is what the clock does now and when it next looks.
type Plan struct {
	Open []Opening
	Pass []Passed
	// Next is each idle task's next slot, absent while it runs.
	Next map[TaskID]time.Time
	// Wake is the earliest Next still ahead, zero when none is.
	Wake time.Time
}

type Opening struct {
	Task    TaskID
	Trigger Trigger
}

// Passed records a missed slot as run, so the next one follows it.
type Passed struct {
	Task TaskID
	Slot time.Time
}

func (k knobs) rule(s state) Overdue {
	switch {
	case s.override != nil:
		return *s.override
	case s.task.Overdue != FollowGlobal:
		return s.task.Overdue
	}
	return k.overdue
}

// plan decides, from states in catalog order, which runs open now, which
// missed slots pass, and when each idle task is next due. Window tasks
// open one at a time.
func plan(now time.Time, k knobs, states []state) Plan {
	p := Plan{Next: map[TaskID]time.Time{}}
	windowBusy := slices.ContainsFunc(states, func(s state) bool { return s.open && inWindow(s.task) })
	for _, s := range states {
		if s.open {
			continue
		}
		trigger, next, missed := k.decide(now, s)
		trigger, windowBusy = serialize(s.task, trigger, windowBusy)
		if missed != nil {
			p.Pass = append(p.Pass, Passed{Task: s.task.ID, Slot: *missed})
		}
		if trigger == "" && s.returned {
			trigger = ByReturn
		}
		if trigger != "" {
			p.Open = append(p.Open, Opening{Task: s.task.ID, Trigger: trigger})
			continue
		}
		p.Next[s.task.ID] = next
		if next.After(now) && (p.Wake.IsZero() || next.Before(p.Wake)) {
			p.Wake = next
		}
	}
	return p
}

// serialize holds a window task's slot run while another window task runs.
func serialize(t Task, trigger Trigger, windowBusy bool) (Trigger, bool) {
	switch {
	case !inWindow(t) || (trigger != BySchedule && trigger != ByCatchUp):
		return trigger, windowBusy
	case windowBusy:
		return "", true
	}
	return trigger, true
}

func inWindow(t Task) bool {
	_, ok := t.Cadence.(InWindow)
	return ok
}

// decide is the slot run s opens now, if any, its next slot, and the
// missed slot it passes under Wait.
func (k knobs) decide(now time.Time, s state) (trigger Trigger, next time.Time, pass *time.Time) {
	switch {
	case s.cut:
		return ByCatchUp, now, nil
	case s.atStart:
		return BySchedule, now, nil
	}
	interval := s.task.Setting().Value(s.every)
	due := s.firstSeen
	switch {
	case s.slot.IsZero():
	case inWindow(s.task):
		due = k.opening(s.slot, int(interval/day))
	default:
		due = k.after(s, interval)
	}
	switch {
	case due.After(now):
		return "", due, nil
	case s.slot.IsZero() || due.Before(k.started):
		return k.missed(now, s, due, interval)
	case inWindow(s.task) && !now.Before(k.closing(k.lastOpening(now))):
		return "", k.opening(k.lastOpening(now), 1), nil
	}
	return BySchedule, due, nil
}

// missed is a slot missed while the clock was down, or a task's first:
// Catch up runs it once now, and Wait passes the latest missed slot.
func (k knobs) missed(now time.Time, s state, due time.Time, interval time.Duration) (trigger Trigger, next time.Time, pass *time.Time) {
	if k.rule(s) == CatchUp {
		return ByCatchUp, due, nil
	}
	if inWindow(s.task) {
		missed := k.lastOpening(now)
		return "", k.opening(missed, int(interval/day)), &missed
	}
	missed := due.Add(now.Sub(due).Truncate(interval))
	return "", missed.Add(interval + k.jitter(s, interval)), &missed
}

const day = 24 * time.Hour

// after is the first interval slot past the task's latest run, so a slot
// that fell due mid-run is skipped rather than queued.
func (k knobs) after(s state, interval time.Duration) time.Time {
	next := s.slot.Add(interval + k.jitter(s, interval))
	if next.Before(s.lastEnded) {
		late := s.lastEnded.Sub(next)
		next = next.Add((late + interval - 1).Truncate(interval))
	}
	return next
}

func (k knobs) jitter(s state, interval time.Duration) time.Duration {
	spread := min(interval*time.Duration(k.jitterPercent)/100, k.jitterCap)
	return time.Duration(s.jitter * float64(spread))
}

// opening is the window's opening days after the day of t, each in the
// zone's wall-clock time, so a DST change keeps it.
func (k knobs) opening(t time.Time, days int) time.Time {
	y, m, d := t.In(k.zone).Date()
	return time.Date(y, m, d+days, k.window.Start/60, k.window.Start%60, 0, 0, k.zone)
}

func (k knobs) closing(opening time.Time) time.Time {
	days := 0
	if k.window.End <= k.window.Start {
		days = 1
	}
	y, m, d := opening.In(k.zone).Date()
	return time.Date(y, m, d+days, k.window.End/60, k.window.End%60, 0, 0, k.zone)
}

func (k knobs) lastOpening(now time.Time) time.Time {
	opening := k.opening(now, 0)
	if opening.After(now) {
		return k.opening(now, -1)
	}
	return opening
}
