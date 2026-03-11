package module

import "time"

// ScheduledTask contains configuration for a scheduled task.
type ScheduledTask struct {
	Name     string
	Interval time.Duration
	RunFunc  func() error
}

// RouteGroup is an abstraction over the HTTP router for module route registration.
type RouteGroup interface {
	GET(path string, handler any)
	POST(path string, handler any)
	PUT(path string, handler any)
	DELETE(path string, handler any)
}
