// Package enginetest is an in-memory torrent engine behind the engine
// interface. It starts no goroutine and no thread, so a testing/synctest
// bubble can hold it, which libtorrent's threads cannot (ADR 0014). A
// torrent added in seed mode seeds; any other downloads and never finishes.
// It reads the files a torrent lists from its metainfo.
package enginetest

import (
	"net/netip"
	"path"
	"slices"
	"sync"

	"github.com/jatassi/slipstream/internal/bencode"
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
	torrents map[engine.Handle]torrent
}

type torrent struct {
	status engine.Status
	files  []engine.File
}

// Open builds an engine listening on cfg's interface, or on port 6881 of
// its address when it asks for any port.
func Open(cfg engine.Config) (engine.Engine, error) {
	listen := cfg.ListenInterface
	if listen.Port() == 0 {
		listen = netip.AddrPortFrom(listen.Addr(), defaultPort)
	}
	return &Engine{listen: listen, events: make(chan engine.Event, events), torrents: map[engine.Handle]torrent{}}, nil
}

func (e *Engine) Add(p engine.AddParams) (engine.Handle, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return 0, engine.ErrClosed
	}
	m, err := bencode.DecodeMetainfo(p.Metainfo)
	if err != nil {
		return 0, &engine.Error{Category: engine.CategoryBdecode, Text: err.Error()}
	}
	e.next++
	state := engine.StateDownloading
	if p.SeedMode {
		state = engine.StateSeeding
	}
	e.torrents[e.next] = torrent{engine.Status{State: state}, files(m.Info)}
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
	t, ok := e.torrents[h]
	if !ok {
		return engine.Status{}, engine.ErrUnknownTorrent
	}
	return t.status, nil
}

func (e *Engine) Files(h engine.Handle) ([]engine.File, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil, engine.ErrClosed
	}
	t, ok := e.torrents[h]
	if !ok {
		return nil, engine.ErrUnknownTorrent
	}
	return slices.Clone(t.files), nil
}

// files lists the metainfo's files as given: unlike libtorrent, the
// in-memory engine sanitizes no name.
func files(info bencode.Info) []engine.File {
	out := make([]engine.File, len(info.Files))
	for i, f := range info.Files {
		out[i] = engine.File{Index: i, Path: path.Join(append([]string{info.Name}, f.Path...)...), Size: f.Length}
	}
	return out
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
