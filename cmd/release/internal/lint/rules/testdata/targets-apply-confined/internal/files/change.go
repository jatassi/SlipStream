package files

import "example.test/slip/internal/targets"

func Change(c targets.Change) error { return targets.Apply(c) }
