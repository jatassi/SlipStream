package tasks

import (
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/health"
)

// Warning is a Health warning a task's latest run raises. It is derived
// on read and stored nowhere, so the next success clears it.
type Warning struct {
	Task     TaskID
	Category string
	Name     string
	// Failed: the latest run failed; else a unit is overdue for success.
	Failed bool
	// At is when the failed run ended, or the overdue unit last succeeded.
	At    time.Time
	Cause string
	// Unit is the overdue unit's label.
	Unit string
	// Text is the Tasks page's sentence.
	Text string
}

const defaultCategory = string(health.ScheduledTasksID)

type latest struct {
	result Result
	ended  time.Time
	cause  string
}

type unitSince struct {
	label string
	at    time.Time
}

func warn(t Task, run latest, units []unitSince, after time.Duration, now time.Time, zone *time.Location) (Warning, bool) {
	w := Warning{Task: t.ID, Category: categoryOf(t), Name: t.Name, Cause: run.cause}
	switch run.result {
	case RunFailed:
		w.Failed, w.At = true, run.ended
		w.Text = fmt.Sprintf("%s failed at %s: %s", t.Name, run.ended.In(zone).Format("15:04"), run.cause)
		return w, true
	case RunPartial:
		for _, u := range units {
			if !now.Before(u.at.Add(after)) {
				w.At, w.Unit = u.at, u.label
				w.Text = fmt.Sprintf("%s hasn't finished %s since %s", t.Name, u.label, u.at.In(zone).Format("Jan 2 15:04"))
				return w, true
			}
		}
	}
	return Warning{}, false
}

// nextOverdue is the earliest instant after now at which a unit of a
// partial run goes overdue, zero when none will.
func nextOverdue(result Result, units []unitSince, after time.Duration, now time.Time) time.Time {
	var next time.Time
	if result != RunPartial {
		return next
	}
	for _, u := range units {
		if due := u.at.Add(after); due.After(now) && (next.IsZero() || due.Before(next)) {
			next = due
		}
	}
	return next
}

func categoryOf(t Task) string {
	if t.Health == "" {
		return defaultCategory
	}
	return t.Health
}
