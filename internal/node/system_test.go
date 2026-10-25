package node

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/api/apitest"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/node/internal/logs"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodesystem"
)

// claimKey claims n through c and returns the Admin's API key.
func claimKey(t *testing.T, n *Node, c client) string {
	t.Helper()
	return decode[struct {
		APIKey string `json:"apiKey"`
	}](t, c.claim(n.Ready().SetupCode, "1234"), http.StatusCreated).APIKey
}

// selfID is the Controller's own row's id.
func selfID(t *testing.T, c client, apiKey string) string {
	t.Helper()
	for _, row := range decode[struct {
		Nodes []nodeRow `json:"nodes"`
	}](t, c.admin(apiKey, http.MethodGet, "/api/v1/nodes", ""), http.StatusOK).Nodes {
		if row.Self {
			return row.ID
		}
	}
	t.Fatal("listNodes has no row for the Controller itself")
	return ""
}

// fieldOf is key's field in a getNodeConfig or updateNodeConfig answer.
func fieldOf(t *testing.T, r reply, key string) map[string]any {
	t.Helper()
	for _, s := range decode[struct {
		Sections []struct {
			ID     string           `json:"id"`
			Fields []map[string]any `json:"fields"`
		} `json:"sections"`
	}](t, r, http.StatusOK).Sections {
		for _, f := range s.Fields {
			if f["key"] == key {
				return f
			}
		}
	}
	t.Fatalf("no field %s in %s", key, r.body)
	return nil
}

// fieldLine is a field's saved and running values, whether it waits for
// a restart, and when it applies.
func fieldLine(f map[string]any) string {
	return fmt.Sprint(f["value"], " ", f["running"], " ", f["pendingRestart"], " ", f["applies"])
}

