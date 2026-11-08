package rtorrent

import (
	"strings"

	"github.com/jatassi/slipstream/fakes/internal/torrentclient"
)

// localError is a command's own error, in rTorrent's words, whose fault
// code each version gives.
type localError string

func (e localError) Error() string { return string(e) }

const (
	errWrongTarget localError = "Target of wrong type to command."
	errTooFew      localError = "Too few arguments."
	errNotString   localError = "Not a string."
)

var global = map[string]func(c *rtorrent, args []any) (any, error){
	"system.client_version": func(c *rtorrent, _ []any) (any, error) { return c.version.client, nil },
	"directory.default":     func(c *rtorrent, _ []any) (any, error) { return c.engine.DefaultFolder(), nil },
	"load.raw":              func(c *rtorrent, args []any) (any, error) { return c.queue(args, false) },
	"load.raw_start":        func(c *rtorrent, args []any) (any, error) { return c.queue(args, true) },
}

// queue is apply_load with raw data: the data, then commands such as
// d.directory.set=/downloads/Movies run on the new download. It answers 0
// at once and loads after the request. A command it can't run drops the
// load, as rTorrent drops a download whose command fails.
func (c *rtorrent) queue(args []any, start bool) (any, error) {
	if len(args) == 0 {
		return nil, errTooFew
	}
	data, ok := args[0].(string)
	if !ok {
		return nil, errNotString
	}
	l := load{metainfo: []byte(data), start: start}
	for _, raw := range args[1:] {
		cmd, ok := raw.(string)
		if !ok {
			return nil, errNotString
		}
		key, arg, _ := strings.Cut(cmd, "=")
		value, parsed := parseArg(arg)
		switch {
		case !parsed:
			return int64(0), nil
		case key == "d.directory.set":
			l.folder = value
		case key == "d.custom1.set":
			l.label = value
		default:
			return int64(0), nil
		}
	}
	c.loads = append(c.loads, l)
	return int64(0), nil
}

// parseArg is rTorrent's parse_string for a command's one argument: a
// quoted string, or text up to a comma, ; or space, with \ escaping the
// next character in both. False is anything else.
func parseArg(s string) (string, bool) {
	s = strings.TrimLeft(s, " \t")
	quoted := strings.HasPrefix(s, `"`)
	if quoted {
		s = s[1:]
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case quoted && ch == '"':
			return b.String(), strings.TrimLeft(s[i+1:], " \t") == ""
		case !quoted && strings.IndexByte(", \t;", ch) >= 0:
			return b.String(), strings.TrimLeft(s[i:], " \t") == ""
		case ch == '\\':
			i++
			if i == len(s) {
				return "", false
			}
			ch = s[i]
		}
		b.WriteByte(ch)
	}
	return b.String(), !quoted
}

