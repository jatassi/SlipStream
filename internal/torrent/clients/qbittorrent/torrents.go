package qbittorrent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

// batch is how many hashes one torrents/info asks for, which keeps its
// URL inside a reverse proxy's limits.
const batch = 100

// json is a call whose 200 response is JSON, decoded into out.
func (c *client) json(ctx context.Context, rq request, out any) error {
	res, err := c.do(ctx, rq)
	if err != nil {
		return err
	}
	if res.Status != http.StatusOK {
		return fmt.Errorf("the client responded to %s with status %d", rq.call, res.Status)
	}
	if err := json.Unmarshal(res.Body, out); err != nil {
		return fmt.Errorf("read the client's %s: %w", rq.call, err)
	}
	return nil
}

// act is a call whose response has no body: 200, or 204 from 5.2.
func (c *client) act(ctx context.Context, rq request) error {
	res, err := c.do(ctx, rq)
	if err != nil {
		return err
	}
	if res.Status != http.StatusOK && res.Status != http.StatusNoContent {
		return fmt.Errorf("the client responded to %s with status %d", rq.call, res.Status)
	}
	return nil
}

type preferences struct {
	SavePath string `json:"save_path"`
	// RatioAct is max_ratio_act: 0 stops, 1 removes, 2 super seeds and 3
	// removes with the files.
	RatioAct   int  `json:"max_ratio_act"`
	RatioOn    bool `json:"max_ratio_enabled"`
	SeedingOn  bool `json:"max_seeding_time_enabled"`
	InactiveOn bool `json:"max_inactive_seeding_time_enabled"`
}

func (p preferences) removesAtLimit() bool {
	return (p.RatioAct == 1 || p.RatioAct == 3) && (p.RatioOn || p.SeedingOn || p.InactiveOn)
}

func (c *client) preferences(ctx context.Context) (preferences, error) {
	var p preferences
	err := c.json(ctx, get("app/preferences", nil), &p)
	return p, err
}

func (c *client) Test(ctx context.Context, _ clients.Label) (clients.Tested, error) {
	if err := c.versions(ctx); err != nil {
		return clients.Tested{}, err
	}
	if c.version.Less(clients.QBittorrent.Facts().Floor) {
		return clients.Tested{}, &clients.Problem{Cause: clients.VersionTooOld, Version: c.version}
	}
	p, err := c.preferences(ctx)
	if err != nil {
		return clients.Tested{}, err
	}
	tested := clients.Tested{Version: c.version}
	if p.removesAtLimit() {
		tested.Warnings = []clients.Warning{clients.RemovesAtLimit}
	}
	return tested, nil
}

// DefaultFolder is the category's save path, else the default save path.
func (c *client) DefaultFolder(ctx context.Context, label clients.Label) (string, error) {
	var categories map[string]struct {
		SavePath string `json:"savePath"`
	}
	if err := c.json(ctx, get("torrents/categories", nil), &categories); err != nil {
		return "", err
	}
	if cat := categories[label.String()]; cat.SavePath != "" {
		return cat.SavePath, nil
	}
	p, err := c.preferences(ctx)
	return p.SavePath, err
}

// Add posts the .torrent with its category, which qBittorrent creates
// when it has none by that name. Automatic management is off whenever a
// folder is sent, so the folder holds. A duplicate is a 409 from 5.2 and
// "Fails." before: the next poll resolves both.
func (c *client) Add(ctx context.Context, a clients.Add) error {
	if err := c.versions(ctx); err != nil {
		return err
	}
	stopped := "paused"
	if c.renamed() {
		stopped = "stopped"
	}
	fields := [][2]string{{"category", a.Label.String()}, {stopped, strconv.FormatBool(a.Paused)}}
	if a.Folder != "" {
		fields = append(fields, [2]string{"savepath", a.Folder}, [2]string{"autoTMM", "false"})
	}
	res, err := c.do(ctx, request{method: http.MethodPost, call: "torrents/add", body: func() (io.Reader, string, error) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		part, err := w.CreateFormFile("torrents", a.Hash.String()+".torrent")
		if err != nil {
			return nil, "", err //nolint:wrapcheck // build names the call
		}
		if _, err := part.Write(a.Metainfo); err != nil {
			return nil, "", err //nolint:wrapcheck // build names the call
		}
		for _, f := range fields {
			if err := w.WriteField(f[0], f[1]); err != nil {
				return nil, "", err //nolint:wrapcheck // build names the call
			}
		}
		return &b, w.FormDataContentType(), w.Close()
	}})
	switch {
	case err != nil:
		return err
	case res.Status == http.StatusConflict, res.Status == http.StatusOK && string(res.Body) == "Fails.":
		return nil
	case res.Status == http.StatusOK:
		c.ran[a.Hash] = map[bool]clients.Run{false: clients.Started, true: clients.Paused}[a.Paused]
		return nil
	case res.Status == http.StatusUnsupportedMediaType:
		return fmt.Errorf("the client refused the torrent: %s", res.Body)
	}
	return fmt.Errorf("the client responded to torrents/add with status %d", res.Status)
}

