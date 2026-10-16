package tasks

import (
	"time"

	"github.com/jatassi/slipstream/internal/setting"
)

// knobs are the schedule settings and the clock's own facts a plan reads.
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

// state is one catalog task as the clock last read it.
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
// missed slots pass, and when each idle task is next due.
func plan(now time.Time, k knobs, states []state) Plan {
	p := Plan{Next: map[TaskID]time.Time{}}
	windowBusy := false
	for _, s := range states {
		if _, ok := s.task.Cadence.(InWindow); ok && s.open {
			windowBusy = true
		}
	}
	for _, s := range states {
		if s.open {
			continue
		}
		id := s.task.ID
		_, inWindow := s.task.Cadence.(InWindow)
		open := func(t Trigger) {
			p.Open = append(p.Open, Opening{Task: id, Trigger: t})
			windowBusy = windowBusy || inWindow
		}
		blocked := inWindow && windowBusy
		if (s.cut || s.atStart) && !blocked {
			if s.cut {
				open(ByCatchUp)
			} else {
				open(BySchedule)
			}
			continue
		}
		next, opened := k.slot(now, s, blocked, open, &p)
		if !opened && s.returned {
			open(ByReturn)
			continue
		}
		if !opened {
			p.Next[id] = next
			if next.After(now) && (p.Wake.IsZero() || next.Before(p.Wake)) {
				p.Wake = next
			}
		}
	}
	return p
}

// slot opens s's due slot or passes a missed one, and answers the task's
// next slot when it opened nothing.
func (k knobs) slot(now time.Time, s state, blocked bool, open func(Trigger), p *Plan) (time.Time, bool) {
	interval := s.task.Setting().Value(s.every)
	_, inWindow := s.task.Cadence.(InWindow)
	due := s.firstSeen
	if !s.slot.IsZero() {
		if inWindow {
			due = k.opening(s.slot, int(interval/day))
		} else {
			due = k.after(s, interval)
		}
	}
	if due.After(now) {
		return due, false
	}
	if s.slot.IsZero() || due.Before(k.started) {
		if k.rule(s) == CatchUp {
			if blocked {
				return due, false
			}
			open(ByCatchUp)
			return time.Time{}, true
		}
		var missed, next time.Time
		if inWindow {
			missed = k.lastOpening(now)
			next = k.opening(missed, int(interval/day))
		} else {
			missed = due.Add(now.Sub(due) / interval * interval)
			next = missed.Add(interval + k.jitter(s, interval))
		}
		p.Pass = append(p.Pass, Passed{Task: s.task.ID, Slot: missed})
		return next, false
	}
	if inWindow {
		opening := k.lastOpening(now)
		if now.Before(k.closing(opening)) {
			if blocked {
				return due, false
			}
			open(BySchedule)
			return time.Time{}, true
		}
		return k.opening(opening, 1), false
	}
	open(BySchedule)
	return time.Time{}, true
}

const day = 24 * time.Hour

// after is the first interval slot past the task's latest run, so a slot
// that fell due mid-run is skipped rather than queued.
func (k knobs) after(s state, interval time.Duration) time.Time {
	next := s.slot.Add(interval + k.jitter(s, interval))
	if next.Before(s.lastEnded) {
		late := s.lastEnded.Sub(next)
		next = next.Add((late + interval - 1) / interval * interval)
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

// lastOpening is the latest window opening at or before now.
func (k knobs) lastOpening(now time.Time) time.Time {
	opening := k.opening(now, 0)
	if opening.After(now) {
		return k.opening(now, -1)
	}
	return opening
}
