package e2e

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// configField is as much of a setting field as the tier asserts.
type configField struct {
	Key     string `json:"key"`
	Value   any    `json:"value"`
	Running any    `json:"running"`
}

type logRecord struct {
	Level     string `json:"level"`
	Subsystem string `json:"subsystem"`
	Message   string `json:"msg"`
}

// followLogs opens streamNodeLogs of the Node id on home and returns the
// data of each event as it comes.
func followLogs(t *testing.T, run *world.Run, id string) *bufio.Scanner {
	t.Helper()
	m := run.Manifest()
	req, err := http.NewRequestWithContext(worldtest.Context(t), http.MethodGet, api(t, run).Node("home").URL+"/api/v1/nodes/"+id+"/logs/stream", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Api-Key", m.APIKey)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("home streamNodeLogs %s = %d %s, want 200 text/event-stream", id, res.StatusCode, res.Header.Get("Content-Type"))
	}
	s := bufio.NewScanner(res.Body)
	s.Buffer(nil, 1<<20)
	return s
}

// awaitRecord reads s, SSE events or the log file's lines, until a record
// equal to want.
func awaitRecord(t *testing.T, s *bufio.Scanner, want logRecord) {
	t.Helper()
	for s.Scan() {
		var r logRecord
		if json.Unmarshal([]byte(strings.TrimPrefix(s.Text(), "data: ")), &r) == nil && r == want {
			return
		}
	}
	t.Fatalf("the log stream ended before %+v: %v", want, s.Err())
}

// failHandshake opens a connection to addr's link listener and closes it
// before TLS, which the Node logs at debug.
func failHandshake(t *testing.T, addr string) {
	t.Helper()
	c, err := new(net.Dialer).DialContext(worldtest.Context(t), "tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Write([]byte("not tls\n")); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestTheControllerManagesAWorkersSystem(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "home")
	a := api(t, run)
	box := a.Node("seedbox")
	worker := "/api/v1/nodes/" + box.NodeID
	linkUp(t, a, box.NodeID, "")

	var config struct {
		Sections []struct {
			Fields []configField `json:"fields"`
		} `json:"sections"`
	}
	r := a.Admin("home", http.MethodPatch, worker+"/config", `{"log.level":"debug"}`)
	if err := json.Unmarshal([]byte(r.Body), &config); r.Status != http.StatusOK || err != nil {
		t.Fatalf("home updateNodeConfig seedbox = %d %s, want 200", r.Status, r.Body)
	}
	var fields []configField
	for _, s := range config.Sections {
		fields = append(fields, s.Fields...)
	}
	if !slices.Contains(fields, configField{Key: "log.level", Value: "debug", Running: "debug"}) {
		t.Fatalf("seedbox's config after the change = %s, want log.level debug saved and running", r.Body)
	}

	stream := followLogs(t, run, box.NodeID)
	failHandshake(t, box.Listeners["link"])
	awaitRecord(t, stream, logRecord{Level: "debug", Subsystem: "link", Message: "link handshake failed"})

	r = a.Admin("home", http.MethodGet, worker+"/logs/archive", "")
	z, err := zip.NewReader(strings.NewReader(r.Body), int64(len(r.Body)))
	if r.Status != http.StatusOK || err != nil {
		t.Fatalf("home downloadNodeLogs seedbox = %d, %v; want 200 and a zip", r.Status, err)
	}
	var names []string
	for _, f := range z.File {
		names = append(names, f.Name)
	}
	if !slices.Equal(names, []string{"slipstream.log"}) {
		t.Fatalf("seedbox's log archive holds %v, want [slipstream.log]", names)
	}
	f, err := z.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	logged := bufio.NewScanner(f)
	logged.Buffer(nil, 1<<20)
	awaitRecord(t, logged, logRecord{Level: "DEBUG", Subsystem: "link", Message: "link handshake failed"})
}

func TestAStoppedWorkerIsOffline(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "home")
	a := api(t, run)
	box := a.Node("seedbox")
	worker := "/api/v1/nodes/" + box.NodeID
	act(t, run, "seedbox", world.StopNode)
	worldtest.Until(t, "home lists seedbox's link down", func() (bool, error) {
		return row(t, a.Nodes("home"), box.NodeID).Link.State == "down", nil
	})

	offline := problem(http.StatusServiceUnavailable, "node_offline", "The Node is offline")
	for _, op := range []struct{ method, path, body string }{
		{http.MethodGet, worker + "/system", ""},
		{http.MethodGet, worker + "/config", ""},
		{http.MethodPatch, worker + "/config", `{"log.level":"debug"}`},
		{http.MethodPost, worker + "/restart", ""},
		{http.MethodGet, worker + "/logs/stream", ""},
		{http.MethodGet, worker + "/logs/archive", ""},
	} {
		worldtest.Expect(t, "home "+op.method+" "+op.path, a.Admin("home", op.method, op.path, op.body), offline)
	}
}