// torrent is one torrents/info row.
type torrent struct {
	Hash       string  `json:"hash"`
	Name       string  `json:"name"`
	State      string  `json:"state"`
	Size       int64   `json:"size"`
	Left       int64   `json:"amount_left"`
	SavePath   string  `json:"save_path"`
	Category   string  `json:"category"`
	Completion int64   `json:"completion_on"`
	Uploaded   int64   `json:"uploaded"`
	Downloaded int64   `json:"downloaded"`
	Active     int64   `json:"time_active"`
	Seeding    int64   `json:"seeding_time"`
	Ratio      float64 `json:"ratio"`
	// MaxRatio and MaxSeeding are the limits that apply to it, the global
	// ones or its own, -1 for none. MaxSeeding is in minutes.
	MaxRatio   float64 `json:"max_ratio"`
	MaxSeeding int64   `json:"max_seeding_time"`
	Down       int64   `json:"dlspeed"`
	Up         int64   `json:"upspeed"`
	Seeds      int     `json:"num_seeds"`
	Leeches    int     `json:"num_leechs"`
	Priority   int     `json:"priority"`
}

func (c *client) Poll(ctx context.Context, hashes []bencode.InfoHash) (map[bencode.InfoHash]clients.Status, error) {
	out := map[bencode.InfoHash]clients.Status{}
	if len(hashes) == 0 {
		_, err := c.text(ctx, get("app/webapiVersion", nil))
		return out, err
	}
	for start := 0; start < len(hashes); start += batch {
		var ts []torrent
		if err := c.json(ctx, get("torrents/info", url.Values{"hashes": {joinHashes(hashes[start:min(start+batch, len(hashes))])}}), &ts); err != nil {
			return nil, err
		}
		for _, t := range ts {
			h, s, err := c.report(ctx, t)
			if err != nil {
				return nil, err
			}
			out[h] = s
		}
	}
	return out, nil
}

// report is one torrents/info row by its hash, the run state a check
// hides taken from what the torrent was last seen or asked to do.
func (c *client) report(ctx context.Context, t torrent) (bencode.InfoHash, clients.Status, error) {
	h, err := bencode.ParseInfoHash(t.Hash)
	if err != nil {
		return h, clients.Status{}, fmt.Errorf("read the client's torrents/info: %w", err)
	}
	s, err := status(t)
	if err != nil {
		return h, clients.Status{}, err
	}
	if run, ok := c.ran[h]; ok && s.Activity == clients.Checking {
		s.Run = run
	}
	c.ran[h] = s.Run
	s.Trackers, err = c.trackersOf(ctx, h, t.State)
	return h, s, err
}

// row is what one qBittorrent state says.
type row struct {
	run      clients.Run
	activity clients.Activity
	// finished is the completion signal: any *UP state but checkingUP.
	finished bool
}

const (
	started = clients.Started
	paused  = clients.Paused
)

// states is qBittorrent's states, both the paused* spellings before
// WebAPI 2.11.0 and the stopped* ones from it. error and missingFiles are
// local errors, read apart.
var states = map[string]row{
	"downloading": {started, clients.Downloading, false}, "forcedDL": {started, clients.Downloading, false},
	"metaDL": {started, clients.Downloading, false}, "forcedMetaDL": {started, clients.Downloading, false},
	"stalledDL": {started, clients.Downloading, false}, "unknown": {started, clients.Downloading, false},
	"pausedDL": {paused, clients.Downloading, false}, "stoppedDL": {paused, clients.Downloading, false},
	"queuedDL": {started, clients.Queued, false}, "checkingDL": {started, clients.Checking, false},
	"checkingResumeData": {started, clients.Checking, false}, "checkingUP": {started, clients.Checking, false},
	"uploading": {started, clients.Seeding, true}, "forcedUP": {started, clients.Seeding, true},
	"stalledUP": {started, clients.Seeding, true}, "queuedUP": {started, clients.Queued, true},
	"pausedUP": {paused, clients.Seeding, true}, "stoppedUP": {paused, clients.Seeding, true},
	"moving": {started, clients.Moving, false},
}

// localErrors are the states of a local error, by the words qBittorrent's
// own torrent list shows for them, since the API gives no reason.
var localErrors = map[string]string{"error": "Errored", "missingFiles": "Missing Files"}