// ringHead is the seq of the newest record in n's ring.
func ringHead(t *testing.T, n *Node) string {
	t.Helper()
	var after uint64
	for {
		b, err := n.logs.Read(t.Context(), after, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(b.Records) == 0 {
			return strconv.FormatUint(after, 10)
		}
		after = b.Records[len(b.Records)-1].Seq
	}
}

type event struct {
	id, name string
	record   streamedRecord
	reset    streamedReset
}

type streamedRecord struct {
	Seq   uint64         `json:"seq"`
	Msg   string         `json:"msg"`
	Attrs map[string]any `json:"attrs"`
}

type streamedReset struct {
	Reason    string    `json:"reason"`
	StartedAt time.Time `json:"startedAt"`
}

func (c client) logStream(apiKey, path, lastEventID string) *apitest.Stream {
	c.t.Helper()
	k := call{method: http.MethodGet, path: path, headers: map[string]string{"X-Api-Key": apiKey}}
	if lastEventID != "" {
		k.headers["Last-Event-ID"] = lastEventID
	}
	res := c.send(k) //nolint:bodyclose // the Stream closes it
	if res.StatusCode != http.StatusOK {
		c.t.Fatalf("GET %s = %d", path, res.StatusCode)
	}
	s := apitest.NewStream(res.Body)
	c.t.Cleanup(func() { _ = s.Close() })
	return s
}

// nextEvent is the stream's next record or reset, past keepalives.
func nextEvent(t *testing.T, s *apitest.Stream) event {
	t.Helper()
	for {
		var e event
		var data string
		for line := range strings.SplitSeq(next(t, s), "\n") {
			name, value, _ := strings.Cut(line, ": ")
			switch name {
			case "id":
				e.id = value
			case "event":
				e.name = value
			case "data":
				data = value
			}
		}
		var err error
		switch e.name {
		case "":
			continue
		case "log.record":
			err = json.Unmarshal([]byte(data), &e.record)
		case "log.reset":
			err = json.Unmarshal([]byte(data), &e.reset)
		}
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
}

// nextReset is the next reset on s, past any records.
func nextReset(t *testing.T, s *apitest.Stream) streamedReset {
	t.Helper()
	for {
		if e := nextEvent(t, s); e.name == "log.reset" {
			return e.reset
		}
	}
}

// nextProbe is the next record logged as "probe".
func nextProbe(t *testing.T, s *apitest.Stream) event {
	t.Helper()
	for {
		if e := nextEvent(t, s); e.name == "log.record" && e.record.Msg == "probe" {
			return e
		}
	}
}

// TestTheLogSectionAppliesWithNoRestart runs on the OS network, since
// rotation needs the rotating log.
func TestTheLogSectionAppliesWithNoRestart(t *testing.T) {
	env, _ := testEnv(t)
	n := startTest(t, env)
	c := client{t: t, c: &http.Client{}, base: "http://" + n.Ready().Listeners.HTTP.String()}
	key := claimKey(t, n, c)
	path := "/api/v1/nodes/" + selfID(t, c, key) + "/config"

	f := fieldOf(t, c.admin(key, http.MethodPatch, path, `{"log.level":"debug"}`), "log.level")
	if got := fieldLine(f); got != "debug debug false live" {
		t.Fatalf("log.level = %s, want debug saved and running", got)
	}
	n.rootLog.Debug("probe")
	if got := debugRecords(t, env); !slices.Contains(got, "probe") {
		t.Fatalf("debug records = %v, want probe among them", got)
	}

	fieldOf(t, c.admin(key, http.MethodPatch, path, `{"log.max_size_mb":1}`), "log.max_size_mb")
	pad := strings.Repeat("x", 1024)
	for range 2048 {
		n.rootLog.Info("filler", "pad", pad)
	}
	entries, err := os.ReadDir(env.DataDir.LogsDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 2 {
		t.Fatalf("the logs dir holds %d files after 2 MB at a 1 MB limit, want a rotated one beside slipstream.log", len(entries))
	}
}

func TestTheConfigHoldsARestartRowAndRefusesAsAWhole(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env, host := inMemory(fallbackEnv(t, "10"))
		n, c := startOn(t, env, host)
		key := claimKey(t, n, c)
		self := selfID(t, c, key)
		path := "/api/v1/nodes/" + self + "/config"

		f := fieldOf(t, c.admin(key, http.MethodPatch, path, `{"http.listen":9000}`), "http.listen")
		if got := fieldLine(f); got != "9000 8484 true restart" {
			t.Fatalf("http.listen = %s, want 9000 saved, 8484 running, pending a restart", got)
		}
		system := decode[struct {
			Listeners struct {
				HTTP string `json:"http"`
			} `json:"listeners"`
		}](t, c.admin(key, http.MethodGet, "/api/v1/nodes/"+self+"/system", ""), http.StatusOK)
		if system.Listeners.HTTP != "127.0.0.1:8484" {
			t.Fatalf("listeners.http = %s after the change, want 127.0.0.1:8484 until a restart", system.Listeners.HTTP)
		}

		expect(t, c.admin(key, http.MethodPatch, path, `{"http.listen":8485}`), 422,
			`{"status":422,"code":"port_used_by_node","title":"The Node uses this port already","detail":"The Node link already uses port 8485.","errors":[{"location":"body.http.listen","message":"The Node link already uses port 8485."}]}`+"\n")
		saved := readConfigYAML(t, env.configFile())
		expect(t, c.admin(key, http.MethodPatch, path, `{"log.level":"warn","log.max_size_mb":5000}`), 422,
			`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"Between 1 and 1000 MB.","errors":[{"location":"body.log.max_size_mb","message":"Between 1 and 1000 MB."}]}`+"\n")
		if got := readConfigYAML(t, env.configFile()); got != saved {
			t.Fatalf("config.yaml = %q after a refused change, want %q", got, saved)
		}
		if f := fieldOf(t, c.admin(key, http.MethodGet, path, ""), "log.level"); f["value"] != "info" || n.cfg.Log.Level != logs.Info {
			t.Fatalf("log.level = %v after a refused change, want info", f)
		}

		lib := n.lib.Load().db
		session := identitytest.Session(t, lib, identitytest.Member(t, lib, "member@example.com"))
		expect(t, c.do(call{method: http.MethodPatch, path: path, body: strings.NewReader(`{"log.level":"warn"}`), session: session}), 403,
			problem(403, "needs_admin", "Only an Admin can do this"))
	})
}

// settingRefused is a 422 refusing key's value with sentence.
func settingRefused(code, title, key, sentence string) string {
	return fmt.Sprintf(`{"status":422,"code":%q,"title":%q,"detail":%q,"errors":[{"location":"body.%s","message":%q}]}`, code, title, sentence, key, sentence) + "\n"
}

func TestTheTorrentRowsTakeAnAddressAndRefuseAFolderTheyCantUse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env := fallbackEnv(t, "1")
		env.Environ = []string{"SLIPSTREAM_LINK_LISTEN=127.0.0.1:8485"}
		env, host := inMemory(env)
		n, c := startOn(t, env, host)
		key := claimKey(t, n, c)
		path := "/api/v1/nodes/" + selfID(t, c, key) + "/config"

		var torrent []string
		for _, s := range decode[struct {
			Sections []struct {
				ID     string           `json:"id"`
				Fields []map[string]any `json:"fields"`
			} `json:"sections"`
		}](t, c.admin(key, http.MethodGet, path, ""), http.StatusOK).Sections {
			if s.ID == "torrent" {
				for _, f := range s.Fields {
					torrent = append(torrent, fmt.Sprint(f["key"], " ", f["kind"]))
				}
			}
		}
		if want := []string{"torrent.listen address", "torrent.downloads folder"}; !slices.Equal(torrent, want) {
			t.Fatalf("the torrent section holds %v, want %v", torrent, want)
		}

		f := fieldOf(t, c.admin(key, http.MethodPatch, path, `{"torrent.listen":"[::]:51413"}`), "torrent.listen")
		if got := fieldLine(f); got != "[::]:51413 0.0.0.0:6881 true restart" {
			t.Fatalf("torrent.listen = %s, want [::]:51413 saved, 0.0.0.0:6881 running, pending a restart", got)
		}
		saved := readConfigYAML(t, env.configFile())
		if !strings.Contains(saved, "[::]:51413") {
			t.Fatalf("config.yaml = %q, want the new listen address", saved)
		}

		const outOfRange, portUsed = "setting_out_of_range", "port_used_by_node"
		const outOfRangeTitle, portUsedTitle = "The value is outside the setting's range", "The Node uses this port already"
		for _, c2 := range []struct{ value, code, title, sentence string }{
			{"51413", outOfRange, outOfRangeTitle, "An address and port, such as 0.0.0.0:6881."},
			{"[::]:0", outOfRange, outOfRangeTitle, "A port between 1 and 65535."},
			{"127.0.0.1:8485", portUsed, portUsedTitle, "The Node link already uses port 8485."},
			{"127.0.0.1:8484", portUsed, portUsedTitle, "The web server already uses port 8484."},
		} {
			expect(t, c.admin(key, http.MethodPatch, path, `{"torrent.listen":"`+c2.value+`"}`), 422,
				settingRefused(c2.code, c2.title, "torrent.listen", c2.sentence))
		}
		expect(t, c.admin(key, http.MethodPatch, path, `{"torrent.downloads":"downloads"}`), 422,
			settingRefused("folder_unusable", "SlipStream can't use this folder", "torrent.downloads", "Can't use this folder because it isn't a full path."))
		if got := readConfigYAML(t, env.configFile()); got != saved {
			t.Fatalf("config.yaml = %q after refused changes, want %q", got, saved)
		}
	})
}

