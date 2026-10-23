package systemapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/pb/slipstream/system/v1/systemv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/system"
)

// host is a Node's system with scripted answers.
type host struct {
	mu        sync.Mutex
	factsErr  error
	config    nodesystem.Config
	set       [][]nodesystem.Change
	refusal   *nodesystem.Refusal
	restarted bool
	batches   []system.LogBatch
	reads     []uint64
	// drained ends the stream once every batch was read.
	drained func()
}

func (h *host) Facts() (system.Facts, error) { return facts, h.factsErr }

func (h *host) Config() (nodesystem.Config, error) { return h.config, nil }

func (h *host) SetConfig(changes []nodesystem.Change) (nodesystem.Config, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.refusal != nil {
		return nodesystem.Config{}, h.refusal
	}
	h.set = append(h.set, changes)
	return h.config, nil
}

func (h *host) Restart() (nodesystem.Restarting, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.restarted {
		return nodesystem.Restarting{}, nodesystem.ErrAlreadyRestarting
	}
	h.restarted = true
	return nodesystem.Restarting{StartedAt: facts.StartedAt, BoundPort: 8486, NextPort: 9000}, nil
}

func (h *host) ReadLogs(ctx context.Context, after uint64, _ time.Duration) (system.LogBatch, error) {
	h.mu.Lock()
	h.reads = append(h.reads, after)
	if len(h.batches) == 0 {
		h.mu.Unlock()
		h.drained()
		<-ctx.Done()
		return system.LogBatch{}, fmt.Errorf("read the log: %w", ctx.Err())
	}
	b := h.batches[0]
	h.batches = h.batches[1:]
	h.mu.Unlock()
	return b, nil
}

func (h *host) Archive() (nodesystem.Archive, error) { panic("unused") }

func (h *host) Reach(context.Context, nodesystem.Target) nodesystem.Reached { panic("unused") }

func (h *host) Probe() ([]role.ProbeResult, error) { panic("unused") }

// self is a roster in which every id is this Node, called in process.
type self struct{ key link.Key }

func (s self) Key(role.NodeID) (link.Key, bool) { return s.key, true }
func (self) Serves(role.Name, role.NodeID) bool { return false }

// controllerOf is a claimed Controller whose Nodes reach h for any id,
// through the in-process call path.
func controllerOf(t *testing.T, h *host) (n *node, apiKey string) {
	t.Helper()
	roster := self{key: link.Key{1}}
	dispatch := rpc.NewDispatch(nil, roster.key, roster.key, func(role.Name) bool { return true })
	systemv1connect.NodeSystemService.Mount(dispatch, nodesystem.Serve(h, nodesystem.ServeTimings{LogPoll: time.Second, LogArchiveTTL: time.Minute}))
	cluster := rpc.NewCluster(dispatch, roster)
	systems := nodesystem.NewNodes(
		func(id role.NodeID) systemv1connect.NodeSystemServiceHandler {
			return systemv1connect.NodeSystemService.On(cluster, id)
		},
		func(context.Context, role.NodeID) (string, error) { return "box", nil },
		nodesystem.FollowTimings{RetryMin: time.Second, RetryMax: time.Minute},
		rand.New(rand.NewPCG(1, 2)), //nolint:gosec // fixed delays
	)
	n = newController(t, facts, systems)
	apiKey = n.claim()
	return n, apiKey
}

const boxPath = "/api/v1/nodes/0b0b0b0b-0000-4000-8000-000000000001"

