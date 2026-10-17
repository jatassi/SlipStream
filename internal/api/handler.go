package api

import (
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
)

const (
	apiKeyHeader    = "X-Api-Key" //nolint:gosec // a header's name, not a credential
	setupCodeHeader = "X-Setup-Code"
	apiTree         = "/api/"
	notEmbedded     = "the web app is not embedded in this build; use the Vite dev server"
)

var securityHeaders = []struct{ name, value string }{
	{"Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'"},
	{"X-Content-Type-Options", "nosniff"},
	{"X-Frame-Options", "DENY"},
	{"Referrer-Policy", "no-referrer"},
	{"Cross-Origin-Opener-Policy", "same-origin"},
	{"Cross-Origin-Resource-Policy", "same-origin"},
	{"Permissions-Policy", "camera=(), microphone=(), geolocation=()"},
}

// Handler serves the operations under /api/v1, answers any other /api/
// path 404 not_found, and serves the web app everywhere else. Outermost
// first: security headers, CSRF, the mux. Each operation authenticates
// its caller in its own access check.
func (r *Router) Handler() http.Handler {
	r.mux.Handle(apiTree, NotFound)
	r.mux.Handle("/", webApp(r.services.Web))
	return wrapWriter(withSecurityHeaders(r.crossOriginProtection(r.mux)))
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		h := w.Header()
		for _, sh := range securityHeaders {
			h.Set(sh.name, sh.value)
		}
		if strings.HasPrefix(req.URL.Path, apiTree) {
			h.Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, req)
	})
}

// crossOriginProtection refuses an unsafe request another site's page
// sent. A request with X-Api-Key passes: it authenticates by the key alone,
// and no browser sends that header across origins without the CORS
// preflight SlipStream never answers.
func (r *Router) crossOriginProtection(next http.Handler) http.Handler {
	cop := http.NewCrossOriginProtection()
	for _, o := range r.services.TrustedOrigins {
		if err := cop.AddTrustedOrigin(o.String()); err != nil {
			panic("api: trusted origin " + o.String() + ": " + err.Error())
		}
	}
	cop.SetDenyHandler(CrossOrigin)
	protected := cop.Handler(next)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get(apiKeyHeader) != "" {
			next.ServeHTTP(w, req)
			return
		}
		protected.ServeHTTP(w, req)
	})
}

// caller resolves req's Caller once its operation is known. A public
// operation's caller presents nothing, so it never reads or slides a
// session, whatever cookie or key the request carries. Neither does any
// caller of a Node whose library.db is closed, which holds no session or
// API key to match.
func (r *Router) caller(w http.ResponseWriter, req *http.Request, authenticates bool) (identity.Caller, bool) {
	addr, https := proxies(r.services.TrustedProxies).clientOf(req)
	creds := identity.Request{Address: addr, HTTPS: https, Host: req.Host, UserAgent: req.UserAgent()}
	if !authenticates {
		return creds.Unauthenticated(), true
	}
	ids, open := r.services.Node.Identity()
	if !open {
		return creds.Unauthenticated(), true
	}
	creds.APIKey = req.Header.Get(apiKeyHeader)
	if c, err := req.Cookie(identity.CookieName); err == nil {
		creds.Session = c.Value
	}
	caller, cookie, err := ids.Authenticate(req.Context(), creds)
	if err != nil {
		r.logFailure(req.Context(), "authentication failed", "method", req.Method, "path", req.URL.Path, "error", err)
		writeProblem(w, Internal.problem())
		return identity.Caller{}, false
	}
	if cookie != nil {
		http.SetCookie(w, cookie)
	}
	return caller, true
}

// contentTypes covers the extensions Go's built-in MIME table lacks, which
// would otherwise depend on the host's MIME files.
var contentTypes = map[string]string{".webmanifest": "application/manifest+json"}

// webApp serves dist with an index.html fallback on GET and HEAD. Vite
// names everything under assets/ by its content hash, so those files never
// change.
func webApp(dist fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet && req.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(path.Clean(req.URL.Path), "/")
		f, ok := openFile(dist, name)
		if !ok {
			name = "index.html"
			f, ok = openFile(dist, name)
		}
		w.Header().Set("Cache-Control", "no-cache")
		if !ok {
			http.Error(w, notEmbedded, http.StatusNotFound)
			return
		}
		defer func() { _ = f.Close() }()
		if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		if ct, ok := contentTypes[path.Ext(name)]; ok {
			w.Header().Set("Content-Type", ct)
		}
		http.ServeContent(w, req, name, time.Time{}, f)
	})
}

type seekFile interface {
	fs.File
	io.ReadSeeker
}

// openFile opens a regular file of dist; embed.FS files can seek.
func openFile(dist fs.FS, name string) (seekFile, bool) {
	f, err := dist.Open(name)
	if err != nil {
		return nil, false
	}
	info, err := f.Stat()
	sf, seeks := f.(seekFile)
	if err != nil || info.IsDir() || !seeks {
		_ = f.Close()
		return nil, false
	}
	return sf, true
}

// responseWriter is the writer every handler sees. It keeps the flushing
// and deadlines of net/http's writer: Huma's body read timeout sets the
// read deadline, and its sse package flushes each event and sets a write
// deadline per event, printing to stderr when it cannot.
type responseWriter struct {
	http.ResponseWriter
}

func wrapWriter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		next.ServeHTTP(responseWriter{w}, req)
	})
}

func (w responseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w responseWriter) Flush() { _ = http.NewResponseController(w.ResponseWriter).Flush() }

func (w responseWriter) SetReadDeadline(t time.Time) error {
	return http.NewResponseController(w.ResponseWriter).SetReadDeadline(t) //nolint:wrapcheck // Huma checks only that it fails
}

func (w responseWriter) SetWriteDeadline(t time.Time) error {
	return http.NewResponseController(w.ResponseWriter).SetWriteDeadline(t) //nolint:wrapcheck // Huma checks only that it fails
}