func flag(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

type torrentCommand = func(c *rtorrent, t torrentclient.Torrent, h *held, args []any) (any, error)

func get(f func(torrentclient.Torrent, *held) any) torrentCommand {
	return func(_ *rtorrent, t torrentclient.Torrent, h *held, _ []any) (any, error) { return f(t, h), nil }
}

func act(f func(c *rtorrent, t torrentclient.Torrent, h *held)) torrentCommand {
	return func(c *rtorrent, t torrentclient.Torrent, h *held, _ []any) (any, error) {
		f(c, t, h)
		return int64(0), nil
	}
}

var onTorrent = map[string]torrentCommand{
	"d.name":  get(func(t torrentclient.Torrent, _ *held) any { return t.Name }),
	"d.state": get(func(_ torrentclient.Torrent, h *held) any { return flag(h.started) }),
	"d.is_active": get(func(t torrentclient.Torrent, _ *held) any {
		return flag(t.State == torrentclient.Downloading || t.State == torrentclient.Seeding)
	}),
	"d.hashing":         get(hashing),
	"d.complete":        get(func(t torrentclient.Torrent, _ *held) any { return flag(t.Done == t.Size) }),
	"d.size_bytes":      get(func(t torrentclient.Torrent, _ *held) any { return t.Size }),
	"d.completed_bytes": get(func(t torrentclient.Torrent, _ *held) any { return t.Done }),
	"d.directory":       get(func(t torrentclient.Torrent, h *held) any { return directory(t, h) }),
	"d.is_multi_file":   get(func(_ torrentclient.Torrent, h *held) any { return flag(h.multi) }),
	"d.custom1": get(func(t torrentclient.Torrent, _ *held) any {
		if len(t.Labels) == 0 {
			return ""
		}
		return t.Labels[0]
	}),
	"d.timestamp.finished": get(func(t torrentclient.Torrent, _ *held) any {
		if t.CompletedAt.IsZero() {
			return int64(0)
		}
		return t.CompletedAt.Unix()
	}),
	"d.up.total":        get(func(t torrentclient.Torrent, _ *held) any { return t.Uploaded }),
	"d.down.total":      get(func(t torrentclient.Torrent, _ *held) any { return t.Downloaded }),
	"d.up.rate":         get(func(t torrentclient.Torrent, _ *held) any { return t.Rates.Up }),
	"d.down.rate":       get(func(t torrentclient.Torrent, _ *held) any { return t.Rates.Down }),
	"d.peers_connected": get(func(t torrentclient.Torrent, _ *held) any { return int64(t.Rates.Peers) }),
	"d.peers_complete":  get(func(t torrentclient.Torrent, _ *held) any { return int64(t.Rates.Seeds) }),
	"d.message":         func(c *rtorrent, t torrentclient.Torrent, _ *held, _ []any) (any, error) { return c.message(t), nil },
	"d.custom1.set": func(c *rtorrent, t torrentclient.Torrent, _ *held, args []any) (any, error) {
		label, ok := oneString(args)
		if !ok {
			return nil, errNotString
		}
		return int64(0), c.engine.SetLabels(t.Hash, []string{label})
	},
	"d.start": act(func(c *rtorrent, t torrentclient.Torrent, h *held) {
		h.started = true
		if !h.paused {
			_ = c.engine.Start(t.Hash)
		}
	}),
	"d.stop": act(func(c *rtorrent, t torrentclient.Torrent, h *held) {
		h.started, h.paused = false, false
		_ = c.engine.Stop(t.Hash)
	}),
	"d.pause": act(func(c *rtorrent, t torrentclient.Torrent, h *held) {
		if h.started {
			h.paused = true
			_ = c.engine.Stop(t.Hash)
		}
	}),
	"d.resume": act(func(c *rtorrent, t torrentclient.Torrent, h *held) {
		h.paused = false
		if h.started {
			_ = c.engine.Start(t.Hash)
		}
	}),
	"d.check_hash": act(func(c *rtorrent, t torrentclient.Torrent, h *held) {
		if t.State != torrentclient.Checking {
			h.rehash = true
			_ = c.engine.Verify(t.Hash)
		}
	}),
	"d.tracker_announce": act(func(c *rtorrent, t torrentclient.Torrent, _ *held) { _ = c.engine.Announce(t.Hash) }),
	"d.erase": act(func(c *rtorrent, t torrentclient.Torrent, _ *held) {
		_ = c.engine.Remove(t.Hash, false)
		delete(c.held, t.Hash)
	}),
	"f.multicall": files,
}

func oneString(args []any) (string, bool) {
	if len(args) != 1 {
		return "", false
	}
	s, ok := args[0].(string)
	return s, ok
}

// hashing is d.hashing: 1 for the initial check, 3 for a d.check_hash, 0
// for none.
func hashing(t torrentclient.Torrent, h *held) any {
	switch {
	case t.State != torrentclient.Checking:
		h.rehash = false
		return int64(0)
	case h.rehash:
		return int64(3)
	}
	return int64(1)
}

// message is d.message: a local error in the engine's words, or a
// tracker's failure as Download::receive_tracker_msg words it. libtorrent
// passes on no warning, except that 0.16 reads a warning saying the
// torrent is unregistered as a failure.
func (c *rtorrent) message(t torrentclient.Torrent) string {
	tr := t.Tracker
	switch {
	case t.LocalError != "":
		return t.LocalError
	case tr.Message == "":
		return ""
	case tr.Error:
		return `Tracker: [Failure reason "` + tr.Message + `"]`
	case c.version.warns && (strings.Contains(tr.Message, "unregistered") || strings.Contains(tr.Message, "not registered") || strings.Contains(tr.Message, "torrent cannot be found")):
		return "Tracker: [Tracker warning: " + tr.Message + "]"
	}
	return ""
}

// files is f.multicall: a pattern, "" for every file, then f.* commands
// such as f.path=, each file answered as an array in metainfo order.
// f.path is under d.directory, so a multi-file torrent's name isn't in it.
func files(_ *rtorrent, t torrentclient.Torrent, h *held, args []any) (any, error) {
	if len(args) == 0 {
		return nil, errTooFew
	}
	if pattern, ok := args[0].(string); !ok || pattern != "" {
		return nil, localError(`fake-rtorrent plays f.multicall with the pattern "" only`)
	}
	out := []any{}
	offset := int64(0)
	for _, f := range t.Files {
		first := offset / h.pieceLength
		last := (offset + f.Size + h.pieceLength - 1) / h.pieceLength
		completed := int64(0)
		for i := first; i < last; i++ {
			if (i+1)*h.pieceLength <= t.Done || t.Done == t.Size {
				completed++
			}
		}
		filePath := f.Path
		if h.multi {
			filePath = strings.TrimPrefix(f.Path, t.Name+"/")
		}
		fields := map[string]any{"f.path=": filePath, "f.size_bytes=": f.Size, "f.size_chunks=": last - first, "f.completed_chunks=": completed}
		row := []any{}
		for _, raw := range args[1:] {
			cmd, ok := raw.(string)
			v, known := fields[cmd]
			if !ok || !known {
				return nil, localError(`Command "` + strings.TrimSuffix(cmd, "=") + `" does not exist.`)
			}
			row = append(row, v)
		}
		out = append(out, row)
		offset += f.Size
	}
	return out, nil
}
