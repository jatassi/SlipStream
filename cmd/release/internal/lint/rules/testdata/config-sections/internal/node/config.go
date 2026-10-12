package node

import (
	"example.com/app/internal/tmdb"
	eng "example.com/app/internal/torrent/engine"
)

type Config struct {
	Torrent   eng.Settings
	Endpoints struct{ TMDB tmdb.Endpoints }
	Timings   struct{ Node Timings }
}

func Defaults() Config {
	var c Config
	c.Torrent = eng.DefaultSettings()
	c.Endpoints.TMDB = tmdb.DefaultEndpoints()
	c.Timings.Node = DefaultTimings()
	return c
}
