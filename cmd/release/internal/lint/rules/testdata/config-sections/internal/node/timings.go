package node

import "example.com/app/internal/torrent/engine"

type Timings struct{}

func DefaultTimings() Timings { return Timings{} }

func listen() engine.Settings { return engine.DefaultSettings() }
