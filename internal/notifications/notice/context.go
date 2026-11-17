package notice

import "time"

// Context is what Render may read besides the facts.
type Context struct {
	Now  time.Time
	Zone *time.Location
	// Sender is the name of the Node that sends this message, for Test.
	Sender string
}

// Clock is t's time of day in the Zone: "14:02".
func (c Context) Clock(t time.Time) string { return t.In(c.Zone).Format("15:04") }

// Day is t's date in the Zone: "1 December".
func (c Context) Day(t time.Time) string { return t.In(c.Zone).Format("2 January") }
