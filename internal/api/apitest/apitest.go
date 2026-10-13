// Package apitest holds the access walk: every operation in the document,
// called as nobody, a Member, an Admin and the API key.
package apitest

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/identity"
)

// Credentials are the walk's callers. Member and Admin each return a fresh
// session token per call, so an operation that signs out can't sign the
// walk out.
type Credentials struct {
	Member func() string
	Admin  func() string
	APIKey string
}

// pass is any status but 401 and 403.
var pass = want{}

type want struct {
	status int
	code   string
}

func (w want) String() string {
	if w == pass {
		return "not 401 or 403"
	}
	return fmt.Sprintf("%d %s", w.status, w.code)
}

func (w want) met(status int, code string) bool {
	if w == pass {
		return status != http.StatusUnauthorized && status != http.StatusForbidden
	}
	return status == w.status && code == w.code
}

var callers = []string{"nobody", "member", "admin", "api key"}

var alreadyClaimed, workerNode = want{409, "already_claimed"}, want{409, "worker_node"}

// walk is what each caller, in callers' order, must get at each level. A
// setup operation is walked on a claimed Node.
var walk = map[string][]want{
	"public":  {pass, pass, pass, pass},
	"setup":   {alreadyClaimed, alreadyClaimed, alreadyClaimed, alreadyClaimed},
	"account": {{401, "signed_out"}, pass, pass, {403, "needs_account"}},
	"admin":   {{401, "signed_out"}, {403, "needs_admin"}, pass, pass},
}

// workerWalk is the walk on a Worker, which holds no Account: every Account
// and Admin check answers worker_node before reading a credential.
var workerWalk = map[string][]want{
	"public":  {pass, pass, pass, pass},
	"setup":   {alreadyClaimed, alreadyClaimed, alreadyClaimed, alreadyClaimed},
	"account": {workerNode, workerNode, workerNode, workerNode},
	"admin":   {workerNode, workerNode, workerNode, workerNode},
}

// madeUp are credentials no Node issued, which is all a Worker's caller can
// hold.
var madeUp = Credentials{ //nolint:gosec // credentials no Node issued
	Member: func() string { return "made-up-member-session" },
	Admin:  func() string { return "made-up-admin-session" },
	APIKey: "ssk_made-up",
}

const zeroUUID = "00000000-0000-0000-0000-000000000000"

var pathParam = regexp.MustCompile(`\{[^}]+\}`)

type document struct {
	Servers []struct {
		URL string `json:"url"`
	} `json:"servers"`
	Paths map[string]map[string]json.RawMessage `json:"paths"`
}

type operation struct {
	ID     string  `json:"operationId"`
	Access *string `json:"x-slipstream-access"`
}

var methods = map[string]string{
	"get": http.MethodGet, "head": http.MethodHead, "post": http.MethodPost, "put": http.MethodPut,
	"patch": http.MethodPatch, "delete": http.MethodDelete, "options": http.MethodOptions, "trace": http.MethodTrace,
}

// CheckAccess sends every operation in doc to handler as each caller, with
// path parameters set to the zero UUID and no body, which is enough because
// the gate answers before Huma parses anything. A stream is cut after its
// status line. It returns one line per breach.
func CheckAccess(handler http.Handler, doc []byte, c Credentials) []string {
	return checkAll(handler, doc, c, walk)
}

// CheckWorkerAccess is CheckAccess on a Worker, with made-up credentials.
func CheckWorkerAccess(handler http.Handler, doc []byte) []string {
	return checkAll(handler, doc, madeUp, workerWalk)
}

func checkAll(handler http.Handler, doc []byte, c Credentials, walk map[string][]want) []string {
	var d document
	if err := json.Unmarshal(doc, &d); err != nil {
		return []string{"the document is not JSON: " + err.Error()}
	}
	prefix := ""
	if len(d.Servers) > 0 {
		prefix = d.Servers[0].URL
	}
	var breaches []string
	for _, p := range slices.Sorted(maps.Keys(d.Paths)) {
		for _, m := range slices.Sorted(maps.Keys(d.Paths[p])) {
			method, ok := methods[m]
			if !ok {
				continue
			}
			var op operation
			if err := json.Unmarshal(d.Paths[p][m], &op); err != nil {
				return append(breaches, fmt.Sprintf("%s %s%s: %v", method, prefix, p, err))
			}
			target := prefix + pathParam.ReplaceAllString(p, zeroUUID)
			breaches = append(breaches, check(handler, method, prefix+p, target, op, c, walk)...)
		}
	}
	return breaches
}

func check(handler http.Handler, method, route, target string, op operation, c Credentials, walk map[string][]want) []string {
	if op.Access == nil {
		return []string{fmt.Sprintf("%s %s: operation %s has no x-slipstream-access", method, route, op.ID)}
	}
	wants, ok := walk[*op.Access]
	if !ok {
		return []string{fmt.Sprintf("%s %s: operation %s has x-slipstream-access %q, which is no level", method, route, op.ID, *op.Access)}
	}
	var breaches []string
	for i, who := range callers {
		status, code := send(handler, method, target, func(h http.Header) {
			switch who {
			case "member":
				h.Set("Cookie", identity.CookieName+"="+c.Member())
			case "admin":
				h.Set("Cookie", identity.CookieName+"="+c.Admin())
			case "api key":
				h.Set("X-Api-Key", c.APIKey)
			}
		})
		if !wants[i].met(status, code) {
			breaches = append(breaches, fmt.Sprintf("%s %s as %s: got %d %s, want %s", method, route, who, status, code, wants[i]))
		}
	}
	return breaches
}

func send(handler http.Handler, method, target string, credential func(http.Header)) (status int, code string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequestWithContext(ctx, method, target, http.NoBody)
	credential(req.Header)
	w := &cutWriter{ResponseRecorder: httptest.NewRecorder(), cut: cancel}
	handler.ServeHTTP(w, req)
	var p struct {
		Code string `json:"code"`
	}
	if strings.Contains(w.Header().Get("Content-Type"), "json") {
		_ = json.Unmarshal(w.Body.Bytes(), &p)
	}
	return w.Code, p.Code
}

// cutWriter ends a stream once its status line is written.
type cutWriter struct {
	*httptest.ResponseRecorder
	cut context.CancelFunc
}

func (w *cutWriter) WriteHeader(status int) {
	w.ResponseRecorder.WriteHeader(status)
	if strings.HasPrefix(w.Header().Get("Content-Type"), "text/event-stream") {
		w.cut()
	}
}

func (w *cutWriter) Flush() {
	w.WriteHeader(http.StatusOK)
	w.ResponseRecorder.Flush()
}
