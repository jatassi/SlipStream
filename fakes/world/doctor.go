package world

import (
	"bufio"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	gobuildinfo "debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/internal/control"
	"github.com/jatassi/slipstream/fakes/world/internal/proc"
	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/worktree"
)

// Report is doctor's verdict on a run: each check's result, and the fault
// rules still set, which a run faulted on purpose holds without failing.
type Report struct {
	OK           bool          `json:"ok"`
	Checks       []CheckResult `json:"checks"`
	ActiveFaults []ActiveFault `json:"activeFaults"`
}

type CheckResult struct {
	N    int    `json:"n"`
	Name string `json:"name"`
	OK   bool   `json:"ok"`
	// Failures say why the check failed. Notes are what it reports without
	// failing, such as a Node an action stopped or a front a rule holds.
	Failures []string `json:"failures,omitempty"`
	Notes    []string `json:"notes,omitempty"`
}

// ActiveFault is a rule still set, with the seq of the fault.set that set
// it.
type ActiveFault struct {
	Seq    uint64 `json:"seq"`
	ID     string `json:"id"`
	Target string `json:"target"`
	Origin string `json:"origin,omitempty"`
	Fault  string `json:"fault"`
}

const (
	rebuild        = "run go run ./cmd/release build and relaunch"
	slipstreamMain = "github.com/jatassi/slipstream/cmd/slipstream"
	// probeTimeout bounds each request and handshake, so one stalled
	// answer fails its check and not every check after it.
	probeTimeout = 10 * time.Second
)

var doctorChecks = []struct {
	name  string
	check func(*doctor, *CheckResult)
}{
	{"the run is ours", (*doctor).runIsOurs},
	{"the build matches the worktree", (*doctor).buildMatches},
	{"the cluster matches the Scenario", (*doctor).clusterMatches},
	{"every fake answers", (*doctor).fakesAnswer},
	{"Admin-entered addresses point at fakes", (*doctor).enteredPointAtFakes},
	{"the API key works", (*doctor).apiKeyWorks},
}

type doctor struct {
	ctx      context.Context
	worktree string
	m        Manifest
	active   []ActiveFault
	client   *http.Client
	tls      *tls.Config

	system *systemReply
}

type systemReply struct {
	status int
	body   []byte
	err    error
}

// Doctor runs six read-only checks on the live run in runDir against the
// git worktree that holds dir. It never signs in.
func Doctor(ctx context.Context, runDir, dir string) Report {
	d := &doctor{ctx: ctx, worktree: dir}
	r := Report{OK: true, Checks: []CheckResult{}, ActiveFaults: []ActiveFault{}}
	err := d.load(runDir)
	if err == nil {
		r.ActiveFaults = append(r.ActiveFaults, d.active...)
	}
	for i, c := range doctorChecks {
		res := CheckResult{N: i + 1, Name: c.name}
		switch {
		case err == nil:
			c.check(d, &res)
		case i == 0:
			res.Failures = []string{err.Error()}
		default:
			res.Failures = []string{"not run: the run is not ours"}
		}
		res.OK = len(res.Failures) == 0
		r.OK = r.OK && res.OK
		r.Checks = append(r.Checks, res)
	}
	return r
}

