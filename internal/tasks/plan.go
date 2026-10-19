package tasks

import (
	"hash/fnv"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
)

type knobs struct {
	zone          *time.Location
	window        setting.Span
	jitterPercent int64
	jitterCap     time.Duration
	overdue       Overdue
	started       time.Time
	install       role.NodeID
}

type state struct {
	task      Task
	every     *int64
	override  *Overdue
	firstSeen time.Time
	slot      time.Time
	jitter    float64
	lastEnded time.Time
	open      bool
	cut       bool
	atStart   bool
	returned  bool
}

type Plan struct {
	Open []Opening
	Pass []Passed
	Next map[TaskID]time.Time
	Wake time.Time
}

type Opening struct {
	Task    TaskID
	Trigger Trigger
}

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

func plan(now time.Time, k knobs, states []state) Plan {
	p := Plan{Next: map[TaskID]time.Time{}}
	windowBusy := slices.ContainsFunc(states, func(s state) bool { return s.open && inWindow(s.task) })
	for _, s := range states {
		if s.open {
			continue
		}
		trigger, next, missed := k.decide(now, s)
		trigger, windowBusy = serialize(s.task, trigger, windowBusy)
		if trigger == "" && k.spreading(now, s.task, next) {
			windowBusy = true
		}
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
		due = k.start(s.task, k.opening(s.slot, int(interval/day)))
	default:
		due = k.after(s, interval)
	}
	switch {
	case due.After(now):
		return "", due, nil
	case s.slot.IsZero() || due.Before(k.started):
		return k.missed(now, s, due, interval)
	case inWindow(s.task) && !now.Before(k.closing(k.lastOpening(now))):
		return "", k.start(s.task, k.opening(k.lastOpening(now), 1)), nil
	}
	return BySchedule, due, nil
}

func (k knobs) missed(now time.Time, s state, due time.Time, interval time.Duration) (trigger Trigger, next time.Time, pass *time.Time) {
	if k.rule(s) == CatchUp {
		return ByCatchUp, due, nil
	}
	if inWindow(s.task) {
		missed := k.lastOpening(now)
		return "", k.start(s.task, k.opening(missed, int(interval/day))), &missed
	}
	missed := due.Add(now.Sub(due).Truncate(interval))
	return "", missed.Add(interval + k.jitter(s, interval)), &missed
}

const day = 24 * time.Hour

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

// start is when a window task starts in the window that opens at opening:
// its spread's offset after, the same for the same date and install.
func (k knobs) start(t Task, opening time.Time) time.Time {
	w, _ := t.Cadence.(InWindow)
	spread := min(w.Spread, k.closing(opening).Sub(opening)) / time.Second
	if spread <= 0 {
		return opening
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(opening.In(k.zone).Format(time.DateOnly) + k.install.String()))
	return opening.Add(time.Duration(h.Sum64()%uint64(spread)) * time.Second) //nolint:gosec // G115: the remainder is below spread, which fits a Duration
}

// spreading reports a window task waiting for its offset in the open
// window, which holds the window for the tasks after it.
func (k knobs) spreading(now time.Time, t Task, next time.Time) bool {
	return inWindow(t) && next.After(now) && next.Before(k.closing(k.lastOpening(now)))
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
