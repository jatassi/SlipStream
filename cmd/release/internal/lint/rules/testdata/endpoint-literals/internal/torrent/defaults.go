package torrent

import "github.com/example/tmdb"

func DefaultEndpoints() tmdb.Endpoints {
	return tmdb.Endpoints{API: "router.utorrent.com:6881"}
}

func DefaultTimings() string {
	return "dht.transmissionbt.com:6881"
}
