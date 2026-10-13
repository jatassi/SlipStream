// Package enginetest is an in-memory torrent engine behind the engine
// interface. It starts no goroutine and no thread, so a testing/synctest
// bubble can hold it, which libtorrent's threads cannot (ADR 0014). A
// torrent added in seed mode seeds; any other downloads and never finishes.
package enginetest

import (
	"net/netip"
	"sync"

	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// events is how many events wait for a reader before the engine drops
// them and sends Dropped.
const events = 64

// defaultPort is the port a listen interface with port 0 reports.
const defaultPort = 6881

type Engine struct {
	listen netip.AddrPort
	events chan engine.Event

	mu       sync.Mutex
	closed   bool
	dropped  bool
	next     engine.Handle
	torrents map[engine.Handle]engine.Status
}

// Open builds an engine listening on cfg's interface, or on port 6881 of
// its address when it asks for any port.
func Open(cfg engine.Config) (engine.Engine, error) {
	listen := cfg.ListenInterface
	if listen.Port() == 0 {
		listen = netip.AddrPortFrom(listen.Addr(), defaultPort)
	}
	return &Engine{listen: listen, events: make(chan engine.Event, events), torrents: map[engine.Handle]engine.Status{}}, nil
}

func (e *Engine) Add(p engine.AddParams) (engine.Handle, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return 0, engine.ErrClosed
	}
	e.next++
	state := engine.StateDownloading
	if p.SeedMode {
		state = engine.StateSeeding
	}
	e.torrents[e.next] = engine.Status{State: state}
	e.emit(engine.Added{Torrent: e.next})
	e.emit(engine.StateChanged{Torrent: e.next, State: state})
	return e.next, nil
}

func (e *Engine) ConnectPeer(h engine.Handle, _ netip.AddrPort) error {
	_, err := e.Status(h)
	return err
}

func (e *Engine) Status(h engine.Handle) (engine.Status, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return engine.Status{}, engine.ErrClosed
	}
	s, ok := e.torrents[h]
	if !ok {
		return engine.Status{}, engine.ErrUnknownTorrent
	}
	return s, nil
}

func (e *Engine) Remove(h engine.Handle) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return engine.ErrClosed
	}
	if _, ok := e.torrents[h]; !ok {
		return engine.ErrUnknownTorrent
	}
	delete(e.torrents, h)
	e.emit(engine.Removed{Torrent: h})
	return nil
}

func (e *Engine) ListenAddr() (netip.AddrPort, error) { return e.listen, nil }

func (e *Engine) Events() <-chan engine.Event { return e.events }

func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.closed {
		e.closed = true
		close(e.events)
	}
	return nil
}

// emit queues ev without blocking. Once a reader falls behind it drops
// events, and the next one it can queue is Dropped.
func (e *Engine) emit(ev engine.Event) {
	if e.dropped {
		select {
		case e.events <- engine.Dropped{}:
			e.dropped = false
		default:
			return
		}
	}
	select {
	case e.events <- ev:
	default:
		e.dropped = true
	}
}
