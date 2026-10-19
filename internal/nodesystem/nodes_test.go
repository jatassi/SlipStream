package nodesystem_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/pb/slipstream/system/v1/systemv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/system"
)

// host is a Node's system as a test scripts it.
type host struct {
	mu         sync.Mutex
	facts      system.Facts
	factsErr   error
	config     nodesystem.Config
	refusal    *nodesystem.Refusal
	set        [][]nodesystem.Change
	restarted  bool
	reads      []uint64
	batches    []readAnswer
	archive    []byte
	archiveDir string
	reached    nodesystem.Reached
	reaches    []nodesystem.Target
}

type readAnswer struct {
	batch system.LogBatch
	err   error
}

func (h *host) Facts() (system.Facts, error) { return h.facts, h.factsErr }

func (h *host) Config() (nodesystem.Config, error) { return h.config, nil }

func (h *host) SetConfig(_ context.Context, changes []nodesystem.Change) (nodesystem.Config, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.set = append(h.set, changes)
	if h.refusal != nil {
		return nodesystem.Config{}, h.refusal
	}
	return h.config, nil
}

func (h *host) Restart() (nodesystem.Restarting, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.restarted {
		return nodesystem.Restarting{}, nodesystem.ErrAlreadyRestarting
	}
	h.restarted = true
	return nodesystem.Restarting{StartedAt: time.Unix(1_800_000_000, 0), BoundPort: 8486, NextPort: 9000}, nil
}

// ReadLogs answers the next scripted batch, then waits out ctx.
func (h *host) ReadLogs(ctx context.Context, after uint64, _ time.Duration) (system.LogBatch, error) {
	h.mu.Lock()
	h.reads = append(h.reads, after)
	if len(h.batches) == 0 {
		h.mu.Unlock()
		<-ctx.Done()
		return system.LogBatch{}, fmt.Errorf("read the log: %w", ctx.Err())
	}
	next := h.batches[0]
	h.batches = h.batches[1:]
	h.mu.Unlock()
	return next.batch, next.err
}

func (h *host) Archive() (nodesystem.Archive, error) {
	f, err := os.CreateTemp(h.archiveDir, "logs-*.zip")
	if err != nil {
		return nodesystem.Archive{}, fmt.Errorf("zip the logs: %w", err)
	}
	if _, err := f.WriteAt(h.archive, 0); err != nil {
		return nodesystem.Archive{}, fmt.Errorf("zip the logs: %w", err)
	}
	return nodesystem.Archive{File: f, Size: int64(len(h.archive)), Stamp: "20261016-120000"}, nil
}

func (h *host) Reach(_ context.Context, t nodesystem.Target) nodesystem.Reached {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reaches = append(h.reaches, t)
	return h.reached
}

// pair is a Controller reaching a Worker that serves h over linktest.
type pair struct {
	*linktest.Pair
	nodes  *nodesystem.Nodes
	worker role.NodeID
}

var errNoSuchNode = fmt.Errorf("roster: %w", nodesystem.ErrNotFound)

func newPair(t *testing.T, h *host, workerVersion string) *pair {
	t.Helper()
	cm, wm := http.NewServeMux(), http.NewServeMux()
	p := &pair{
		Pair:   linktest.New(t, linktest.Side{Lockstep: cm}, linktest.Side{Version: workerVersion, Lockstep: wm}),
		worker: role.NodeID{UUID: storage.NewUUID()},
	}
	controller := rpc.NewDispatch(p.Controller.Link, p.Controller.Key, p.Controller.Key, func(role.Name) bool { return true })
	worker := rpc.NewDispatch(p.Worker.Link, p.Worker.Key, p.Controller.Key, func(role.Name) bool { return true })
	cm.Handle("/", controller)
	wm.Handle("/", worker)
	systemv1connect.NodeSystemService.Mount(worker, nodesystem.Serve(h, nodesystem.ServeTimings{LogPoll: time.Second, LogArchiveTTL: 10 * time.Minute}))
	cluster := rpc.NewCluster(controller, roster{p.worker: p.Worker.Key})
	p.nodes = nodesystem.NewNodes(
		func(id role.NodeID) systemv1connect.NodeSystemServiceHandler {
			return systemv1connect.NodeSystemService.On(cluster, id)
		},
		func(_ context.Context, id role.NodeID) (string, error) {
			if id != p.worker {
				return "", errNoSuchNode
			}
			return "box", nil
		},
		nodesystem.FollowTimings{RetryMin: time.Second, RetryMax: 30 * time.Second},
		rand.New(rand.NewPCG(1, 2)), //nolint:gosec // a fixed draw for fixed delays
	)
	return p
}

