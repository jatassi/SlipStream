package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"math"
	"net/http"
	"path"
	"reflect"
	"regexp"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/system"
)

const (
	// Prefix is where the operations live; Op paths are relative to it.
	Prefix = "/api/v1"
	// version is info.version, constant so a stamped and an unstamped build
	// produce the same document.
	version = "1"

	defaultBodyLimit = 2 << 20
	accessExtension  = "x-slipstream-access"
	schemaPrefix     = "#/components/schemas/"
)

// Services are what serving needs. The document needs none of them.
type Services struct {
	Node           Node
	Gate           *identity.Gate
	Web            fs.FS
	TrustedOrigins []Origin
	TrustedProxies []Proxy
	Log            *slog.Logger
}

// Node is the Node serving the API as the access checks read it, on every
// request: its claim, and its identity service once library.db is open.
type Node interface {
	Claim(context.Context) (system.Claim, error)
	Identity() (*identity.Service, bool)
}

// Router holds every registered operation. Owners register through Owner,
// then the Node serves Handler and surface renders Document.
type Router struct {
	services Services
	huma     huma.API
	mux      *http.ServeMux
	owners   map[string]bool
	ops      map[string]string
	// streams ends at EndStreams, and every stream's context with it.
	streams    context.Context //nolint:containedctx // the lifetime of every stream the Router serves
	endStreams context.CancelFunc
}

func NewRouter(s Services) *Router {
	mux := http.NewServeMux()
	registry := huma.NewMapRegistry(schemaPrefix, schemaName)
	h := huma.NewAPI(huma.Config{
		OpenAPI: &huma.OpenAPI{
			OpenAPI: "3.1.0",
			Info:    &huma.Info{Title: "SlipStream", Version: version},
			Servers: []*huma.Server{{URL: Prefix}},
			Components: &huma.Components{
				Schemas: registry,
				SecuritySchemes: map[string]*huma.SecurityScheme{
					schemeSession:   {Type: "apiKey", In: "cookie", Name: identity.CookieName},
					schemeAPIKey:    {Type: "apiKey", In: "header", Name: apiKeyHeader},
					schemeSetupCode: {Type: "apiKey", In: "header", Name: setupCodeHeader},
				},
			},
		},
		Formats:       huma.DefaultFormats,
		DefaultFormat: "application/json",
	}, humago.NewAdapter(mux, Prefix))
	registry.Schema(reflect.TypeFor[Problem](), true, "Problem")
	streams, endStreams := context.WithCancel(context.Background())
	return &Router{services: s, huma: h, mux: mux, owners: map[string]bool{}, ops: map[string]string{}, streams: streams, endStreams: endStreams}
}

var tagShape = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

// Owner is an owner's handle on the router. tag names the owner and is the
// tag of its operations; floor is the lowest level they may use.
func (r *Router) Owner(tag string, floor Floor) *Owner {
	if !tagShape.MatchString(tag) {
		panic(fmt.Sprintf("api: owner tag %q must match %s", tag, tagShape))
	}
	if floor.level() == nil {
		panic("api: owner " + tag + ": its floor is " + zeroLevel)
	}
	if r.owners[tag] {
		panic("api: owner tag " + tag + " is used by two owners")
	}
	r.owners[tag] = true
	return &Owner{router: r, tag: tag, floor: floor.level()}
}

type Owner struct {
	router *Router
	tag    string
	floor  *access
}

type Op struct {
	// ID is lowerCamelCase and verb-first: listNodes, getSystem.
	ID     string
	Method string
	// Path is relative to Prefix: /nodes/{id}.
	Path    string
	Summary string
	Status  int
	// Also are statuses beside Status that the handler selects through
	// its output's Status field. Each answers Status's body.
	Also []int
	// Errors are the owner's codes the handler may return, beyond its
	// level's codes, invalid and internal.
	Errors []*Code
	// MaxBody raises the 2 MiB body limit for an operation that needs more.
	MaxBody int64
	// Gzip compresses the response for a client that accepts gzip, for an
	// operation that answers megabytes.
	Gzip bool
}

var idShape = regexp.MustCompile(`^[a-z]+[A-Z][A-Za-z0-9]*$`)

type principalKey struct{}

// Register adds an operation. Its handler receives the principal its
// level admits. Register panics at start on an id that is not
// lowerCamelCase or is taken, a level below the owner's floor, an unlimited
// or overflowing body limit, a GET or HEAD with a request body, or an
// output embedding an unexported struct.
func Register[P, I, O any](o *Owner, level Level[P], op Op, handler func(context.Context, P, *I) (*O, error)) {
	r := o.router
	hop, declared := humaOperation(o, level, op, reflect.TypeFor[I]())
	mustDocumentEmbedded(reflect.TypeFor[O](), map[reflect.Type]bool{})
	documentAlso(r.huma.OpenAPI().Components.Schemas, &hop, op, reflect.TypeFor[O]())
	huma.Register(r.huma, hop, func(ctx context.Context, in *I) (out *O, err error) {
		p := principal[P](ctx, op)
		defer func() {
			if v := recover(); v != nil {
				r.logPanic(op, v)
				out, err = nil, Internal.problem()
			}
		}()
		out, err = handler(ctx, p, in)
		if err != nil {
			return nil, r.failure(ctx, op, declared, err)
		}
		return out, nil
	})
}

