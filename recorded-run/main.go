// Command recorder runs the S43 (Member access to Media Servers) recorded
// Plex run against real plex.tv, one step per subcommand. It keeps the
// device key, tokens and ids in a state directory outside the repo and
// appends every HTTP call, and every finding, to a JSONL log with tokens,
// PIN codes, emails, usernames, ids, machine ids and addresses replaced by
// stable labels. Standard library only: go run main.go <command>.
package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	webHost = envOr("PLEX_MEMBERS_WEB", "https://plex.tv")
	apiHost = envOr("PLEX_MEMBERS_API", "https://clients.plex.tv")
	pmsHost = envOr("PLEX_MEMBERS_PMS", "http://127.0.0.1:32400")
)

const (
	authPage     = "https://app.plex.tv/auth"
	product      = "Recorded run"
	productVer   = "0.0.1"
	container    = "recorded-run-pms"
	sectionTitle = "Member access test"
	logName      = "plex-members-run.jsonl"
	resourcesQ   = "/api/v2/resources?includeHttps=1&includeRelay=1&includeIPv6=1"
	pendingPath  = "/api/v2/shared_servers/invites/received/pending"
	members      = "BCDE"
)

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// account is one sign-in: A with the JWT PIN, or a Member with a legacy
// PIN. Slot B2 is account B signed in a second time on plex.tv.
type account struct {
	Slot      string    `json:"slot"`
	Who       string    `json:"who"`
	PinHost   string    `json:"pin_host,omitempty"`
	ClientID  string    `json:"client_id"`
	KeySeed   []byte    `json:"key_seed,omitempty"`
	Token     string    `json:"token,omitempty"`
	TokenAt   time.Time `json:"token_at"`
	SignedOut bool      `json:"signed_out,omitempty"`
	ID        string    `json:"id,omitempty"`
	UUID      string    `json:"uuid,omitempty"`
	Username  string    `json:"username,omitempty"`
	Email     string    `json:"email,omitempty"`
	Title     string    `json:"title,omitempty"`
	PlexPass  bool      `json:"plex_pass,omitempty"`
}

type secret struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type state struct {
	Accounts      map[string]*account `json:"accounts"`
	MachineID     string              `json:"machine_id,omitempty"`
	ServerToken   string              `json:"server_token,omitempty"`
	ServerVersion string              `json:"server_version,omitempty"`
	SectionKey    string              `json:"section_key,omitempty"`
	SectionIDs    []string            `json:"section_ids,omitempty"`
	// Shares are the first shared server id plex.tv gave each Member.
	Shares map[string]string `json:"shares"`
	// Managed are the managed Home users q6-full made, for cleanup.
	Managed []string `json:"managed,omitempty"`
	Secrets []secret `json:"secrets"`
}

type run struct {
	ctx     context.Context
	dir     string
	st      *state
	pending []map[string]any
	http    *http.Client
	rd      *redactor
}

// who is the client identifier and token a call goes out with.
type who struct{ clientID, token string }

type reply struct {
	status int
	raw    []byte
	tree   any
	err    error
	entry  map[string]any
}

func (rp reply) ok() bool { return rp.err == nil && rp.status/100 == 2 }

type stopRule struct{ why string }

func (s stopRule) Error() string { return "stop rule: " + s.why }

var commands = map[string]func(*run, []string) error{
	"init":      cmdInit,
	"import":    cmdImport,
	"signin":    cmdSignin,
	"media":     cmdMedia,
	"claim":     cmdClaim,
	"server":    cmdServer,
	"precheck":  cmdPrecheck,
	"q1":        cmdQ1,
	"q2":        cmdQ2,
	"q3":        cmdQ3,
	"q4":        cmdQ4,
	"q5":        cmdQ5,
	"q6":        cmdQ6,
	"q6-remove": cmdQ6Remove,
	"q6-full":   cmdQ6Full,
	"q7":        cmdQ7,
	"q8":        cmdQ8,
	"cleanup":   cmdCleanup,
	"note":      cmdNote,
	"status":    cmdStatus,
	"export":    cmdExport,
}

const usageText = `usage: go run main.go [-state DIR] <command> [args]

In order (RUNBOOK.md has the detail):
  init                      make the state dir, A's device key and client identifier
  import [-file F] [-token-at RFC3339]
                            read B-E (and A's legacy token, as slot AL) from accounts.json
                            and read /api/v2/user with each token
  signin A                  JWT PIN with A's device key; approve as A
  media                     two short clips for the "Member access test" section
  claim                     claim token for A and the docker run for the server
  server                    find the claimed server, make the section, read plex.tv section ids
  signin B|C|D|E [-host clients|plex] [-again]
                            fallback to import: legacy PIN without a JWK under its own client
                            identifier; -host plex signs in on plex.tv as slot B2, C2, ...
  precheck                  records that C is A's friend and D has A's pending Home invite
  q1 .. q7                  one spec question each (q6-remove and q6-full follow q6)
  cleanup [-all] [-keep-admin]
                            remove shares, invites, Home users, friendships, the server
                            device and Member tokens (B and B2 kept for q8 unless -all)
  q8 [-force]               eight days after signin B: token still works, sign-out scope
  note <step> <text...>     a finding line for something seen by hand
  status                    what is done, by label
  export -date YYYY-MM-DD [-out DIR]
                            audits the log for raw secrets and copies it out
`

