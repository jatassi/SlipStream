package transmission

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/fakes/internal/torrentclient"
	"github.com/jatassi/slipstream/internal/bencode"
)

// legacy is each snake_case name's legacy spelling. The legacy protocol
// spells a torrent's fields in camelCase and most other names in
// kebab-case, so download_dir has two spellings, and torrentLegacy holds
// the torrent fields' one.
var legacy = map[string]string{
	"session_get": "session-get", "torrent_add": "torrent-add", "torrent_get": "torrent-get", "torrent_set": "torrent-set",
	"torrent_start": "torrent-start", "torrent_stop": "torrent-stop", "torrent_verify": "torrent-verify",
	"torrent_reannounce": "torrent-reannounce", "torrent_remove": "torrent-remove",

	"rpc_version": "rpc-version", "rpc_version_semver": "rpc-version-semver", "download_dir": "download-dir",
	"seed_ratio_limit": "seedRatioLimit", "seed_ratio_limited": "seedRatioLimited",
	"download_queue_enabled": "download-queue-enabled", "download_queue_size": "download-queue-size",
	"delete_local_data": "delete-local-data", "torrent_added": "torrent-added", "torrent_duplicate": "torrent-duplicate",

	"hash_string": "hashString", "percent_done": "percentDone", "left_until_done": "leftUntilDone",
	"size_when_done": "sizeWhenDone", "total_size": "totalSize", "error_string": "errorString", "is_finished": "isFinished",
	"done_date": "doneDate", "upload_ratio": "uploadRatio", "uploaded_ever": "uploadedEver", "downloaded_ever": "downloadedEver",
	"seconds_seeding": "secondsSeeding", "seconds_downloading": "secondsDownloading", "rate_download": "rateDownload",
	"rate_upload": "rateUpload", "peers_connected": "peersConnected", "peers_sending_to_us": "peersSendingToUs",
	"peers_getting_from_us": "peersGettingFromUs", "queue_position": "queuePosition", "bytes_completed": "bytesCompleted",
}

var torrentLegacy = func() map[string]string {
	m := maps.Clone(legacy)
	m["download_dir"] = "downloadDir"
	return m
}()

func reverse(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[v] = k
	}
	return out
}

var (
	fromLegacy        = reverse(legacy)
	torrentFromLegacy = reverse(torrentLegacy)
)

type request struct {
	JSONRPC   string          `json:"jsonrpc"`
	Method    string          `json:"method"`
	Arguments json.RawMessage `json:"arguments"`
	Params    json.RawMessage `json:"params"`
	Tag       json.RawMessage `json:"tag"`
	ID        json.RawMessage `json:"id"`
}

// rpcError is a failed call: a JSON-RPC code and message, and the legacy
// result string, which is also JSON-RPC's error_string unless bare.
type rpcError struct {
	code    int
	message string
	text    string
	bare    bool
}

func (e *rpcError) Error() string { return e.text }

var (
	errMethod  = &rpcError{code: -32601, message: "Method not found", text: "method name not recognized", bare: true}
	errCorrupt = &rpcError{code: 9, message: "invalid or corrupt torrent file", text: "invalid or corrupt torrent file"}
	errEmpty   = &rpcError{code: -32602, message: "Invalid params", text: "labels cannot be empty"}
	errComma   = &rpcError{code: -32602, message: "Invalid params", text: "labels cannot contain comma (,) character"}
	errParams  = &rpcError{code: -32602, message: "Invalid params", text: "the arguments aren't an object of the method's"}
)