// humaOperation is op as Huma registers it, behind level's access check,
// and every code it may answer.
func humaOperation[P any](o *Owner, level Level[P], op Op, input reflect.Type) (huma.Operation, []*Code) {
	o.check(level.access, op, input)
	declared := slices.Concat(level.codes, op.Errors, []*Code{Internal})
	if input.NumField() > 0 {
		declared = append(declared, Invalid)
	}
	limit := op.MaxBody
	if limit == 0 {
		limit = defaultBodyLimit
	}
	middlewares := huma.Middlewares{admitMiddleware(o.router, level)}
	if op.Gzip {
		middlewares = append(middlewares, gzipMiddleware(o.router))
	}
	return huma.Operation{
		OperationID:   op.ID,
		Method:        op.Method,
		Path:          op.Path,
		Summary:       op.Summary,
		Tags:          []string{o.tag},
		DefaultStatus: op.Status,
		Security:      level.security,
		Extensions:    map[string]any{accessExtension: level.name},
		Responses:     errorResponses(o.router.huma.OpenAPI().Components.Schemas, declared),
		// Huma answers 413 when it reads exactly MaxBodyBytes, so the limit
		// itself is MaxBodyBytes-1.
		MaxBodyBytes: limit + 1,
		Middlewares:  middlewares,
	}, declared
}

// documentAlso adds each of op.Also to hop's responses with the body of
// output, which must carry the Status field that selects it.
func documentAlso(registry huma.Registry, hop *huma.Operation, op Op, output reflect.Type) {
	if len(op.Also) == 0 {
		return
	}
	body, hasBody := output.FieldByName("Body")
	if _, hasStatus := output.FieldByName("Status"); !hasStatus || !hasBody {
		panic(op.ID + ": an operation answering more than one status has an output with Status and Body")
	}
	schema := huma.SchemaFromField(registry, body, op.ID+"Response")
	for _, status := range op.Also {
		hop.Responses[strconv.Itoa(status)] = &huma.Response{
			Description: http.StatusText(status),
			Content:     map[string]*huma.MediaType{"application/json": {Schema: schema}},
		}
	}
}

func principal[P any](ctx context.Context, op Op) P {
	p, ok := ctx.Value(principalKey{}).(P)
	if !ok {
		panic("api: " + op.ID + " ran without its access check")
	}
	return p
}

func (r *Router) logPanic(op Op, v any) {
	r.services.Log.Error("operation panicked", "operation", op.ID, "method", op.Method, "path", Prefix+op.Path, "panic", v, "stack", string(debug.Stack()))
}

func (o *Owner) check(level *access, op Op, input reflect.Type) {
	r := o.router
	if level == nil {
		panic(op.ID + ": " + zeroLevel)
	}
	if !idShape.MatchString(op.ID) {
		panic(fmt.Sprintf("api: operation id %q is not lowerCamelCase and verb-first", op.ID))
	}
	route := fmt.Sprintf("%s %s%s by owner %s", op.Method, Prefix, op.Path, o.tag)
	if other, taken := r.ops[op.ID]; taken {
		panic(fmt.Sprintf("api: operation id %s is registered twice: %s and %s", op.ID, other, route))
	}
	r.ops[op.ID] = route
	if level.rank < o.floor.rank {
		var admitted []string
		for _, l := range levels[o.floor.rank:] {
			admitted = append(admitted, l.name)
		}
		panic(fmt.Sprintf("%s: owner %s admits %s operations only; level is %s", op.ID, o.tag, strings.Join(admitted, " and "), level.name))
	}
	if op.MaxBody < 0 {
		panic(op.ID + ": an operation's body limit is never unlimited")
	}
	if op.MaxBody == math.MaxInt64 {
		panic(op.ID + ": a body limit of math.MaxInt64 overflows; raise it only as far as the operation needs")
	}
	_, body := input.FieldByName("Body")
	_, raw := input.FieldByName("RawBody")
	if (op.Method == http.MethodGet || op.Method == http.MethodHead) && (body || raw) {
		panic(op.ID + ": a " + op.Method + " operation takes no request body")
	}
}

