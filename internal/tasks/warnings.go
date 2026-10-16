package tasks

import (
	"fmt"
	"time"
)

// Warning is a Health warning a task's latest run raises. It is derived
// on read and stored nowhere, so the next success clears it.
type Warning struct {
	Task     TaskID
	Category string
	Text     string
}

const defaultCategory = "scheduled-tasks"

// latest is a task's latest finished run.
type latest struct {
	result Result
	ended  time.Time
	cause  string
}

// since is a unit's last success, or when it was first seen if it never
// succeeded.
type since struct {
	label string
	at    time.Time
}

// warn derives t's warning: a Failed run raises one; a Partial run raises
// one only when a unit has gone intervals of the task's interval without
// success. Skipped and Cancelled runs raise nothing.
func warn(t Task, run latest, units []since, interval time.Duration, intervals int64, now time.Time, zone *time.Location) (Warning, bool) {
	w := Warning{Task: t.ID, Category: t.Health}
	if w.Category == "" {
		w.Category = defaultCategory
	}
	switch run.result {
	case RunFailed:
		w.Text = fmt.Sprintf("%s failed at %s: %s", t.Name, run.ended.In(zone).Format("15:04"), run.cause)
		return w, true
	case RunPartial:
		for _, u := range units {
			if now.Sub(u.at) > time.Duration(intervals)*interval {
				w.Text = fmt.Sprintf("%s has not finished %s since %s", t.Name, u.label, u.at.In(zone).Format("Jan 2 15:04"))
				return w, true
			}
		}
	}
	return Warning{}, false
}
