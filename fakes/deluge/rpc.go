package deluge

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/torrentclient"
	"github.com/jatassi/slipstream/internal/bencode"
)

// rpcError is a JSON-RPC error: 1 not authenticated, 2 unknown method, 3
// a local method's exception, 4 a daemon's failure and 5 a request the
// web UI can't decode.
type rpcError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

var (
	notAuthenticated = &rpcError{Message: "Not authenticated", Code: 1}
	unknownMethod    = &rpcError{Message: "Unknown method", Code: 2}
)

// failure is code 4 in the words json_api.py gives it: the Twisted
// Failure's class and its repr around the daemon's exception.
func failure(class, message string) *rpcError {
	return &rpcError{Code: 4, Message: fmt.Sprintf(
		"Failure: [Failure instance: Traceback (failure with no frames): <class 'deluge.error.%s'>: %s\n]", class, message)}
}

// undecodable is code 5, which answers with a null id.
func undecodable(message string) *rpcError {
	return &rpcError{Code: 5, Message: "JSONException: " + message}
}

// params are a call's positional parameters.
type params []json.RawMessage

func (p params) arg(i int, out any) error {
	if i >= len(p) {
		return fmt.Errorf("missing %d required positional argument", i+1)
	}
	if err := json.Unmarshal(p[i], out); err != nil {
		return fmt.Errorf("argument %d: %w", i+1, err)
	}
	return nil
}

// local is a web UI method. An open one needs no session.
type local struct {
	open bool
	call func(w *web, rw http.ResponseWriter, session string, p params) (any, *rpcError)
}

var locals = map[string]local{
	"auth.login": {true, (*web).login},
	"auth.check_session": {true, func(_ *web, _ http.ResponseWriter, session string, _ params) (any, *rpcError) {
		return session != "", nil
	}},
	"auth.delete_session": {false, func(w *web, _ http.ResponseWriter, session string, _ params) (any, *rpcError) {
		delete(w.sessions, session)
		return true, nil
	}},
	"web.connected": {false, func(w *web, _ http.ResponseWriter, _ string, _ params) (any, *rpcError) {
		return w.connected >= 0, nil
	}},
	"web.get_hosts":       {false, (*web).hosts},
	"web.connect":         {false, (*web).connectHost},
	"web.get_host_status": {false, (*web).hostStatus},
}

// remote is a daemon method, run on the connected daemon.
type remote func(d *daemon, p params) (any, *rpcError)

var core = map[string]remote{
	"daemon.get_version":       (*daemon).getVersion,
	"daemon.info":              (*daemon).getVersion,
	"core.get_enabled_plugins": (*daemon).plugins,
	"core.get_config_value":    (*daemon).configValue,
	"core.get_config_values":   (*daemon).configValues,
	"core.add_torrent_file":    (*daemon).add,
	"core.get_torrents_status": (*daemon).torrentsStatus,
	"core.get_torrent_status":  (*daemon).torrentStatus,
	"core.pause_torrents":      (*daemon).pause,
	"core.resume_torrents":     (*daemon).resume,
	"core.force_recheck":       each((*torrentclient.Engine).Verify),
	"core.force_reannounce":    each((*torrentclient.Engine).Announce),
	"core.remove_torrent":      (*daemon).remove,
}

var label = map[string]remote{
	"label.get_labels":  (*daemon).getLabels,
	"label.add":         (*daemon).addLabel,
	"label.get_options": (*daemon).labelOptions,
	"label.set_torrent": (*daemon).setLabel,
}

// remoteMethod is method among the methods of the daemon the web UI last
// connected to.
func (w *web) remoteMethod(method string) (remote, bool) {
	if w.methods < 0 {
		return nil, false
	}
	if m, ok := core[method]; ok {
		return m, true
	}
	m, ok := label[method]
	return m, ok && !w.daemons[w.methods].LabelOff
}