func (d *daemon) rpc(w http.ResponseWriter, r *http.Request) {
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"jsonrpc": "2.0", "error": map[string]any{"code": -32700, "message": "Parse error"}, "id": nil})
		return
	}
	json2 := d.version.json2 && req.JSONRPC == "2.0"
	method, args := req.Method, req.Arguments
	if json2 {
		args = req.Params
	} else {
		method = fromLegacy[method]
		args = rekey(args, fromLegacy)
	}
	if len(bytes.TrimSpace(args)) == 0 {
		args = json.RawMessage("{}")
	}
	result, err := d.call(r.Context(), method, args, !json2)
	switch {
	case json2 && len(req.ID) == 0:
		w.WriteHeader(http.StatusNoContent)
	case json2 && err != nil:
		e := map[string]any{"code": err.code, "message": err.message}
		if !err.bare {
			e["data"] = map[string]any{"error_string": err.text}
		}
		writeJSON(w, map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": e})
	case json2:
		writeJSON(w, map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	case err != nil:
		writeJSON(w, tagged(req.Tag, map[string]any{"result": err.text, "arguments": map[string]any{}}))
	default:
		names := legacy
		if method == "torrent_get" {
			names = torrentLegacy
		}
		writeJSON(w, tagged(req.Tag, map[string]any{"result": "success", "arguments": rekeyValue(result, names)}))
	}
}

// tagged is a legacy response with the request's tag, which it carries
// only when the request had one.
func tagged(tag json.RawMessage, res map[string]any) map[string]any {
	if len(tag) > 0 {
		res["tag"] = tag
	}
	return res
}

func writeJSON(w http.ResponseWriter, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	_, _ = w.Write(body)
}

type ids struct {
	IDs []json.RawMessage `json:"ids"`
}

func (i *ids) named() []json.RawMessage { return i.IDs }

// call runs method with args, whose names are snake_case.
func (d *daemon) call(ctx context.Context, method string, args json.RawMessage, legacy bool) (any, *rpcError) {
	switch method {
	case "session_get":
		return d.sessionGet(), nil
	case "torrent_add":
		return d.add(ctx, args)
	case "torrent_get":
		return d.torrentGet(args, legacy)
	case "torrent_set":
		return d.torrentSet(args)
	case "torrent_remove":
		var a struct {
			ids
			Delete bool `json:"delete_local_data"`
		}
		return d.each(args, &a, func(h bencode.InfoHash) error {
			if err := d.engine.Remove(h, a.Delete); err != nil {
				return err
			}
			d.removed.Record(ctx, Removal{InfoHash: h.String(), Data: a.Delete})
			return nil
		})
	}
	act, ok := map[string]func(bencode.InfoHash) error{
		"torrent_start": d.engine.Start, "torrent_stop": d.engine.Stop, "torrent_verify": d.engine.Verify,
		"torrent_reannounce": d.engine.Announce,
	}[method]
	if !ok {
		return nil, errMethod
	}
	var a ids
	return d.each(args, &a, act)
}

func (d *daemon) torrentGet(args json.RawMessage, legacy bool) (any, *rpcError) {
	var a struct {
		ids
		Fields []string `json:"fields"`
	}
	if json.Unmarshal(args, &a) != nil {
		return nil, errParams
	}
	if legacy {
		for i, f := range a.Fields {
			if snake, ok := torrentFromLegacy[f]; ok {
				a.Fields[i] = snake
			}
		}
	}
	return d.get(a.IDs, a.Fields), nil
}

func (d *daemon) torrentSet(args json.RawMessage) (any, *rpcError) {
	var a struct {
		ids
		Labels *[]string `json:"labels"`
	}
	if json.Unmarshal(args, &a) != nil {
		return nil, errParams
	}
	if a.Labels == nil {
		return map[string]any{}, nil
	}
	if err := checkLabels(*a.Labels); err != nil {
		return nil, err
	}
	for _, t := range d.selected(a.IDs) {
		_ = d.engine.SetLabels(t.Hash, *a.Labels)
	}
	return map[string]any{}, nil
}

func (d *daemon) sessionGet() map[string]any {
	return map[string]any{
		"version": d.version.name, "rpc_version": d.version.rpc, "rpc_version_semver": d.version.semver,
		"download_dir":     d.engine.DefaultFolder(),
		"seed_ratio_limit": d.SeedRatio, "seed_ratio_limited": d.SeedRatio > 0,
		"download_queue_enabled": d.Slots > 0, "download_queue_size": d.Slots,
	}
}

func (d *daemon) add(ctx context.Context, args json.RawMessage) (any, *rpcError) {
	var a struct {
		Metainfo string    `json:"metainfo"`
		Dir      string    `json:"download_dir"`
		Labels   *[]string `json:"labels"`
		Paused   bool      `json:"paused"`
	}
	if json.Unmarshal(args, &a) != nil {
		return nil, errParams
	}
	var labels []string
	if a.Labels != nil {
		if err := checkLabels(*a.Labels); err != nil {
			return nil, err
		}
		labels = *a.Labels
	}
	meta, err := base64.StdEncoding.DecodeString(a.Metainfo)
	if err != nil {
		return nil, errCorrupt
	}
	t, duplicate, err := d.engine.Add(meta, a.Dir, labels, a.Paused)
	if err != nil {
		return nil, errCorrupt
	}
	key := "torrent_added"
	if duplicate {
		key = "torrent_duplicate"
	} else {
		d.added.Record(ctx, Add{InfoHash: t.Hash.String(), Name: t.Name, Folder: t.Folder, Labels: labels, Paused: a.Paused})
	}
	return map[string]any{key: map[string]any{"id": t.ID, "name": t.Name, "hash_string": t.Hash.String()}}, nil
}

func checkLabels(labels []string) *rpcError {
	for _, l := range labels {
		switch {
		case l == "":
			return errEmpty
		case strings.Contains(l, ","):
			return errComma
		}
	}
	return nil
}

// each decodes args into a and runs act on each torrent its ids name
// that the daemon lists. An id it doesn't list is skipped, as Transmission
// skips it.
func (d *daemon) each(args json.RawMessage, a interface{ named() []json.RawMessage }, act func(bencode.InfoHash) error) (any, *rpcError) {
	if json.Unmarshal(args, a) != nil {
		return nil, errParams
	}
	for _, t := range d.selected(a.named()) {
		_ = act(t.Hash)
	}
	return map[string]any{}, nil
}

// selected is each torrent ids names, by hash or by id, or every torrent
// when ids is absent.
func (d *daemon) selected(ids []json.RawMessage) []torrentclient.Torrent {
	all := d.engine.List(nil)
	if ids == nil {
		return all
	}
	var out []torrentclient.Torrent
	for _, t := range all {
		if slices.ContainsFunc(ids, func(raw json.RawMessage) bool { return names(raw, t) }) {
			out = append(out, t)
		}
	}
	return out
}

func names(raw json.RawMessage, t torrentclient.Torrent) bool {
	var id int
	if json.Unmarshal(raw, &id) == nil {
		return id == t.ID
	}
	var hash string
	if json.Unmarshal(raw, &hash) != nil {
		return false
	}
	h, err := bencode.ParseInfoHash(hash)
	return err == nil && h == t.Hash
}

func (d *daemon) get(ids []json.RawMessage, fields []string) map[string]any {
	torrents := []map[string]any{}
	for _, t := range d.selected(ids) {
		all := d.fields(t)
		row := map[string]any{}
		for _, f := range fields {
			if v, ok := all[f]; ok {
				row[f] = v
			}
		}
		torrents = append(torrents, row)
	}
	return map[string]any{"torrents": torrents}
}

// Transmission's tr_torrent_activity.
const (
	stopped     = 0
	verifying   = 2
	downloadQ   = 3
	downloading = 4
	seeding     = 6
)

func (d *daemon) fields(t torrentclient.Torrent) map[string]any {
	status := map[torrentclient.State]int{
		torrentclient.Stopped: stopped, torrentclient.Checking: verifying, torrentclient.Queued: downloadQ, torrentclient.Downloading: downloading, torrentclient.Seeding: seeding,
	}[t.State]
	code, text := 0, ""
	switch {
	case t.LocalError != "":
		code, text = 3, t.LocalError
	case t.Tracker.Message != "" && t.Tracker.Error:
		code, text = 2, t.Tracker.Message
	case t.Tracker.Message != "":
		code, text = 1, t.Tracker.Message
	}
	files := make([]map[string]any, len(t.Files))
	for i, f := range t.Files {
		files[i] = map[string]any{"name": f.Path, "length": f.Size, "bytes_completed": f.Done}
	}
	ratio, percent := -1.0, 0.0
	if t.Size > 0 {
		ratio, percent = float64(t.Uploaded)/float64(t.Size), float64(t.Done)/float64(t.Size)
	}
	labels := t.Labels
	if labels == nil {
		labels = []string{}
	}
	return map[string]any{
		"id": t.ID, "hash_string": t.Hash.String(), "name": t.Name, "status": status,
		"percent_done": percent, "left_until_done": t.Size - t.Done, "size_when_done": t.Size, "total_size": t.Size,
		"download_dir": t.Folder, "labels": labels, "error": code, "error_string": text,
		"is_finished": t.SeedGoalMet, "done_date": unix(t.CompletedAt), "upload_ratio": ratio,
		"uploaded_ever": t.Uploaded, "downloaded_ever": t.Downloaded,
		"seconds_seeding": int64(t.Seeding.Seconds()), "seconds_downloading": int64(t.Downloading.Seconds()),
		"rate_download": t.Rates.Down, "rate_upload": t.Rates.Up, "peers_connected": t.Rates.Peers,
		"peers_sending_to_us": t.Rates.Seeds, "peers_getting_from_us": t.Rates.Peers - t.Rates.Seeds,
		"queue_position": t.Order, "files": files,
	}
}

// rekey renames every object key in raw found in names.
func rekey(raw json.RawMessage, names map[string]string) json.RawMessage {
	if len(bytes.TrimSpace(raw)) == 0 {
		return raw
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if dec.Decode(&v) != nil {
		return raw
	}
	out, err := json.Marshal(rekeyValue(v, names))
	if err != nil {
		return raw
	}
	return out
}

func rekeyValue(v any, names map[string]string) any {
	switch v := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, x := range v {
			if n, ok := names[k]; ok {
				k = n
			}
			out[k] = rekeyValue(x, names)
		}
		return out
	case []map[string]any:
		out := make([]any, len(v))
		for i, x := range v {
			out[i] = rekeyValue(x, names)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, x := range v {
			out[i] = rekeyValue(x, names)
		}
		return out
	}
	return v
}
