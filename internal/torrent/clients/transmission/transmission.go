// Package transmission speaks Transmission's RPC, 4.0 and later: the
// legacy protocol, or JSON-RPC 2.0 with snake_case names from RPC version
// 6.0.0 (Transmission 4.1), picked once from the first 409's
// X-Transmission-Rpc-Version.
package transmission

import (
	"bytes"
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

const (
	sessionHeader = "X-Transmission-Session-Id"
	versionHeader = "X-Transmission-Rpc-Version"
	// limit bounds a response, which holds at most the torrents asked for.
	limit = 32 << 20
)

// json2Since is the first RPC version that speaks JSON-RPC 2.0.
var json2Since, _ = clients.ParseVersion("6.0.0")

// Open starts a session with the Transmission at s.Address. It does no IO.
func Open(s clients.Session) clients.Client { return open(s) }

func open(s clients.Session) *client {
	return &client{s: s, ran: map[bencode.InfoHash]clients.Run{}}
}

type client struct {
	s clients.Session
	// id is X-Transmission-Session-Id, "" until the first 409.
	id string
	// picked is set by the first 409, and json2 from its header.
	picked, json2 bool
	// version is the version session_get read, zero before.
	version clients.Version
	tag     int
	// admitted is a session that got a response past the rpc_whitelist.
	// The whitelist's 403 and the anti-brute-force 403 have one body, and
	// the whitelist doesn't change while the daemon runs, so a 403 to an
	// admitted session is the lockout and any other the whitelist.
	admitted bool
	// ran is what each torrent was last seen or asked to do, since
	// Transmission reports a torrent it verifies as verifying whether it
	// was stopped or started.
	ran map[bencode.InfoHash]clients.Run
}

// call sends method with args, whose names are snake_case, and decodes
// the result into out. It stores the session id and retries once on a
// 409. 401 is a refused sign-in, 403 the whitelist or the lockout, and
// 421 the host check.
func (c *client) call(ctx context.Context, method string, args map[string]any, out any) error {
	for retried := false; ; retried = true {
		res, err := c.send(ctx, method, args)
		if err != nil {
			return err
		}
		admitted := c.admitted
		c.admitted = admitted || res.Status != http.StatusForbidden
		switch res.Status {
		case http.StatusOK:
			return c.decode(method, res.Body, out)
		case http.StatusConflict:
			if retried {
				return errors.New("the client refused the session id it sent")
			}
			c.id = res.Header.Get(sessionHeader)
			if !c.picked {
				c.picked = true
				v, err := clients.ParseVersion(res.Header.Get(versionHeader))
				c.json2 = err == nil && !v.Less(json2Since)
			}
			continue
		case http.StatusUnauthorized:
			return &clients.Problem{Cause: clients.WrongCredentials}
		case http.StatusForbidden:
			if admitted {
				return &clients.Problem{Cause: clients.LockedOut}
			}
			return &clients.Problem{Cause: clients.AddressRefused}
		case http.StatusMisdirectedRequest:
			host, _, _ := net.SplitHostPort(c.s.Address.HostPort())
			return &clients.Problem{Cause: clients.HostRefused, Host: host}
		}
		return fmt.Errorf("the client responded to %s with status %d", method, res.Status)
	}
}

func (c *client) send(ctx context.Context, method string, args map[string]any) (clients.Response, error) {
	c.tag++
	msg := map[string]any{"method": legacy(method), "arguments": kebabKeys(args), "tag": c.tag}
	if c.json2 {
		msg = map[string]any{"jsonrpc": "2.0", "method": method, "params": args, "id": c.tag}
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return clients.Response{}, fmt.Errorf("encode %s: %w", method, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.s.Address.String(), bytes.NewReader(body))
	if err != nil {
		return clients.Response{}, fmt.Errorf("build %s: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.id != "" {
		req.Header.Set(sessionHeader, c.id)
	}
	if c.s.Username != "" || c.s.Secrets.Password != "" {
		req.SetBasicAuth(c.s.Username, c.s.Secrets.Password)
	}
	return clients.Do(ctx, c.s, req, limit)
}

func (c *client) decode(method string, body []byte, out any) error {
	if c.json2 {
		var res struct {
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
				Data    struct {
					ErrorString string `json:"error_string"`
				} `json:"data"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &res); err != nil {
			return fmt.Errorf("read the client's %s: %w", method, err)
		}
		if res.Error != nil {
			return fmt.Errorf("the client refused %s: %s", method, cmp.Or(res.Error.Data.ErrorString, res.Error.Message))
		}
		return unmarshal(method, res.Result, out)
	}
	var res struct {
		Result    string          `json:"result"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return fmt.Errorf("read the client's %s: %w", method, err)
	}
	if res.Result != "success" {
		return fmt.Errorf("the client refused %s: %s", method, res.Result)
	}
	args, err := snakeKeys(res.Arguments)
	if err != nil {
		return fmt.Errorf("read the client's %s: %w", method, err)
	}
	return unmarshal(method, args, out)
}

func unmarshal(method string, raw json.RawMessage, out any) error {
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("read the client's %s: %w", method, err)
	}
	return nil
}

type session struct {
	Version     string `json:"version"`
	DownloadDir string `json:"download_dir"`
}

func (c *client) session(ctx context.Context) (session, error) {
	var s session
	if err := c.call(ctx, "session_get", map[string]any{}, &s); err != nil {
		return session{}, err
	}
	v, err := clients.ParseVersion(s.Version)
	if err != nil {
		return session{}, fmt.Errorf("read the client's version: %w", err)
	}
	c.version = v
	return s, nil
}

func (c *client) Test(ctx context.Context, _ clients.Label) (clients.Tested, error) {
	if _, err := c.session(ctx); err != nil {
		return clients.Tested{}, err
	}
	if c.version.Less(clients.Transmission.Facts().Floor) {
		return clients.Tested{}, &clients.Problem{Cause: clients.VersionTooOld, Version: c.version}
	}
	return clients.Tested{Version: c.version}, nil
}

func (c *client) Version(ctx context.Context) (clients.Version, error) {
	if c.version == (clients.Version{}) {
		if _, err := c.session(ctx); err != nil {
			return clients.Version{}, err
		}
	}
	return c.version, nil
}

func (c *client) DefaultFolder(ctx context.Context, _ clients.Label) (string, error) {
	s, err := c.session(ctx)
	return s.DownloadDir, err
}

func (c *client) Add(ctx context.Context, a clients.Add) error {
	args := map[string]any{
		"metainfo": base64.StdEncoding.EncodeToString(a.Metainfo),
		"labels":   []string{a.Label.String()},
		"paused":   a.Paused,
	}
	if a.Folder != "" {
		args["download_dir"] = a.Folder
	}
	var res struct {
		Added *struct{} `json:"torrent_added"`
	}
	if err := c.call(ctx, "torrent_add", args, &res); err != nil {
		return err
	}
	if res.Added != nil {
		c.ran[a.Hash] = map[bool]clients.Run{false: clients.Started, true: clients.Paused}[a.Paused]
	}
	return nil
}

// torrent is one torrent_get row, by its snake_case names.
type torrent struct {
	Hash        string   `json:"hash_string"`
	Name        string   `json:"name"`
	Status      int      `json:"status"`
	Left        int64    `json:"left_until_done"`
	Size        int64    `json:"size_when_done"`
	Dir         string   `json:"download_dir"`
	Labels      []string `json:"labels"`
	Error       int      `json:"error"`
	ErrorString string   `json:"error_string"`
	// AtGoal is is_finished: the seeding goal was met, never the download
	// complete.
	AtGoal      bool  `json:"is_finished"`
	DoneDate    int64 `json:"done_date"`
	Uploaded    int64 `json:"uploaded_ever"`
	Downloaded  int64 `json:"downloaded_ever"`
	Seeding     int64 `json:"seconds_seeding"`
	Downloading int64 `json:"seconds_downloading"`
	RateDown    int64 `json:"rate_download"`
	RateUp      int64 `json:"rate_upload"`
	Peers       int   `json:"peers_connected"`
	Seeds       int   `json:"peers_sending_to_us"`
	Queue       int   `json:"queue_position"`
	Files       []struct {
		Name string `json:"name"`
		Size int64  `json:"length"`
		Done int64  `json:"bytes_completed"`
	} `json:"files"`
}

var pollFields = []string{
	"hash_string", "name", "status", "left_until_done", "size_when_done", "download_dir", "labels", "error", "error_string",
	"is_finished", "done_date", "uploaded_ever", "downloaded_ever", "seconds_seeding", "seconds_downloading",
	"rate_download", "rate_upload", "peers_connected", "peers_sending_to_us", "queue_position",
}

func (c *client) get(ctx context.Context, hashes []bencode.InfoHash, fields []string) ([]torrent, error) {
	named := make([]string, len(fields))
	for i, f := range fields {
		named[i] = c.field(f)
	}
	var res struct {
		Torrents []torrent `json:"torrents"`
	}
	if err := c.call(ctx, "torrent_get", map[string]any{"fields": named, "ids": hashStrings(hashes)}, &res); err != nil {
		return nil, err
	}
	return res.Torrents, nil
}

func (c *client) Poll(ctx context.Context, hashes []bencode.InfoHash) (map[bencode.InfoHash]clients.Status, error) {
	ts, err := c.get(ctx, hashes, pollFields)
	if err != nil {
		return nil, err
	}
	out := make(map[bencode.InfoHash]clients.Status, len(ts))
	for _, t := range ts {
		h, err := bencode.ParseInfoHash(t.Hash)
		if err != nil {
			return nil, fmt.Errorf("read the client's torrent_get: %w", err)
		}
		s := status(t)
		if s.Activity == clients.Checking {
			s.Run = cmp.Or(c.ran[h], clients.Started)
		}
		c.ran[h] = s.Run
		out[h] = s
	}
	return out, nil
}

// Transmission's tr_torrent_activity.
const (
	stopped = iota
	checkWait
	checking
	downloadWait
	downloading
	seedWait
	seeding
)

// Transmission's tr_stat_errtype.
const (
	trackerWarning = 1
	trackerError   = 2
	localError     = 3
)

// status is t in SlipStream's terms. A torrent is finished when nothing
// is left to download and it is stopped, queued to seed or seeding.
// is_finished alone means its seeding goal was met, so a stopped torrent
// with it was stopped at that goal.
func status(t torrent) clients.Status {
	s := clients.Status{
		Name: t.Name, Size: t.Size, Done: t.Size - t.Left, Run: clients.Started, Folder: t.Dir,
		Finished:       t.Left == 0 && (t.Status == stopped || t.Status == seedWait || t.Status == seeding),
		StoppedAtLimit: t.Status == stopped && t.AtGoal,
		Counters: clients.Counters{
			Uploaded: t.Uploaded, Downloaded: t.Downloaded,
			Active:  time.Duration(t.Downloading+t.Seeding) * time.Second,
			Seeding: time.Duration(t.Seeding) * time.Second,
		},
		Rates: clients.Rates{Down: t.RateDown, Up: t.RateUp, Peers: t.Peers, Seeds: t.Seeds},
	}
	switch t.Status {
	case stopped:
		s.Run, s.Activity = clients.Paused, clients.Downloading
		if t.Left == 0 {
			s.Activity = clients.Seeding
		}
	case checkWait, checking:
		s.Activity = clients.Checking
	case downloadWait, seedWait:
		s.Activity, s.QueuePosition = clients.Queued, t.Queue+1
	case downloading:
		s.Activity = clients.Downloading
	case seeding:
		s.Activity = clients.Seeding
	}
	if len(t.Labels) > 0 {
		s.Label = t.Labels[0]
	}
	if t.DoneDate > 0 {
		s.FinishedAt = time.Unix(t.DoneDate, 0)
	}
	switch t.Error {
	case trackerWarning, trackerError:
		s.Trackers = []clients.TrackerMessage{{Message: t.ErrorString, Error: t.Error == trackerError}}
	case localError:
		s.LocalError = t.ErrorString
	}
	return s
}

func (c *client) Files(ctx context.Context, h bencode.InfoHash) ([]clients.File, error) {
	ts, err := c.get(ctx, []bencode.InfoHash{h}, []string{"files"})
	if err != nil {
		return nil, err
	}
	if len(ts) == 0 {
		return nil, fmt.Errorf("the client doesn't list torrent %s", h)
	}
	files := make([]clients.File, len(ts[0].Files))
	for i, f := range ts[0].Files {
		files[i] = clients.File{Index: i, Path: f.Name, Size: f.Size, Done: f.Done}
	}
	return files, nil
}

var acts = map[clients.Act]string{
	clients.Pause: "torrent_stop", clients.Resume: "torrent_start", clients.Recheck: "torrent_verify", clients.Announce: "torrent_reannounce",
}

func (c *client) Act(ctx context.Context, a clients.Act, hashes []bencode.InfoHash) error {
	if err := c.call(ctx, acts[a], map[string]any{"ids": hashStrings(hashes)}, nil); err != nil {
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
	return c.call(ctx, "torrent_remove", map[string]any{"ids": hashStrings([]bencode.InfoHash{h}), "delete_local_data": data}, nil)
}

// Close drops the session. Transmission has no sign-out.
func (c *client) Close(context.Context) error {
	*c = *open(c.s)
	return nil
}

func hashStrings(hashes []bencode.InfoHash) []string {
	out := make([]string, len(hashes))
	for i, h := range hashes {
		out[i] = h.String()
	}
	return out
}
