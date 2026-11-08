package deluge

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

type config struct {
	DownloadLocation string `json:"download_location"`
	MoveCompleted    bool   `json:"move_completed"`
	StopAtRatio      bool   `json:"stop_seed_at_ratio"`
	RemoveAtRatio    bool   `json:"remove_seed_at_ratio"`
}

// Test signs in and connects, then checks the version, the Label plugin
// and core.conf, and the label's own move where it has one. A Test that
// finds no daemon to connect to still answers the daemons the web UI
// lists, for the Admin to choose from.
func (c *client) Test(ctx context.Context, label clients.Label) (clients.Tested, error) {
	daemons, err := c.signIn(ctx)
	if err != nil {
		return clients.Tested{Daemons: daemons}, err
	}
	tested := clients.Tested{Version: c.version, Daemons: daemons}
	if c.version.Less(clients.Deluge.Facts().Floor) {
		return tested, &clients.Problem{Cause: clients.VersionTooOld, Version: c.version}
	}
	var plugins []string
	if err := c.call(ctx, "core.get_enabled_plugins", []any{}, &plugins); err != nil {
		return tested, err
	}
	if !slices.Contains(plugins, "Label") {
		return tested, &clients.Problem{Cause: clients.LabelPluginOff}
	}
	var cfg config
	if err := c.call(ctx, "core.get_config_values", []any{[]string{"download_location", "move_completed", "stop_seed_at_ratio", "remove_seed_at_ratio"}}, &cfg); err != nil {
		return tested, err
	}
	if cfg.StopAtRatio && cfg.RemoveAtRatio {
		tested.Warnings = []clients.Warning{clients.RemovesAtLimit}
	}
	tested.MovesAfterFinish = cfg.MoveCompleted
	labels, err := c.labels(ctx)
	if err != nil || !slices.Contains(labels, label.String()) {
		return tested, err
	}
	var opts struct {
		Apply bool `json:"apply_move_completed"`
		Move  bool `json:"move_completed"`
	}
	if err := c.call(ctx, "label.get_options", []any{label.String()}, &opts); err != nil {
		return tested, err
	}
	if opts.Apply {
		tested.MovesAfterFinish = opts.Move
	}
	return tested, nil
}

func (c *client) labels(ctx context.Context) ([]string, error) {
	var labels []string
	err := c.call(ctx, "label.get_labels", []any{}, &labels)
	return labels, err
}

// DefaultFolder is core.conf's download_location, since a label has no
// folder of its own.
func (c *client) DefaultFolder(ctx context.Context, _ clients.Label) (string, error) {
	var folder string
	err := c.call(ctx, "core.get_config_value", []any{"download_location"}, &folder)
	return folder, err
}

// Add sends the .torrent with core.add_torrent_file, then labels it. A
// torrent already in the session is code 4 "Torrent already in session",
// which still sets the label when the torrent lacks it, so an Add sent
// again heals a torrent whose label never arrived.
func (c *client) Add(ctx context.Context, a clients.Add) error {
	opts := map[string]any{"add_paused": a.Paused}
	if a.Folder != "" {
		opts["download_location"] = a.Folder
	}
	err := c.call(ctx, "core.add_torrent_file", []any{a.Hash.String() + ".torrent", base64.StdEncoding.EncodeToString(a.Metainfo), opts}, nil)
	switch {
	case failedWith(err, "Torrent already in session"):
		var s struct {
			Label *string `json:"label"`
		}
		if err := c.call(ctx, "core.get_torrent_status", []any{a.Hash.String(), []string{"label"}}, &s); err != nil {
			return err
		}
		if s.Label != nil && *s.Label == a.Label.String() {
			return nil
		}
	case err != nil:
		return err
	}
	return c.label(ctx, a.Hash, a.Label)
}

// label sets the label on h, creating it first when the daemon has none
// by that name. A torrent the daemon doesn't hold is the next poll's.
func (c *client) label(ctx context.Context, h bencode.InfoHash, label clients.Label) error {
	labels, err := c.labels(ctx)
	if err != nil {
		return err
	}
	if !slices.Contains(labels, label.String()) {
		if err := c.call(ctx, "label.add", []any{label.String()}, nil); err != nil {
			return err
		}
	}
	err = c.call(ctx, "label.set_torrent", []any{h.String(), label.String()}, nil)
	if failedWith(err, "Unknown Torrent") {
		return nil
	}
	return err
}

// keys are the status keys a poll reads.
var keys = []string{
	"name", "state", "total_wanted", "total_done", "save_path", "label", "is_finished", "completed_time", "queue",
	"message", "tracker_status", "total_uploaded", "all_time_download", "active_time", "seeding_time",
	"download_payload_rate", "upload_payload_rate", "num_peers", "num_seeds", "ratio", "stop_at_ratio", "stop_ratio",
	"move_completed", "move_completed_path",
}

// torrent is one torrent's core.get_torrents_status row.
type torrent struct {
	Name     string `json:"name"`
	State    string `json:"state"`
	Size     int64  `json:"total_wanted"`
	Done     int64  `json:"total_done"`
	SavePath string `json:"save_path"`
	Label    string `json:"label"`
	Finished bool   `json:"is_finished"`
	// Completed is completed_time in seconds, 0 before.
	Completed float64 `json:"completed_time"`
	// Queue is the place in the queue from 0, -1 for none.
	Queue         int     `json:"queue"`
	Message       string  `json:"message"`
	Tracker       string  `json:"tracker_status"`
	Uploaded      int64   `json:"total_uploaded"`
	Downloaded    int64   `json:"all_time_download"`
	Active        float64 `json:"active_time"`
	Seeding       float64 `json:"seeding_time"`
	Down          float64 `json:"download_payload_rate"`
	Up            float64 `json:"upload_payload_rate"`
	Peers         int     `json:"num_peers"`
	Seeds         int     `json:"num_seeds"`
	Ratio         float64 `json:"ratio"`
	StopAtRatio   bool    `json:"stop_at_ratio"`
	StopRatio     float64 `json:"stop_ratio"`
	MoveCompleted bool    `json:"move_completed"`
	MovePath      string  `json:"move_completed_path"`
}