func admitMiddleware[P any](r *Router, level Level[P]) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		req, w := humago.Unwrap(ctx)
		if err := r.refusedByClaim(ctx.Context(), level.access); err != nil {
			code, ok := identityCode(err)
			if !ok {
				r.logFailure(ctx.Context(), "claim unknown", "method", req.Method, "path", req.URL.Path, "error", err)
				code = Internal
			}
			writeProblem(w, code.problem())
			return
		}
		c, ok := r.caller(w, req, level.authenticates)
		if !ok {
			return
		}
		err := level.admit(ctx.Context(), r.services.Gate, c, ctx.Header(setupCodeHeader), func(p P) {
			next(huma.WithValue(ctx, principalKey{}, p))
		})
		if err == nil {
			return
		}
		code, ok := identityCode(err)
		if !ok {
			panic(fmt.Sprintf("api: %s refused with %v, which maps to no code", ctx.Operation().OperationID, err))
		}
		writeProblem(w, code.problem())
	}
}

// refusedByClaim answers an Account or Admin check before any credential
// is read: an Unclaimed Node has no Account yet, and a Worker's live on its
// Controller.
func (r *Router) refusedByClaim(ctx context.Context, a *access) error {
	if !a.byClaim {
		return nil
	}
	claim, err := r.services.Node.Claim(ctx)
	switch {
	case err != nil:
		return err
	case claim == system.Unclaimed:
		return identity.ErrSignedOut
	case claim == system.Worker:
		return identity.ErrWorkerNode
	}
	return nil
}

// failure turns a handler's error into what the client sees. A code the
// operation declares answers as itself; anything else answers internal
// and logs the real error.
func (r *Router) failure(ctx context.Context, op Op, declared []*Code, err error) error {
	attrs := []any{"method", op.Method, "path", Prefix + op.Path}
	code, isCode := err.(*Code) //nolint:errorlint // a handler returns its code itself, never wrapped
	var e *explained
	if x, ok := err.(*explained); ok { //nolint:errorlint // as for a bare code
		code, isCode, e = x.code, true, x
	}
	switch {
	case isCode && slices.Contains(declared, code):
		return e.answer(code)
	case isCode:
		r.services.Log.Error("operation "+op.ID+" returned undeclared code "+code.name, attrs...)
	default:
		r.logFailure(ctx, "operation failed", append(attrs, "operation", op.ID, "error", err)...)
	}
	return Internal.problem()
}

// logFailure logs at error level, or at debug once ctx is done: a client
// that went away is no failure of the operation.
func (r *Router) logFailure(ctx context.Context, msg string, attrs ...any) {
	level := slog.LevelError
	if ctx.Err() != nil {
		level = slog.LevelDebug
	}
	r.services.Log.Log(ctx, level, msg, attrs...)
}

// errorResponses documents each status's codes, and the members they
// carry, so a client's generated types know exactly which codes an
// operation answers.
func errorResponses(registry huma.Registry, codes []*Code) map[string]*huma.Response {
	byStatus := map[int][]any{}
	carried := map[int]map[string]reflect.Type{}
	for _, c := range codes {
		if !slices.Contains(byStatus[c.status], any(c.name)) {
			byStatus[c.status] = append(byStatus[c.status], c.name)
		}
		if c.member == "" {
			continue
		}
		if carried[c.status] == nil {
			carried[c.status] = map[string]reflect.Type{}
		}
		if t, ok := carried[c.status][c.member]; ok && t != c.carries {
			panic(fmt.Sprintf("api: two %d codes carry %s as %s and %s", c.status, c.member, t, c.carries))
		}
		carried[c.status][c.member] = c.carries
	}
	responses := map[string]*huma.Response{}
	for _, status := range slices.Sorted(maps.Keys(byStatus)) {
		names := byStatus[status]
		slices.SortFunc(names, func(a, b any) int { return strings.Compare(fmt.Sprint(a), fmt.Sprint(b)) })
		properties := map[string]*huma.Schema{"code": {Type: "string", Enum: names}}
		for member, t := range carried[status] {
			properties[member] = registry.Schema(t, true, "")
		}
		responses[strconv.Itoa(status)] = &huma.Response{
			Description: http.StatusText(status),
			Content: map[string]*huma.MediaType{problemType: {Schema: &huma.Schema{AllOf: []*huma.Schema{
				{Ref: schemaPrefix + "Problem"},
				{Type: "object", Properties: properties},
			}}}},
		}
	}
	return responses
}

// schemaName prefixes Huma's name with the owner a type belongs to, read
// from its package with any api suffix dropped: identityapi.Me becomes
// IdentityMe. Two owners' types never share a name.
func schemaName(t reflect.Type, hint string) string {
	name := huma.DefaultSchemaNamer(t, hint)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Name() == "" || t.PkgPath() == "" {
		return name
	}
	owner := strings.TrimSuffix(path.Base(t.PkgPath()), "api")
	if owner == "" {
		return name
	}
	return strings.ToUpper(owner[:1]) + owner[1:] + name
}

// Document is the OpenAPI document: OpenAPI 3.1 as indented JSON with a
// trailing newline, the same bytes on every call.
func (r *Router) Document() []byte {
	b, err := json.MarshalIndent(r.huma.OpenAPI(), "", "  ")
	if err != nil {
		panic(fmt.Sprintf("api: render the OpenAPI document: %v", err))
	}
	return append(b, '\n')
}
