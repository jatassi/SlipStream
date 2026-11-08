package qbittorrent

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/torrentclient"
	"github.com/jatassi/slipstream/internal/bencode"
)

func (c *client) endpoints() map[string]endpoint {
	post := http.MethodPost
	e := map[string]endpoint{
		"auth/login":  {post, (*client).login},
		"auth/logout": {post, (*client).logout},
		"app/version": {"", func(c *client, w http.ResponseWriter, _ *http.Request, _ *session) {
			text(w, http.StatusOK, c.version.app)
		}},
		"app/webapiVersion": {"", func(c *client, w http.ResponseWriter, _ *http.Request, _ *session) {
			text(w, http.StatusOK, c.version.api)
		}},
		"app/preferences":     {"", (*client).preferences},
		"torrents/categories": {"", (*client).listCategories},
		"torrents/add":        {post, (*client).add},
		"torrents/info":       {"", (*client).info},
		"torrents/files":      {"", (*client).files},
		"torrents/trackers":   {"", (*client).trackers},
		"torrents/delete":     {post, (*client).delete},
		"torrents/recheck":    {post, each((*torrentclient.Engine).Verify)},
		"torrents/reannounce": {post, each((*torrentclient.Engine).Announce)},
	}
	stop, start := "torrents/pause", "torrents/resume"
	if c.version.stop {
		stop, start = "torrents/stop", "torrents/start"
	}
	e[stop] = endpoint{post, each((*torrentclient.Engine).Stop)}
	e[start] = endpoint{post, each((*torrentclient.Engine).Start)}
	return e
}

