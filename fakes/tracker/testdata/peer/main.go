// Command peer is one libtorrent peer in its own process, so OpenSSL reads
// SSL_CERT_FILE from the environment the test gives it as a Node's does.
// It listens on its first argument, adds the torrent file in its second
// with the folder in its third as the save path, seeding it with a fourth
// argument seed, and prints announced after its first tracker reply and
// finished once it holds the whole payload, exiting once the tracker has
// also answered its completed announce. libtorrent sends that announce as
// the torrent turns finished, while Finished waits for the payload's files
// to be released, so the answer can come first. A failed announce prints
// the tracker error and exits 1.
package main

import (
	"fmt"
	"math/rand/v2"
	"net/netip"
	"os"

	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	e, err := libtorrent.Open(engine.Config{
		ListenInterface: netip.AddrPortFrom(netip.MustParseAddr(os.Args[1]), 0),
		Rand:            rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())), //nolint:gosec // picks listen ports
	})
	if err != nil {
		return err
	}
	defer func() { _ = e.Close() }()
	meta, err := os.ReadFile(os.Args[2])
	if err != nil {
		return err
	}
	seed := len(os.Args) > 4 && os.Args[4] == "seed"
	h, err := e.Add(engine.AddParams{Metainfo: meta, SavePath: os.Args[3], SeedMode: seed})
	if err != nil {
		return err
	}
	var announced, complete, completedAnswered, finished bool
	for ev := range e.Events() {
		switch ev := ev.(type) {
		case engine.StateChanged:
			if ev.Torrent == h && (ev.State == engine.StateFinished || ev.State == engine.StateSeeding) {
				complete = true
			}
		case engine.TrackerReplied:
			if ev.Torrent != h {
				continue
			}
			if !announced {
				announced = true
				fmt.Println("announced")
			}
			completedAnswered = complete
		case engine.TrackerFailed:
			if ev.Torrent == h {
				return fmt.Errorf("tracker failed: %s", ev.Err.Text)
			}
		case engine.Finished:
			if ev.Torrent == h && !seed {
				finished = true
				fmt.Println("finished")
			}
		}
		if finished && completedAnswered {
			return nil
		}
	}
	return nil
}