func TestANodesConfigIsChangedEveryKeyOrNone(t *testing.T) {
	field := json.RawMessage(`{"key":"log.level","kind":"choice","value":"debug"}`)
	h := &host{config: nodesystem.Config{Sections: []nodesystem.Section{{ID: nodesystem.Log, Fields: []json.RawMessage{field}}}}}
	n, key := controllerOf(t, h)
	admin := map[string]string{"X-Api-Key": key}
	want := `{"sections":[{"id":"log","fields":[{"key":"log.level","kind":"choice","value":"debug"}]}]}` + "\n"

	expect(t, n.do(http.MethodGet, boxPath+"/config", "", admin), 200, want)
	expect(t, n.do(http.MethodPatch, boxPath+"/config", `{"log.level":"debug","http.listen":9000,"log.compress":false,"log.max_files":null}`, admin), 200, want)
	if got := fmt.Sprint(h.set); got != fmt.Sprint([][]nodesystem.Change{{
		{Key: "http.listen", Value: nodesystem.Number(9000)},
		{Key: "log.compress", Value: nodesystem.Switch(false)},
		{Key: "log.level", Value: nodesystem.Word("debug")},
		{Key: "log.max_files", Value: nodesystem.Reset()},
	}}) {
		t.Fatalf("the Node was asked for %v", got)
	}

	for _, tc := range []struct {
		refusal nodesystem.Refusal
		status  int
		want    string
	}{
		{nodesystem.Refusal{Key: "http.listen", Reason: nodesystem.SetByEnvironment, Sentence: "Set by SLIPSTREAM_HTTP_LISTEN.", Variable: "SLIPSTREAM_HTTP_LISTEN"}, 409,
			`{"status":409,"code":"set_by_environment","title":"An environment variable sets this setting","detail":"Set by SLIPSTREAM_HTTP_LISTEN.","errors":[{"location":"body.http.listen","message":"Set by SLIPSTREAM_HTTP_LISTEN."}]}`},
		{nodesystem.Refusal{Key: "log.max_size_mb", Reason: nodesystem.OutOfRange, Sentence: "Between 1 and 1000 MB."}, 422,
			`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"Between 1 and 1000 MB.","errors":[{"location":"body.log.max_size_mb","message":"Between 1 and 1000 MB."}]}`},
		{nodesystem.Refusal{Key: "http.listen", Reason: nodesystem.PortUsedByNode, Sentence: "The link uses port 8485."}, 422,
			`{"status":422,"code":"port_used_by_node","title":"The Node uses this port already","detail":"The link uses port 8485.","errors":[{"location":"body.http.listen","message":"The link uses port 8485."}]}`},
		{nodesystem.Refusal{Key: "log.colour", Reason: nodesystem.UnknownSetting, Sentence: "No setting has this key."}, 422,
			`{"status":422,"code":"unknown_setting","title":"No setting has this key","detail":"No setting has this key.","errors":[{"location":"body.log.colour","message":"No setting has this key."}]}`},
		{nodesystem.Refusal{Reason: nodesystem.NotWritable, Sentence: "Edit it by hand."}, 409,
			`{"status":409,"code":"config_not_writable","title":"SlipStream can't rewrite config.yaml","detail":"Edit it by hand."}`},
	} {
		h.refusal = &tc.refusal
		expect(t, n.do(http.MethodPatch, boxPath+"/config", `{"log.level":"warn"}`, admin), tc.status, tc.want+"\n")
	}
	expect(t, n.do(http.MethodPatch, boxPath+"/config", `{"log.max_size_mb":1.5}`, admin), 422,
		`{"status":422,"code":"invalid","title":"Some fields are invalid","detail":"A whole number.","errors":[{"location":"body.log.max_size_mb","message":"A whole number."}]}`+"\n")
}

func TestANodeRestartsOnceAndAnOfflineOneAnswers503(t *testing.T) {
	h := &host{}
	n, key := controllerOf(t, h)
	admin := map[string]string{"X-Api-Key": key}

	expect(t, n.do(http.MethodPost, boxPath+"/restart", "", admin), 202, `{"startedAt":"2026-10-12T09:30:00Z","boundPort":8486,"nextPort":9000}`+"\n")
	expect(t, n.do(http.MethodPost, boxPath+"/restart", "", admin), 409, `{"status":409,"code":"already_restarting","title":"The Node is already restarting"}`+"\n")

	h.factsErr = fmt.Errorf("%w: stopping", nodesystem.ErrOffline)
	expect(t, n.do(http.MethodGet, boxPath+"/system", "", admin), 503, `{"status":503,"code":"node_offline","title":"The Node is offline"}`+"\n")
	expect(t, n.do(http.MethodGet, boxPath+"/logs/stream", "", admin), 503, `{"status":503,"code":"node_offline","title":"The Node is offline"}`+"\n")
}

func TestANodesLogStreamsAsRecordsAndResets(t *testing.T) {
	at := time.Date(2026, 10, 12, 9, 31, 0, 0, time.UTC)
	record := func(seq uint64, msg string) system.LogRecord {
		return system.LogRecord{Seq: seq, Time: at, Level: "info", Subsystem: "grabs", Message: msg, Attrs: json.RawMessage(`{"n":1}`)}
	}
	h := &host{batches: []system.LogBatch{
		{Records: []system.LogRecord{record(4, "a"), record(5, "b")}},
		{},
		{Reset: &system.LogReset{Reason: system.Restarted, StartedAt: at, After: 100}, Records: []system.LogRecord{record(101, "c")}},
	}}
	n, key := controllerOf(t, h)
	ctx, cancel := context.WithCancel(t.Context())
	h.drained = cancel
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, boxPath+"/logs/stream", nil)
	req.Header.Set("X-Api-Key", key)
	req.Header.Set("Last-Event-ID", "3")
	w := httptest.NewRecorder()
	n.h.ServeHTTP(w, req)

	rec := func(seq int, msg string) string {
		return fmt.Sprintf("id: %d\nevent: log.record\ndata: {\"seq\":%d,\"time\":\"2026-10-12T09:31:00Z\",\"level\":\"info\",\"subsystem\":\"grabs\",\"msg\":%q,\"attrs\":{\"n\":1}}\n\n", seq, seq, msg)
	}
	want := rec(4, "a") + rec(5, "b") + ": keepalive\n\n" +
		"id: 100\nevent: log.reset\ndata: {\"reason\":\"restarted\",\"startedAt\":\"2026-10-12T09:31:00Z\"}\n\n" + rec(101, "c")
	if w.Code != 200 || w.Header().Get("Content-Type") != "text/event-stream" || w.Body.String() != want {
		t.Fatalf("stream = %d %s\n%s\nwant\n%s", w.Code, w.Header().Get("Content-Type"), w.Body, want)
	}
	if fmt.Sprint(h.reads) != "[3 5 5 101]" {
		t.Fatalf("the Node's log was read after %v, want [3 5 5 101]", h.reads)
	}
}