func writeJSON(w http.ResponseWriter, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (c *client) signedIn(w http.ResponseWriter) {
	if c.version.v52 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	text(w, http.StatusOK, "Ok.")
}

func (c *client) login(w http.ResponseWriter, r *http.Request, s *session) {
	if s != nil {
		c.signedIn(w)
		return
	}
	from := remote(r)
	if until, ok := c.banned[from]; ok && time.Now().Before(until) {
		text(w, http.StatusForbidden, "Your IP address has been banned after too many failed authentication attempts.")
		return
	}
	if r.PostFormValue("username") == c.Username && r.PostFormValue("password") == c.Password {
		delete(c.failures, from)
		c.start(w)
		c.signedIn(w)
		return
	}
	if c.BanAfter > 0 {
		c.failures[from]++
		if c.failures[from] >= c.BanAfter {
			delete(c.failures, from)
			c.banned[from] = time.Now().Add(banFor)
		}
	}
	if c.version.v52 {
		text(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	text(w, http.StatusOK, "Fails.")
}

func (c *client) logout(w http.ResponseWriter, _ *http.Request, s *session) {
	delete(c.sessions, s.id)
	http.SetCookie(w, &http.Cookie{Name: c.cookieName(), Path: "/", Expires: time.Now().Add(-24 * time.Hour)}) //nolint:gosec // qBittorrent's own expired cookie
	done(w)
}

func (c *client) preferences(w http.ResponseWriter, _ *http.Request, _ *session) {
	ratio := -1.0
	if c.SeedRatio > 0 {
		ratio = c.SeedRatio
	}
	writeJSON(w, map[string]any{
		"save_path": c.Folder, "temp_path_enabled": c.MoveFor > 0, "auto_tmm_enabled": false,
		"max_ratio_enabled": c.SeedRatio > 0, "max_ratio": ratio, "max_ratio_act": int(c.AtLimit),
		"max_seeding_time_enabled": false, "max_seeding_time": -1,
		"queueing_enabled": c.Slots > 0, "max_active_downloads": c.Slots, "web_ui_port": c.Port,
	})
}

func (c *client) listCategories(w http.ResponseWriter, _ *http.Request, _ *session) {
	out := map[string]any{}
	for n, save := range c.categories {
		cat := map[string]any{"name": n, "savePath": save}
		if c.version.v52 {
			cat["downloadPath"] = nil
		}
		out[n] = cat
	}
	writeJSON(w, out)
}

func (c *client) absolute(p string) bool {
	if c.Windows {
		return len(p) > 2 && p[1] == ':' && (p[2] == '\\' || p[2] == '/')
	}
	return strings.HasPrefix(p, "/")
}

func (c *client) join(dir, rel string) string {
	sep := "/"
	if c.Windows {
		sep, rel = `\`, strings.ReplaceAll(rel, "/", `\`)
	}
	return strings.TrimSuffix(dir, sep) + sep + rel
}

func parseBool(s string) bool { return strings.EqualFold(s, "true") }

// maxContent is qBittorrent's RequestParser::MAX_CONTENT_SIZE.
const maxContent = 64 << 20

// add is torrents/add for .torrent files. The category is created when
// it doesn't exist. Automatic management is off, qBittorrent's default,
// so a torrent with no save path goes to the default save path.
func (c *client) add(w http.ResponseWriter, r *http.Request, _ *session) {
	r.Body = http.MaxBytesReader(w, r.Body, maxContent)
	if err := r.ParseMultipartForm(maxContent); err != nil { //nolint:gosec // the body is bounded above
		text(w, http.StatusBadRequest, "Bad Request")
		return
	}
	category, save := r.PostFormValue("category"), strings.TrimSpace(r.PostFormValue("savepath"))
	stopped := "paused"
	if c.version.stop {
		stopped = "stopped"
	}
	paused := parseBool(r.PostFormValue(stopped))
	folder := save
	switch {
	case save == "":
		folder = c.Folder
	case !c.absolute(save):
		folder = c.join(c.Folder, save)
	}
	var added []string
	failed := 0
	for _, f := range r.MultipartForm.File["torrents"] {
		meta, err := readPart(f)
		m, decodeErr := bencode.DecodeMetainfo(meta)
		if err != nil || decodeErr != nil {
			text(w, http.StatusUnsupportedMediaType, fmt.Sprintf("Error: '%s' is not a valid torrent file.", f.Filename))
			return
		}
		if _, exists := c.categories[category]; category != "" && !exists {
			c.categories[category] = ""
		}
		var labels []string
		if category != "" {
			labels = []string{category}
		}
		t, duplicate, err := c.engine.Add(meta, folder, labels, paused)
		if err != nil || duplicate {
			failed++
			continue
		}
		c.announce[t.Hash] = m.Announce
		added = append(added, t.Hash.String())
	}
	switch {
	case c.version.v52 && len(added) == 0:
		text(w, http.StatusConflict, "Conflict")
	case c.version.v52:
		writeJSON(w, map[string]any{"success_count": len(added), "failure_count": failed, "pending_count": 0, "added_torrent_ids": added})
	case len(added) == 0:
		text(w, http.StatusOK, "Fails.")
	default:
		text(w, http.StatusOK, "Ok.")
	}
}

// hashes is the torrents a hashes parameter names, split at |.
func hashes(param string) []bencode.InfoHash {
	var named []bencode.InfoHash
	for p := range strings.SplitSeq(param, "|") {
		if h, err := bencode.ParseInfoHash(p); err == nil {
			named = append(named, h)
		}
	}
	return named
}

func each(act func(*torrentclient.Engine, bencode.InfoHash) error) func(*client, http.ResponseWriter, *http.Request, *session) {
	return func(c *client, w http.ResponseWriter, r *http.Request, _ *session) {
		if _ = r.ParseForm(); !r.PostForm.Has("hashes") {
			text(w, http.StatusBadRequest, "Missing required parameters")
			return
		}
		named := hashes(r.PostFormValue("hashes"))
		for _, t := range c.engine.List(nil) {
			if slices.Contains(named, t.Hash) {
				_ = act(c.engine, t.Hash)
			}
		}
		done(w)
	}
}

func (c *client) delete(w http.ResponseWriter, r *http.Request, _ *session) {
	_ = r.ParseForm()
	if !r.PostForm.Has("hashes") || !r.PostForm.Has("deleteFiles") {
		text(w, http.StatusBadRequest, "Missing required parameters")
		return
	}
	data := parseBool(r.PostFormValue("deleteFiles"))
	named := hashes(r.PostFormValue("hashes"))
	for _, t := range c.engine.List(nil) {
		if slices.Contains(named, t.Hash) {
			_ = c.engine.Remove(t.Hash, data)
		}
	}
	done(w)
}

func (c *client) info(w http.ResponseWriter, r *http.Request, _ *session) {
	all := c.engine.List(nil)
	var named []bencode.InfoHash
	if p := r.FormValue("hashes"); p != "" {
		named = hashes(p)
	}
	position := map[bencode.InfoHash]int{}
	for _, t := range all {
		if c.Slots > 0 && t.Done < t.Size {
			position[t.Hash] = len(position) + 1
		}
	}
	out := []map[string]any{}
	for _, t := range all {
		if named == nil || slices.Contains(named, t.Hash) {
			out = append(out, c.torrent(t, position[t.Hash]))
		}
	}
	writeJSON(w, out)
}

// state is the torrent's state in qBittorrent's words.
func (c *client) state(t torrentclient.Torrent) string {
	suffix := "DL"
	if t.Done == t.Size {
		suffix = "UP"
	}
	stopped := "paused"
	if c.version.stop {
		stopped = "stopped"
	}
	stalled := t.Tracker.Error || (t.State == torrentclient.Downloading && t.Rates.Down == 0) ||
		(t.State == torrentclient.Seeding && t.Rates.Up == 0)
	switch {
	case t.LocalError != "":
		return "error"
	case t.State == torrentclient.Checking:
		return "checking" + suffix
	case t.State == torrentclient.Stopped:
		return stopped + suffix
	case t.State == torrentclient.Queued:
		return "queued" + suffix
	case t.State == torrentclient.Seeding && time.Now().Before(t.CompletedAt.Add(c.MoveFor)):
		return "moving"
	case stalled:
		return "stalled" + suffix
	case suffix == "UP":
		return "uploading"
	}
	return "downloading"
}

func (c *client) torrent(t torrentclient.Torrent, position int) map[string]any {
	progress, ratio := 0.0, 0.0
	if t.Size > 0 {
		progress = float64(t.Done) / float64(t.Size)
	}
	if base := max(t.Downloaded, t.Done); base > 0 {
		ratio = float64(t.Uploaded) / float64(base)
	}
	maxRatio := -1.0
	if c.SeedRatio > 0 {
		maxRatio = c.SeedRatio
	}
	completion := int64(-1)
	if !t.CompletedAt.IsZero() {
		completion = t.CompletedAt.Unix()
	}
	content := c.join(t.Folder, t.Name)
	category := ""
	if len(t.Labels) > 0 {
		category = t.Labels[0]
	}
	rates := t.Rates
	if t.Tracker.Error {
		rates = torrentclient.Rates{}
	}
	out := map[string]any{
		"hash": t.Hash.String(), "infohash_v1": t.Hash.String(), "name": t.Name, "size": t.Size, "total_size": t.Size,
		"progress": progress, "amount_left": t.Size - t.Done, "completed": t.Done, "state": c.state(t),
		"dlspeed": rates.Down, "upspeed": rates.Up, "num_seeds": rates.Seeds, "num_leechs": rates.Peers - rates.Seeds,
		"priority": position, "category": category, "tags": "", "save_path": t.Folder, "content_path": content,
		"completion_on": completion, "downloaded": t.Downloaded, "uploaded": t.Uploaded, "ratio": ratio,
		"time_active": int64((t.Downloading + t.Seeding).Seconds()), "seeding_time": int64(t.Seeding.Seconds()),
		"max_ratio": maxRatio, "max_seeding_time": -1, "ratio_limit": -2, "seeding_time_limit": -2, "auto_tmm": false,
	}
	if c.version.stop {
		out["root_path"] = ""
		if len(t.Files) > 1 || (len(t.Files) == 1 && strings.Contains(t.Files[0].Path, "/")) {
			out["root_path"] = content
		}
	}
	return out
}

// torrentOf is the torrent the hash parameter names, false after a 404.
func (c *client) torrentOf(w http.ResponseWriter, r *http.Request) (torrentclient.Torrent, bool) {
	h, err := bencode.ParseInfoHash(r.FormValue("hash"))
	if err == nil {
		if t, ok := c.engine.Get(h); ok {
			return t, true
		}
	}
	text(w, http.StatusNotFound, "Not Found")
	return torrentclient.Torrent{}, false
}

func (c *client) files(w http.ResponseWriter, r *http.Request, _ *session) {
	t, ok := c.torrentOf(w, r)
	if !ok {
		return
	}
	out := make([]map[string]any, len(t.Files))
	for i, f := range t.Files {
		progress := 0.0
		if f.Size > 0 {
			progress = float64(f.Done) / float64(f.Size)
		}
		out[i] = map[string]any{"index": i, "name": f.Path, "size": f.Size, "progress": progress, "priority": 1, "availability": 1}
	}
	if len(out) > 0 {
		out[0]["is_seed"] = t.Done == t.Size
	}
	writeJSON(w, out)
}

// qBittorrent's TrackerEntryStatus states.
const (
	trackerWorking    = 2
	trackerNotWorking = 4
	trackerError      = 5
)

func (c *client) trackers(w http.ResponseWriter, r *http.Request, _ *session) {
	t, ok := c.torrentOf(w, r)
	if !ok {
		return
	}
	out := []map[string]any{}
	for _, sticky := range []string{"** [DHT] **", "** [PeX] **", "** [LSD] **"} {
		out = append(out, map[string]any{"url": sticky, "tier": "", "status": trackerWorking, "msg": "", "num_peers": 0})
	}
	status := trackerWorking
	if t.Tracker.Error {
		status = trackerNotWorking
		if c.version.v52 {
			status = trackerError
		}
	}
	out = append(out, map[string]any{"url": c.announce[t.Hash], "tier": 0, "status": status, "msg": t.Tracker.Message, "num_peers": 0})
	writeJSON(w, out)
}

func readPart(f *multipart.FileHeader) ([]byte, error) {
	part, err := f.Open()
	if err != nil {
		return nil, err //nolint:wrapcheck // the caller answers a 415
	}
	defer func() { _ = part.Close() }()
	return io.ReadAll(part) //nolint:wrapcheck // the caller answers a 415
}