type roster map[role.NodeID]link.Key

func (r roster) Key(node role.NodeID) (link.Key, bool) {
	k, ok := r[node]
	return k, ok
}

func (roster) Serves(role.Name, role.NodeID) bool { return false }

var boxFacts = system.Facts{
	DataDir:        "/srv/box",
	LogsDir:        "/srv/box/logs",
	PID:            77,
	Seed:           "s",
	StartedAt:      time.Unix(1_800_000_000, 5).UTC(),
	HTTPConfigured: netip.MustParseAddrPort("[::]:8484"),
	Listeners: system.Listeners{
		HTTP:    netip.MustParseAddrPort("[::]:8486"),
		Torrent: netip.MustParseAddrPort("0.0.0.0:6881"),
		Link:    netip.MustParseAddrPort("[::]:8485"),
		Control: "/srv/box/run/control.sock",
	},
}

func TestTheControllerReadsAWorkersFacts(t *testing.T) {
	facts := boxFacts
	h := &host{facts: facts}
	synctest.Test(t, func(t *testing.T) {
		p := newPair(t, h, "")
		p.Connect(t)
		ctx := t.Context()

		got, err := p.nodes.Facts(ctx, p.worker)
		if err != nil || !got.StartedAt.Equal(facts.StartedAt) {
			t.Fatalf("Facts = %+v, %v; want %+v", got, err, facts)
		}
		got.StartedAt = facts.StartedAt
		if !reflect.DeepEqual(got, facts) {
			t.Fatalf("Facts = %+v; want %+v", got, facts)
		}

		h.factsErr = fmt.Errorf("%w: starting", nodesystem.ErrOffline)
		if _, err := p.nodes.Facts(ctx, p.worker); !errors.Is(err, nodesystem.ErrOffline) {
			t.Fatalf("Facts of a starting Node = %v, want %v", err, nodesystem.ErrOffline)
		}
		stranger := role.NodeID{UUID: storage.NewUUID()}
		if _, err := p.nodes.Facts(ctx, stranger); !errors.Is(err, nodesystem.ErrNotFound) {
			t.Fatalf("Facts of a Node outside the roster = %v, want %v", err, nodesystem.ErrNotFound)
		}
	})
}

func TestTheControllerChangesAWorkersConfig(t *testing.T) {
	field := json.RawMessage(`{"key":"log.level","kind":"choice","value":"debug"}`)
	h := &host{config: nodesystem.Config{Sections: []nodesystem.Section{{ID: nodesystem.Log, Fields: []json.RawMessage{field}}}}}
	changes := []nodesystem.Change{
		{Key: "log.level", Value: nodesystem.Word("debug")},
		{Key: "http.listen", Value: nodesystem.Number(9000)},
		{Key: "log.compress", Value: nodesystem.Switch(false)},
		{Key: "log.max_files", Value: nodesystem.Reset()},
	}
	synctest.Test(t, func(t *testing.T) {
		p := newPair(t, h, "")
		p.Connect(t)
		ctx := t.Context()

		config, err := p.nodes.SetConfig(ctx, p.worker, changes)
		if err != nil || config.Revision() != h.config.Revision() || string(config.Sections[0].Fields[0]) != string(field) {
			t.Fatalf("SetConfig = %+v, %v", config, err)
		}
		if fmt.Sprint(h.set[0]) != fmt.Sprint(changes) {
			t.Fatalf("the Worker was asked for %v, want %v", h.set[0], changes)
		}

		h.refusal = &nodesystem.Refusal{Key: "http.listen", Reason: nodesystem.SetByEnvironment, Sentence: "Set by SLIPSTREAM_HTTP_LISTEN.", Variable: "SLIPSTREAM_HTTP_LISTEN"}
		_, err = p.nodes.SetConfig(ctx, p.worker, changes[1:2])
		if refused, ok := errors.AsType[*nodesystem.Refusal](err); !ok || *refused != *h.refusal {
			t.Fatalf("a refused SetConfig = %#v, want %#v", err, h.refusal)
		}
	})
}

func TestTheControllerRestartsAWorkerOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPair(t, &host{facts: boxFacts}, "")
		p.Connect(t)
		ctx := t.Context()

		restarting, err := p.nodes.Restart(ctx, p.worker)
		if want := (nodesystem.Restarting{StartedAt: time.Unix(1_800_000_000, 0), BoundPort: 8486, NextPort: 9000}); err != nil || restarting != want {
			t.Fatalf("Restart = %+v, %v; want %+v", restarting, err, want)
		}
		if _, err := p.nodes.Restart(ctx, p.worker); !errors.Is(err, nodesystem.ErrAlreadyRestarting) {
			t.Fatalf("a second Restart = %v, want %v", err, nodesystem.ErrAlreadyRestarting)
		}
	})
}

func TestAWorkerOfferedNoSessionOrAnotherVersionIsUnreachable(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPair(t, &host{}, "")
		if _, err := p.nodes.Facts(t.Context(), p.worker); !errors.Is(err, nodesystem.ErrOffline) {
			t.Fatalf("Facts with no session = %v, want %v", err, nodesystem.ErrOffline)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		p := newPair(t, &host{}, "0.0.0-other")
		p.Connect(t)
		if _, err := p.nodes.Facts(t.Context(), p.worker); !errors.Is(err, nodesystem.ErrMismatched) {
			t.Fatalf("Facts across versions = %v, want %v", err, nodesystem.ErrMismatched)
		}
	})
}

func record(seq uint64, msg string) system.LogRecord {
	return system.LogRecord{Seq: seq, Time: time.Unix(1_800_000_000, 0), Level: "info", Subsystem: "grabs", Message: msg, Attrs: json.RawMessage(`{"n":1}`)}
}

