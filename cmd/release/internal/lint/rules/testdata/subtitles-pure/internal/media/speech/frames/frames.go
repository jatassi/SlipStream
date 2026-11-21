package frames

import "time"

func Age(t time.Time) time.Duration { return time.Until(t) }