func TestRestartThroughTheAPIAnswersBeforeTheNodeStops(t *testing.T) {
	env, _ := testEnv(t)
	env, next := restartable(t, env)
	n := startTest(t, env)
	c := client{t: t, c: &http.Client{}, base: "http://" + n.Ready().Listeners.HTTP.String()}
	key := claimKey(t, n, c)

	r := c.admin(key, http.MethodPost, "/api/v1/nodes/"+selfID(t, c, key)+"/restart", "")
	accepted := decode[struct {
		StartedAt time.Time `json:"startedAt"`
		BoundPort uint16    `json:"boundPort"`
		NextPort  *uint16   `json:"nextPort"`
	}](t, r, http.StatusAccepted)
	if !accepted.StartedAt.Equal(n.started) || accepted.BoundPort != n.Ready().Listeners.HTTP.Port() || accepted.NextPort != nil {
		t.Fatalf("restartNode = %s, want startedAt %s and boundPort %d", r.body, n.started, n.Ready().Listeners.HTTP.Port())
	}
	if _, err := n.host.Restart(); !errors.Is(err, nodesystem.ErrAlreadyRestarting) {
		t.Fatalf("a second restart = %v, want %v", err, nodesystem.ErrAlreadyRestarting)
	}
	waitDone(t, n)
	var successor *Node
	select {
	case successor = <-next:
	case <-time.After(30 * time.Second):
		t.Fatal("no successor within 30s")
	}
	if successor == nil {
		t.FailNow()
	}
	sc := client{t: t, c: &http.Client{}, base: "http://" + successor.Ready().Listeners.HTTP.String()}
	later := decode[struct {
		StartedAt time.Time `json:"startedAt"`
	}](t, sc.admin(key, http.MethodGet, "/api/v1/system", ""), http.StatusOK)
	if !later.StartedAt.After(accepted.StartedAt) {
		t.Fatalf("the successor started at %s, want after %s", later.StartedAt, accepted.StartedAt)
	}
}