func (d *doctor) load(runDir string) error {
	var err error
	if d.m, err = ReadManifest(runDir); err != nil {
		return fmt.Errorf("the run is not live: %w", err)
	}
	live, err := LiveFaults(runDir)
	if err != nil {
		return err
	}
	for _, rec := range live {
		var a ActiveFault
		if err := json.Unmarshal(rec.Data, &a); err != nil {
			return fmt.Errorf("%s %d: %w", rec.Kind, rec.Seq, err)
		}
		a.Seq = rec.Seq
		d.active = append(d.active, a)
	}
	pem, err := os.ReadFile(d.m.CA.File)
	if err != nil {
		return fmt.Errorf("the run CA: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		return fmt.Errorf("the run CA %s holds no certificate", d.m.CA.File)
	}
	d.tls = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	d.client = &http.Client{Transport: &http.Transport{TLSClientConfig: d.tls}}
	return nil
}

func fail(c *CheckResult, format string, args ...any) {
	c.Failures = append(c.Failures, fmt.Sprintf(format, args...))
}

func note(c *CheckResult, format string, args ...any) {
	c.Notes = append(c.Notes, fmt.Sprintf(format, args...))
}

// controller is the Controller's manifest entry, or false for a run with
// no Controller or no API key to ask it with.
func (d *doctor) controller() (Node, bool) {
	i := slices.IndexFunc(d.m.Nodes, func(n Node) bool { return n.Setup == scenario.SetupController })
	if i < 0 || d.m.APIKey == "" {
		return Node{}, false
	}
	return d.m.Nodes[i], true
}

func (d *doctor) get(target, apiKey string) (status int, body []byte, err error) {
	ctx, cancel := context.WithTimeout(d.ctx, probeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
	if err != nil {
		return 0, nil, fmt.Errorf("GET %s: %w", target, err)
	}
	if apiKey != "" {
		req.Header.Set("X-Api-Key", apiKey)
	}
	res, err := d.client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("GET %s: %w", target, err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err = io.ReadAll(res.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("GET %s: %w", target, err)
	}
	return res.StatusCode, body, nil
}

// adminGet is a GET of path on the Controller with the API key, which
// must answer 200.
func (d *doctor) adminGet(path string) ([]byte, error) {
	n, _ := d.controller()
	status, body, err := d.get(n.URL+path, d.m.APIKey)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET %s on %s = %d %s", path, n.Name, status, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// systemReport is the Controller's GET /api/v1/system with the API key,
// sent once for checks 1 and 6.
func (d *doctor) systemReport() systemReply {
	if d.system == nil {
		n, _ := d.controller()
		status, body, err := d.get(n.URL+"/api/v1/system", d.m.APIKey)
		d.system = &systemReply{status, body, err}
	}
	return *d.system
}

func (d *doctor) runIsOurs(c *CheckResult) {
	if !proc.Alive(d.m.Supervisor.PID) {
		fail(c, "the supervisor (PID %d) is not running", d.m.Supervisor.PID)
	} else if err := d.ping(); err != nil {
		fail(c, "the control socket %s does not answer: %v", d.m.Supervisor.Control, err)
	}
	controller, hasController := d.controller()
	for _, n := range d.m.Nodes {
		switch {
		case n.State != NodeRunning:
			note(c, "node %s is %s", n.Name, n.State)
		case !proc.Alive(n.PID):
			fail(c, "node %s (PID %d) is not running", n.Name, n.PID)
		case hasController && n.Name == controller.Name:
			d.checkSystem(c, n)
		default:
			if status, body, err := d.get(n.URL+"/api/v1/status", ""); err != nil || status != http.StatusOK {
				fail(c, "node %s: GET /api/v1/status = %d %s %v", n.Name, status, strings.TrimSpace(string(body)), err)
			}
		}
	}
}

func (d *doctor) ping() error {
	ctx, cancel := context.WithTimeout(d.ctx, probeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://fakeworld/v1", http.NoBody)
	if err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	_, _, err = control.Do(ctx, d.m.Supervisor.Control, req)
	return err
}

// checkSystem checks the Controller n reports the PID the manifest names
// and a data dir under the run's scratch dir.
func (d *doctor) checkSystem(c *CheckResult, n Node) {
	sys := d.systemReport()
	if sys.err != nil || sys.status != http.StatusOK {
		fail(c, "node %s: GET /api/v1/system = %d %s %v", n.Name, sys.status, strings.TrimSpace(string(sys.body)), sys.err)
		return
	}
	var report struct {
		PID     int    `json:"pid"`
		DataDir string `json:"dataDir"`
	}
	if err := json.Unmarshal(sys.body, &report); err != nil {
		fail(c, "node %s: GET /api/v1/system: %v", n.Name, err)
		return
	}
	if report.PID != n.PID {
		fail(c, "node %s answers as PID %d, not the manifest's %d", n.Name, report.PID, n.PID)
	}
	if rel, err := filepath.Rel(d.m.ScratchDir, report.DataDir); err != nil || !filepath.IsLocal(rel) {
		fail(c, "node %s runs on data dir %s, outside the run's scratch dir %s", n.Name, report.DataDir, d.m.ScratchDir)
	}
}

func (d *doctor) buildMatches(c *CheckResult) {
	path := d.m.Binary.Path
	if sum, err := fileSHA256(path); err != nil {
		fail(c, "%v", err)
	} else if sum != d.m.Binary.SHA256 {
		fail(c, "%s hashes to %s, not the manifest's %s", path, sum, d.m.Binary.SHA256)
	}
	if info, err := gobuildinfo.ReadFile(path); err != nil {
		fail(c, "%s: %v", path, err)
	} else if info.Path != slipstreamMain {
		fail(c, "%s is built from %s, not %s", path, info.Path, slipstreamMain)
	}
	stamp, err := buildinfo.ReadFile(path)
	if errors.Is(err, buildinfo.ErrUnstamped) {
		fail(c, "%s is an unstamped build; %s", path, rebuild)
		return
	}
	if err != nil {
		fail(c, "%v", err)
		return
	}
	root, err := worktree.Root(d.ctx, d.worktree)
	if err != nil {
		fail(c, "%v", err)
		return
	}
	head, err := worktree.Stamp(d.ctx, root)
	if err != nil {
		fail(c, "%v", err)
		return
	}
	if stamp.Commit != head.Commit {
		fail(c, "the binary was built at commit %s, and the worktree's HEAD is %s; %s", stamp.Commit, head.Commit, rebuild)
	}
	if stamp.Fingerprint != head.Fingerprint {
		fail(c, "worktree changed since the build (fingerprint %s vs %s); %s", Abbrev(stamp.Fingerprint), Abbrev(head.Fingerprint), rebuild)
	}
}

// Abbrev is a fingerprint as doctor writes it: its first four characters
// and an ellipsis, or "clean" whole.
func Abbrev(fingerprint string) string {
	if fingerprint == worktree.CleanFingerprint {
		return fingerprint
	}
	return fingerprint[:min(4, len(fingerprint))] + "…"
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("hash the binary: %w", err)
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash the binary: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

type nodeRow struct {
	ID    string   `json:"id"`
	Roles []string `json:"roles"`
	Link  struct {
		State string `json:"state"`
	} `json:"link"`
}

func (d *doctor) clusterMatches(c *CheckResult) {
	if _, ok := d.controller(); !ok {
		note(c, "the run has no Controller")
		return
	}
	body, err := d.adminGet("/api/v1/nodes")
	if err != nil {
		fail(c, "%v", err)
		return
	}
	var list struct {
		Nodes []nodeRow `json:"nodes"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		fail(c, "listNodes: %v", err)
		return
	}
	for _, n := range d.m.Nodes {
		if n.Setup == scenario.SetupNone {
			continue
		}
		if n.State != NodeRunning {
			note(c, "node %s is %s", n.Name, n.State)
			continue
		}
		i := slices.IndexFunc(list.Nodes, func(r nodeRow) bool { return r.ID == n.NodeID })
		if i < 0 {
			fail(c, "node %s (%s) is not paired", n.Name, n.NodeID)
			continue
		}
		row, link := list.Nodes[i], "up"
		if n.Setup == scenario.SetupController {
			link = "local"
		}
		if row.Link.State != link {
			fail(c, "node %s's link is %s, not %s", n.Name, row.Link.State, link)
		}
		if !slices.Equal(row.Roles, n.Roles) {
			fail(c, "node %s holds %v, not the Scenario's %v", n.Name, row.Roles, n.Roles)
		}
	}
}

// holding is the active fault that covers the front f, if any.
func (d *doctor) holding(f Front) (ActiveFault, bool) {
	for _, a := range d.active {
		if (fault.Rule{Target: fault.Target(a.Target), Origin: a.Origin}).Covers(fault.Front{Target: fault.Target(f.Target), Origin: f.Node}) {
			return a, true
		}
	}
	return ActiveFault{}, false
}

func frontName(f Front) string {
	if f.Node == "" {
		return f.Target
	}
	return f.Target + "@" + f.Node
}

func (d *doctor) fakesAnswer(c *CheckResult) {
	held := map[string]bool{}
	for _, f := range d.m.Fronts {
		if a, ok := d.holding(f); ok {
			note(c, "%s is held by fault %s (seq %d)", frontName(f), a.ID, a.Seq)
			held[frontName(f)] = true
			continue
		}
		if err := d.handshake(f); err != nil {
			fail(c, "%s: %v", frontName(f), err)
		}
	}
	for _, f := range d.m.Fakes {
		if held[f.Name+"@"+events.World] {
			continue
		}
		if err := d.answers(f.URL); err != nil {
			fail(c, "%s: %v", f.Name, err)
		}
	}
}

func (d *doctor) answers(rawURL string) error {
	addr, ok := strings.CutPrefix(rawURL, "smtp://")
	if !ok {
		_, _, err := d.get(rawURL+"/", "")
		return err
	}
	ctx, cancel := context.WithTimeout(d.ctx, probeTimeout)
	defer cancel()
	conn, err := new(net.Dialer).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	defer func() { _ = conn.Close() }()
	deadline, _ := ctx.Deadline()
	_ = conn.SetReadDeadline(deadline)
	greeting, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return fmt.Errorf("read the greeting of %s: %w", addr, err)
	}
	if !strings.HasPrefix(greeting, "220 ") {
		return fmt.Errorf("%s greets %q, not 220", addr, strings.TrimSpace(greeting))
	}
	return nil
}

// handshake dials the front f, and completes a TLS handshake with the run
// CA when f serves HTTPS.
func (d *doctor) handshake(f Front) error {
	ctx, cancel := context.WithTimeout(d.ctx, probeTimeout)
	defer cancel()
	var dial interface {
		DialContext(ctx context.Context, network, address string) (net.Conn, error)
	} = &net.Dialer{}
	if strings.HasPrefix(f.URL, "https:") {
		dial = &tls.Dialer{Config: d.tls}
	}
	conn, err := dial.DialContext(ctx, "tcp", f.Address)
	if err != nil {
		return fmt.Errorf("dial %s: %w", f.Address, err)
	}
	return conn.Close() //nolint:wrapcheck // closing a probe that connected has nothing to add
}

func (d *doctor) enteredPointAtFakes(c *CheckResult) {
	if _, ok := d.controller(); !ok {
		note(c, "the run has no Controller to ask")
		return
	}
	for _, spec := range fakes {
		if spec.Entered == nil || !slices.ContainsFunc(d.m.Fakes, func(f Fake) bool { return f.Name == spec.Name }) {
			continue
		}
		entered, err := spec.Entered(d.adminGet)
		if err != nil {
			fail(c, "%s: %v", spec.Name, err)
			continue
		}
		for _, e := range entered {
			d.checkEntered(c, spec.Name, e.What, e.URL, e.NodeID)
		}
	}
}

// checkEntered checks the URL an Admin entered as what, which the Node
// with id dials, is the fake target's front for that Node.
func (d *doctor) checkEntered(c *CheckResult, target, what, rawURL, id string) {
	i := slices.IndexFunc(d.m.Nodes, func(n Node) bool { return n.NodeID == id })
	if i < 0 {
		fail(c, "%s %s is dialed from %s, no Node of the run", what, rawURL, id)
		return
	}
	name := d.m.Nodes[i].Name
	j := slices.IndexFunc(d.m.Fronts, func(f Front) bool { return f.Target == target && f.Node == name })
	u, err := url.Parse(rawURL)
	if j < 0 || err != nil || u.Host != d.m.Fronts[j].Address {
		fail(c, "%s %s is not %s's front for %s", what, rawURL, target, name)
	}
}

func (d *doctor) apiKeyWorks(c *CheckResult) {
	if _, ok := d.controller(); !ok {
		note(c, "the run has no API key")
		return
	}
	if sys := d.systemReport(); sys.err != nil || sys.status != http.StatusOK {
		fail(c, "GET /api/v1/system with the API key = %d %s %v", sys.status, strings.TrimSpace(string(sys.body)), sys.err)
	}
}

// String is the report as doctor prints it: one line per check, then
// each active fault.
func (r Report) String() string {
	var b strings.Builder
	for _, c := range r.Checks {
		mark := "ok  "
		if !c.OK {
			mark = "FAIL"
		}
		fmt.Fprintf(&b, "%s %d %s\n", mark, c.N, c.Name)
		for _, f := range c.Failures {
			fmt.Fprintf(&b, "       %s\n", f)
		}
		for _, n := range c.Notes {
			fmt.Fprintf(&b, "       (%s)\n", n)
		}
	}
	for _, a := range r.ActiveFaults {
		at := a.Target
		if a.Origin != "" {
			at += "@" + a.Origin
		}
		fmt.Fprintf(&b, "active fault %s (seq %d): %s %s\n", a.ID, a.Seq, at, a.Fault)
	}
	return b.String()
}
