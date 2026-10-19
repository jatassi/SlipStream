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

func warn(t Task, run latest, units []unitSince, interval time.Duration, intervals int64, now time.Time, zone *time.Location) (Warning, bool) {
	w := Warning{Task: t.ID, Category: categoryOf(t), Name: t.Name, Cause: run.cause}
	switch run.result {
	case RunFailed:
		w.Failed, w.At = true, run.ended
		w.Text = fmt.Sprintf("%s failed at %s: %s", t.Name, run.ended.In(zone).Format("15:04"), run.cause)
		return w, true
	case RunPartial:
		for _, u := range units {
			if now.Sub(u.at) > time.Duration(intervals)*interval {
				w.At, w.Unit = u.at, u.label
				w.Text = fmt.Sprintf("%s has not finished %s since %s", t.Name, u.label, u.at.In(zone).Format("Jan 2 15:04"))
				return w, true
			}
		}
	}
	return Warning{}, false
}

func categoryOf(t Task) string {
	if t.Health == "" {
		return defaultCategory
	}
	return t.Health
}