// controllerInMemory is a claimed Controller in the caller's bubble, with
// its Admin's API key and its own row's path.
func controllerInMemory(t *testing.T, env Env) (n *Node, c client, apiKey, self string) {
	t.Helper()
	n, c = startInMemory(t, env)
	apiKey = claimKey(t, n, c)
	return n, c, apiKey, "/api/v1/nodes/" + selfID(t, c, apiKey)
}

// probes are the next count events on s, each a probe record numbered
// from 1 with its seq as its id, in rising order.
func probes(t *testing.T, s *apitest.Stream, count int) []event {
	t.Helper()
	var out []event
	for i := range count {
		e := nextEvent(t, s)
		if e.name != "log.record" || e.record.Msg != "probe" || e.record.Attrs["n"] != float64(i+1) || e.id != strconv.FormatUint(e.record.Seq, 10) {
			t.Fatalf("event %d = %+v, want probe %d with its seq as its id", i, e, i+1)
		}
		if i > 0 && e.record.Seq <= out[i-1].record.Seq {
			t.Fatalf("seq %d follows %d", e.record.Seq, out[i-1].record.Seq)
		}
		out = append(out, e)
	}
	return out
}

func TestTheLogStreamsAndResumesAfterTheLastEventID(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env, _ := testEnv(t)
		n, c, key, self := controllerInMemory(t, env)
		path := self + "/logs/stream"

		s := c.logStream(key, path+"?after="+ringHead(t, n), "")
		for i := range 5 {
			n.rootLog.Info("probe", "n", i+1)
		}
		sent := probes(t, s, 5)
		_ = s.Close()

		resumed := c.logStream(key, path, sent[2].id)
		for _, want := range sent[3:] {
			if e := nextEvent(t, resumed); e.id != want.id || e.record.Msg != "probe" {
				t.Fatalf("after Last-Event-ID %s came %+v, want %s", sent[2].id, e, want.id)
			}
		}
		_ = resumed.Close()

		restarted := c.logStream(key, path, "1")
		if e := nextEvent(t, restarted); e.name != "log.reset" || e.reset.Reason != "restarted" || !e.reset.StartedAt.Equal(n.started) {
			t.Fatalf("an id from an earlier process gave %+v, want reset restarted at %s", e, n.started)
		}
	})
}

func TestALappedReadResetsToTheRingAndAJoinCodeStreamsMasked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env, _ := testEnv(t)
		n, c, key, self := controllerInMemory(t, env)
		pub := n.key.Public()
		code, err := joincode.New(pub[:], [joincode.SecretSize]byte{7}, []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:8485")})
		if err != nil {
			t.Fatal(err)
		}
		fieldOf(t, c.admin(key, http.MethodPatch, self+"/config", `{"log.viewer_lines":100}`), "log.viewer_lines")
		synctest.Wait()
		after := ringHead(t, n)
		for i := range 150 {
			n.rootLog.Info("probe", "n", i+1, "code", code)
		}

		s := c.logStream(key, self+"/logs/stream?after="+after, "")
		reset := nextEvent(t, s)
		if reset.name != "log.reset" || reset.reset.Reason != "lapped" {
			t.Fatalf("a lapped read began with %+v, want reset lapped", reset)
		}
		var ring []string
		for range 100 {
			e := nextEvent(t, s)
			ring = append(ring, fmt.Sprint(e.record.Msg, " ", e.record.Attrs["n"], " ", e.record.Attrs["code"]))
			if len(ring) == 1 && e.id != strconv.FormatUint(mustUint(t, reset.id)+1, 10) {
				t.Fatalf("the ring's oldest is %s after a reset with id %s", e.id, reset.id)
			}
		}
		if first, last := "probe 51 "+code.Redacted(), "probe 150 "+code.Redacted(); ring[0] != first || ring[99] != last {
			t.Fatalf("the ring runs %q to %q, want %q to %q", ring[0], ring[99], first, last)
		}
	})
}