func main() {
	fs := flag.NewFlagSet("recorder", flag.ExitOnError)
	home, _ := os.UserHomeDir()
	dir := fs.String("state", filepath.Join(home, ".slipstream-recorded-runs", "plex-members"), "state directory")
	fs.Usage = func() { fmt.Fprint(os.Stderr, usageText) }
	_ = fs.Parse(os.Args[1:])
	args := fs.Args()
	if len(args) == 0 {
		fs.Usage()
		os.Exit(2)
	}
	cmd, ok := commands[args[0]]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		fs.Usage()
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	r, err := open(ctx, *dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	err = cmd(r, args[1:])
	if cerr := r.close(); cerr != nil {
		fmt.Fprintln(os.Stderr, "error saving the run:", cerr)
		os.Exit(1)
	}
	var sr stopRule
	if errors.As(err, &sr) {
		fmt.Fprintf(os.Stderr, "\n*** STOP RULE: %s\n*** Stop the run here and report. Run cleanup, then hand back the log.\n", r.redactor().str(sr.why))
		os.Exit(3)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func open(ctx context.Context, dir string) (*run, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	st := &state{Accounts: map[string]*account{}, Shares: map[string]string{}}
	raw, err := os.ReadFile(filepath.Join(dir, "state.json"))
	switch {
	case err == nil:
		if err := json.Unmarshal(raw, st); err != nil {
			return nil, fmt.Errorf("read state.json: %w", err)
		}
	case !errors.Is(err, os.ErrNotExist):
		return nil, err
	}
	if st.Shares == nil {
		st.Shares = map[string]string{}
	}
	return &run{ctx: ctx, dir: dir, st: st, http: &http.Client{Timeout: 60 * time.Second}}, nil
}

// close writes the pending log lines, redacted with every label the
// command learned, and saves the state.
func (r *run) close() error {
	ferr := r.flush()
	raw, err := json.MarshalIndent(r.st, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(r.dir, "state.json.tmp")
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(r.dir, "state.json")); err != nil {
		return err
	}
	return ferr
}

func (r *run) flush() error {
	if len(r.pending) == 0 {
		return nil
	}
	f, err := os.OpenFile(filepath.Join(r.dir, logName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	rd := r.redactor()
	for _, e := range r.pending {
		if err := enc.Encode(rd.entry(e)); err != nil {
			return err
		}
	}
	r.pending = nil
	return nil
}

// label gives value a stable label, the first one it was given.
func (r *run) label(value, base string) string {
	if value == "" {
		return ""
	}
	for _, s := range r.st.Secrets {
		if s.Value == value {
			return s.Label
		}
	}
	taken := func(l string) bool {
		for _, s := range r.st.Secrets {
			if s.Label == l {
				return true
			}
		}
		return false
	}
	l := "<" + base + ">"
	for n := 2; taken(l); n++ {
		l = fmt.Sprintf("<%s-%d>", base, n)
	}
	r.st.Secrets = append(r.st.Secrets, secret{Value: value, Label: l})
	r.rd = nil
	return l
}

func (r *run) redactor() *redactor {
	if r.rd == nil {
		r.rd = newRedactor(r.st.Secrets)
	}
	return r.rd
}

// --- accounts ---

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

func (r *run) slot(name string) *account {
	a := r.st.Accounts[name]
	if a == nil {
		a = &account{Slot: name, Who: name[:1], ClientID: newUUID()}
		r.st.Accounts[name] = a
	}
	r.label(a.ClientID, "client-id-"+name)
	return a
}

// member is who's signed-in slot: the clients.plex.tv one, else the
// plex.tv one.
func (r *run) member(w string) (*account, error) {
	for _, s := range []string{w, w + "2"} {
		if a := r.st.Accounts[s]; a != nil && a.Token != "" && a.ID != "" && !a.SignedOut {
			return a, nil
		}
	}
	return nil, fmt.Errorf("%s isn't signed in: run signin %s first", w, w)
}

func (r *run) admin() (*account, error) {
	a := r.st.Accounts["A"]
	if a == nil || a.Token == "" || a.SignedOut {
		return nil, errors.New("A isn't signed in: run signin A first")
	}
	return a, nil
}

func (r *run) machine() (string, error) {
	if r.st.MachineID == "" || len(r.st.SectionIDs) == 0 {
		return "", errors.New("no server yet: run server first")
	}
	return r.st.MachineID, nil
}

func as(a *account) who { return who{clientID: a.ClientID, token: a.Token} }

func ids(a *account) []string {
	var out []string
	for _, v := range []string{a.ID, a.UUID, a.Username, a.Email} {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

// --- HTTP ---

func platform() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	}
	return "Linux"
}

func (r *run) call(step string, w who, method, u string, body any) reply {
	rp := r.do(step, w, method, u, body)
	r.keep(rp)
	return rp
}

func (r *run) keep(rp reply) { r.pending = append(r.pending, rp.entry) }

// do makes one call and builds its log line without keeping it.
func (r *run) do(step string, w who, method, u string, body any) reply {
	var in []byte
	if body != nil {
		var err error
		if in, err = json.Marshal(body); err != nil {
			panic(err)
		}
	}
	rp := reply{}
	entry := map[string]any{"at": now(), "step": step, "method": method, "url": u, "request_body": string(in)}
	rp.entry = entry
	reader := io.Reader(http.NoBody)
	if in != nil {
		reader = bytes.NewReader(in)
	}
	req, err := http.NewRequestWithContext(r.ctx, method, u, reader)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range map[string]string{
		"X-Plex-Client-Identifier": w.clientID,
		"X-Plex-Product":           product,
		"X-Plex-Version":           productVer,
		"X-Plex-Platform":          platform(),
		"X-Plex-Device-Name":       product,
		"X-Plex-Token":             w.token,
	} {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
	headers := map[string]string{}
	for k := range req.Header {
		headers[k] = req.Header.Get(k)
	}
	entry["request_headers"] = headers
	start := time.Now()
	resp, err := r.http.Do(req)
	entry["ms"] = time.Since(start).Milliseconds()
	if err != nil {
		rp.err = err
		entry["error"] = err.Error()
		fmt.Printf("%-44s %s %s -> %v\n", step, method, r.redactor().str(u), err)
		return rp
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	rp.status, rp.raw = resp.StatusCode, raw
	entry["status"] = resp.StatusCode
	rh := map[string][]string{}
	for _, k := range []string{"Content-Type", "Retry-After"} {
		if v := resp.Header.Values(k); len(v) > 0 {
			rh[k] = v
		}
	}
	entry["response_headers"] = rh
	tree, kind := parse(raw)
	rp.tree = tree
	switch kind {
	case "json":
		entry["response_json"] = tree
	case "xml":
		entry["response_xml"] = tree
	case "text":
		entry["response_text"] = truncate(string(raw), 4000)
	}
	fmt.Printf("%-44s %s %s -> %d\n", step, method, r.redactor().str(u), resp.StatusCode)
	return rp
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

func (r *run) finding(step, name string, v any) {
	r.pending = append(r.pending, map[string]any{"at": now(), "finding": name, "step": step, "value": v})
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(r.redactor().value("", v))
	fmt.Printf("  finding %s: %s", name, b.String())
}

func (r *run) sleep(d time.Duration) error {
	select {
	case <-r.ctx.Done():
		return r.ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// errText is plex.tv's refusal in short: the errors array, an XML error,
// or the start of the body.
func errText(rp reply) string {
	if rp.ok() {
		return ""
	}
	if rp.err != nil {
		return rp.err.Error()
	}
	if m, ok := rp.tree.(map[string]any); ok {
		if errs, ok := m["errors"].([]any); ok && len(errs) > 0 {
			if e, ok := errs[0].(map[string]any); ok {
				return strings.TrimSpace(get(e, "code") + " " + get(e, "message"))
			}
		}
		if s := get(m, "error", "_text", "message"); s != "" {
			return s
		}
	}
	return truncate(strings.TrimSpace(string(rp.raw)), 300)
}

// --- bodies ---

func parse(raw []byte) (any, string) {
	t := bytes.TrimSpace(raw)
	if len(t) == 0 {
		return nil, "empty"
	}
	switch t[0] {
	case '{', '[':
		dec := json.NewDecoder(bytes.NewReader(t))
		dec.UseNumber()
		var v any
		if dec.Decode(&v) == nil {
			return v, "json"
		}
	case '<':
		if v, err := parseXML(t); err == nil && v != nil {
			return v, "xml"
		}
	}
	return nil, "text"
}

// parseXML turns XML into maps: "_tag", the attributes as keys, and
// "_children".
func parseXML(raw []byte) (map[string]any, error) {
	d := xml.NewDecoder(bytes.NewReader(raw))
	var stack []map[string]any
	var root map[string]any
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return root, nil
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := map[string]any{"_tag": t.Name.Local}
			for _, a := range t.Attr {
				n[a.Name.Local] = a.Value
			}
			if len(stack) > 0 {
				p := stack[len(stack)-1]
				ch, _ := p["_children"].([]any)
				p["_children"] = append(ch, n)
			} else {
				root = n
			}
			stack = append(stack, n)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if s := strings.TrimSpace(string(t)); s != "" && len(stack) > 0 {
				stack[len(stack)-1]["_text"] = s
			}
		}
	}
}

func sval(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		return strconv.FormatBool(t)
	}
	return ""
}

func get(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s := sval(m[k]); s != "" {
			return s
		}
	}
	return ""
}

func truthy(s string) bool { return s == "1" || s == "true" }

func sortedKeys(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// walk visits every map in v, parents first.
func walk(v any, fn func(map[string]any)) {
	switch t := v.(type) {
	case map[string]any:
		fn(t)
		for _, k := range sortedKeys(t) {
			walk(t[k], fn)
		}
	case []any:
		for _, x := range t {
			walk(x, fn)
		}
	}
}

func mapsIn(xs []any) []map[string]any {
	var out []map[string]any
	for _, x := range xs {
		if m, ok := x.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// items are a listing's entries: XML children, a JSON array, or the
// arrays inside a JSON object.
func items(v any) []map[string]any {
	switch t := v.(type) {
	case []any:
		return mapsIn(t)
	case map[string]any:
		if c, ok := t["_children"].([]any); ok {
			return mapsIn(c)
		}
		if mc, ok := t["MediaContainer"]; ok {
			return items(mc)
		}
		var out []map[string]any
		for _, k := range sortedKeys(t) {
			if arr, ok := t[k].([]any); ok {
				out = append(out, mapsIn(arr)...)
			}
		}
		return out
	}
	return nil
}

func set(vals ...string) map[string]bool {
	s := map[string]bool{}
	for _, v := range vals {
		if v != "" {
			s[v] = true
		}
	}
	return s
}

// has is whether any scalar under v is one of vals.
func has(v any, vals map[string]bool) bool {
	switch t := v.(type) {
	case map[string]any:
		for k, x := range t {
			if k != "_tag" && has(x, vals) {
				return true
			}
		}
	case []any:
		for _, x := range t {
			if has(x, vals) {
				return true
			}
		}
	default:
		s := sval(v)
		return s != "" && vals[s]
	}
	return false
}

func mentioning(tree any, vals ...string) []map[string]any {
	s := set(vals...)
	var out []map[string]any
	for _, it := range items(tree) {
		if has(it, s) {
			out = append(out, it)
		}
	}
	return out
}

// childIDs are the ids under m, as tag:id.
func childIDs(m map[string]any) []string {
	var out []string
	for _, k := range sortedKeys(m) {
		walk(m[k], func(c map[string]any) {
			if id := get(c, "id"); id != "" {
				out = append(out, get(c, "_tag")+":"+id)
			}
		})
	}
	return out
}

func tagged(tree any, tag string) []map[string]any {
	var out []map[string]any
	walk(tree, func(m map[string]any) {
		if get(m, "_tag") == tag {
			out = append(out, m)
		}
	})
	return out
}

// shareID is the shared server id in a share's response: a SharedServer
// element, or an object carrying the invitee.
func shareID(tree any) string {
	id := ""
	walk(tree, func(m map[string]any) {
		if id != "" {
			return
		}
		_, inv := m["invitedId"]
		_, uid := m["userID"]
		_, acc := m["acceptedAt"]
		if get(m, "_tag") == "SharedServer" || inv || uid || acc {
			id = get(m, "id")
		}
	})
	if m, ok := tree.(map[string]any); ok && id == "" {
		id = get(m, "id")
	}
	return id
}

// pendingIDs are the sharedServers ids in a Member's pending invites.
func pendingIDs(tree any) []string {
	var out []string
	walk(tree, func(m map[string]any) {
		if ss, ok := m["sharedServers"].([]any); ok {
			for _, s := range mapsIn(ss) {
				out = append(out, get(s, "id"))
			}
		}
	})
	return out
}

func findServer(tree any, mid string) map[string]any {
	var hit map[string]any
	walk(tree, func(m map[string]any) {
		if hit == nil && get(m, "clientIdentifier", "machineIdentifier") == mid {
			hit = m
		}
	})
	return hit
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func refused(status int) bool { return status == 401 || status == 403 }

// --- redaction ---

var piiKeys = set("email", "invitedEmail", "username", "friendlyName", "thumb", "avatar", "uuid",
	"authToken", "accessToken", "authenticationToken", "token", "publicAddress", "address", "host",
	"uri", "localAddresses", "clientIdentifier", "machineIdentifier", "pin", "qr", "homeEmail", "code")

type sweep struct {
	re    *regexp.Regexp
	label string
}

var sweeps = []sweep{
	{regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`), "<unlabelled-jwt>"},
	{regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`), "<email>"},
	{regexp.MustCompile(`[0-9a-f-]+\.[0-9a-f]{32}\.plex\.direct`), "<plex-direct-host>"},
	{regexp.MustCompile(`claim-[A-Za-z0-9_-]{12,}`), "<claim-token>"},
}

var ipv4 = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)

type redactor struct {
	exact map[string]string
	re    *regexp.Regexp
}

func isWord(b byte) bool {
	return b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

// newRedactor matches every secret in one pass, longest first, so a
// replacement is never matched again. Secrets under four characters only
// replace a whole value.
func newRedactor(secrets []secret) *redactor {
	rd := &redactor{exact: map[string]string{}}
	var alts []string
	sorted := append([]secret(nil), secrets...)
	sort.SliceStable(sorted, func(i, j int) bool { return len(sorted[i].Value) > len(sorted[j].Value) })
	for _, s := range sorted {
		rd.exact[s.Value] = s.Label
		if len(s.Value) < 4 {
			continue
		}
		p := regexp.QuoteMeta(s.Value)
		if isWord(s.Value[0]) {
			p = `\b` + p
		}
		if isWord(s.Value[len(s.Value)-1]) {
			p += `\b`
		}
		alts = append(alts, p)
	}
	if len(alts) > 0 {
		rd.re = regexp.MustCompile(strings.Join(alts, "|"))
	}
	return rd
}

func (rd *redactor) str(s string) string {
	if l, ok := rd.exact[s]; ok {
		return l
	}
	if rd.re != nil {
		s = rd.re.ReplaceAllStringFunc(s, func(m string) string { return rd.exact[m] })
	}
	for _, sw := range sweeps {
		s = sw.re.ReplaceAllString(s, sw.label)
	}
	return ipv4.ReplaceAllStringFunc(s, func(m string) string {
		if m == "127.0.0.1" || m == "0.0.0.0" {
			return m
		}
		return "<ipv4>"
	})
}

func isLabel(s string) bool {
	return strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">") && !strings.Contains(s, " ")
}

func (rd *redactor) value(key string, v any) any {
	if key == "location" {
		return "<removed>"
	}
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, x := range t {
			out[k] = rd.value(k, x)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = rd.value(key, x)
		}
		return out
	case []string:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = rd.value(key, x)
		}
		return out
	case string:
		s := rd.str(t)
		if piiKeys[key] && s != "" && !isLabel(s) {
			return "<string>"
		}
		return s
	case json.Number:
		if l, ok := rd.exact[t.String()]; ok {
			return l
		}
		return t
	}
	return v
}

// shape keeps a field's type and drops its value, as the JWT run did for
// /api/v2/user, with subscription kept as returned.
func (rd *redactor) shape(key string, v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, x := range t {
			if k == "subscription" {
				out[k] = rd.value(k, x)
			} else {
				out[k] = rd.shape(k, x)
			}
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = rd.shape(key, x)
		}
		return out
	case string:
		return "<string>"
	case json.Number:
		return 0
	}
	return v
}

func (rd *redactor) entry(e map[string]any) map[string]any {
	out := map[string]any{}
	u, _ := e["url"].(string)
	userBody := false
	if pu, err := url.Parse(u); err == nil {
		userBody = strings.HasSuffix(pu.Path, "/api/v2/user")
	}
	for k, v := range e {
		switch k {
		case "url", "error", "response_text":
			out[k] = rd.str(v.(string))
		case "request_headers":
			h := map[string]string{}
			for hk, hv := range v.(map[string]string) {
				h[hk] = rd.str(hv)
			}
			out[k] = h
		case "request_body":
			s := v.(string)
			if tree, kind := parse([]byte(s)); kind == "json" {
				var b bytes.Buffer
				enc := json.NewEncoder(&b)
				enc.SetEscapeHTML(false)
				_ = enc.Encode(rd.value("", tree))
				s = strings.TrimSpace(b.String())
			} else {
				s = rd.str(s)
			}
			out[k] = s
		case "response_json", "response_xml":
			if userBody {
				out[k] = rd.shape("", v)
			} else {
				out[k] = rd.value("", v)
			}
		case "value":
			out[k] = rd.value("", v)
		default:
			out[k] = v
		}
	}
	return out
}

// --- the JWT device key, as internal/providers/plextv signs it ---

func jwk(key ed25519.PrivateKey) map[string]string {
	pub, _ := key.Public().(ed25519.PublicKey)
	x := base64.RawURLEncoding.EncodeToString(pub)
	sum := sha256.Sum256([]byte(`{"crv":"Ed25519","kty":"OKP","x":"` + x + `"}`))
	kid := base64.RawURLEncoding.EncodeToString(sum[:])
	return map[string]string{"kty": "OKP", "crv": "Ed25519", "x": x, "kid": kid, "alg": "EdDSA", "use": "sig"}
}

func deviceJWT(key ed25519.PrivateKey, clientID string) string {
	n := time.Now()
	h, _ := json.Marshal(map[string]string{"alg": "EdDSA", "kid": jwk(key)["kid"], "typ": "JWT"})
	p, _ := json.Marshal(map[string]any{"aud": "plex.tv", "iss": clientID, "iat": n.Unix(), "exp": n.Add(5 * time.Minute).Unix()})
	in := base64.RawURLEncoding.EncodeToString(h) + "." + base64.RawURLEncoding.EncodeToString(p)
	return in + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, []byte(in)))
}

func isJWT(t string) bool { return strings.Count(t, ".") == 2 && strings.HasPrefix(t, "eyJ") }

// --- commands ---

func cmdInit(r *run, _ []string) error {
	a := r.slot("A")
	if len(a.KeySeed) == 0 {
		seed := make([]byte, ed25519.SeedSize)
		_, _ = rand.Read(seed)
		a.KeySeed = seed
	}
	fmt.Printf("state: %s\nlog:   %s\n", filepath.Join(r.dir, "state.json"), filepath.Join(r.dir, logName))
	fmt.Println("A has a device key and its own client identifier. Next: import")
	return nil
}

func printAuth(as, clientID, code string) {
	q := url.Values{"clientID": {clientID}, "code": {code}, "context[device][product]": {product}}
	fmt.Printf("\n>>> Ask Jackson: open this in a NEW private window (close every other private window first) and sign in as %s:\n\n    %s\n\n", as, authPage+"#?"+q.Encode())
}

func cmdSignin(r *run, args []string) error {
	if len(args) == 0 {
		return errors.New("signin needs A, B, C, D or E")
	}
	w := strings.ToUpper(args[0])
	fs := flag.NewFlagSet("signin", flag.ExitOnError)
	host := fs.String("host", "clients", "clients (clients.plex.tv) or plex (plex.tv), Members only")
	again := fs.Bool("again", false, "sign in again even if the slot has a token")
	_ = fs.Parse(args[1:])
	if w == "A" {
		return r.signinAdmin(*again)
	}
	if len(w) != 1 || !strings.Contains(members, w) {
		return fmt.Errorf("unknown account %q", w)
	}
	switch *host {
	case "clients":
		return r.signinMember(w, w, apiHost, *again)
	case "plex":
		return r.signinMember(w, w+"2", webHost, *again)
	}
	return fmt.Errorf("-host is clients or plex, not %q", *host)
}

func (r *run) signinAdmin(again bool) error {
	a := r.slot("A")
	if len(a.KeySeed) == 0 {
		return errors.New("no device key: run init first")
	}
	if a.Token != "" && !a.SignedOut && !again {
		fmt.Println("A is already signed in (use -again to replace the token)")
		return nil
	}
	key := ed25519.NewKeyFromSeed(a.KeySeed)
	step := "signin-A"
	rp := r.call(step+":pin-create", who{clientID: a.ClientID}, "POST", apiHost+"/api/v2/pins", map[string]any{"jwk": jwk(key), "strong": true})
	m, _ := rp.tree.(map[string]any)
	if !rp.ok() || m == nil {
		return fmt.Errorf("JWT PIN refused: %d %s", rp.status, errText(rp))
	}
	id, code := get(m, "id"), get(m, "code")
	r.label(code, "pin-code-A")
	r.finding(step, "pin", map[string]any{"kind": "jwt", "via": "clients.plex.tv", "id": id, "code_len": len(code), "expires_in": get(m, "expiresIn")})
	printAuth("A (the throwaway Admin)", a.ClientID, code)
	tok, err := r.poll(step, a, apiHost, id, func() string { return deviceJWT(key, a.ClientID) })
	if err != nil {
		return err
	}
	a.Token, a.TokenAt, a.SignedOut, a.PinHost = tok, time.Now().UTC(), false, "clients.plex.tv"
	r.label(tok, "token-A")
	r.finding(step, "token", map[string]any{"slot": "A", "is_jwt": isJWT(tok), "len": len(tok)})
	return r.readUser(step, a, apiHost, true)
}

func (r *run) signinMember(w, slotName, base string, again bool) error {
	a := r.slot(slotName)
	a.Who = w
	if a.Token != "" && !a.SignedOut && !again {
		fmt.Printf("%s is already signed in (use -again to replace the token)\n", slotName)
		return nil
	}
	hostName := strings.TrimPrefix(base, "https://")
	step := "signin-" + slotName
	rp := r.call(step+":pin-create", who{clientID: a.ClientID}, "POST", base+"/api/v2/pins?strong=true", nil)
	m, _ := rp.tree.(map[string]any)
	r.finding(step, "legacy-pin-create", map[string]any{"via": hostName, "status": rp.status, "error": errText(rp)})
	if !rp.ok() || m == nil {
		return fmt.Errorf("legacy PIN refused on %s: %d %s (try signin %s -host plex)", hostName, rp.status, errText(rp), w)
	}
	id, code := get(m, "id"), get(m, "code")
	r.label(code, "pin-code-"+slotName)
	r.finding(step, "pin", map[string]any{"kind": "legacy", "via": hostName, "id": id, "code_len": len(code), "expires_in": get(m, "expiresIn")})
	printAuth(w, a.ClientID, code)
	tok, err := r.poll(step, a, base, id, nil)
	if err != nil {
		return err
	}
	a.Token, a.TokenAt, a.SignedOut, a.PinHost = tok, time.Now().UTC(), false, hostName
	r.label(tok, "token-"+slotName)
	r.finding(step, "token", map[string]any{"slot": slotName, "pin_host": hostName, "is_jwt": isJWT(tok), "len": len(tok)})
	return r.readUser(step, a, base, true)
}

// poll waits for the PIN's authToken, logging the first poll, each change
// of status and the last.
func (r *run) poll(step string, a *account, base, id string, device func() string) (string, error) {
	wait, last, n := 2*time.Second, -1, 0
	start := time.Now()
	for time.Since(start) < 30*time.Minute {
		u := base + "/api/v2/pins/" + id
		if device != nil {
			u += "?deviceJWT=" + device()
		}
		rp := r.do(step+":pin-poll", who{clientID: a.ClientID}, "GET", u, nil)
		n++
		tok := ""
		if m, ok := rp.tree.(map[string]any); ok {
			tok = get(m, "authToken")
		}
		if tok != "" {
			r.label(tok, "token-"+a.Slot)
		}
		if rp.status != last || tok != "" {
			r.keep(rp)
		}
		last = rp.status
		if tok != "" {
			r.finding(step, "pin-polls", map[string]any{"polls": n, "seconds": int(time.Since(start).Seconds())})
			return tok, nil
		}
		switch {
		case rp.status == http.StatusTooManyRequests:
			wait = min(wait*2, 30*time.Second)
		case rp.status == http.StatusNotFound:
			return "", errors.New("the PIN expired or is gone: run signin again")
		case errors.Is(rp.err, context.Canceled):
			return "", rp.err
		}
		if n%15 == 0 {
			fmt.Printf("  still waiting for approval (%ds)\n", int(time.Since(start).Seconds()))
		}
		if err := r.sleep(wait); err != nil {
			return "", err
		}
	}
	return "", errors.New("no approval within 30 minutes")
}

// readUser reads the account behind a's token. A token approved as an
// account another slot holds is refused, and signed out when this run
// made it.
func (r *run) readUser(step string, a *account, base string, ours bool) error {
	rp := r.call(step+":user", as(a), "GET", base+"/api/v2/user", nil)
	m, _ := rp.tree.(map[string]any)
	if !rp.ok() || m == nil {
		return fmt.Errorf("/api/v2/user refused the new token: %d %s", rp.status, errText(rp))
	}
	id := get(m, "id")
	for _, o := range r.st.Accounts {
		if o.ID == "" || o == a {
			continue
		}
		if (o.Who == a.Who) != (o.ID == id) {
			if !ours {
				return fmt.Errorf("slot %s's token belongs to the account of slot %s (%s): fix accounts.json", a.Slot, o.Slot, o.Who)
			}
			r.signout(step, a)
			a.Token = ""
			return fmt.Errorf("the PIN was approved as the wrong account (slot %s is %s, this sign-in is %s's): the token is signed out, run signin again in a fresh private window", o.Slot, o.Who, a.Who)
		}
	}
	a.ID, a.UUID, a.Username, a.Email = id, get(m, "uuid"), get(m, "username"), get(m, "email")
	a.Title = get(m, "title")
	sub, _ := m["subscription"].(map[string]any)
	a.PlexPass = sub != nil && truthy(get(sub, "active"))
	r.label(a.ID, "id-"+a.Who)
	r.label(a.UUID, "uuid-"+a.Who)
	r.label(a.Email, "email-"+a.Who)
	r.label(a.Username, "username-"+a.Who)
	r.label(a.Title, "title-"+a.Who)
	r.label(get(m, "friendlyName"), "friendly-name-"+a.Who)
	_, hasConfirmed := m["confirmed"]
	_, hasVerified := m["emailVerified"]
	r.finding(step, "account", map[string]any{
		"slot": a.Slot, "plex_pass": a.PlexPass, "has_confirmed": hasConfirmed, "confirmed": get(m, "confirmed"),
		"has_emailVerified": hasVerified, "emailVerified": get(m, "emailVerified"),
	})
	fmt.Printf("%s signed in as %s\n", a.Slot, r.redactor().str(a.Username))
	return nil
}

// signout signs a's token out and answers /api/v2/user's status after.
func (r *run) signout(step string, a *account) int {
	rp := r.call(step+":signout-"+a.Slot, as(a), "DELETE", apiHost+"/api/v2/users/signout", nil)
	if rp.status == http.StatusNotFound {
		rp = r.call(step+":signout-"+a.Slot+" on plex.tv", as(a), "DELETE", webHost+"/api/v2/users/signout", nil)
	}
	chk := r.call(step+":user-after-signout-"+a.Slot, as(a), "GET", apiHost+"/api/v2/user", nil)
	a.SignedOut = chk.status == http.StatusUnauthorized
	r.finding(step, "signout", map[string]any{"slot": a.Slot, "status": rp.status, "user_after": chk.status})
	return chk.status
}

func cmdMedia(r *run, _ []string) error {
	dir := filepath.Join(r.dir, "media", "clips")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if ff, err := exec.LookPath("ffmpeg"); err == nil {
		for i, src := range []string{"testsrc=duration=6:size=640x360:rate=24", "smptebars=duration=6:size=640x360:rate=24"} {
			out := filepath.Join(dir, fmt.Sprintf("Clip %d.mp4", i+1))
			cmd := exec.Command(ff, "-y", "-loglevel", "error", "-f", "lavfi", "-i", src, "-f", "lavfi", "-i", "sine=frequency=440:duration=6",
				"-c:v", "mpeg4", "-q:v", "5", "-c:a", "aac", "-shortest", out)
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("ffmpeg: %w", err)
			}
		}
		fmt.Printf("made two 6-second clips with ffmpeg in %s\n", dir)
		return nil
	}
	// Blender Foundation trailers, CC BY 3.0. URLs from memory: if one 404s,
	// install ffmpeg (brew install ffmpeg) and run media again.
	for name, u := range map[string]string{
		"Sintel trailer.mp4":         "https://download.blender.org/durian/trailer/sintel_trailer-480p.mp4",
		"Big Buck Bunny trailer.mov": "https://download.blender.org/peach/trailer/trailer_480p.mov",
	} {
		req, _ := http.NewRequestWithContext(r.ctx, "GET", u, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			_ = resp.Body.Close()
			return fmt.Errorf("%s answered %d: install ffmpeg and run media again", u, resp.StatusCode)
		}
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			_ = resp.Body.Close()
			return err
		}
		_, err = io.Copy(f, resp.Body)
		_ = resp.Body.Close()
		_ = f.Close()
		if err != nil {
			return err
		}
	}
	fmt.Printf("downloaded two Creative Commons trailers into %s\n", dir)
	return nil
}

func cmdClaim(r *run, _ []string) error {
	a, err := r.admin()
	if err != nil {
		return err
	}
	rp := r.call("claim:token", as(a), "GET", webHost+"/api/claim/token.json", nil)
	tok := ""
	if m, ok := rp.tree.(map[string]any); ok {
		tok = get(m, "token")
	}
	r.label(tok, "claim-token")
	r.finding("claim", "claim-with-jwt", map[string]any{"status": rp.status, "got_token": tok != "", "error": errText(rp)})
	if tok == "" {
		tok = "<paste the code from https://plex.tv/claim, signed in as A>"
		fmt.Println("plex.tv gave no claim token for A's JWT. Get one by hand at https://plex.tv/claim in A's private window.")
	}
	clips := filepath.Join(r.dir, "media", "clips")
	fmt.Printf(`
The claim token lasts 4 minutes. Run now (Colima running: colima start):

docker rm -f %[1]s 2>/dev/null; docker volume rm %[1]s-config %[1]s-transcode 2>/dev/null
docker run -d --name %[1]s --hostname %[1]s \
  -p 32400:32400/tcp -e TZ=UTC -e PLEX_CLAIM='%[2]s' \
  -v %[1]s-config:/config -v %[1]s-transcode:/transcode \
  -v '%[3]s:/data:ro' \
  plexinc/pms-docker:latest
docker logs -f %[1]s   # Ctrl-C once it says the server started

Then: ./rec server
`, container, tok, clips)
	return nil
}

func versionAtLeast(v string, major, minor int) bool {
	p := strings.SplitN(v, ".", 3)
	if len(p) < 2 {
		return false
	}
	a, _ := strconv.Atoi(p[0])
	b, _ := strconv.Atoi(p[1])
	return a > major || a == major && b >= minor
}

func cmdServer(r *run, _ []string) error {
	a, err := r.admin()
	if err != nil {
		return err
	}
	var srv map[string]any
	for i := 0; i < 18 && srv == nil; i++ {
		rp := r.do("server:resources", as(a), "GET", apiHost+resourcesQ, nil)
		for _, it := range items(rp.tree) {
			if strings.Contains(get(it, "provides"), "server") && truthy(get(it, "owned")) && get(it, "name") == container {
				if srv == nil || truthy(get(it, "presence")) {
					srv = it
				}
			}
		}
		if srv != nil || i == 0 {
			r.keep(rp)
		}
		if srv == nil {
			fmt.Println("  plex.tv doesn't list the server yet, trying again in 10 s")
			if err := r.sleep(10 * time.Second); err != nil {
				return err
			}
		}
	}
	if srv == nil {
		return fmt.Errorf("plex.tv lists no server named %s for A after 3 minutes: check docker logs %s", container, container)
	}
	r.st.MachineID = get(srv, "clientIdentifier")
	r.st.ServerToken = get(srv, "accessToken")
	r.st.ServerVersion = get(srv, "productVersion")
	r.label(r.st.MachineID, "machine-id")
	r.label(r.st.ServerToken, "server-token")
	r.label(get(srv, "publicAddress"), "public-address")
	ok := versionAtLeast(r.st.ServerVersion, 1, 43)
	r.finding("server", "server", map[string]any{"version": r.st.ServerVersion, "at_least_1_43": ok})
	if !ok {
		return fmt.Errorf("server %s is older than the 1.43 floor: pull a newer plexinc/pms-docker", r.st.ServerVersion)
	}
	srvWho := who{clientID: a.ClientID, token: r.st.ServerToken}
	id := r.call("server:identity", srvWho, "GET", pmsHost+"/identity", nil)
	local := ""
	walk(id.tree, func(m map[string]any) {
		if local == "" {
			local = get(m, "machineIdentifier")
		}
	})
	r.finding("server", "local-identity", map[string]any{"status": id.status, "same_machine": local == r.st.MachineID})
	key := r.sectionKey(srvWho)
	if key == "" {
		q := url.Values{"name": {sectionTitle}, "type": {"movie"}, "agent": {"tv.plex.agents.none"},
			"scanner": {"Plex Video Files Scanner"}, "language": {"xn"}, "location": {"/data"}}
		mk := r.call("server:section-create", srvWho, "POST", pmsHost+"/library/sections?"+q.Encode(), nil)
		r.finding("server", "section-create", map[string]any{"status": mk.status, "error": errText(mk)})
		key = r.sectionKey(srvWho)
	}
	if key == "" {
		return fmt.Errorf("no %q section: make it by hand at %s/web (Other Videos, folder /data) and run server again", sectionTitle, pmsHost)
	}
	r.st.SectionKey = key
	r.call("server:section-refresh", srvWho, "GET", pmsHost+"/library/sections/"+key+"/refresh", nil)
	n := 0
	for i := 0; i < 12 && n < 2; i++ {
		rp := r.do("server:section-items", srvWho, "GET", pmsHost+"/library/sections/"+key+"/all", nil)
		n = 0
		walk(rp.tree, func(m map[string]any) {
			if get(m, "ratingKey") != "" {
				n++
			}
		})
		if n >= 2 || i == 0 {
			r.keep(rp)
		}
		if n < 2 {
			if err := r.sleep(5 * time.Second); err != nil {
				return err
			}
		}
	}
	r.finding("server", "section-items", map[string]any{"items": n})
	return r.plexSectionIDs(a)
}

func (r *run) sectionKey(w who) string {
	rp := r.call("server:sections", w, "GET", pmsHost+"/library/sections", nil)
	key := ""
	walk(rp.tree, func(m map[string]any) {
		if key == "" && get(m, "title") == sectionTitle {
			key = get(m, "key")
		}
	})
	return key
}

// plexSectionIDs waits for plex.tv to list the section, by its plex.tv id.
func (r *run) plexSectionIDs(a *account) error {
	u := webHost + "/api/servers/" + r.st.MachineID
	last := -1
	for i := 0; i < 18; i++ {
		rp := r.do("server:plex-tv-sections", as(a), "GET", u, nil)
		pid := ""
		walk(rp.tree, func(m map[string]any) {
			if pid == "" && get(m, "title") == sectionTitle {
				pid = get(m, "id")
			}
		})
		if rp.status != last || pid != "" {
			r.keep(rp)
		}
		last = rp.status
		if refused(rp.status) {
			r.finding("server", "stop-rule", map[string]any{"route": u, "status": rp.status, "error": errText(rp)})
			return stopRule{fmt.Sprintf("plex.tv refused A's JWT on the section route (%d)", rp.status)}
		}
		if pid != "" {
			r.st.SectionIDs = []string{pid}
			r.finding("server", "plex-section", map[string]any{"plex_tv_id": pid, "server_key": r.st.SectionKey, "status": rp.status})
			return nil
		}
		fmt.Println("  plex.tv doesn't list the section yet, trying again in 10 s")
		if err := r.sleep(10 * time.Second); err != nil {
			return err
		}
	}
	return fmt.Errorf("plex.tv lists no %q section after 3 minutes", sectionTitle)
}

// listing reads one list as w and records which of its entries mention m,
// and whether those carry share.
func (r *run) listing(step, name string, w who, u string, m *account, share string) (reply, []map[string]any) {
	rp := r.call(step+":"+name, w, "GET", u, nil)
	hits := mentioning(rp.tree, ids(m)...)
	var matches []any
	for _, h := range hits {
		matches = append(matches, map[string]any{
			"tag": get(h, "_tag"), "id": get(h, "id"), "child_ids": childIDs(h),
			"has_share_id": share != "" && has(h, set(share)),
			"pending":      get(h, "pending"), "home": get(h, "home"), "accepted_at": get(h, "acceptedAt"),
		})
	}
	entries := 0
	if rp.ok() {
		entries = len(items(rp.tree))
	}
	r.finding(step+":"+name, "listing", map[string]any{
		"url": u, "status": rp.status, "entries": entries, "mentions": m.Who, "matched": len(hits) > 0, "matches": matches,
	})
	return rp, hits
}

func (r *run) share(step string, a, m *account, base string) reply {
	sections := make([]any, 0, len(r.st.SectionIDs))
	for _, s := range r.st.SectionIDs {
		sections = append(sections, json.Number(s))
	}
	body := map[string]any{
		"server_id":        r.st.MachineID,
		"shared_server":    map[string]any{"library_section_ids": sections, "invited_id": json.Number(m.ID)},
		"sharing_settings": map[string]any{},
	}
	rp := r.call(step, as(a), "POST", base+"/api/servers/"+r.st.MachineID+"/shared_servers", body)
	id := ""
	if rp.ok() {
		id = shareID(rp.tree)
		if _, had := r.st.Shares[m.Who]; !had && id != "" {
			r.st.Shares[m.Who] = id
		}
		r.label(id, "share-"+m.Who)
	}
	r.finding(step, "share", map[string]any{"to": m.Who, "via": strings.TrimPrefix(base, "https://"), "status": rp.status, "id": id, "error": errText(rp)})
	return rp
}

func (r *run) resources(step string, m *account) {
	rp := r.call(step+":resources-"+m.Slot, as(m), "GET", apiHost+resourcesQ, nil)
	srv := findServer(rp.tree, r.st.MachineID)
	v := map[string]any{"member": m.Who, "status": rp.status, "lists_server": srv != nil}
	if srv != nil {
		v["owned"] = get(srv, "owned")
		r.label(get(srv, "accessToken"), "server-token-"+m.Who)
	}
	r.finding(step, "member-sees-server", v)
}

func cmdPrecheck(r *run, _ []string) error {
	a, err := r.admin()
	if err != nil {
		return err
	}
	c, err := r.member("C")
	if err != nil {
		return err
	}
	d, err := r.member("D")
	if err != nil {
		return err
	}
	r.listing("precheck", "users on plex.tv", as(a), webHost+"/api/users/", c, "")
	r.listing("precheck", "friends-v2 on clients", as(a), apiHost+"/api/v2/friends", c, "")
	r.listing("precheck", "invites-requested on plex.tv", as(a), webHost+"/api/invites/requested", d, "")
	r.listing("precheck", "home-users on plex.tv", as(a), webHost+"/api/home/users", d, "")
	r.listing("precheck", "pending-v2 on clients(D)", as(d), apiHost+pendingPath, a, "")
	r.listing("precheck", "invites-requests-v1 on plex.tv(D)", as(d), webHost+"/api/invites/requests", a, "")
	fmt.Println("\nC must show as matched in users or friends, D in invites-requested or home-users. If not, redo the pre-steps.")
	return nil
}

func cmdQ1(r *run, _ []string) error {
	a, err := r.admin()
	if err != nil {
		return err
	}
	mid, err := r.machine()
	if err != nil {
		return err
	}
	b, err := r.member("B")
	if err != nil {
		return err
	}
	routes := []struct{ name, path string }{
		{"user", "/api/v2/user"},
		{"server", "/api/servers/" + mid},
		{"server-v2", "/api/v2/servers/" + mid},
		{"shared-servers", "/api/servers/" + mid + "/shared_servers"},
		{"users", "/api/users/"},
		{"home-users", "/api/home/users"},
		{"invites-requested", "/api/invites/requested"},
		{"owned-accepted", "/api/v2/shared_servers/owned/accepted"},
		{"owned-pending", "/api/v2/shared_servers/owned/pending"},
		{"friends-v2", "/api/v2/friends"},
	}
	stop := ""
	for _, rt := range routes {
		for _, h := range []struct{ name, base string }{{"plex.tv", webHost}, {"clients.plex.tv", apiHost}} {
			step := "q1:" + rt.name + " on " + h.name
			rp := r.call(step, as(a), "GET", h.base+rt.path, nil)
			_, kind := parse(rp.raw)
			r.finding(step, "jwt-route", map[string]any{"route": rt.path, "via": h.name, "method": "GET", "status": rp.status, "body": kind, "error": errText(rp)})
			if h.name == "plex.tv" && (rt.name == "server" || rt.name == "shared-servers") && refused(rp.status) {
				stop = fmt.Sprintf("GET plex.tv%s answered %d to A's JWT", rt.path, rp.status)
			}
		}
	}
	if r.st.Shares["B"] != "" {
		fmt.Println("B already has a share from an earlier q1: not sharing again")
	} else {
		rp := r.share("q1:share-B on plex.tv", a, b, webHost)
		if refused(rp.status) {
			stop = fmt.Sprintf("POST plex.tv shared_servers answered %d to A's JWT", rp.status)
		}
	}
	if stop != "" {
		r.finding("q1", "stop-rule", map[string]any{"why": stop})
		return stopRule{stop}
	}
	fmt.Println("\nB's share stays pending for q2. Next: q2")
	return nil
}

func cmdQ2(r *run, _ []string) error {
	a, err := r.admin()
	if err != nil {
		return err
	}
	mid, err := r.machine()
	if err != nil {
		return err
	}
	b, err := r.member("B")
	if err != nil {
		return err
	}
	share := r.st.Shares["B"]
	if share == "" {
		return errors.New("B has no share yet: run q1 first")
	}
	for _, l := range []struct{ name, u string }{
		{"shared-servers-v1 on plex.tv", webHost + "/api/servers/" + mid + "/shared_servers"},
		{"shared-servers-v1 on clients", apiHost + "/api/servers/" + mid + "/shared_servers"},
		{"invites-requested on plex.tv", webHost + "/api/invites/requested"},
		{"invites-requested on clients", apiHost + "/api/invites/requested"},
		{"users on plex.tv", webHost + "/api/users/"},
		{"owned-pending on clients", apiHost + "/api/v2/shared_servers/owned/pending"},
		{"owned-accepted on clients", apiHost + "/api/v2/shared_servers/owned/accepted"},
	} {
		r.listing("q2", l.name, as(a), l.u, b, share)
	}
	fmt.Println("\nThe listing whose match has has_share_id true answers question 2. Next: q3")
	return nil
}

func cmdQ3(r *run, _ []string) error {
	var slots []*account
	for _, s := range []string{"AL", "B", "B2", "C", "C2", "D", "D2", "E", "E2"} {
		if a := r.st.Accounts[s]; a != nil && a.Token != "" && !a.SignedOut {
			slots = append(slots, a)
		}
	}
	if len(slots) == 0 {
		return errors.New("no Member is signed in")
	}
	for _, a := range slots {
		v := map[string]any{"slot": a.Slot, "pin_host": a.PinHost, "is_jwt": isJWT(a.Token), "len": len(a.Token)}
		for _, h := range []struct{ name, base string }{{"plex.tv", webHost}, {"clients.plex.tv", apiHost}} {
			rp := r.call("q3:user-"+a.Slot+" on "+h.name, as(a), "GET", h.base+"/api/v2/user", nil)
			v["user_on_"+h.name] = rp.status
		}
		r.finding("q3", "legacy-token", v)
	}
	if b, b2 := r.st.Accounts["B"], r.st.Accounts["B2"]; b != nil && b2 != nil && b.ID != "" && b2.ID != "" {
		r.finding("q3", "b-on-both-hosts", map[string]any{"same_account": b.ID == b2.ID, "distinct_tokens": b.Token != b2.Token})
	}
	return nil
}

func cmdQ4(r *run, _ []string) error {
	a, err := r.admin()
	if err != nil {
		return err
	}
	mid, err := r.machine()
	if err != nil {
		return err
	}
	b, err := r.member("B")
	if err != nil {
		return err
	}
	share := r.st.Shares["B"]
	if share == "" {
		return errors.New("B has no share yet: run q1 first")
	}
	pend := r.call("q4:pending on clients", as(b), "GET", apiHost+pendingPath, nil)
	other := r.call("q4:pending on plex.tv", as(b), "GET", webHost+pendingPath, nil)
	r.finding("q4", "pending-hosts", map[string]any{"clients.plex.tv": pend.status, "plex.tv": other.status})
	pids := pendingIDs(pend.tree)
	r.finding("q4", "pending-id", map[string]any{"share_id": share, "pending_ids": pids, "equal": contains(pids, share)})
	tries := []string{share}
	for _, p := range pids {
		if p != share {
			tries = append(tries, p)
		}
	}
	accepted := false
	for _, id := range tries {
		acc := r.call("q4:accept", as(b), "POST", apiHost+"/api/v2/shared_servers/"+id+"/accept", nil)
		r.finding("q4", "accept", map[string]any{"id": id, "is_share_id": id == share, "status": acc.status, "error": errText(acc)})
		if acc.ok() {
			accepted = true
			break
		}
	}
	after := r.call("q4:pending-after on clients", as(b), "GET", apiHost+pendingPath, nil)
	r.finding("q4", "pending-after-accept", map[string]any{"status": after.status, "pending_ids": pendingIDs(after.tree)})
	r.listing("q4", "owned-accepted on clients", as(a), apiHost+"/api/v2/shared_servers/owned/accepted", b, share)
	r.listing("q4", "shared-servers-v1 on plex.tv", as(a), webHost+"/api/servers/"+mid+"/shared_servers", b, share)
	r.resources("q4", b)
	if !accepted {
		return errors.New("no accept succeeded: see the findings, then carry on with q5")
	}
	fmt.Println("\nNext: q5")
	return nil
}

func cmdQ5(r *run, _ []string) error {
	a, err := r.admin()
	if err != nil {
		return err
	}
	mid, err := r.machine()
	if err != nil {
		return err
	}
	var ms []*account
	for _, w := range "BCD" {
		m, err := r.member(string(w))
		if err != nil {
			return err
		}
		ms = append(ms, m)
	}
	b, c, d := ms[0], ms[1], ms[2]
	r.listing("q5", "users-before on plex.tv(C)", as(a), webHost+"/api/users/", c, "")
	r.share("q5:second-share-B", a, b, webHost)
	r.share("q5:share-C-friend-no-share", a, c, webHost)
	r.share("q5:share-D-pending-home-invitee", a, d, webHost)
	r.listing("q5", "shared-servers-v1 on plex.tv(C)", as(a), webHost+"/api/servers/"+mid+"/shared_servers", c, r.st.Shares["C"])
	r.listing("q5", "shared-servers-v1 on plex.tv(D)", as(a), webHost+"/api/servers/"+mid+"/shared_servers", d, r.st.Shares["D"])
	r.listing("q5", "users on plex.tv(D)", as(a), webHost+"/api/users/", d, r.st.Shares["D"])
	fmt.Println("\nC's share stays pending for q7. Next: q6")
	return nil
}

func (r *run) homeInvite(step string, a, m *account) reply {
	rp := r.call(step+"-by-username", as(a), "POST", webHost+"/api/home/users?invitedEmail="+url.QueryEscape(m.Username), nil)
	by := "username"
	if !rp.ok() {
		rp = r.call(step+"-by-email", as(a), "POST", webHost+"/api/home/users?invitedEmail="+url.QueryEscape(m.Email), nil)
		by = "email"
	}
	r.finding(step, "home-invite", map[string]any{"to": m.Who, "by": by, "status": rp.status, "error": errText(rp), "admin_plex_pass": a.PlexPass})
	return rp
}

func cmdQ6(r *run, _ []string) error {
	a, err := r.admin()
	if err != nil {
		return err
	}
	if _, err := r.machine(); err != nil {
		return err
	}
	e, err := r.member("E")
	if err != nil {
		return err
	}
	d, err := r.member("D")
	if err != nil {
		return err
	}
	r.finding("q6", "admin-plex-pass", map[string]any{"plex_pass": a.PlexPass})
	r.listing("q6", "home-users-before on plex.tv", as(a), webHost+"/api/home/users", e, "")
	r.homeInvite("q6:home-invite-E", a, e)
	r.listing("q6", "home-users-after-invite on plex.tv", as(a), webHost+"/api/home/users", e, "")
	r.listing("q6", "invites-requested on plex.tv(E)", as(a), webHost+"/api/invites/requested", e, "")
	for _, m := range []*account{e, d} {
		r.listing("q6", "pending-v2 on clients("+m.Who+")", as(m), apiHost+pendingPath, a, "")
		r.listing("q6", "invites-requests-v1 on plex.tv("+m.Who+")", as(m), webHost+"/api/invites/requests", a, "")
	}
	r.share("q6:share-E-home", a, e, webHost)
	pend := r.call("q6:pending-E-after-share", as(e), "GET", apiHost+pendingPath, nil)
	pids := pendingIDs(pend.tree)
	r.finding("q6", "home-share-pending", map[string]any{"pending_ids": pids, "share_id": r.st.Shares["E"], "equal": contains(pids, r.st.Shares["E"])})
	for _, id := range pids {
		acc := r.call("q6:accept-E", as(e), "POST", apiHost+"/api/v2/shared_servers/"+id+"/accept", nil)
		r.finding("q6", "accept", map[string]any{"id": id, "status": acc.status, "error": errText(acc)})
	}
	reqs := r.call("q6:invites-requests-E-after-accept", as(e), "GET", webHost+"/api/invites/requests", nil)
	for _, inv := range tagged(reqs.tree, "Invite") {
		if !truthy(get(inv, "home")) {
			continue
		}
		q := url.Values{"friend": {get(inv, "friend")}, "server": {get(inv, "server")}, "home": {get(inv, "home")}}
		acc := r.call("q6:accept-home-v1-E", as(e), "PUT", webHost+"/api/invites/requests/"+get(inv, "id")+"?"+q.Encode(), nil)
		r.finding("q6", "accept-home-v1", map[string]any{"status": acc.status, "error": errText(acc)})
	}
	r.listing("q6", "home-users-final on plex.tv", as(a), webHost+"/api/home/users", e, "")
	r.listing("q6", "invites-requested-final on plex.tv(E)", as(a), webHost+"/api/invites/requested", e, "")
	r.resources("q6", e)
	fmt.Println("\nNext: q6-remove")
	return nil
}

func cmdQ6Remove(r *run, _ []string) error {
	a, err := r.admin()
	if err != nil {
		return err
	}
	mid, err := r.machine()
	if err != nil {
		return err
	}
	e, err := r.member("E")
	if err != nil {
		return err
	}
	del := r.call("q6-remove:home-user-E", as(a), "DELETE", webHost+"/api/home/users/"+e.ID, nil)
	r.finding("q6-remove", "home-remove", map[string]any{"status": del.status, "error": errText(del)})
	r.listing("q6-remove", "home-users on plex.tv", as(a), webHost+"/api/home/users", e, "")
	r.listing("q6-remove", "shared-servers-v1 on plex.tv", as(a), webHost+"/api/servers/"+mid+"/shared_servers", e, r.st.Shares["E"])
	r.listing("q6-remove", "users on plex.tv", as(a), webHost+"/api/users/", e, r.st.Shares["E"])
	r.resources("q6-remove", e)
	fmt.Println("\nmember-sees-server false means removing a Home user ends its shares. Next: q6-full, or q7")
	return nil
}

// cmdQ6Full fills A's Home with managed users until plex.tv refuses, then
// asks it to take C as well, then deletes the managed users.
func cmdQ6Full(r *run, _ []string) error {
	a, err := r.admin()
	if err != nil {
		return err
	}
	c, err := r.member("C")
	if err != nil {
		return err
	}
	made := 0
	for i := 1; i <= 16; i++ {
		t := fmt.Sprintf("Member access filler %d", i)
		rp := r.call("q6-full:managed-user", as(a), "POST", webHost+"/api/home/users?title="+url.QueryEscape(t), nil)
		if !rp.ok() {
			r.finding("q6-full", "home-full", map[string]any{"managed_made": made, "status": rp.status, "error": errText(rp)})
			break
		}
		id := ""
		walk(rp.tree, func(m map[string]any) {
			if id == "" && (get(m, "_tag") == "User" || get(m, "title") == t) {
				id = get(m, "id")
			}
		})
		if id != "" {
			r.st.Managed = append(r.st.Managed, id)
		}
		made++
	}
	r.listing("q6-full", "home-users on plex.tv", as(a), webHost+"/api/home/users", c, "")
	r.homeInvite("q6-full:home-invite-C-when-full", a, c)
	var left []string
	for _, id := range r.st.Managed {
		rp := r.call("q6-full:managed-user-delete", as(a), "DELETE", webHost+"/api/home/users/"+id, nil)
		if !rp.ok() && rp.status != http.StatusNotFound {
			left = append(left, id)
		}
	}
	r.st.Managed = left
	r.finding("q6-full", "managed-cleanup", map[string]any{"left": len(left)})
	return nil
}

func cmdQ7(r *run, _ []string) error {
	a, err := r.admin()
	if err != nil {
		return err
	}
	mid, err := r.machine()
	if err != nil {
		return err
	}
	b, err := r.member("B")
	if err != nil {
		return err
	}
	c, err := r.member("C")
	if err != nil {
		return err
	}
	cs, bs := r.st.Shares["C"], r.st.Shares["B"]
	if cs == "" || bs == "" {
		return errors.New("B or C has no share: run q1 and q5 first")
	}
	path := "/api/servers/" + mid + "/shared_servers/"
	d1 := r.call("q7:delete-pending-C on clients", as(a), "DELETE", apiHost+path+cs, nil)
	r.finding("q7", "delete-before-accept", map[string]any{"via": "clients.plex.tv", "status": d1.status, "error": errText(d1)})
	if !d1.ok() {
		d2 := r.call("q7:delete-pending-C on plex.tv", as(a), "DELETE", webHost+path+cs, nil)
		r.finding("q7", "delete-before-accept", map[string]any{"via": "plex.tv", "status": d2.status, "error": errText(d2)})
	}
	_, h1 := r.listing("q7", "shared-servers-v1 on plex.tv(C)", as(a), webHost+"/api/servers/"+mid+"/shared_servers", c, cs)
	_, h2 := r.listing("q7", "invites-requested on plex.tv(C)", as(a), webHost+"/api/invites/requested", c, cs)
	r.listing("q7", "pending-v2 on clients(C)", as(c), apiHost+pendingPath, a, cs)
	if len(h1)+len(h2) > 0 {
		for _, inv := range h2 {
			q := url.Values{"friend": {get(inv, "friend")}, "server": {get(inv, "server")}, "home": {get(inv, "home")}}
			cn := r.call("q7:cancel-invite-C", as(a), "DELETE", webHost+"/api/invites/requested/"+get(inv, "id")+"?"+q.Encode(), nil)
			r.finding("q7", "cancel-invite", map[string]any{"status": cn.status, "error": errText(cn)})
		}
		r.listing("q7", "shared-servers-v1-after-cancel on plex.tv(C)", as(a), webHost+"/api/servers/"+mid+"/shared_servers", c, cs)
		r.listing("q7", "invites-requested-after-cancel on plex.tv(C)", as(a), webHost+"/api/invites/requested", c, cs)
		r.listing("q7", "pending-v2-after-cancel on clients(C)", as(c), apiHost+pendingPath, a, cs)
	}
	r.listing("q7", "users on plex.tv(C-still-friend)", as(a), webHost+"/api/users/", c, "")
	r.listing("q7", "friends-v2 on clients(C-still-friend)", as(a), apiHost+"/api/v2/friends", c, "")
	d3 := r.call("q7:delete-accepted-B on plex.tv", as(a), "DELETE", webHost+path+bs, nil)
	r.finding("q7", "delete-after-accept", map[string]any{"status": d3.status, "error": errText(d3)})
	r.listing("q7", "shared-servers-v1 on plex.tv(B)", as(a), webHost+"/api/servers/"+mid+"/shared_servers", b, bs)
	r.listing("q7", "users on plex.tv(B-still-friend)", as(a), webHost+"/api/users/", b, "")
	r.listing("q7", "friends-v2 on clients(B-still-friend)", as(a), apiHost+"/api/v2/friends", b, "")
	r.resources("q7", b)
	fmt.Println("\nNext: cleanup (B and B2 tokens stay for q8)")
	return nil
}

func cmdQ8(r *run, args []string) error {
	fs := flag.NewFlagSet("q8", flag.ExitOnError)
	force := fs.Bool("force", false, "run before eight days have passed")
	_ = fs.Parse(args)
	b := r.st.Accounts["B"]
	if b == nil || b.Token == "" || b.SignedOut {
		return errors.New("B has no live token: q8 already ran, or B was never imported")
	}
	days := time.Since(b.TokenAt).Hours() / 24
	if days < 8 && !*force {
		return fmt.Errorf("B's token is %.1f days old: run q8 on or after %s", days, b.TokenAt.Add(8*24*time.Hour).Local().Format(time.RFC1123))
	}
	slots := []*account{b}
	if b2 := r.st.Accounts["B2"]; b2 != nil && b2.Token != "" {
		slots = append(slots, b2)
	}
	for _, a := range slots {
		v := map[string]any{"slot": a.Slot, "days": fmt.Sprintf("%.1f", time.Since(a.TokenAt).Hours()/24)}
		for _, h := range []struct{ name, base string }{{"plex.tv", webHost}, {"clients.plex.tv", apiHost}} {
			rp := r.call("q8:user-"+a.Slot+" on "+h.name, as(a), "GET", h.base+"/api/v2/user", nil)
			v["user_on_"+h.name] = rp.status
		}
		r.finding("q8", "token-after-days", v)
	}
	after := r.signout("q8", b)
	if len(slots) > 1 {
		chk := r.call("q8:user-B2-after-B-signout", as(slots[1]), "GET", apiHost+"/api/v2/user", nil)
		r.finding("q8", "signout-scope", map[string]any{"B_after": after, "B2_after": chk.status})
		r.signout("q8", slots[1])
	}
	fmt.Println("\nNext: cleanup -all, then export")
	return nil
}

func cmdCleanup(r *run, args []string) error {
	fs := flag.NewFlagSet("cleanup", flag.ExitOnError)
	all := fs.Bool("all", false, "also sign out B and B2 (normally kept for q8)")
	keepAdmin := fs.Bool("keep-admin", false, "leave A's JWT signed in")
	_ = fs.Parse(args)
	ours := map[string]bool{}
	for _, a := range r.st.Accounts {
		if a.Who != "A" && a.ID != "" {
			ours[a.ID] = true
		}
	}
	for _, id := range r.st.Managed {
		ours[id] = true
	}
	a, aerr := r.admin()
	if aerr == nil {
		r.cleanupAdmin(a, ours)
	} else {
		fmt.Println("A is signed out: skipping the Admin-side cleanup")
	}
	for _, s := range []string{"AL", "C", "C2", "D", "D2", "E", "E2", "B", "B2"} {
		m := r.st.Accounts[s]
		if m == nil || m.Token == "" || m.SignedOut {
			continue
		}
		if (s == "B" || s == "B2") && !*all {
			fmt.Printf("keeping %s's token for q8\n", s)
			continue
		}
		r.signout("cleanup", m)
	}
	if aerr == nil && !*keepAdmin {
		r.signout("cleanup", a)
	}
	fmt.Printf(`
Stop and delete the server:

docker rm -f %[1]s
docker volume rm %[1]s-config %[1]s-transcode
rm -rf '%[2]s'

Delete accounts A, C, D and E by hand at https://app.plex.tv (Account > Delete account) when the run is
written up. B waits for q8.
`, container, filepath.Join(r.dir, "media"))
	return nil
}

func (r *run) cleanupAdmin(a *account, ours map[string]bool) {
	mid := r.st.MachineID
	ok := func(rp reply) bool { return rp.ok() || rp.status == http.StatusNotFound }
	if mid != "" {
		path := "/api/servers/" + mid + "/shared_servers"
		ss := r.call("cleanup:shared-servers", as(a), "GET", webHost+path, nil)
		var shares []string
		for _, s := range tagged(ss.tree, "SharedServer") {
			shares = append(shares, get(s, "id"))
		}
		acc := r.call("cleanup:owned-accepted", as(a), "GET", apiHost+"/api/v2/shared_servers/owned/accepted", nil)
		for _, it := range items(acc.tree) {
			if get(it, "machineIdentifier") == mid && !contains(shares, get(it, "id")) {
				shares = append(shares, get(it, "id"))
			}
		}
		for _, id := range shares {
			rp := r.call("cleanup:delete-share", as(a), "DELETE", webHost+path+"/"+id, nil)
			r.finding("cleanup", "delete-share", map[string]any{"id": id, "status": rp.status, "done": ok(rp)})
		}
	}
	inv := r.call("cleanup:invites-requested", as(a), "GET", webHost+"/api/invites/requested", nil)
	for _, i := range tagged(inv.tree, "Invite") {
		q := url.Values{"friend": {get(i, "friend")}, "server": {get(i, "server")}, "home": {get(i, "home")}}
		rp := r.call("cleanup:cancel-invite", as(a), "DELETE", webHost+"/api/invites/requested/"+get(i, "id")+"?"+q.Encode(), nil)
		r.finding("cleanup", "cancel-invite", map[string]any{"status": rp.status, "done": ok(rp)})
	}
	home := r.call("cleanup:home-users", as(a), "GET", webHost+"/api/home/users", nil)
	for _, u := range tagged(home.tree, "User") {
		id := get(u, "id")
		if truthy(get(u, "admin")) || !ours[id] {
			continue
		}
		rp := r.call("cleanup:home-user-delete", as(a), "DELETE", webHost+"/api/home/users/"+id, nil)
		r.finding("cleanup", "home-user-delete", map[string]any{"id": id, "status": rp.status, "done": ok(rp)})
	}
	for _, w := range members {
		m := r.st.Accounts[string(w)]
		if m == nil || m.ID == "" {
			continue
		}
		rp := r.call("cleanup:remove-friend-"+m.Who, as(a), "DELETE", webHost+"/api/v2/sharings/"+m.ID, nil)
		if !ok(rp) {
			rp = r.call("cleanup:remove-friend-v1-"+m.Who, as(a), "DELETE", webHost+"/api/friends/"+m.ID, nil)
		}
		r.finding("cleanup", "remove-friend", map[string]any{"member": m.Who, "status": rp.status, "done": ok(rp)})
	}
	if mid != "" {
		dev := r.call("cleanup:devices", as(a), "GET", webHost+"/devices.xml", nil)
		for _, d := range tagged(dev.tree, "Device") {
			if get(d, "clientIdentifier") != mid {
				continue
			}
			rp := r.call("cleanup:server-device-delete", as(a), "DELETE", webHost+"/devices/"+get(d, "id")+".xml", nil)
			r.finding("cleanup", "server-device-delete", map[string]any{"status": rp.status, "done": ok(rp)})
		}
	}
	left := map[string]any{}
	if mid != "" {
		ss := r.call("cleanup:shared-servers-after", as(a), "GET", webHost+"/api/servers/"+mid+"/shared_servers", nil)
		left["shares"] = len(tagged(ss.tree, "SharedServer"))
	}
	inv = r.call("cleanup:invites-requested-after", as(a), "GET", webHost+"/api/invites/requested", nil)
	left["invites"] = len(tagged(inv.tree, "Invite"))
	home = r.call("cleanup:home-users-after", as(a), "GET", webHost+"/api/home/users", nil)
	n := 0
	for _, u := range tagged(home.tree, "User") {
		if !truthy(get(u, "admin")) {
			n++
		}
	}
	left["home_users"] = n
	users := r.call("cleanup:users-after", as(a), "GET", webHost+"/api/users/", nil)
	left["friends"] = len(tagged(users.tree, "User"))
	r.finding("cleanup", "left", left)
}

func cmdNote(r *run, args []string) error {
	if len(args) < 2 {
		return errors.New("note <step> <text...>")
	}
	r.finding(args[0], "note", strings.Join(args[1:], " "))
	return nil
}

func cmdStatus(r *run, _ []string) error {
	rd := r.redactor()
	fmt.Printf("state %s\n\n", r.dir)
	var slots []string
	for s := range r.st.Accounts {
		slots = append(slots, s)
	}
	sort.Strings(slots)
	for _, s := range slots {
		a := r.st.Accounts[s]
		st := "not signed in"
		switch {
		case a.Token != "" && a.SignedOut:
			st = "signed out"
		case a.Token != "":
			st = fmt.Sprintf("token from %s (%s, %.1f days old)", a.PinHost, map[bool]string{true: "JWT", false: "legacy"}[isJWT(a.Token)], time.Since(a.TokenAt).Hours()/24)
		}
		fmt.Printf("%-3s %-18s %-14s plex_pass=%-5v %s\n", s, rd.str(a.Username), rd.str(a.ID), a.PlexPass, st)
	}
	fmt.Printf("\nserver %s version %s section key %s plex.tv ids %v\n", rd.str(r.st.MachineID), r.st.ServerVersion, r.st.SectionKey, r.st.SectionIDs)
	for _, w := range members {
		if id := r.st.Shares[string(w)]; id != "" {
			fmt.Printf("share %s %s\n", string(w), rd.str(id))
		}
	}
	if b := r.st.Accounts["B"]; b != nil && b.Token != "" {
		fmt.Printf("\nq8 can run from %s\n", b.TokenAt.Add(8*24*time.Hour).Local().Format(time.RFC1123))
	}
	return nil
}

// cmdExport checks the log for any raw secret the state knows and for
// emails, JWTs, claim tokens and IPv4 addresses, then copies it out.
func cmdExport(r *run, args []string) error {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	date := fs.String("date", "", "run date, YYYY-MM-DD")
	out := fs.String("out", filepath.Join(r.dir, "out"), "directory to copy into")
	_ = fs.Parse(args)
	if _, err := time.Parse("2006-01-02", *date); err != nil {
		return errors.New("export needs -date YYYY-MM-DD")
	}
	if err := r.flush(); err != nil {
		return err
	}
	run, err := os.ReadFile(filepath.Join(r.dir, logName))
	if err != nil {
		return err
	}
	signin, err := r.redactFile(filepath.Join(r.dir, "signin.jsonl"))
	if err != nil {
		return err
	}
	raw := append(signin, run...)
	text := string(raw)
	var hits []string
	values := append([]secret(nil), r.st.Secrets...)
	for _, a := range r.st.Accounts {
		values = append(values, secret{Value: a.Token, Label: "token of " + a.Slot})
	}
	for _, s := range values {
		if len(s.Value) < 4 {
			continue
		}
		re := regexp.MustCompile(`(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(s.Value) + `($|[^A-Za-z0-9_])`)
		if n := len(re.FindAllStringIndex(text, -1)); n > 0 {
			hits = append(hits, fmt.Sprintf("%s appears raw %d times", s.Label, n))
		}
	}
	for _, sw := range sweeps {
		if n := len(sw.re.FindAllStringIndex(text, -1)); n > 0 {
			hits = append(hits, fmt.Sprintf("%d strings look like %s", n, sw.label))
		}
	}
	for _, m := range ipv4.FindAllString(text, -1) {
		if m != "127.0.0.1" && m != "0.0.0.0" {
			hits = append(hits, "an IPv4 address appears: "+m)
			break
		}
	}
	if len(hits) > 0 {
		return fmt.Errorf("the log is not clean:\n  %s", strings.Join(hits, "\n  "))
	}
	if err := os.MkdirAll(*out, 0o700); err != nil {
		return err
	}
	dst := filepath.Join(*out, "plex-members-run-"+*date+".jsonl")
	if err := os.WriteFile(dst, raw, 0o600); err != nil {
		return err
	}
	fmt.Printf("clean: %d lines copied to %s\n", strings.Count(text, "\n"), dst)
	return nil
}

// redactFile is a JSONL log written before this recorder, such as the
// sign-ins in signin.jsonl, with every line redacted like the recorder's
// own. A missing file is empty.
func (r *run) redactFile(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rd := r.redactor()
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		tree, kind := parse([]byte(line))
		m, ok := tree.(map[string]any)
		if kind != "json" || !ok {
			if err := enc.Encode(map[string]any{"unparsed_line": rd.str(line)}); err != nil {
				return nil, err
			}
			continue
		}
		u := ""
		for _, k := range []string{"url", "URL", "path"} {
			if s, ok := m[k].(string); ok {
				u = s
			}
		}
		out := map[string]any{}
		for k, v := range m {
			lk := strings.ToLower(k)
			if strings.HasSuffix(u, "/api/v2/user") && (strings.Contains(lk, "response") || lk == "body") {
				out[k] = rd.shape(k, v)
			} else {
				out[k] = rd.value(k, v)
			}
		}
		if err := enc.Encode(out); err != nil {
			return nil, err
		}
	}
	return b.Bytes(), nil
}

// accountsFile reads accounts.json, keyed by label A-E, whichever of an
// object of labels or an array of entries with a label field it is, and
// whatever spelling its field names use.
func accountsFile(path string) (map[string]map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	tree, kind := parse(raw)
	if kind != "json" {
		return nil, fmt.Errorf("%s isn't JSON", path)
	}
	norm := func(k string) string {
		var b strings.Builder
		for _, c := range strings.ToLower(k) {
			if c >= 'a' && c <= 'z' {
				b.WriteRune(c)
			}
		}
		switch n := b.String(); n {
		case "clientidentifier", "clientid", "xplexclientidentifier", "client":
			return "client"
		case "token", "authtoken", "legacytoken", "xplextoken":
			return "token"
		case "obtainedat", "tokenat", "at", "createdat", "signedinat", "time", "approvedat":
			return "at"
		default:
			return n
		}
	}
	entry := func(m map[string]any) map[string]string {
		e := map[string]string{}
		for k, v := range m {
			if s := sval(v); s != "" {
				e[norm(k)] = s
			}
		}
		return e
	}
	out := map[string]map[string]string{}
	var add func(v any)
	add = func(v any) {
		switch t := v.(type) {
		case []any:
			for _, m := range mapsIn(t) {
				e := entry(m)
				l := strings.ToUpper(e["label"] + e["slot"] + e["account"] + e["name"])
				if len(l) == 1 {
					out[l] = e
				}
			}
		case map[string]any:
			for k, x := range t {
				switch xv := x.(type) {
				case map[string]any:
					if len(k) == 1 {
						out[strings.ToUpper(k)] = entry(xv)
					}
				case []any:
					add(xv)
				}
			}
		}
	}
	add(tree)
	return out, nil
}

// cmdImport takes the Members' legacy sign-ins already made on the Mac,
// and A's legacy token as slot AL, then reads /api/v2/user with each.
func cmdImport(r *run, args []string) error {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	file := fs.String("file", filepath.Join(r.dir, "accounts.json"), "accounts file")
	tokenAt := fs.String("token-at", "", "when the tokens were issued, RFC 3339 (default: the file's own field, else its modification time)")
	_ = fs.Parse(args)
	entries, err := accountsFile(*file)
	if err != nil {
		return err
	}
	info, err := os.Stat(*file)
	if err != nil {
		return err
	}
	for _, l := range "A" + members {
		w := string(l)
		e := entries[w]
		if e == nil || e["token"] == "" || e["client"] == "" {
			fmt.Printf("%s: no token and client identifier in %s, skipped\n", w, *file)
			continue
		}
		name := w
		if w == "A" {
			name = "AL"
		}
		at := info.ModTime().UTC()
		for _, s := range []string{*tokenAt, e["at"]} {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				at = t.UTC()
				break
			}
		}
		a := &account{Slot: name, Who: w, PinHost: "clients.plex.tv", ClientID: e["client"], Token: e["token"], TokenAt: at}
		r.st.Accounts[name] = a
		r.label(a.ClientID, "client-id-"+name)
		r.label(a.Token, "token-"+name)
		r.label(e["email"], "email-"+w)
		r.label(e["username"], "username-"+w)
		if err := r.readUser("import-"+name, a, apiHost, false); err != nil {
			return err
		}
		r.finding("import-"+name, "imported", map[string]any{
			"slot": name, "is_jwt": isJWT(a.Token), "len": len(a.Token), "token_at": at.Format(time.RFC3339),
			"username_matches_file": e["username"] == "" || e["username"] == a.Username,
			"email_matches_file":    e["email"] == "" || e["email"] == a.Email,
		})
	}
	return nil
}