func (c *client) Poll(ctx context.Context, hashes []bencode.InfoHash) (map[bencode.InfoHash]clients.Status, error) {
	if err := c.onDaemon(ctx); err != nil {
		return nil, err
	}
	ids := make([]string, len(hashes))
	for i, h := range hashes {
		ids[i] = h.String()
	}
	var rows map[string]torrent
	if err := c.call(ctx, "core.get_torrents_status", []any{map[string]any{"id": ids}, keys}, &rows); err != nil {
		return nil, err
	}
	out := make(map[bencode.InfoHash]clients.Status, len(rows))
	for id, t := range rows {
		h, err := bencode.ParseInfoHash(id)
		if err != nil {
			return nil, fmt.Errorf("read the client's core.get_torrents_status: %w", err)
		}
		if out[h], err = status(id, t); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// row is what one Deluge state says, Paused and Error read apart.
type row struct {
	run      clients.Run
	activity clients.Activity
}

var states = map[string]row{
	"Downloading": {clients.Started, clients.Downloading}, "Allocating": {clients.Started, clients.Downloading},
	"Seeding": {clients.Started, clients.Seeding}, "Checking": {clients.Started, clients.Checking},
	"Queued": {clients.Started, clients.Queued}, "Moving": {clients.Started, clients.Moving},
}

// status is t in SlipStream's terms. It is finished when is_finished
// holds, it isn't Moving and, with move completed on, it is in the move's
// folder.
func status(id string, t torrent) (clients.Status, error) {
	r, ok := states[t.State]
	switch {
	case t.State == "Paused", t.State == "Error":
		r = row{clients.Paused, clients.Downloading}
		if t.Done == t.Size {
			r.activity = clients.Seeding
		}
	case !ok:
		return clients.Status{}, fmt.Errorf("the client reports torrent %s in state %q, which SlipStream doesn't know", id, t.State)
	}
	finished := t.Finished && t.State != "Moving" && (!t.MoveCompleted || t.SavePath == t.MovePath)
	s := clients.Status{
		Name: t.Name, Size: t.Size, Done: t.Done, Run: r.run, Activity: r.activity, Finished: finished,
		Folder: t.SavePath, Label: t.Label,
		StoppedAtLimit: finished && t.State == "Paused" && t.StopAtRatio && t.Ratio >= t.StopRatio,
		Counters: clients.Counters{
			Uploaded: t.Uploaded, Downloaded: t.Downloaded,
			Active: seconds(t.Active), Seeding: seconds(t.Seeding),
		},
		Rates: clients.Rates{Down: int64(t.Down), Up: int64(t.Up), Peers: t.Peers + t.Seeds, Seeds: t.Seeds},
	}
	if t.State == "Error" {
		s.LocalError = t.Message
	}
	if t.State == "Queued" && t.Queue >= 0 {
		s.QueuePosition = t.Queue + 1
	}
	if t.Completed > 0 {
		s.FinishedAt = time.Unix(int64(t.Completed), 0)
	}
	if message, ok := strings.CutPrefix(t.Tracker, "Error: "); ok {
		s.Trackers = []clients.TrackerMessage{{Message: message, Error: true}}
	} else if message, ok := strings.CutPrefix(t.Tracker, "Warning: "); ok {
		s.Trackers = []clients.TrackerMessage{{Message: message}}
	}
	return s, nil
}

func seconds(s float64) time.Duration { return time.Duration(s * float64(time.Second)) }

func (c *client) Files(ctx context.Context, h bencode.InfoHash) ([]clients.File, error) {
	var s struct {
		Files []struct {
			Index int    `json:"index"`
			Path  string `json:"path"`
			Size  int64  `json:"size"`
		} `json:"files"`
		Progress []float64 `json:"file_progress"`
	}
	if err := c.call(ctx, "core.get_torrent_status", []any{h.String(), []string{"files", "file_progress"}}, &s); err != nil {
		return nil, err
	}
	if s.Files == nil {
		return nil, fmt.Errorf("the client doesn't list torrent %s", h)
	}
	if len(s.Progress) != len(s.Files) {
		return nil, fmt.Errorf("the client lists %d files of torrent %s and the progress of %d", len(s.Files), h, len(s.Progress))
	}
	files := make([]clients.File, len(s.Files))
	for i, f := range s.Files {
		files[i] = clients.File{Index: f.Index, Path: f.Path, Size: f.Size, Done: int64(math.Round(s.Progress[i] * float64(f.Size)))}
	}
	return files, nil
}

var acts = map[clients.Act]string{
	clients.Pause: "core.pause_torrents", clients.Resume: "core.resume_torrents",
	clients.Recheck: "core.force_recheck", clients.Announce: "core.force_reannounce",
}

func (c *client) Act(ctx context.Context, a clients.Act, hashes []bencode.InfoHash) error {
	ids := make([]string, len(hashes))
	for i, h := range hashes {
		ids[i] = h.String()
	}
	return c.call(ctx, acts[a], []any{ids}, nil)
}

// Remove is core.remove_torrent. A torrent the daemon no longer holds is
// already removed.
func (c *client) Remove(ctx context.Context, h bencode.InfoHash, data bool) error {
	err := c.call(ctx, "core.remove_torrent", []any{h.String(), data}, nil)
	if failedWith(err, "not in session") {
		return nil
	}
	return err
}