func mustUint(t *testing.T, s string) uint64 {
	t.Helper()
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// archived is the zip of path's download: its file name and entries.
func archived(t *testing.T, c client, apiKey, path string) (name string, entries []*zip.File) {
	t.Helper()
	r := c.admin(apiKey, http.MethodGet, path, "")
	if r.status != http.StatusOK || r.header.Get("Content-Type") != "application/zip" || r.header.Get("Content-Length") != strconv.Itoa(len(r.body)) {
		t.Fatalf("GET %s = %d %v", path, r.status, r.header)
	}
	disposition := regexp.MustCompile(`^attachment; filename=(slipstream-logs-[a-z]+-\d{8}-\d{6}\.zip)$`).FindStringSubmatch(r.header.Get("Content-Disposition"))
	if disposition == nil {
		t.Fatalf("Content-Disposition = %q", r.header.Get("Content-Disposition"))
	}
	z, err := zip.NewReader(strings.NewReader(r.body), int64(len(r.body)))
	if err != nil {
		t.Fatal(err)
	}
	return disposition[1], z.File
}

func contents(t *testing.T, f *zip.File) []byte {
	t.Helper()
	rc, err := f.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// rotatedLogs are a compressed and a plain file in the logs dir, as
// rotation leaves them.
var rotatedLogs = map[string]string{
	"slipstream-2026-10-01T00-00-00.000.log.gz": "\x1f\x8b gzip bytes",
	"slipstream-2026-10-02T00-00-00.000.log":    `{"msg":"old"}` + "\n",
}

func TestTheArchiveZipsTheLogFolder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env, _ := testEnv(t)
		env.Environ = append(env.Environ, "SLIPSTREAM_NAME=home")
		if err := os.MkdirAll(env.DataDir.LogsDir(), 0o700); err != nil {
			t.Fatal(err)
		}
		for name, body := range rotatedLogs {
			writeFile(t, filepath.Join(env.DataDir.LogsDir(), name), body)
		}
		_, c, key, self := controllerInMemory(t, env)
		path := self + "/logs/archive"

		head := c.do(call{method: http.MethodHead, path: path, headers: map[string]string{"X-Api-Key": key}})
		if head.status != http.StatusOK || head.header.Get("Content-Type") != "application/zip" || head.body != "" {
			t.Fatalf("HEAD = %d %v %q, want 200 application/zip and no body", head.status, head.header, head.body)
		}
		name, entries := archived(t, c, key, path)
		var names []string
		for _, e := range entries {
			names = append(names, e.Name)
		}
		if want := []string{"slipstream.log", "slipstream-2026-10-01T00-00-00.000.log.gz", "slipstream-2026-10-02T00-00-00.000.log"}; !strings.HasPrefix(name, "slipstream-logs-home-") || !slices.Equal(names, want) {
			t.Fatalf("%s holds %v, want slipstream-logs-home-<stamp>.zip holding %v", name, names, want)
		}
		current, err := os.ReadFile(filepath.Join(env.DataDir.LogsDir(), "slipstream.log"))
		if err != nil {
			t.Fatal(err)
		}
		if first := contents(t, entries[0]); len(first) == 0 || !bytes.HasPrefix(current, first) {
			t.Fatalf("the zip's slipstream.log is %d bytes, not the file's first", len(first))
		}
		gz, plain := entries[1], entries[2]
		if string(contents(t, gz)) != rotatedLogs[gz.Name] || string(contents(t, plain)) != rotatedLogs[plain.Name] || gz.Method != zip.Store {
			t.Fatalf("the rotated files were zipped as %q (method %d) and %q", contents(t, gz), gz.Method, contents(t, plain))
		}
	})
}