// status is t in SlipStream's terms.
func status(t torrent) (clients.Status, error) {
	r, ok := states[t.State]
	words, failed := localErrors[t.State]
	if failed {
		r = row{run: paused, activity: clients.Downloading}
		if t.Left == 0 {
			r.activity = clients.Seeding
		}
	} else if !ok {
		return clients.Status{}, fmt.Errorf("the client reports torrent %s in state %q, which SlipStream doesn't know", t.Hash, t.State)
	}
	s := clients.Status{
		Name: t.Name, Size: t.Size, Done: t.Size - t.Left, Run: r.run, Activity: r.activity, Finished: r.finished,
		Folder: t.SavePath, Label: t.Category, LocalError: words,
		StoppedAtLimit: r.finished && r.run == paused &&
			(t.MaxRatio >= 0 && t.Ratio >= t.MaxRatio || t.MaxSeeding >= 0 && t.Seeding >= t.MaxSeeding*60),
		Counters: clients.Counters{
			Uploaded: t.Uploaded, Downloaded: t.Downloaded,
			Active: time.Duration(t.Active) * time.Second, Seeding: time.Duration(t.Seeding) * time.Second,
		},
		Rates: clients.Rates{Down: t.Down, Up: t.Up, Peers: t.Seeds + t.Leeches, Seeds: t.Seeds},
	}
	if r.activity == clients.Queued {
		s.QueuePosition = t.Priority
	}
	if t.Completion > 0 {
		s.FinishedAt = time.Unix(t.Completion, 0)
	}
	return s, nil
}

// trackersOf is the tracker messages of a torrent in state, read from
// torrents/trackers only for a torrent in error or stalled, and at most
// once each TrackerEvery.
func (c *client) trackersOf(ctx context.Context, h bencode.InfoHash, state string) ([]clients.TrackerMessage, error) {
	if state != "error" && state != "stalledDL" && state != "stalledUP" {
		delete(c.trackers, h)
		return nil, nil
	}
	last, ok := c.trackers[h]
	if ok && time.Since(last.at) < c.s.TrackerEvery {
		return last.messages, nil
	}
	var rows []struct {
		URL    string `json:"url"`
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	res, err := c.do(ctx, get("torrents/trackers", url.Values{"hash": {h.String()}}))
	switch {
	case err != nil:
		return nil, err
	case res.Status == http.StatusNotFound:
		return nil, nil
	case res.Status != http.StatusOK:
		return nil, fmt.Errorf("the client responded to torrents/trackers with status %d", res.Status)
	}
	if err := json.Unmarshal(res.Body, &rows); err != nil {
		return nil, fmt.Errorf("read the client's torrents/trackers: %w", err)
	}
	var messages []clients.TrackerMessage
	for _, r := range rows {
		if r.Msg != "" && !strings.HasPrefix(r.URL, "** [") {
			messages = append(messages, clients.TrackerMessage{Message: r.Msg, Error: r.Status >= trackerNotWorking})
		}
	}
	c.trackers[h] = trackers{at: time.Now(), messages: messages}
	return messages, nil
}

// trackerNotWorking is the first of qBittorrent's tracker states that
// fail an announce: not working, then from WebAPI 2.13.0 tracker error
// and unreachable.
const trackerNotWorking = 4

func (c *client) Files(ctx context.Context, h bencode.InfoHash) ([]clients.File, error) {
	var rows []struct {
		Index    int     `json:"index"`
		Name     string  `json:"name"`
		Size     int64   `json:"size"`
		Progress float64 `json:"progress"`
	}
	if err := c.json(ctx, get("torrents/files", url.Values{"hash": {h.String()}}), &rows); err != nil {
		return nil, err
	}
	files := make([]clients.File, len(rows))
	for i, r := range rows {
		files[i] = clients.File{Index: r.Index, Path: r.Name, Size: r.Size, Done: int64(math.Round(r.Progress * float64(r.Size)))}
	}
	return files, nil
}

// acts is each Act's call before WebAPI 2.11.0 and from it.
var acts = map[clients.Act][2]string{
	clients.Pause: {"pause", "stop"}, clients.Resume: {"resume", "start"},
	clients.Recheck: {"recheck", "recheck"}, clients.Announce: {"reannounce", "reannounce"},
}

func (c *client) Act(ctx context.Context, a clients.Act, hashes []bencode.InfoHash) error {
	if err := c.versions(ctx); err != nil {
		return err
	}
	call := acts[a][0]
	if c.renamed() {
		call = acts[a][1]
	}
	if err := c.act(ctx, post("torrents/"+call, url.Values{"hashes": {joinHashes(hashes)}})); err != nil {
		return err
	}
	if run, ok := map[clients.Act]clients.Run{clients.Pause: clients.Paused, clients.Resume: clients.Started}[a]; ok {
		for _, h := range hashes {
			c.ran[h] = run
		}
	}
	return nil
}

func (c *client) Remove(ctx context.Context, h bencode.InfoHash, data bool) error {
	return c.act(ctx, post("torrents/delete", url.Values{"hashes": {h.String()}, "deleteFiles": {strconv.FormatBool(data)}}))
}

func joinHashes(hashes []bencode.InfoHash) string {
	s := make([]string, len(hashes))
	for i, h := range hashes {
		s[i] = h.String()
	}
	return strings.Join(s, "|")
}
