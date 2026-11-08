package rtorrent

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

// quoted is s as one argument of an rTorrent command string, which splits
// bare text at commas and spaces.
func quoted(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// Add sends load.raw_start, or load.raw for a paused add, with
// d.directory.set to the folder, so rTorrent adds the name of a
// multi-file torrent as the other kinds do, and d.custom1.set to the
// label. rTorrent loads the torrent after it responds, so the hash shows
// on a later poll, and it drops a torrent it already holds without
// running the commands.
func (c *client) Add(ctx context.Context, a clients.Add) error {
	method := "load.raw_start"
	if a.Paused {
		method = "load.raw"
	}
	params := []any{"", a.Metainfo}
	if a.Folder != "" {
		params = append(params, "d.directory.set="+quoted(a.Folder))
	}
	params = append(params, "d.custom1.set="+quoted(a.Label.String()))
	_, err := c.send(ctx, method, params)
	return err
}

// fields are the d.* calls a poll makes for each hash.
var fields = []string{
	"d.name", "d.state", "d.is_active", "d.hashing", "d.complete", "d.size_bytes", "d.completed_bytes",
	"d.directory", "d.is_multi_file", "d.custom1", "d.message", "d.timestamp.finished",
	"d.up.total", "d.down.total", "d.up.rate", "d.down.rate", "d.peers_connected", "d.peers_complete",
}

// Poll is one system.multicall of fields over hashes. A hash rTorrent
// doesn't hold faults each of its calls.
func (c *client) Poll(ctx context.Context, hashes []bencode.InfoHash) (map[bencode.InfoHash]clients.Status, error) {
	calls := make([]call, 0, len(hashes)*len(fields))
	for _, h := range hashes {
		for _, f := range fields {
			calls = append(calls, call{method: f, params: []any{target(h)}})
		}
	}
	results, err := c.multicall(ctx, calls)
	if err != nil {
		return nil, err
	}
	out := make(map[bencode.InfoHash]clients.Status, len(hashes))
	for i, h := range hashes {
		row := results[i*len(fields) : (i+1)*len(fields)]
		if missing(asError(row[0])) {
			continue
		}
		v := values{byName: make(map[string]any, len(fields))}
		for j, f := range fields {
			if err := asError(row[j]); err != nil {
				return nil, err
			}
			v.byName[f] = row[j]
		}
		s := status(&v)
		if v.err != nil {
			return nil, fmt.Errorf("read the client's torrent %s: %w", h, v.err)
		}
		out[h] = s
	}
	return out, nil
}

func asError(v any) error {
	if f, ok := v.(*fault); ok {
		return f
	}
	return nil
}

// values are one torrent's answers by method, and the first that isn't
// of the type its reader wants.
type values struct {
	byName map[string]any
	err    error
}

func (v *values) str(method string) string {
	s, ok := v.byName[method].(string)
	if !ok && v.err == nil {
		v.err = fmt.Errorf("%s is %v, not a string", method, v.byName[method])
	}
	return s
}

func (v *values) num(method string) int64 {
	n, ok := v.byName[method].(int64)
	if !ok && v.err == nil {
		v.err = fmt.Errorf("%s is %v, not a number", method, v.byName[method])
	}
	return n
}

func (v *values) count(method string) int {
	n := v.num(method)
	if n < 0 || n > math.MaxInt32 {
		if v.err == nil {
			v.err = fmt.Errorf("%s is %d, not a count", method, n)
		}
		return 0
	}
	return int(n)
}

// status is a torrent in SlipStream's terms. d.state 0 is stopped, and a
// started torrent that is neither active nor hashing is paused. rTorrent
// has no queue and no moving state, so d.complete alone is finished.
// Seeding time counts from d.timestamp.finished.
func status(v *values) clients.Status {
	complete := v.num("d.complete") == 1
	s := clients.Status{
		Name: v.str("d.name"), Size: v.num("d.size_bytes"), Done: v.num("d.completed_bytes"),
		Run: clients.Started, Activity: clients.Downloading, Finished: complete,
		Folder: v.str("d.directory"), Label: v.str("d.custom1"),
		Counters: clients.Counters{Uploaded: v.num("d.up.total"), Downloaded: v.num("d.down.total")},
		Rates: clients.Rates{
			Down: v.num("d.down.rate"), Up: v.num("d.up.rate"),
			Peers: v.count("d.peers_connected"), Seeds: v.count("d.peers_complete"),
		},
	}
	hashing := v.num("d.hashing") != 0
	switch {
	case hashing:
		s.Activity = clients.Checking
	case complete:
		s.Activity = clients.Seeding
	}
	if v.num("d.state") == 0 || v.num("d.is_active") == 0 && !hashing {
		s.Run = clients.Paused
	}
	if v.num("d.is_multi_file") == 1 {
		s.Folder = parent(s.Folder)
	}
	if finished := v.num("d.timestamp.finished"); finished > 0 {
		s.FinishedAt = time.Unix(finished, 0)
		if complete {
			s.Counters.Seeding = time.Since(s.FinishedAt)
		}
	}
	s.LocalError, s.Trackers = message(v.str("d.message"))
	return s
}

// parent is the folder d.directory.set was given for a multi-file
// torrent, whose d.directory ends in its own folder.
func parent(directory string) string {
	if i := strings.LastIndexByte(directory, '/'); i > 0 {
		return directory[:i]
	}
	return "/"
}

// message reads d.message: a tracker's failure as rTorrent words it,
// `Tracker: [Failure reason "..."]`, or from 0.16 `Tracker: [Tracker
// warning: ...]` for a warning that says the torrent is unregistered.
// Any other message is a local error.
func message(m string) (string, []clients.TrackerMessage) {
	inner, ok := strings.CutPrefix(m, "Tracker: [")
	if !ok {
		return m, nil
	}
	inner = strings.TrimSuffix(inner, "]")
	if reason, ok := strings.CutPrefix(inner, `Failure reason "`); ok {
		inner = strings.TrimSuffix(reason, `"`)
	} else if warning, ok := strings.CutPrefix(inner, "Tracker warning: "); ok {
		inner = warning
	}
	return "", []clients.TrackerMessage{{Message: inner, Error: true}}
}

// Files is f.multicall under d.directory. f.path is under the torrent's
// own folder for a multi-file torrent, so the folder's name goes first,
// and a file's done bytes are its share of its completed chunks.
func (c *client) Files(ctx context.Context, h bencode.InfoHash) ([]clients.File, error) {
	results, err := c.multicall(ctx, []call{
		{method: "d.directory", params: []any{target(h)}},
		{method: "d.is_multi_file", params: []any{target(h)}},
		{method: "f.multicall", params: []any{target(h), "", "f.path=", "f.size_bytes=", "f.size_chunks=", "f.completed_chunks="}},
	})
	if err != nil {
		return nil, err
	}
	for _, r := range results {
		if err := asError(r); missing(err) {
			return nil, fmt.Errorf("the client doesn't list torrent %s", h)
		} else if err != nil {
			return nil, err
		}
	}
	directory, _ := results[0].(string)
	prefix := ""
	if multi, _ := results[1].(int64); multi == 1 {
		prefix = directory[strings.LastIndexByte(directory, '/')+1:] + "/"
	}
	rows, ok := results[2].([]any)
	if !ok {
		return nil, fmt.Errorf("the client responded to f.multicall with %v", results[2])
	}
	files := make([]clients.File, len(rows))
	for i, raw := range rows {
		row, _ := raw.([]any)
		v := values{byName: map[string]any{}}
		for j, f := range []string{"f.path", "f.size_bytes", "f.size_chunks", "f.completed_chunks"} {
			if j < len(row) {
				v.byName[f] = row[j]
			}
		}
		f := clients.File{Index: i, Path: prefix + v.str("f.path"), Size: v.num("f.size_bytes")}
		if chunks := v.num("f.size_chunks"); chunks > 0 {
			f.Done = min(f.Size*v.num("f.completed_chunks")/chunks, f.Size)
		}
		if v.err != nil {
			return nil, fmt.Errorf("read the client's files of torrent %s: %w", h, v.err)
		}
		files[i] = f
	}
	return files, nil
}

// acts are the d.* calls of each act. Resume starts a stopped torrent and
// resumes one paused in the client with d.pause.
var acts = map[clients.Act][]string{
	clients.Pause: {"d.stop"}, clients.Resume: {"d.start", "d.resume"},
	clients.Recheck: {"d.check_hash"}, clients.Announce: {"d.tracker_announce"},
}

// Act is one system.multicall over hashes. A hash rTorrent no longer
// holds is skipped.
func (c *client) Act(ctx context.Context, a clients.Act, hashes []bencode.InfoHash) error {
	var calls []call
	for _, h := range hashes {
		for _, m := range acts[a] {
			calls = append(calls, call{method: m, params: []any{target(h)}})
		}
	}
	results, err := c.multicall(ctx, calls)
	if err != nil {
		return err
	}
	for _, r := range results {
		if err := asError(r); err != nil && !missing(err) {
			return err
		}
	}
	return nil
}

// Remove is d.erase, which never deletes data, so the Node deletes a
// removed torrent's files itself when data is set. A torrent rTorrent no
// longer holds is already removed.
func (c *client) Remove(ctx context.Context, h bencode.InfoHash, _ bool) error {
	_, err := c.send(ctx, "d.erase", []any{target(h)})
	if missing(err) {
		return nil
	}
	return err
}