// settingsChanged waits for node.settings-changed about node on sub.
func settingsChanged(t *testing.T, sub *live.Subscription, node string) {
	t.Helper()
	for {
		msg, err := sub.Next(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if v, ok := msg.Data.(nodes.SettingsChanged); ok && v.NodeID == node {
			return
		}
	}
}

// TestTheControllerManagesAWorkersSystem runs a Controller and a Worker on
// one in-memory network: the Controller changes the Worker's settings,
// follows its log through a restart and downloads its archive, and a
// stopped Worker answers node_offline.
func TestTheControllerManagesAWorkersSystem(t *testing.T) {
	net := fault.NewNetwork()
	homeEnv, homeHost := machine(t, net, "127.0.0.2", "home")
	boxEnv, boxHost := machine(t, net, "127.0.0.3", "box")
	synctest.Test(t, func(t *testing.T) {
		boxEnv, next := restartable(t, boxEnv)
		home, hc := startOn(t, homeEnv, homeHost)
		apiKey, code := claimedHome(t, home, hc)
		box, bc := startOn(t, boxEnv, boxHost)
		id := joinBox(t, box, bc, boxEnv.DataDir, code, apiKey).NodeID
		synctest.Wait()
		worker := "/api/v1/nodes/" + id

		sub := home.hub.Subscribe(live.Viewer{Admin: true}, "")
		defer sub.Close()
		f := fieldOf(t, hc.admin(apiKey, http.MethodPatch, worker+"/config", `{"log.level":"debug"}`), "log.level")
		box.cfgMu.Lock()
		level := box.cfg.Log.Level
		box.cfgMu.Unlock()
		if f["value"] != "debug" || f["running"] != "debug" || level != logs.Debug {
			t.Fatalf("the Worker's log.level = %v, running %s; want debug", f, level)
		}
		settingsChanged(t, sub, id)
		facts := decode[struct {
			Facts struct {
				HTTPAddress string `json:"httpAddress"`
			} `json:"facts"`
		}](t, hc.admin(apiKey, http.MethodGet, worker, ""), http.StatusOK).Facts
		if facts.HTTPAddress != box.Ready().Listeners.HTTP.String() {
			t.Fatalf("the Worker's facts give httpAddress %q, want %s", facts.HTTPAddress, box.Ready().Listeners.HTTP)
		}

		s := hc.logStream(apiKey, worker+"/logs/stream?after="+ringHead(t, box), "")
		box.rootLog.Info("probe", "n", 1)
		if e := nextProbe(t, s); e.record.Attrs["n"] != 1.0 {
			t.Fatalf("followed %+v, want probe 1", e)
		}

		name, entries := archived(t, hc, apiKey, worker+"/logs/archive")
		if !strings.HasPrefix(name, "slipstream-logs-box-") || len(entries) == 0 || entries[0].Name != "slipstream.log" {
			t.Fatalf("the Worker's archive %s holds %d entries", name, len(entries))
		}

		decode[struct{}](t, hc.admin(apiKey, http.MethodPost, worker+"/restart", ""), http.StatusAccepted)
		waitDone(t, box)
		successor := <-next
		if successor == nil {
			t.FailNow()
		}
		if reset := nextReset(t, s); reset.Reason != "restarted" || !reset.StartedAt.Equal(successor.started) {
			t.Fatalf("the stream reset with %+v, want restarted at %s", reset, successor.started)
		}
		successor.rootLog.Info("probe", "n", 2)
		if e := nextProbe(t, s); e.record.Attrs["n"] != 2.0 {
			t.Fatalf("followed %+v after the restart, want probe 2", e)
		}

		successor.Stop()
		waitDone(t, successor)
		synctest.Wait()
		expect(t, hc.admin(apiKey, http.MethodGet, worker+"/system", ""), 503, problem(503, "node_offline", "The Node is offline"))
		expect(t, hc.admin(apiKey, http.MethodGet, worker+"/logs/stream", ""), 503, problem(503, "node_offline", "The Node is offline"))
	})
}
