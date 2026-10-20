// Package seeders seeds a world's seeded releases: one libtorrent session
// apiece on 127.0.0.1, which a Node reaches through its seeder front.
package seeders

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/netip"
	"os"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
)

// Seeders is the running sessions, by release name.
type Seeders struct {
	sessions map[string]engine.Engine
}

// certFile guards SSL_CERT_FILE around a session's start: libtorrent reads
// the trust store from it once, when the session starts, so the process
// environment is the only way to trust the run CA.
var certFile sync.Mutex

// Start seeds each seeded release of c and returns once each seeder's
// started announce reached the tracker, which trusts caFile.
func Start(ctx context.Context, c *releases.Catalog, caFile string) (*Seeders, error) {
	s := &Seeders{sessions: map[string]engine.Engine{}}
	for _, r := range c.Releases {
		if !r.Seeded {
			continue
		}
		ses, err := open(caFile)
		if err != nil {
			return nil, errors.Join(fmt.Errorf("seeder %s: %w", r.Name, err), s.Close())
		}
		s.sessions[r.Name] = ses
		if err := seed(ctx, ses, r); err != nil {
			return nil, errors.Join(fmt.Errorf("seeder %s: %w", r.Name, err), s.Close())
		}
	}
	return s, nil
}

func open(caFile string) (engine.Engine, error) {
	certFile.Lock()
	defer certFile.Unlock()
	was, set := os.LookupEnv("SSL_CERT_FILE")
	if err := os.Setenv("SSL_CERT_FILE", caFile); err != nil {
		return nil, fmt.Errorf("trust the run CA: %w", err)
	}
	ses, err := libtorrent.Open(engine.Config{
		ListenInterface: netip.MustParseAddrPort("127.0.0.1:0"),
		AnnounceFloor:   5 * time.Minute,
		Rand:            rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())), //nolint:gosec // picks listen ports
	})
	restore := os.Unsetenv("SSL_CERT_FILE")
	if set {
		restore = os.Setenv("SSL_CERT_FILE", was)
	}
	if err != nil {
		return nil, fmt.Errorf("open a session: %w", err)
	}
	if restore != nil {
		return nil, errors.Join(fmt.Errorf("restore SSL_CERT_FILE: %w", restore), ses.Close())
	}
	if err := ses.Configure(engine.Limits{}); err != nil {
		return nil, errors.Join(fmt.Errorf("dial TCP only: %w", err), ses.Close())
	}
	return ses, nil
}

// seed adds r in seed mode and waits for the tracker's reply to its
// started announce.
func seed(ctx context.Context, ses engine.Engine, r releases.Release) error {
	h, err := ses.Add(engine.AddParams{Metainfo: r.Metainfo, SavePath: r.Dir, SeedMode: true})
	if err != nil {
		return fmt.Errorf("add the torrent: %w", err)
	}
	if err := ses.Resume(h); err != nil {
		return fmt.Errorf("start the torrent: %w", err)
	}
	for {
		select {
		case ev, ok := <-ses.Events():
			if !ok {
				return errors.New("the session closed before the tracker replied")
			}
			if done, err := announced(ev, h); done {
				return err
			}
		case <-ctx.Done():
			return fmt.Errorf("await the started announce: %w", ctx.Err())
		}
	}
}

// announced reports whether ev ends the wait for torrent h's started
// announce, with the error it ends in.
func announced(ev engine.Event, h engine.Handle) (bool, error) {
	switch ev := ev.(type) {
	case engine.TrackerReplied:
		return ev.Torrent == h, nil
	case engine.TrackerFailed:
		return ev.Torrent == h, fmt.Errorf("announce: %w", ev.Err)
	case engine.TorrentFailed:
		return ev.Torrent == h, fmt.Errorf("seed: %w", ev.Err)
	}
	return false, nil
}

// Addr is where the seeder of release listens.
func (s *Seeders) Addr(release string) (netip.AddrPort, error) {
	ses, ok := s.sessions[release]
	if !ok {
		return netip.AddrPort{}, fmt.Errorf("no seeder seeds %s", release)
	}
	addr, err := ses.ListenAddr()
	if err != nil {
		return netip.AddrPort{}, fmt.Errorf("seeder %s: %w", release, err)
	}
	return addr, nil
}

// Close stops every session.
func (s *Seeders) Close() error {
	var errs []error
	for _, ses := range s.sessions {
		errs = append(errs, ses.Close())
	}
	return errors.Join(errs...)
}