func (s *server) handle(rw http.ResponseWriter, r *http.Request) {
	w, ok := s.clients[fake.OriginOf(r.Context())]
	if !ok {
		http.Error(rw, "fake-deluge plays no Deluge for this Node.", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	w.mu.Lock()
	id, result, rpcErr, wait := w.serve(rw, r, body)
	w.mu.Unlock()
	if wait {
		<-r.Context().Done()
		return
	}
	reply(rw, id, result, rpcErr)
}

// serve is one request under the web UI's lock. wait is a call to a
// daemon the web UI is no longer connected to, which never answers.
func (w *web) serve(rw http.ResponseWriter, r *http.Request, body []byte) (id json.RawMessage, result any, rpcErr *rpcError, wait bool) {
	w.sweep()
	contentType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if contentType != "application/json" {
		return nil, nil, undecodable("Invalid JSON request content-type: " + contentType), false
	}
	var req map[string]json.RawMessage
	if json.Unmarshal(body, &req) != nil {
		return nil, nil, undecodable("JSON not decodable"), false
	}
	for _, key := range []string{"method", "params", "id"} {
		if _, ok := req[key]; !ok {
			return nil, nil, undecodable(fmt.Sprintf("Invalid JSON request, missing param '%s' in %s", key, body)), false
		}
	}
	id = req["id"]
	var method string
	var p params
	if json.Unmarshal(req["method"], &method) != nil || json.Unmarshal(req["params"], &p) != nil {
		return id, nil, &rpcError{Code: 3, Message: "TypeError: the method or params aren't a string and a list"}, false
	}
	session := w.session(rw, r)
	signedIn := session != ""
	if l, ok := locals[method]; ok {
		if !l.open && !signedIn {
			return id, nil, notAuthenticated, false
		}
		result, rpcErr = l.call(w, rw, session, p)
		return id, result, rpcErr, false
	}
	m, ok := w.remoteMethod(method)
	switch {
	case !ok:
		return id, nil, unknownMethod, false
	case !signedIn:
		return id, nil, notAuthenticated, false
	case w.connected < 0:
		return id, nil, nil, true
	}
	result, rpcErr = m(w.daemons[w.connected], p)
	return id, result, rpcErr, false
}

func reply(rw http.ResponseWriter, id json.RawMessage, result any, rpcErr *rpcError) {
	if id == nil {
		id = json.RawMessage("null")
	}
	body, err := json.Marshal(map[string]any{"result": result, "error": rpcErr, "id": id})
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	_, _ = rw.Write(body)
}

func typeError(err error) *rpcError {
	return &rpcError{Code: 3, Message: "TypeError: " + err.Error()}
}

func (w *web) login(rw http.ResponseWriter, _ string, p params) (any, *rpcError) {
	var password string
	if err := p.arg(0, &password); err != nil {
		return nil, typeError(err)
	}
	if password != w.Password {
		return false, nil
	}
	w.newSession(rw)
	return true, nil
}

func (w *web) hosts(http.ResponseWriter, string, params) (any, *rpcError) {
	out := [][]any{}
	for _, d := range w.daemons {
		out = append(out, []any{d.id, d.Host, d.Port, "localclient"})
	}
	return out, nil
}

func (w *web) daemonByID(p params) (at int, id string, rpcErr *rpcError) {
	if err := p.arg(0, &id); err != nil {
		return -1, "", typeError(err)
	}
	return slices.IndexFunc(w.daemons, func(d *daemon) bool { return d.id == id }), id, nil
}

// connectHost is web.connect: the daemon's methods, or null when the host
// id is unknown, as the connect's errback swallows the failure.
func (w *web) connectHost(_ http.ResponseWriter, _ string, p params) (any, *rpcError) {
	i, _, rpcErr := w.daemonByID(p)
	if rpcErr != nil || i < 0 {
		return nil, rpcErr
	}
	w.connect(i)
	var methods []string
	for m := range core {
		methods = append(methods, m)
	}
	if !w.daemons[i].LabelOff {
		for m := range label {
			methods = append(methods, m)
		}
	}
	sort.Strings(methods)
	return methods, nil
}

func (w *web) hostStatus(_ http.ResponseWriter, _ string, p params) (any, *rpcError) {
	i, id, rpcErr := w.daemonByID(p)
	switch {
	case rpcErr != nil:
		return nil, rpcErr
	case i < 0:
		return []any{id, "Offline", ""}, nil
	case i == w.connected && ipAddress(w.daemons[i].Host):
		return []any{id, "Connected", w.version}, nil
	}
	return []any{id, "Online", w.version}, nil
}

func (d *daemon) getVersion(params) (any, *rpcError) { return d.version, nil }

func (d *daemon) plugins(params) (any, *rpcError) {
	if d.LabelOff {
		return []string{}, nil
	}
	return []string{"Label"}, nil
}

func (d *daemon) config() map[string]any {
	ratio := 2.0
	if d.SeedRatio > 0 {
		ratio = d.SeedRatio
	}
	return map[string]any{
		"download_location": d.Folder, "move_completed": d.MoveTo != "", "move_completed_path": d.MoveTo,
		"stop_seed_at_ratio": d.SeedRatio > 0, "stop_seed_ratio": ratio, "remove_seed_at_ratio": d.RemoveAtRatio,
		"max_active_downloading": d.Slots, "add_paused": false,
	}
}

func (d *daemon) configValue(p params) (any, *rpcError) {
	var key string
	if err := p.arg(0, &key); err != nil {
		return nil, failure("WrappedException", "TypeError: "+err.Error())
	}
	return d.config()[key], nil
}

func (d *daemon) configValues(p params) (any, *rpcError) {
	var keys []string
	if err := p.arg(0, &keys); err != nil {
		return nil, failure("WrappedException", "TypeError: "+err.Error())
	}
	cfg, out := d.config(), map[string]any{}
	for _, k := range keys {
		out[k] = cfg[k]
	}
	return out, nil
}

type addOptions struct {
	DownloadLocation  *string `json:"download_location"`
	AddPaused         *bool   `json:"add_paused"`
	MoveCompleted     *bool   `json:"move_completed"`
	MoveCompletedPath *string `json:"move_completed_path"`
}

// add is core.add_torrent_file: the hash, or AddTorrentError for a
// torrent already in the session. Options left out take core.conf's.
func (d *daemon) add(p params) (any, *rpcError) {
	var filedump string
	var opts addOptions
	if err := errors.Join(p.arg(1, &filedump), p.arg(2, &opts)); err != nil {
		return nil, failure("WrappedException", "TypeError: "+err.Error())
	}
	meta, err := base64.StdEncoding.DecodeString(filedump)
	if err != nil {
		return nil, failure("AddTorrentError", "Unable to add torrent, decoding filedump failed: "+err.Error())
	}
	folder, paused := d.Folder, false
	if opts.DownloadLocation != nil {
		folder = *opts.DownloadLocation
	}
	if opts.AddPaused != nil {
		paused = *opts.AddPaused
	}
	move, moveTo := d.MoveTo != "", d.MoveTo
	if opts.MoveCompleted != nil {
		move = *opts.MoveCompleted
	}
	if opts.MoveCompletedPath != nil {
		moveTo = *opts.MoveCompletedPath
	}
	t, duplicate, err := d.engine.Add(meta, folder, nil, paused)
	switch {
	case errors.Is(err, torrentclient.ErrCorrupt):
		return nil, failure("AddTorrentError", "Unable to add torrent, decoding filedump failed: "+err.Error())
	case err != nil:
		return nil, failure("AddTorrentError", "Unable to add torrent to session: "+err.Error())
	case duplicate:
		return nil, failure("AddTorrentError", fmt.Sprintf("Torrent already in session (%s).", t.Hash))
	}
	d.paused[t.Hash] = paused
	if move && moveTo != "" && moveTo != folder {
		d.moveTo[t.Hash] = moveTo
	}
	return t.Hash.String(), nil
}

func ids(raw []string) []bencode.InfoHash {
	var out []bencode.InfoHash
	for _, s := range raw {
		if h, err := bencode.ParseInfoHash(s); err == nil {
			out = append(out, h)
		}
	}
	return out
}

// torrentsStatus is core.get_torrents_status filtered by id, the only
// filter this fake plays, each torrent with the keys asked for.
func (d *daemon) torrentsStatus(p params) (any, *rpcError) {
	var filter struct {
		ID []string `json:"id"`
	}
	var keys []string
	if err := errors.Join(p.arg(0, &filter), p.arg(1, &keys)); err != nil {
		return nil, failure("WrappedException", "TypeError: "+err.Error())
	}
	var named []bencode.InfoHash
	if filter.ID != nil {
		named = ids(filter.ID)
		if named == nil {
			named = []bencode.InfoHash{}
		}
	}
	all := d.engine.List(nil)
	out := map[string]any{}
	for _, t := range all {
		if named == nil || slices.Contains(named, t.Hash) {
			out[t.Hash.String()] = d.status(t, all, keys)
		}
	}
	return out, nil
}

// torrentStatus is core.get_torrent_status: {} for a torrent the daemon
// doesn't have.
func (d *daemon) torrentStatus(p params) (any, *rpcError) {
	var id string
	var keys []string
	if err := errors.Join(p.arg(0, &id), p.arg(1, &keys)); err != nil {
		return nil, failure("WrappedException", "TypeError: "+err.Error())
	}
	all := d.engine.List(nil)
	for _, t := range all {
		if t.Hash.String() == id {
			return d.status(t, all, keys), nil
		}
	}
	return map[string]any{}, nil
}

// state is the torrent's state in Deluge's words, by update_state's order.
func (d *daemon) state(t torrentclient.Torrent) string {
	_, moving := d.moveTo[t.Hash]
	switch {
	case t.LocalError != "":
		return "Error"
	case moving && t.Done == t.Size:
		return "Moving"
	case t.State == torrentclient.Checking && d.paused[t.Hash]:
		return "Paused"
	case t.State == torrentclient.Checking:
		return "Checking"
	case t.State == torrentclient.Stopped:
		return "Paused"
	case t.State == torrentclient.Queued:
		return "Queued"
	case t.State == torrentclient.Seeding:
		return "Seeding"
	}
	return "Downloading"
}

func (d *daemon) status(t torrentclient.Torrent, all []torrentclient.Torrent, keys []string) map[string]any {
	moveTo, moving := d.moveTo[t.Hash]
	if !moving {
		moveTo = d.MoveTo
	}
	ratio := -1.0
	if t.Done > 0 {
		ratio = float64(t.Uploaded) / float64(t.Done)
	}
	message := "OK"
	if t.LocalError != "" {
		message = t.LocalError
	}
	completed := int64(0)
	if !t.CompletedAt.IsZero() {
		completed = t.CompletedAt.Unix()
	}
	progress := 0.0
	if t.Size > 0 {
		progress = 100 * float64(t.Done) / float64(t.Size)
	}
	cfg := d.config()
	files, fileProgress := fileStatus(t)
	every := map[string]any{
		"hash": t.Hash.String(), "name": t.Name, "state": d.state(t), "paused": t.State == torrentclient.Stopped,
		"total_size": t.Size, "total_wanted": t.Size, "total_done": t.Done, "progress": progress,
		"save_path": t.Folder, "download_location": t.Folder, "is_finished": t.Done == t.Size && !moving,
		"completed_time": completed, "queue": d.queue(t, all), "message": message, "tracker_status": trackerStatus(t.Tracker),
		"total_uploaded": t.Uploaded, "all_time_download": t.Downloaded,
		"active_time": int64((t.Downloading + t.Seeding).Seconds()), "seeding_time": int64(t.Seeding.Seconds()),
		"download_payload_rate": t.Rates.Down, "upload_payload_rate": t.Rates.Up,
		"num_peers": t.Rates.Peers - t.Rates.Seeds, "num_seeds": t.Rates.Seeds, "ratio": ratio,
		"stop_at_ratio": cfg["stop_seed_at_ratio"], "stop_ratio": cfg["stop_seed_ratio"], "remove_at_ratio": d.RemoveAtRatio,
		"move_completed": moving, "move_completed_path": moveTo, "files": files, "file_progress": fileProgress,
	}
	if !d.LabelOff {
		every["label"] = ""
		if len(t.Labels) > 0 {
			every["label"] = t.Labels[0]
		}
	}
	out := map[string]any{}
	for _, k := range keys {
		if v, ok := every[k]; ok {
			out[k] = v
		}
	}
	return out
}

// queue is the torrent's place among the incomplete ones from 0, -1 for
// a complete one or with no queue.
func (d *daemon) queue(t torrentclient.Torrent, all []torrentclient.Torrent) int {
	if d.Slots == 0 || t.Done == t.Size {
		return -1
	}
	n := 0
	for _, other := range all {
		if other.Hash == t.Hash {
			break
		}
		if other.Done < other.Size {
			n++
		}
	}
	return n
}

func trackerStatus(tr torrentclient.Tracker) string {
	switch {
	case tr.Message == "":
		return "Announce OK"
	case tr.Error:
		return "Error: " + tr.Message
	}
	return "Warning: " + tr.Message
}

func fileStatus(t torrentclient.Torrent) (files []map[string]any, progress []float64) {
	files = make([]map[string]any, len(t.Files))
	progress = make([]float64, len(t.Files))
	offset := int64(0)
	for i, f := range t.Files {
		files[i] = map[string]any{"index": i, "path": f.Path, "size": f.Size, "offset": offset}
		offset += f.Size
		if f.Size > 0 {
			progress[i] = float64(f.Done) / float64(f.Size)
		}
	}
	return files, progress
}

func each(act func(*torrentclient.Engine, bencode.InfoHash) error) remote {
	return func(d *daemon, p params) (any, *rpcError) {
		var raw []string
		if err := p.arg(0, &raw); err != nil {
			return nil, failure("WrappedException", "TypeError: "+err.Error())
		}
		for _, s := range raw {
			h, err := bencode.ParseInfoHash(s)
			if err != nil || act(d.engine, h) != nil {
				return nil, failure("WrappedException", fmt.Sprintf("KeyError: '%s'", s))
			}
		}
		return nil, nil
	}
}

func (d *daemon) pause(p params) (any, *rpcError) {
	return each(func(e *torrentclient.Engine, h bencode.InfoHash) error {
		d.paused[h] = true
		return e.Stop(h)
	})(d, p)
}

// resume is core.resume_torrents, which skips a torrent that has met its
// stop ratio, as Torrent.resume does.
func (d *daemon) resume(p params) (any, *rpcError) {
	return each(func(e *torrentclient.Engine, h bencode.InfoHash) error {
		if t, ok := e.Get(h); ok && t.SeedGoalMet {
			return nil
		}
		d.paused[h] = false
		return e.Start(h)
	})(d, p)
}

func (d *daemon) remove(p params) (any, *rpcError) {
	var id string
	var data bool
	if err := errors.Join(p.arg(0, &id), p.arg(1, &data)); err != nil {
		return nil, failure("WrappedException", "TypeError: "+err.Error())
	}
	h, err := bencode.ParseInfoHash(id)
	if err != nil || d.engine.Remove(h, data) != nil {
		return nil, failure("InvalidTorrentError", fmt.Sprintf("torrent_id %s not in session.", id))
	}
	delete(d.moveTo, h)
	delete(d.paused, h)
	return true, nil
}

// validLabel is the Label plugin's RE_VALID.
var validLabel = regexp.MustCompile(`^[a-z0-9_\-.]*$`)

func (d *daemon) getLabels(params) (any, *rpcError) {
	out := []string{}
	for l := range d.labels {
		out = append(out, l)
	}
	sort.Strings(out)
	return out, nil
}

// pluginError is an Exception the Label plugin raises, which reaches the
// web UI wrapped.
func pluginError(message string) *rpcError { return failure("WrappedException", message) }

func (d *daemon) addLabel(p params) (any, *rpcError) {
	var l string
	if err := p.arg(0, &l); err != nil {
		return nil, pluginError("TypeError: " + err.Error())
	}
	l = strings.ToLower(l)
	switch _, exists := d.labels[l]; {
	case !validLabel.MatchString(l):
		return nil, pluginError("Invalid label, valid characters:[a-z0-9_-]")
	case l == "":
		return nil, pluginError("Empty Label")
	case exists:
		return nil, pluginError("Label already exists")
	}
	d.labels[l] = ""
	return nil, nil
}

func (d *daemon) labelOptions(p params) (any, *rpcError) {
	var l string
	if err := p.arg(0, &l); err != nil {
		return nil, pluginError("TypeError: " + err.Error())
	}
	move, ok := d.labels[l]
	if !ok {
		return nil, pluginError(fmt.Sprintf("KeyError: '%s'", l))
	}
	return map[string]any{
		"apply_max": false, "max_download_speed": -1, "max_upload_speed": -1, "max_connections": -1, "max_upload_slots": -1,
		"prioritize_first_last": false, "apply_queue": false, "is_auto_managed": false, "stop_at_ratio": false,
		"stop_ratio": 2.0, "remove_at_ratio": false, "apply_move_completed": move != "", "move_completed": move != "",
		"move_completed_path": move, "auto_add": false, "auto_add_trackers": []string{},
	}, nil
}

// setLabel is label.set_torrent, which applies a label's move and puts
// core.conf's back for the label it replaces.
func (d *daemon) setLabel(p params) (any, *rpcError) {
	var id, l string
	if err := errors.Join(p.arg(0, &id), p.arg(1, &l)); err != nil {
		return nil, pluginError("TypeError: " + err.Error())
	}
	if l == "No Label" {
		l = ""
	}
	if _, ok := d.labels[l]; l != "" && !ok {
		return nil, pluginError("Unknown Label")
	}
	h, err := bencode.ParseInfoHash(id)
	t, ok := d.engine.Get(h)
	if err != nil || !ok {
		return nil, pluginError("Unknown Torrent")
	}
	if len(t.Labels) > 0 && d.labels[t.Labels[0]] != "" && t.Done < t.Size {
		delete(d.moveTo, h)
		if d.MoveTo != "" && d.MoveTo != t.Folder {
			d.moveTo[h] = d.MoveTo
		}
	}
	var labels []string
	if l != "" {
		labels = []string{l}
		if move := d.labels[l]; move != "" && t.Done < t.Size {
			d.moveTo[h] = move
		}
	}
	_ = d.engine.SetLabels(h, labels)
	return nil, nil
}