func TestFollowRidesOutAWorkerGoingOffline(t *testing.T) {
	h := &host{batches: []readAnswer{
		{batch: system.LogBatch{Records: []system.LogRecord{record(5, "a"), record(6, "b")}}},
		{err: fmt.Errorf("%w: %w", nodesystem.ErrOffline, errors.New("the log serves no more reads"))},
		{batch: system.LogBatch{}},
		{batch: system.LogBatch{Reset: &system.LogReset{Reason: system.Restarted, StartedAt: time.Unix(1_800_000_100, 0), After: 100}, Records: []system.LogRecord{record(101, "c")}}},
	}}
	synctest.Test(t, func(t *testing.T) {
		p := newPair(t, h, "")
		p.Connect(t)
		ctx, cancel := context.WithCancel(t.Context())
		var sent []string
		err := p.nodes.Follow(ctx, p.worker, 0, func(b system.LogBatch) error {
			var line []byte
			if r := b.Reset; r != nil {
				line = fmt.Appendf(line, "reset %s at %d after %d;", r.Reason, r.StartedAt.Unix(), r.After)
			}
			for _, r := range b.Records {
				line = fmt.Appendf(line, " %d %s %s %s %s", r.Seq, r.Level, r.Subsystem, r.Message, r.Attrs)
			}
			sent = append(sent, string(line))
			if len(sent) == 3 {
				cancel()
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		want := []string{
			` 5 info grabs a {"n":1} 6 info grabs b {"n":1}`,
			``,
			`reset restarted at 1800000100 after 100; 101 info grabs c {"n":1}`,
		}
		if fmt.Sprint(sent) != fmt.Sprint(want) || fmt.Sprint(h.reads) != "[0 6 6 6]" {
			t.Fatalf("sent %q after reads %v; want %q after [0 6 6 6]", sent, h.reads, want)
		}
	})
}

func TestAnArchiveCrossesTheLinkInChunks(t *testing.T) {
	content := bytes.Repeat([]byte("0123456789abcdef"), 80_000)
	h := &host{archive: content, archiveDir: t.TempDir()}
	synctest.Test(t, func(t *testing.T) {
		p := newPair(t, h, "")
		p.Connect(t)
		a, err := p.nodes.Archive(t.Context(), p.worker)
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(a.Body)
		if err != nil || a.Name != "slipstream-logs-box-20261016-120000.zip" || a.Size != int64(len(content)) || !bytes.Equal(got, content) {
			t.Fatalf("archive %s of %d bytes read %d bytes, %v", a.Name, a.Size, len(got), err)
		}
		if left := archives(t, h.archiveDir); left != 0 {
			t.Fatalf("%d archives left after the last chunk, want 0", left)
		}

		if _, err := p.nodes.Archive(t.Context(), p.worker); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10*time.Minute + time.Second)
		if left := archives(t, h.archiveDir); left != 0 {
			t.Fatalf("%d archives left after the TTL, want 0", left)
		}
		if _, err := p.nodes.Archive(t.Context(), role.NodeID{UUID: storage.NewUUID()}); !errors.Is(err, nodesystem.ErrNotFound) {
			t.Fatalf("the archive of a Node outside the roster = %v, want %v", err, nodesystem.ErrNotFound)
		}
	})
}

func TestASlowArchiveReadOutlastsTheTTLWhileEachChunkComesInTime(t *testing.T) {
	content := bytes.Repeat([]byte("0123456789abcdef"), 80_000)
	h := &host{archive: content, archiveDir: t.TempDir()}
	synctest.Test(t, func(t *testing.T) {
		p := newPair(t, h, "")
		p.Connect(t)
		a, err := p.nodes.Archive(t.Context(), p.worker)
		if err != nil {
			t.Fatal(err)
		}
		var got []byte
		chunk := make([]byte, 512<<10)
		for {
			n, err := io.ReadFull(a.Body, chunk)
			got = append(got, chunk[:n]...)
			if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatalf("read after %d bytes: %v", len(got), err)
			}
			time.Sleep(6 * time.Minute)
		}
		if !bytes.Equal(got, content) {
			t.Fatalf("read %d bytes over 12 minutes, want all %d", len(got), len(content))
		}

		if _, err := p.nodes.Archive(t.Context(), p.worker); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10*time.Minute + time.Second)
		if left := archives(t, h.archiveDir); left != 0 {
			t.Fatalf("%d archives left after the TTL idle, want 0", left)
		}
	})
}

func archives(t *testing.T, dir string) int {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "logs-*.zip"))
	if err != nil {
		t.Fatal(err)
	}
	return len(matches)
}

func TestTheControllerRunsAReachTestFromAWorker(t *testing.T) {
	h := &host{reached: nodesystem.Reached{Took: 2 * time.Second, Err: "connect: connection refused"}}
	synctest.Test(t, func(t *testing.T) {
		p := newPair(t, h, "")
		p.Connect(t)
		target := nodesystem.Target{Host: "smtp.example.com", Port: 465, TLS: true}

		got, err := p.nodes.Reach(t.Context(), p.worker, target)
		if err != nil || got != h.reached {
			t.Fatalf("Reach = %+v, %v; want %+v", got, err, h.reached)
		}
		if !slices.Equal(h.reaches, []nodesystem.Target{target}) {
			t.Fatalf("the Worker dialed %v, want %v", h.reaches, target)
		}
		if _, err := p.nodes.Reach(t.Context(), role.NodeID{UUID: storage.NewUUID()}, target); !errors.Is(err, nodesystem.ErrNotFound) {
			t.Fatalf("Reach on a Node the roster lacks = %v, want ErrNotFound", err)
		}
	})
}
