package filestest

import "example.test/slip/internal/targets"

func Apply(c targets.Change) error { return targets.Apply(c) }
